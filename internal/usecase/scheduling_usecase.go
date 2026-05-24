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
	notifUC           iusecase.NotificationUseCase
	inAppNotifUC      iusecase.InAppNotificationUseCase
	jobCheckpointRepo irepository.JobCheckpointRepository
	clinicalRepo      irepository.ClinicalUpdateRepository
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
	jRepo irepository.JobCheckpointRepository,
	cRepo irepository.ClinicalUpdateRepository,
) iusecase.SchedulingUseCase {
	return &schedulingUseCase{
		db:           db,
		referralRepo: rRepo,
		triageRepo:   tRepo,
		scheduleRepo: sRepo,
		overrideRepo: ovRepo,
		deptRepo:     deptRepo,
		configRepo:        configRepo,
		auditRepo:         auditRepo,
		notifUC:           notifUC,
		inAppNotifUC:      inAppNotifUC,
		jobCheckpointRepo: jRepo,
		clinicalRepo:      cRepo,
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
		overbookLimit := u.getOverbookLimit(ctx, sched.OverbookLimit)
		resp = append(resp, dto.CapacityStatusResponse{
			Date:          date,
			TotalCapacity: sched.MaxSlots,
			OverbookLimit: overbookLimit,
			BookedSlots:   sched.BookedSlots,
			IsFull:        sched.BookedSlots >= (sched.MaxSlots + overbookLimit),
		})
	}
	return resp, nil
}

func (u *schedulingUseCase) ScheduleAppointment(ctx context.Context, referralID, userID uuid.UUID, req dto.SchedulingRequest) (bool, error) {
	ref, err := u.referralRepo.GetReferralByID(ctx, referralID)
	if err != nil {
		return false, err
	}

	if ref.Status != entity.StatusAccepted && ref.Status != entity.StatusScheduled {
		return false, errors.New("only accepted or scheduled referrals can be scheduled")
	}

	if req.AppointmentDate.Before(time.Now().Truncate(24 * time.Hour)) {
		return false, errors.New("appointment date cannot be in the past")
	}

	queue, err := u.triageRepo.GetByReferralID(ctx, referralID)
	if err != nil {
		return false, err
	}

	if queue.ArrivalStatus != entity.ArrivalExpected && queue.ArrivalStatus != entity.ArrivalMissed {
		return false, errors.New("cannot schedule appointment: patient has already arrived or been admitted")
	}

	wasMissed := queue.ArrivalStatus == entity.ArrivalMissed

	canSchedule, err := u.ValidateCapacity(ctx, ref.TargetHospitalID, ref.TargetDeptID, req.AppointmentDate.Format("2006-01-02"), req.Override)
	if err != nil || !canSchedule {
		return false, errors.New("capacity reached and no override granted")
	}

	err = u.db.Transaction(func(tx *gorm.DB) error {
		sched, err := u.getOrInitSchedule(ctx, ref.TargetHospitalID, ref.TargetDeptID, req.AppointmentDate)
		if err != nil {
			return err
		}

		if err := u.scheduleRepo.IncrementBookedSlots(ctx, sched.ID, sched.Version); err != nil {
			return err
		}

		if wasMissed {
			queue.ArrivalStatus = entity.ArrivalExpected
		}
		queue.AppointmentDate = &req.AppointmentDate
		if err := tx.Save(queue).Error; err != nil {
			return err
		}

		ref.Status = entity.StatusScheduled
		if err := tx.Save(ref).Error; err != nil {
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

		notifType := entity.NotifyScheduling
		eventType := "APPOINTMENT_SCHEDULED"
		content := fmt.Sprintf("Your appointment at %s, %s is confirmed for %s.", hospitalName, deptName, req.AppointmentDate.Format("2006-01-02"))
		
		if wasMissed {
			notifType = entity.NotifyMissedReschedule
			eventType = "MISSED_APPOINTMENT_RESCHEDULED"
			content = fmt.Sprintf("Your missed appointment at %s has been rescheduled to %s.", hospitalName, req.AppointmentDate.Format("2006-01-02"))
		}

		_ = u.notifUC.QueueNotification(ctx, referralID, notifType, content)

		_ = u.inAppNotifUC.CreateForEvent(ctx, eventType, referralID, userID)

		return nil
	})
	if err != nil {
		return false, err
	}

	return wasMissed, nil
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
		overbookLimit := u.getOverbookLimit(ctx, sched.OverbookLimit)
		// If emergency override is requested, allow up to MaxSlots + Overbook
		return sched.BookedSlots < (sched.MaxSlots + overbookLimit), nil
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

func (u *schedulingUseCase) ManualEmergencySchedule(ctx context.Context, referralID uuid.UUID, appointmentDate time.Time, justification string, userID uuid.UUID) (bool, error) {
	ref, err := u.referralRepo.GetReferralByID(ctx, referralID)
	if err != nil {
		return false, err
	}

	if ref.Status != entity.StatusAccepted && ref.Status != entity.StatusScheduled {
		return false, errors.New("only accepted or scheduled referrals can be emergency-scheduled")
	}

	if appointmentDate.Before(time.Now().Truncate(24 * time.Hour)) {
		return false, errors.New("appointment date cannot be in the past")
	}

	queue, err := u.triageRepo.GetByReferralID(ctx, referralID)
	if err != nil {
		return false, err
	}

	if queue.ArrivalStatus != entity.ArrivalExpected && queue.ArrivalStatus != entity.ArrivalMissed {
		return false, errors.New("cannot schedule appointment: patient has already arrived or been admitted")
	}

	if ref.MLSeverityScore == nil {
		defaultScore := 50.0
		ref.MLSeverityScore = &defaultScore
	}

	// 1. Eligibility Check: Critical condition or explicit emergency intent
	isCritical := false
	if ref.ReferralForm != nil && ref.ReferralForm.ConditionAtReferral == "critical" {
		isCritical = true
	}

	if !isCritical && justification == "" {
		return false, errors.New("manual emergency schedule requires a critical condition or explicit justification")
	}

	wasMissed := queue.ArrivalStatus == entity.ArrivalMissed

	// 2. Capacity Check: Allow overbooking up to (max_slots + overbook_limit)
	hospDept, err := u.deptRepo.FindHospitalDepartment(ctx, ref.TargetHospitalID, ref.TargetDeptID)
	if err != nil {
		return false, err
	}

	sched, err := u.scheduleRepo.GetOrCreate(ctx, ref.TargetHospitalID, ref.TargetDeptID, appointmentDate, hospDept.StandardDailyLimit)
	if err != nil {
		return false, err
	}

	overbookLimit := u.getOverbookLimit(ctx, sched.OverbookLimit)
	if sched.BookedSlots >= (sched.MaxSlots + overbookLimit) {
		return false, errors.New("even overbook capacity is full for this date")
	}

	err = u.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 3. Increment BookedSlots
		if err := u.scheduleRepo.IncrementBookedSlots(ctx, sched.ID, sched.Version); err != nil {
			return err
		}

		// 4. Update TriageQueue
		if wasMissed {
			queue.ArrivalStatus = entity.ArrivalExpected
		}
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

		notifType := entity.NotifyScheduling
		eventType := "APPOINTMENT_SCHEDULED"
		message := fmt.Sprintf("Your appointment at %s, %s is confirmed for %s.", hospitalName, deptName, appointmentDate.Format("2006-01-02"))

		if wasMissed {
			notifType = entity.NotifyMissedReschedule
			eventType = "MISSED_APPOINTMENT_RESCHEDULED"
			message = fmt.Sprintf("Your missed appointment at %s has been rescheduled to %s.", hospitalName, appointmentDate.Format("2006-01-02"))
		}

		_ = u.notifUC.QueueNotification(ctx, referralID, notifType, message)

		_ = u.inAppNotifUC.CreateForEvent(ctx, eventType, referralID, userID)

		return nil
	})
	if err != nil {
		return false, err
	}

	return wasMissed, nil
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
	if cfg, err := u.configRepo.GetByKey(ctx, "buffer_days"); err == nil && cfg != nil {
		if val, err := strconv.Atoi(cfg.Value); err == nil {
			bufferDays = val
		}
	}

	horizonDays := 30
	if cfg, err := u.configRepo.GetByKey(ctx, "max_horizon_days"); err == nil && cfg != nil {
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
						_ = u.notifUC.QueueNotification(ctx, q.ReferralID, entity.NotifyScheduling, message)
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

func (u *schedulingUseCase) getOverbookLimit(ctx context.Context, fallback int) int {
	if fallback > 0 {
		return fallback
	}
	if cfg, err := u.configRepo.GetByKey(ctx, "overbook_limit_default"); err == nil && cfg != nil {
		if val, err := strconv.Atoi(cfg.Value); err == nil && val >= 0 {
			return val
		}
	}
	return fallback
}

func (u *schedulingUseCase) ProcessMissedAppointments(ctx context.Context) error {
	enabled, err := u.configRepo.GetBool(ctx, "enable_cron_jobs", false)
	if err != nil || !enabled {
		return nil
	}

	today := time.Now().Truncate(24 * time.Hour)

	queues, err := u.triageRepo.FindMissedByDate(ctx, today)
	if err != nil {
		return err
	}

	for _, q := range queues {
		q.ArrivalStatus = entity.ArrivalMissed
		_ = u.triageRepo.Update(ctx, &q)

		_ = u.inAppNotifUC.CreateForEvent(ctx, "PATIENT_MISSED", q.ReferralID, uuid.Nil)
		
		_ = u.notifUC.QueueNotification(ctx, q.ReferralID, entity.NotifyReschedule, "You have missed your scheduled appointment. Your case has been flagged for review.")
	}

	if u.jobCheckpointRepo != nil {
		_ = u.jobCheckpointRepo.UpdateLastRun(ctx, "missed_appointments", time.Now())
	}

	return nil
}
