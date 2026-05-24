package usecase

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"Hospital-Referral-System/internal/delivery/http/dto"
	"Hospital-Referral-System/internal/domain/entity"
	irepository "Hospital-Referral-System/internal/domain/interfaces/repository"
	iusecase "Hospital-Referral-System/internal/domain/interfaces/usecase"
)

// ErrOverrideExists is returned by CreateOverride when an active override
// is already present for the (hospital, department, date) tuple. The
// caller (handler) should surface this as HTTP 409.
var ErrOverrideExists = errors.New("active capacity override already exists for this date; delete it first to recreate")

// ErrOverrideInsideBuffer is returned by CreateOverride when the target
// date is before today + buffer_days + 1. Surfaced as HTTP 400.
var ErrOverrideInsideBuffer = errors.New("capacity override target date is within the booking buffer; must be at least buffer_days+1 days in the future")

type capacityManagementUseCase struct {
	scheduleRepo irepository.DailyScheduleRepository
	overrideRepo irepository.CapacityOverrideRepository
	deptRepo     irepository.DepartmentRepository
	configRepo   irepository.SystemConfigRepository
	auditRepo    irepository.AuditLogRepository
	inAppNotifUC iusecase.InAppNotificationUseCase
	triageRepo   irepository.TriageQueueRepository
	schedulingUC iusecase.SchedulingUseCase
}

// NewCapacityManagementUseCase wires the capacity manager. schedulingUC
// is required so capacity-detail / calendar views can reuse
// EffectiveCapacity instead of duplicating the SQL; triageRepo is
// required for the staff_assigned and scheduled-patients views.
func NewCapacityManagementUseCase(
	sRepo irepository.DailyScheduleRepository,
	ovRepo irepository.CapacityOverrideRepository,
	deptRepo irepository.DepartmentRepository,
	configRepo irepository.SystemConfigRepository,
	auditRepo irepository.AuditLogRepository,
	inAppNotifUC iusecase.InAppNotificationUseCase,
	triageRepo irepository.TriageQueueRepository,
	schedulingUC iusecase.SchedulingUseCase,
) iusecase.CapacityManagementUseCase {
	return &capacityManagementUseCase{
		scheduleRepo: sRepo,
		overrideRepo: ovRepo,
		deptRepo:     deptRepo,
		configRepo:   configRepo,
		auditRepo:    auditRepo,
		inAppNotifUC: inAppNotifUC,
		triageRepo:   triageRepo,
		schedulingUC: schedulingUC,
	}
}

// GetSchedule returns the immutable history log for the inclusive range.
// No rows are created on this call.
func (u *capacityManagementUseCase) GetSchedule(ctx context.Context, hospitalID, deptID uuid.UUID, startDate, endDate time.Time) ([]entity.DailySchedule, error) {
	return u.scheduleRepo.FindByDeptAndDateRange(ctx, hospitalID, deptID, startDate, endDate)
}

// GetOverrides lists every override (active or inactive) for the dept.
func (u *capacityManagementUseCase) GetOverrides(ctx context.Context, hospitalID, deptID uuid.UUID) ([]entity.CapacityOverride, error) {
	return u.overrideRepo.ListByDept(ctx, hospitalID, deptID)
}

// ListOverridesByYearMonth narrows GetOverrides to a specific calendar
// window. year is required; month is optional (1-12; 0 = whole year).
func (u *capacityManagementUseCase) ListOverridesByYearMonth(ctx context.Context, hospitalID, deptID uuid.UUID, year, month int) ([]entity.CapacityOverride, error) {
	return u.overrideRepo.ListByDeptAndYearMonth(ctx, hospitalID, deptID, year, month)
}

// GetOverride fetches a single override by ID. Returns gorm.ErrRecordNotFound
// when the row is missing so the handler can translate it to a 404.
func (u *capacityManagementUseCase) GetOverride(ctx context.Context, overrideID uuid.UUID) (*entity.CapacityOverride, error) {
	return u.overrideRepo.FindByID(ctx, overrideID)
}

// CreateOverride enforces the Schedule-on-Demand override rules:
//  1. The target date must be at least buffer_days + 1 days in the future
//     so it cannot affect any patient already inside the booking window.
//  2. NewLimit must be >= 0.
//  3. No active override may already exist for that date (immutability:
//     delete the existing one first).
//
// On success a single row is inserted; DailySchedule is NOT touched - the
// override is read live by getEffectiveCapacity on the next booking.
func (u *capacityManagementUseCase) CreateOverride(ctx context.Context, hospitalID, deptID uuid.UUID, date time.Time, newLimit int, reason string, userID uuid.UUID) error {
	if newLimit < 0 {
		return errors.New("new limit must be 0 or greater")
	}

	if _, err := u.deptRepo.FindHospitalDepartment(ctx, hospitalID, deptID); err != nil {
		return err
	}

	bufferDays := 2
	if cfg, err := u.configRepo.GetByKey(ctx, "buffer_days"); err == nil && cfg != nil {
		if val, err := strconv.Atoi(cfg.Value); err == nil && val >= 0 {
			bufferDays = val
		}
	}

	earliest := time.Now().Truncate(24 * time.Hour).AddDate(0, 0, bufferDays+1)
	if date.Before(earliest) {
		return ErrOverrideInsideBuffer
	}

	existing, err := u.overrideRepo.GetActive(ctx, hospitalID, deptID, date)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	if existing != nil {
		return ErrOverrideExists
	}

	override := &entity.CapacityOverride{
		HospitalID:   hospitalID,
		DepartmentID: deptID,
		TargetDate:   date,
		NewLimit:     newLimit,
		Reason:       &reason,
		IsActive:     true,
		SetByID:      userID,
	}

	if err := u.overrideRepo.Create(ctx, override); err != nil {
		return err
	}

	_ = u.inAppNotifUC.CreateForEvent(ctx, "CAPACITY_OVERRIDE_CREATED", uuid.Nil, userID)

	return u.auditRepo.LogWithContext(ctx, userID, entity.ActionOverrideQueue, nil, nil, override)
}

// DeleteOverride deactivates the row (soft delete). Immutability is
// preserved by keeping the row but flipping IsActive = false so historical
// audits still show the value that was in effect.
func (u *capacityManagementUseCase) DeleteOverride(ctx context.Context, overrideID, userID uuid.UUID) error {
	override, err := u.overrideRepo.FindByID(ctx, overrideID)
	if err != nil {
		return err
	}

	override.IsActive = false
	if err := u.overrideRepo.Update(ctx, override); err != nil {
		return err
	}

	_ = u.inAppNotifUC.CreateForEvent(ctx, "CAPACITY_OVERRIDE_DELETED", uuid.Nil, userID)

	return u.auditRepo.LogWithContext(ctx, userID, entity.ActionOverrideQueue, nil, nil, map[string]interface{}{"deleted_id": overrideID})
}

// GetCapacityDetail composes the rich daily capacity view by reusing
// EffectiveCapacity (same source of truth used by the booking engine)
// and decorating it with staff hints from HospitalDepartment + the
// distinct-assigned-doctor count from the triage queue.
func (u *capacityManagementUseCase) GetCapacityDetail(ctx context.Context, hospitalID, deptID uuid.UUID, date time.Time) (*dto.CapacityDetailResponse, error) {
	maxSlots, overbook, booked, err := u.schedulingUC.EffectiveCapacity(ctx, hospitalID, deptID, date)
	if err != nil {
		return nil, err
	}

	staffAssigned, _ := u.triageRepo.CountAssignedDoctorsByDeptAndDate(ctx, hospitalID, deptID, date)

	staffCapacity := 0
	if dept, derr := u.deptRepo.FindHospitalDepartment(ctx, hospitalID, deptID); derr == nil && dept != nil {
		staffCapacity = dept.MaxCapacityOfStaff
	}

	hasOverride := false
	if ov, oerr := u.overrideRepo.GetActive(ctx, hospitalID, deptID, date); oerr == nil && ov != nil {
		hasOverride = true
	}

	available := maxSlots - int(booked)
	if available < 0 {
		available = 0
	}

	return &dto.CapacityDetailResponse{
		Date:           date.Format("2006-01-02"),
		MaxSlots:       maxSlots,
		OverbookLimit:  overbook,
		BookedSlots:    booked,
		StaffCapacity:  staffCapacity,
		StaffAssigned:  staffAssigned,
		AvailableSlots: available,
		IsFull:         booked >= int64(maxSlots+overbook),
		HasOverride:    hasOverride,
	}, nil
}

// GetScheduledPatientsForDate proxies to the triage repo using the same
// arrival-status set (Expected/Arrived/Admitted) used by the capacity
// counter, so the patient list and the booked-count agree.
func (u *capacityManagementUseCase) GetScheduledPatientsForDate(ctx context.Context, hospitalID, deptID uuid.UUID, date time.Time) ([]entity.TriageQueue, error) {
	return u.triageRepo.FindScheduledByDeptAndDate(ctx, hospitalID, deptID, date)
}

// BuildCapacityCalendar returns a per-day rollup for the given (year,
// month). Each day is computed via EffectiveCapacity; HasOverride and
// HasLog are decorated from the override and daily_schedules tables.
// Year is required; month must be in 1-12.
func (u *capacityManagementUseCase) BuildCapacityCalendar(ctx context.Context, hospitalID, deptID uuid.UUID, year, month int) ([]dto.CapacityCalendarDay, error) {
	if year <= 0 {
		return nil, errors.New("year is required")
	}
	if month < 1 || month > 12 {
		return nil, errors.New("month must be between 1 and 12")
	}

	firstOfMonth := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	nextMonth := firstOfMonth.AddDate(0, 1, 0)
	days := int(nextMonth.Sub(firstOfMonth).Hours() / 24)

	out := make([]dto.CapacityCalendarDay, 0, days)
	for i := 0; i < days; i++ {
		date := firstOfMonth.AddDate(0, 0, i)
		maxSlots, overbook, booked, err := u.schedulingUC.EffectiveCapacity(ctx, hospitalID, deptID, date)
		if err != nil {
			continue
		}
		available := maxSlots - int(booked)
		if available < 0 {
			available = 0
		}

		hasOverride := false
		if ov, oerr := u.overrideRepo.GetActive(ctx, hospitalID, deptID, date); oerr == nil && ov != nil {
			hasOverride = true
		}

		hasLog := false
		if logRow, lerr := u.scheduleRepo.FindByDeptAndDate(ctx, hospitalID, deptID, date); lerr == nil && logRow != nil {
			hasLog = true
		}

		out = append(out, dto.CapacityCalendarDay{
			Date:           date.Format("2006-01-02"),
			MaxSlots:       maxSlots,
			OverbookLimit:  overbook,
			BookedSlots:    booked,
			AvailableSlots: available,
			HasOverride:    hasOverride,
			HasLog:         hasLog,
		})
	}
	return out, nil
}

// UpdateStaffCapacity persists a soft staffing hint on HospitalDepartment.
// No booking decision references this value; it is surfaced only through
// GetCapacityDetail.staff_capacity for the UI to render warnings.
func (u *capacityManagementUseCase) UpdateStaffCapacity(ctx context.Context, hospitalID, deptID uuid.UUID, value int, userID uuid.UUID) error {
	if value < 0 {
		return errors.New("max_capacity_of_staff must be 0 or greater")
	}
	if _, err := u.deptRepo.FindHospitalDepartment(ctx, hospitalID, deptID); err != nil {
		return err
	}
	if err := u.deptRepo.UpdateStaffCapacity(ctx, hospitalID, deptID, value); err != nil {
		return err
	}

	_ = u.inAppNotifUC.CreateForEvent(ctx, "STAFF_CAPACITY_UPDATED", uuid.Nil, userID)

	return u.auditRepo.LogWithContext(ctx, userID, entity.ActionUpdateSystemConfig, nil, nil, map[string]interface{}{
		"hospital_id":           hospitalID,
		"department_id":         deptID,
		"max_capacity_of_staff": value,
		"note":                  fmt.Sprintf("staff capacity soft hint set to %d", value),
	})
}
