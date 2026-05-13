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

type schedulingUseCase struct {
	db           *gorm.DB
	referralRepo irepository.ReferralRepository
	triageRepo   irepository.TriageQueueRepository
	scheduleRepo irepository.DailyScheduleRepository
	overrideRepo irepository.CapacityOverrideRepository
	deptRepo     irepository.DepartmentRepository
	configRepo   irepository.SystemConfigRepository
	auditRepo    irepository.AuditLogRepository
	notifUC      iusecase.NotificationUseCase
	inAppNotifUC iusecase.InAppNotificationUseCase
}

func NewSchedulingUseCase(
	db *gorm.DB,
	rRepo irepository.ReferralRepository,
	tRepo irepository.TriageQueueRepository,
	sRepo irepository.DailyScheduleRepository,
	ovRepo irepository.CapacityOverrideRepository,
	deptRepo irepository.DepartmentRepository,
	configRepo irepository.SystemConfigRepository,
	auditRepo irepository.AuditLogRepository,
	notifUC iusecase.NotificationUseCase,
	inAppNotifUC iusecase.InAppNotificationUseCase,
) iusecase.SchedulingUseCase {
	return &schedulingUseCase{
		db:           db,
		referralRepo: rRepo,
		triageRepo:   tRepo,
		scheduleRepo: sRepo,
		overrideRepo: ovRepo,
		deptRepo:     deptRepo,
		configRepo:   configRepo,
		auditRepo:    auditRepo,
		notifUC:      notifUC,
		inAppNotifUC: inAppNotifUC,
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
		if err := u.triageRepo.Update(ctx, queue); err != nil {
			return err
		}

		ref.Status = entity.StatusScheduled
		if err := u.referralRepo.UpdateReferralTransaction(ctx, ref); err != nil {
			return err
		}

		if err := u.auditRepo.LogWithContext(ctx, userID, entity.ActionOverrideQueue, &referralID, nil, req); err != nil {
			return err
		}

		// Queue Notification
		hospitalName := "the hospital"
		deptName := "the department"
		if ref.ReceiverHospital != nil {
			hospitalName = ref.ReceiverHospital.Name
		}
		if ref.TargetDepartment != nil {
			deptName = ref.TargetDepartment.Name
		}

		notifType := entity.NotificationType("SCHEDULING")
		eventType := "APPOINTMENT_SCHEDULED"
		content := fmt.Sprintf("Your appointment at %s, %s is confirmed for %s.", hospitalName, deptName, req.AppointmentDate.Format("2006-01-02"))
		
		if queue.ArrivalStatus == entity.ArrivalMissed {
			notifType = entity.NotificationType("RESCHEDULE")
			eventType = "APPOINTMENT_RESCHEDULED"
			content = fmt.Sprintf("Your appointment at %s, %s has been rescheduled to %s.", hospitalName, deptName, req.AppointmentDate.Format("2006-01-02"))
			// Reset arrival status for rescheduled appointment
			queue.ArrivalStatus = entity.ArrivalExpected
			if err := u.triageRepo.Update(ctx, queue); err != nil {
				return err
			}
		}

		_ = u.notifUC.QueueNotification(ctx, referralID, notifType, content)

		_ = u.inAppNotifUC.CreateForEvent(ctx, eventType, referralID, userID)

		return nil
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
		HospitalID:   hospitalID,
		DepartmentID: deptID,
		TargetDate:   d,
		NewLimit:     newLimit,
		Reason:       &notes,
		IsActive:     true,
		SetByID:      userID,
	}

	if err := u.overrideRepo.Create(ctx, override); err != nil {
		return err
	}

	_ = u.inAppNotifUC.CreateForEvent(ctx, "CAPACITY_OVERRIDE_CREATED", uuid.Nil, userID)

	return u.auditRepo.LogWithContext(ctx, userID, entity.ActionOverrideQueue, nil, nil, override)
}

func (u *schedulingUseCase) ManualEmergencySchedule(ctx context.Context, referralID uuid.UUID, appointmentDate time.Time, justification string, userID uuid.UUID) error {
	ref, err := u.referralRepo.GetReferralByID(ctx, referralID)
	if err != nil {
		return err
	}

	// 1. Eligibility Check: Critical condition or explicit emergency intent
	isCritical := false
	if ref.ReferralForm != nil && ref.ReferralForm.ConditionAtReferral == "critical" {
		isCritical = true
	}

	if !isCritical && justification == "" {
		return errors.New("manual emergency schedule requires a critical condition or explicit justification")
	}

	queue, err := u.triageRepo.GetByReferralID(ctx, referralID)
	if err != nil {
		return err
	}

	// 2. Capacity Check: Allow overbooking up to (max_slots + overbook_limit)
	hospDept, err := u.deptRepo.FindHospitalDepartment(ctx, ref.TargetHospitalID, ref.TargetDeptID)
	if err != nil {
		return err
	}

	sched, err := u.scheduleRepo.GetOrCreate(ctx, ref.TargetHospitalID, ref.TargetDeptID, appointmentDate, hospDept.StandardDailyLimit)
	if err != nil {
		return err
	}

	if sched.BookedSlots >= (sched.MaxSlots + sched.OverbookLimit) {
		return errors.New("even overbook capacity is full for this date")
	}

	return u.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 3. Increment BookedSlots
		if err := u.scheduleRepo.IncrementBookedSlots(ctx, sched.ID, sched.Version); err != nil {
			return err
		}

		// 4. Update TriageQueue
		queue.AppointmentDate = &appointmentDate
		if err := tx.Save(queue).Error; err != nil {
			return err
		}

		// 5. Update Referral Status
		ref.Status = entity.StatusScheduled
		if err := tx.Save(ref).Error; err != nil {
			return err
		}

		if err := u.auditRepo.LogWithContext(ctx, userID, entity.ActionEmergencySchedule, &referralID, nil, map[string]interface{}{
			"appointment_date": appointmentDate.Format("2006-01-02"),
			"justification":    justification,
		}); err != nil {
			return err
		}

		// Queue Notification
		hospitalName := "the hospital"
		deptName := "the department"
		if ref.ReceiverHospital != nil {
			hospitalName = ref.ReceiverHospital.Name
		}
		if ref.TargetDepartment != nil {
			deptName = ref.TargetDepartment.Name
		}

		wasMissed := queue.ArrivalStatus == entity.ArrivalMissed

		// Reset arrival status if previously missed
		if wasMissed {
			queue.ArrivalStatus = entity.ArrivalExpected
		}

		notifType := entity.NotificationType("SCHEDULING")
		eventType := "APPOINTMENT_SCHEDULED"
		message := fmt.Sprintf("Your appointment at %s, %s is confirmed for %s.", hospitalName, deptName, appointmentDate.Format("2006-01-02"))

		if wasMissed {
			notifType = entity.NotificationType("RESCHEDULE")
			eventType = "APPOINTMENT_RESCHEDULED"
			message = fmt.Sprintf("Your appointment at %s, %s has been rescheduled to %s.", hospitalName, deptName, appointmentDate.Format("2006-01-02"))
		}

		_ = u.notifUC.QueueNotification(ctx, referralID, notifType, message)

		_ = u.inAppNotifUC.CreateForEvent(ctx, eventType, referralID, userID)

		return nil
	})
}

func (u *schedulingUseCase) BatchSchedule(ctx context.Context, hospitalID, departmentID, userID uuid.UUID, sendNotifications bool) (*dto.BatchScheduleResult, error) {
	waiting, err := u.triageRepo.FindWaitingByHospitalAndDept(ctx, hospitalID, departmentID)
	if err != nil {
		return nil, err
	}

	dept, err := u.deptRepo.FindHospitalDepartment(ctx, hospitalID, departmentID)
	if err != nil {
		return nil, err
	}

	// Load configuration
	bufferDays := 2
	if cfg, err := u.configRepo.GetByKey(ctx, "buffer_days"); err == nil {
		if val, err := strconv.Atoi(cfg.Value); err == nil {
			bufferDays = val
		}
	}

	horizonDays := 30
	if cfg, err := u.configRepo.GetByKey(ctx, "max_horizon_days"); err == nil {
		if val, err := strconv.Atoi(cfg.Value); err == nil {
			horizonDays = val
		}
	}

	startDate := time.Now().AddDate(0, 0, bufferDays)

	result := &dto.BatchScheduleResult{
		WaitingCount:   len(waiting),
		ScheduledCount: 0,
	}

	for _, q := range waiting {
		// Look for earliest slot
		for i := 0; i < horizonDays; i++ {
			targetDate := startDate.AddDate(0, 0, i)
			
			sched, err := u.scheduleRepo.GetOrCreate(ctx, hospitalID, departmentID, targetDate, dept.StandardDailyLimit)
			if err != nil {
				continue
			}

			// Batch rule: No overbooking
			if sched.BookedSlots < sched.MaxSlots {
				err := u.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
					// Use pessimistic or optimistic locking? Repository uses versioning.
					if err := u.scheduleRepo.IncrementBookedSlots(ctx, sched.ID, sched.Version); err != nil {
						return err
					}

					q.AppointmentDate = &targetDate
					q.ArrivalStatus = entity.ArrivalExpected
					if err := tx.Save(&q).Error; err != nil {
						return err
					}

					ref, err := u.referralRepo.GetReferralByID(ctx, q.ReferralID)
					if err != nil {
						return err
					}

					oldStatus := ref.Status
					ref.Status = entity.StatusScheduled
					if err := tx.Save(ref).Error; err != nil {
						return err
					}

					if err := u.referralRepo.CreateStatusHistory(ctx, &entity.ReferralStatusHistory{
						ReferralID:  q.ReferralID,
						ChangedByID: userID,
						FromStatus:  &oldStatus,
						ToStatus:    entity.StatusScheduled,
					}); err != nil {
						return err
					}

					// Queue Notification
					if sendNotifications {
						hospitalName := "the hospital"
						deptName := "the department"
						if dept.Hospital.Name != "" {
							hospitalName = dept.Hospital.Name
						}
						if dept.Department.Name != "" {
							deptName = dept.Department.Name
						}
						message := fmt.Sprintf("Your appointment at %s, %s is confirmed for %s.", hospitalName, deptName, targetDate.Format("2006-01-02"))
						_ = u.notifUC.QueueNotification(ctx, q.ReferralID, entity.NotificationType("SCHEDULING"), message)
					}

					return nil
				})

				if err == nil {
					result.ScheduledCount++
					result.WaitingCount--
					break
				}
			}
		}
	}

	u.auditRepo.LogWithContext(ctx, userID, entity.ActionBatchSchedule, nil, nil, map[string]interface{}{
		"hospital_id":   hospitalID,
		"department_id": departmentID,
		"result":        result,
	})

	if result.ScheduledCount > 0 {
		// Just use Nil UUID for referral if it's a batch event, or logic inside UseCase handles it
		_ = u.inAppNotifUC.CreateForEvent(ctx, "BATCH_SCHEDULE_COMPLETED", uuid.Nil, userID)
	}

	return result, nil
}

func (u *schedulingUseCase) getOrInitSchedule(ctx context.Context, hospitalID, deptID uuid.UUID, date time.Time) (*entity.DailySchedule, error) {
	dept, err := u.deptRepo.FindHospitalDepartment(ctx, hospitalID, deptID) 
	if err != nil {
		return nil, err
	}

	return u.scheduleRepo.GetOrCreate(ctx, hospitalID, deptID, date, dept.StandardDailyLimit)
}
