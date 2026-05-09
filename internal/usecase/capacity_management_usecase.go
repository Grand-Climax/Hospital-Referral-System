package usecase

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"Hospital-Referral-System/internal/delivery/http/dto"
	"Hospital-Referral-System/internal/domain/entity"
	irepository "Hospital-Referral-System/internal/domain/interfaces/repository"
	iusecase "Hospital-Referral-System/internal/domain/interfaces/usecase"
)

type capacityManagementUseCase struct {
	scheduleRepo irepository.DailyScheduleRepository
	overrideRepo irepository.CapacityOverrideRepository
	deptRepo     irepository.DepartmentRepository
	auditRepo    irepository.AuditLogRepository
	inAppNotifUC iusecase.InAppNotificationUseCase
}

func NewCapacityManagementUseCase(
	sRepo irepository.DailyScheduleRepository,
	ovRepo irepository.CapacityOverrideRepository,
	deptRepo irepository.DepartmentRepository,
	auditRepo irepository.AuditLogRepository,
	inAppNotifUC iusecase.InAppNotificationUseCase,
) iusecase.CapacityManagementUseCase {
	return &capacityManagementUseCase{
		scheduleRepo: sRepo,
		overrideRepo: ovRepo,
		deptRepo:     deptRepo,
		auditRepo:    auditRepo,
		inAppNotifUC: inAppNotifUC,
	}
}

func (u *capacityManagementUseCase) GetSchedule(ctx context.Context, hospitalID, deptID uuid.UUID, startDate, endDate time.Time) ([]entity.DailySchedule, error) {
	return u.scheduleRepo.FindByDeptAndDateRange(ctx, hospitalID, deptID, startDate, endDate)
}

func (u *capacityManagementUseCase) GetOverrides(ctx context.Context, hospitalID, deptID uuid.UUID) ([]entity.CapacityOverride, error) {
	return u.overrideRepo.ListByDept(ctx, hospitalID, deptID)
}

func (u *capacityManagementUseCase) CreateOverride(ctx context.Context, hospitalID, deptID uuid.UUID, date time.Time, newLimit int, reason string, userID uuid.UUID) error {
	if newLimit < 0 {
		return errors.New("new limit must be 0 or greater")
	}

	// Resolve the hospital_department link so we can populate deprecated FK fields safely.
	// Some databases enforce a FK constraint on dept_id even though it's marked deprecated.
	dept, err := u.deptRepo.FindHospitalDepartment(ctx, hospitalID, deptID)
	if err != nil {
		return err
	}

	override := &entity.CapacityOverride{
		HospitalID:   hospitalID,
		DepartmentID: deptID,
		DeptID:       dept.ID,
		TargetDate:   date,
		NewLimit:     newLimit,
		Reason:       &reason,
		IsActive:     true,
		SetByID:      userID,
	}

	if err := u.overrideRepo.Create(ctx, override); err != nil {
		return err
	}

	// Synchronize DailySchedule
	sched, err := u.scheduleRepo.GetOrCreate(ctx, hospitalID, deptID, date, dept.StandardDailyLimit)
	if err != nil {
		return err
	}

	sched.MaxSlots = newLimit
	if err := u.scheduleRepo.Update(ctx, sched); err != nil {
		return err
	}

	_ = u.inAppNotifUC.CreateForEvent(ctx, "CAPACITY_OVERRIDE_CREATED", uuid.Nil, userID)

	return u.auditRepo.LogWithContext(ctx, userID, entity.ActionOverrideQueue, nil, nil, override)
}

func (u *capacityManagementUseCase) UpdateOverride(ctx context.Context, overrideID uuid.UUID, newLimit int, reason string, userID uuid.UUID) error {
	override, err := u.overrideRepo.FindByID(ctx, overrideID)
	if err != nil {
		return err
	}

	override.NewLimit = newLimit
	override.Reason = &reason
	override.SetByID = userID

	if err := u.overrideRepo.Update(ctx, override); err != nil {
		return err
	}

	// Synchronize DailySchedule
	sched, err := u.scheduleRepo.GetByDeptAndDate(ctx, override.HospitalID, override.DepartmentID, override.TargetDate)
	if err == nil {
		sched.MaxSlots = newLimit
		_ = u.scheduleRepo.Update(ctx, sched)
	}

	_ = u.inAppNotifUC.CreateForEvent(ctx, "CAPACITY_OVERRIDE_UPDATED", uuid.Nil, userID)

	return u.auditRepo.LogWithContext(ctx, userID, entity.ActionOverrideQueue, nil, nil, override)
}

func (u *capacityManagementUseCase) DeleteOverride(ctx context.Context, overrideID, userID uuid.UUID) error {
	override, err := u.overrideRepo.FindByID(ctx, overrideID)
	if err != nil {
		return err
	}

	override.IsActive = false
	if err := u.overrideRepo.Update(ctx, override); err != nil {
		return err
	}

	// Revert DailySchedule to standard limit
	dept, err := u.deptRepo.FindHospitalDepartment(ctx, override.HospitalID, override.DepartmentID)
	if err == nil {
		sched, err := u.scheduleRepo.GetByDeptAndDate(ctx, override.HospitalID, override.DepartmentID, override.TargetDate)
		if err == nil {
			sched.MaxSlots = dept.StandardDailyLimit
			_ = u.scheduleRepo.Update(ctx, sched)
		}
	}

	return u.auditRepo.LogWithContext(ctx, userID, entity.ActionOverrideQueue, nil, nil, map[string]interface{}{"deleted_id": overrideID})
}

func (u *capacityManagementUseCase) UpdateMaxSlots(ctx context.Context, scheduleID uuid.UUID, maxSlots int, userID uuid.UUID) error {
	sched, err := u.scheduleRepo.FindByID(ctx, scheduleID)
	if err != nil {
		return err
	}

	oldSlots := sched.MaxSlots
	sched.MaxSlots = maxSlots
	if err := u.scheduleRepo.Update(ctx, sched); err != nil {
		return err
	}
	_ = oldSlots // could be used for audit

	return u.auditRepo.LogWithContext(ctx, userID, entity.ActionUpdateSystemConfig, nil, nil, map[string]interface{}{
		"schedule_id":   scheduleID,
		"hospital_id":   sched.HospitalID,
		"department_id": sched.DepartmentID,
		"date":          sched.ScheduleDate,
		"new_slots":     maxSlots,
	})
}

func (u *capacityManagementUseCase) ExtendSchedules(ctx context.Context) error {
	// 31st day calculation
	targetDate := time.Now().AddDate(0, 0, 31)

	// Fetch all hospital-department links (simplified iterate)
	links, err := u.deptRepo.ListHospitalDepartments(ctx, uuid.Nil) 
	if err != nil {
		return err
	}

	for _, link := range links {
		_, _ = u.scheduleRepo.GetOrCreate(ctx, link.HospitalID, link.DepartmentID, targetDate, link.StandardDailyLimit)
	}
	return nil
}

func (u *capacityManagementUseCase) BatchSchedule(ctx context.Context, hospitalID, deptID, userID uuid.UUID) (*dto.BatchScheduleResult, error) {
	// The core batch logic resides in SchedulingUseCase, but we expose it here if needed for Dept Head
	// For now, returning error to indicate it should be called via SchedulingUseCase if appropriate, 
	// or we can delegate if they are injected into each other.
	// However, the task says to implement it here if preferred. 
	// Let's assume CapacityManagement handles the capacity aspect of batching.
	return nil, errors.New("batch schedule should be called via SchedulingUseCase")
}
