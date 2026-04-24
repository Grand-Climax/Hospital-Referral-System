package usecase

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"Hospital-Referral-System/internal/delivery/http/dto"
	"Hospital-Referral-System/internal/domain/entity"
	irepository "Hospital-Referral-System/internal/domain/interfaces/repository"
	iusecase "Hospital-Referral-System/internal/domain/interfaces/usecase"
)

type schedulingUseCase struct {
	db           *gorm.DB
	referralRepo irepository.ReferralRepository
	triageRepo   irepository.TriageQueueRepository
	scheduleRepo irepository.DailyScheduleRepository
	overrideRepo irepository.CapacityOverrideRepository
	deptRepo     irepository.DepartmentRepository
	auditRepo    irepository.AuditLogRepository
}

func NewSchedulingUseCase(
	db *gorm.DB,
	rRepo irepository.ReferralRepository,
	tRepo irepository.TriageQueueRepository,
	sRepo irepository.DailyScheduleRepository,
	ovRepo irepository.CapacityOverrideRepository,
	deptRepo irepository.DepartmentRepository,
	auditRepo irepository.AuditLogRepository,
) iusecase.SchedulingUseCase {
	return &schedulingUseCase{
		db:           db,
		referralRepo: rRepo,
		triageRepo:   tRepo,
		scheduleRepo: sRepo,
		overrideRepo: ovRepo,
		deptRepo:     deptRepo,
		auditRepo:    auditRepo,
	}
}

func (u *schedulingUseCase) GetCapacityStatus(ctx context.Context, hospitalID, deptID uuid.UUID, dateRangeDays int) ([]dto.CapacityStatusResponse, error) {
	var resp []dto.CapacityStatusResponse
	for i := 0; i < dateRangeDays; i++ {
		date := time.Now().AddDate(0, 0, i+1)
		sched, err := u.getOrInitSchedule(ctx, hospitalID, deptID, date)
		if err != nil {
			continue
		}
		resp = append(resp, dto.CapacityStatusResponse{
			Date:          date,
			TotalCapacity: sched.MaxSlots,
			OverbookLimit: sched.OverbookLimit,
			BookedSlots:   sched.BookedSlots,
			IsFull:        sched.BookedSlots >= (sched.MaxSlots + sched.OverbookLimit),
		})
	}
	return resp, nil
}

func (u *schedulingUseCase) ScheduleAppointment(ctx context.Context, referralID, userID uuid.UUID, req dto.SchedulingRequest) error {
	ref, err := u.referralRepo.GetReferralByID(ctx, referralID)
	if err != nil {
		return err
	}

	canSchedule, err := u.ValidateCapacity(ctx, ref.TargetHospitalID, ref.TargetDeptID, req.AppointmentDate.Format("2006-01-02"), req.Override)
	if err != nil || !canSchedule {
		return errors.New("capacity reached and no override granted")
	}

	return u.db.Transaction(func(tx *gorm.DB) error {
		sched, err := u.getOrInitSchedule(ctx, ref.TargetHospitalID, ref.TargetDeptID, req.AppointmentDate)
		if err != nil {
			return err
		}

		if err := u.scheduleRepo.IncrementBookedSlots(ctx, sched.ID, sched.Version); err != nil {
			return err
		}

		queue, err := u.triageRepo.GetByReferralID(ctx, referralID)
		if err != nil {
			return err
		}
		queue.AppointmentDate = &req.AppointmentDate
		queue.QueueStatus = entity.QueueScheduled
		if err := u.triageRepo.Update(ctx, queue); err != nil {
			return err
		}

		ref.Status = entity.StatusScheduled
		if err := u.referralRepo.UpdateReferralTransaction(ctx, ref); err != nil {
			return err
		}

		return u.auditRepo.LogWithContext(ctx, userID, entity.ActionOverrideQueue, &referralID, nil, req)
	})
}

func (u *schedulingUseCase) ValidateCapacity(ctx context.Context, hospitalID, deptID uuid.UUID, date string, override bool) (bool, error) {
	d, _ := time.Parse("2006-01-02", date)
	
	// Always ensure row exists with correct MaxSlots (synced with overrides)
	dept, err := u.deptRepo.FindHospitalDepartment(ctx, hospitalID, deptID)
	if err != nil {
		return false, err
	}

	sched, err := u.scheduleRepo.GetOrCreate(ctx, hospitalID, deptID, d, dept.StandardDailyLimit)
	if err != nil {
		return false, err
	}

	if override {
		// If emergency override is requested, allow up to MaxSlots + Overbook
		return sched.BookedSlots < (sched.MaxSlots + sched.OverbookLimit), nil
	}

	return sched.BookedSlots < sched.MaxSlots, nil
}

func (u *schedulingUseCase) ManageCapacityOverride(ctx context.Context, hospitalID, deptID, userID uuid.UUID, appointmentDate string, newLimit int, notes string) error {
	d, err := time.Parse("2006-01-02", appointmentDate)
	if err != nil {
		return errors.New("invalid date format: YYYY-MM-DD")
	}

	override := &entity.CapacityOverride{
		HospitalID: hospitalID,
		DeptID:     deptID,
		TargetDate: d,
		NewLimit:   newLimit,
		Reason:     &notes,
		IsActive:   true,
		SetByID:    userID,
	}

	if err := u.overrideRepo.Create(ctx, override); err != nil {
		return err
	}

	return u.auditRepo.LogWithContext(ctx, userID, entity.ActionOverrideQueue, nil, nil, override)
}

func (u *schedulingUseCase) ManualEmergencySchedule(ctx context.Context, referralID uuid.UUID, appointmentDate string, justification string, userID uuid.UUID) error {
	d, err := time.Parse("2006-01-02", appointmentDate)
	if err != nil {
		return err
	}

	queue, err := u.triageRepo.GetByReferralID(ctx, referralID)
	if err != nil {
		return err
	}

	// Look up hospital_id from hospital_departments (dept_id in queue refers to hospital_department link)
	hospDept, err := u.deptRepo.FindHospitalDepartmentByID(ctx, queue.DeptID)
	if err != nil {
		return err
	}

	hospID := hospDept.HospitalID
	dept := hospDept

	// Always ensure row exists
	sched, err := u.scheduleRepo.GetOrCreate(ctx, hospID, queue.DeptID, d, dept.StandardDailyLimit)
	if err != nil {
		return err
	}

	// Rule: Overbooking allowed (up to max + overbook)
	if sched.BookedSlots >= (sched.MaxSlots + sched.OverbookLimit) {
		return errors.New("even overbook capacity is full for this date")
	}

	return u.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Increment BookedSlots
		sched.BookedSlots++
		if err := tx.Save(sched).Error; err != nil {
			return err
		}

		// Update TriageQueue
		queue.AppointmentDate = &d
		queue.QueueStatus = entity.QueueScheduled
		reschedReason := "EMERGENCY_MANUAL"
		queue.RescheduleReason = &reschedReason 
		if err := tx.Save(queue).Error; err != nil {
			return err
		}

		// Referral record status update (optional but consistent)
		err := tx.Model(&entity.Referral{}).Where("id = ?", referralID).Update("status", entity.StatusScheduled).Error
		if err != nil {
			return err
		}

		return u.auditRepo.LogWithContext(ctx, userID, "MANUAL_EMERGENCY_SCHEDULE", nil, nil, map[string]interface{}{
			"referral_id": referralID,
			"date":        appointmentDate,
			"note":        justification,
		})
	})
}

func (u *schedulingUseCase) BatchSchedule(ctx context.Context, hospitalID, deptID, userID uuid.UUID) (*dto.BatchScheduleResult, error) {
	waiting, err := u.triageRepo.GetWaitingByDept(ctx, hospitalID, deptID)
	if err != nil {
		return nil, err
	}

	dept, err := u.deptRepo.FindHospitalDepartment(ctx, hospitalID, deptID)
	if err != nil {
		return nil, err
	}

	const bufferDays = 2 // As requested
	const horizonDays = 14
	startDate := time.Now().AddDate(0, 0, bufferDays)

	result := &dto.BatchScheduleResult{
		WaitingCount:   len(waiting),
		ScheduledCount: 0,
	}

	for _, q := range waiting {
		scheduled := false
		// Look for earliest slot
		for i := 0; i < horizonDays; i++ {
			targetDate := startDate.AddDate(0, 0, i)
			
			// Use GetOrCreate to ensure row exists
			sched, err := u.scheduleRepo.GetOrCreate(ctx, hospitalID, deptID, targetDate, dept.StandardDailyLimit)
			if err != nil {
				continue
			}

			// Batch rule: No overbooking
			if sched.BookedSlots < sched.MaxSlots {
				err := u.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
					// Refresh with lock or hope optimistic versioning suffices
					// Using tx.Save on sched should check version if properly implemented
					sched.BookedSlots++
					if err := tx.Save(sched).Error; err != nil {
						return err
					}

					q.AppointmentDate = &targetDate
					q.QueueStatus = entity.QueueScheduled
					if err := tx.Save(&q).Error; err != nil {
						return err
					}

					return tx.Model(&entity.Referral{}).Where("id = ?", q.ReferralID).Update("status", entity.StatusScheduled).Error
				})

				if err == nil {
					result.ScheduledCount++
					result.WaitingCount--
					scheduled = true
					break
				}
			}
		}
		if !scheduled {
			// Stay in waiting list
		}
	}

	u.auditRepo.LogWithContext(ctx, userID, "BATCH_SCHEDULE_RUN", nil, nil, result)
	return result, nil
}

func (u *schedulingUseCase) getOrInitSchedule(ctx context.Context, hospitalID, deptID uuid.UUID, date time.Time) (*entity.DailySchedule, error) {
	dept, err := u.deptRepo.FindHospitalDepartment(ctx, hospitalID, deptID) 
	if err != nil {
		return nil, err
	}

	return u.scheduleRepo.GetOrCreate(ctx, hospitalID, deptID, date, dept.StandardDailyLimit)
}
