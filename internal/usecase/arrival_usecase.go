 package usecase

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"Hospital-Referral-System/internal/domain/entity"
	irepository "Hospital-Referral-System/internal/domain/interfaces/repository"
	iusecase "Hospital-Referral-System/internal/domain/interfaces/usecase"
)

type arrivalUseCase struct {
	db                 *gorm.DB
	triageRepo         irepository.TriageQueueRepository
	referralRepo       irepository.ReferralRepository
	userRepo           irepository.UserRepository
	referralAccessRepo irepository.ReferralAccessRepository
	clinicalRepo       irepository.ClinicalUpdateRepository
	auditRepo          irepository.AuditLogRepository
	inAppNotifUC       iusecase.InAppNotificationUseCase
}

func NewArrivalUseCase(
	db *gorm.DB,
	tRepo irepository.TriageQueueRepository,
	refRepo irepository.ReferralRepository,
	uRepo irepository.UserRepository,
	accessRepo irepository.ReferralAccessRepository,
	clinRepo irepository.ClinicalUpdateRepository,
	auditRepo irepository.AuditLogRepository,
	inAppNotifUC iusecase.InAppNotificationUseCase,
) iusecase.ArrivalUseCase {
	return &arrivalUseCase{
		db:                 db,
		triageRepo:         tRepo,
		referralRepo:       refRepo,
		userRepo:           uRepo,
		referralAccessRepo: accessRepo,
		clinicalRepo:       clinRepo,
		auditRepo:          auditRepo,
		inAppNotifUC:       inAppNotifUC,
	}
}

func (u *arrivalUseCase) GetTodayAndTomorrowSchedule(ctx context.Context, hospitalID, deptID uuid.UUID) ([]*entity.TriageQueue, error) {
	startDate := time.Now().Truncate(24 * time.Hour)
	endDate := startDate.AddDate(0, 0, 2)
	return u.triageRepo.FindScheduledByHospitalAndDept(ctx, hospitalID, deptID, startDate, endDate)
}

func (u *arrivalUseCase) ConfirmArrival(ctx context.Context, queueID uuid.UUID, userID uuid.UUID) error {
	queue, err := u.triageRepo.FindByID(ctx, queueID)
	if err != nil {
		return err
	}

	if queue.ArrivalStatus == entity.ArrivalArrived {
		return errors.New("patient already marked as arrived")
	}
	if queue.ArrivalStatus == entity.ArrivalMissed {
		return errors.New("patient already marked as missed")
	}

	queue.ArrivalStatus = entity.ArrivalArrived
	queue.QueueStatus = entity.QueueArrived
	now := time.Now()
	queue.ArrivedAt = &now
	queue.MarkedBy = &userID

	if err := u.triageRepo.Update(ctx, queue); err != nil {
		return err
	}

	u.auditRepo.LogWithContext(ctx, userID, entity.ActionConfirmArrival, &queue.ReferralID, nil, map[string]interface{}{
		"queue_id": queueID,
		"status":   "ARRIVED",
	})

	_ = u.inAppNotifUC.CreateForEvent(ctx, "PATIENT_ARRIVED", queue.ReferralID, userID)

	return nil
}

func (u *arrivalUseCase) AssignDoctor(ctx context.Context, queueID uuid.UUID, doctorID uuid.UUID, userID uuid.UUID) error {
	queue, err := u.triageRepo.FindByID(ctx, queueID)
	if err != nil {
		return err
	}

	if queue.AssignedDoctorID != nil {
		return errors.New("a doctor is already assigned to this patient")
	}

	doctor, err := u.userRepo.FindByID(ctx, doctorID)
	if err != nil {
		return errors.New("doctor not found")
	}

	if doctor.Role != entity.RoleReceivingSpecialist {
		return errors.New("selected user is not a receiving specialist")
	}

	if doctor.HospitalID == nil || *doctor.HospitalID != queue.HospitalID {
		return errors.New("doctor does not belong to the same hospital")
	}

	if queue.ArrivalStatus != entity.ArrivalArrived {
		return errors.New("cannot assign doctor before patient arrives")
	}

	now := time.Now()
	queue.AssignedDoctorID = &doctorID
	queue.DoctorAssignedAt = &now

	return u.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(queue).Error; err != nil {
			return err
		}

		access := &entity.ReferralAccess{
			ReferralID: queue.ReferralID,
			UserID:     doctorID,
			AccessType: "TREATING_DOCTOR",
			GrantedBy:  userID,
		}
		if err := tx.Create(access).Error; err != nil {
			return err
		}

		u.auditRepo.LogWithContext(ctx, userID, entity.ActionAssignDoctor, &queue.ReferralID, nil, map[string]interface{}{
			"queue_id":  queueID,
			"doctor_id": doctorID,
		})

		_ = u.inAppNotifUC.CreateForEvent(ctx, "DOCTOR_ASSIGNED", queue.ReferralID, userID)

		return nil
	})
}

func (u *arrivalUseCase) RegisterWalkIn(ctx context.Context, referralID uuid.UUID, hospitalID uuid.UUID, deptID uuid.UUID, userID uuid.UUID) (*entity.TriageQueue, error) {
	ref, err := u.referralRepo.FindByID(ctx, referralID)
	if err != nil {
		return nil, errors.New("referral not found")
	}
	var hospDept entity.HospitalDepartment
	if err := u.db.WithContext(ctx).Where("hospital_id = ? AND department_id = ?", hospitalID, deptID).First(&hospDept).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("department does not exist in this hospital")
		}
		return nil, err
	}

	// Allowed statuses for walk-in
	allowed := false
	for _, s := range []entity.ReferralStatus{entity.StatusAccepted, entity.StatusScheduled} {
		if ref.Status == s {
			allowed = true
			break
		}
	}
	if !allowed {
		return nil, errors.New("referral status does not allow walk-in registration")
	}

	// Calculate score: (ml_severity_score * 0.7) + (waiting_hours_weight * 0.3) + 20
	// For walk-in, we use current severity and 0 waiting weight (new in queue)
	mlScore := 0.0
	if ref.MLSeverityScore != nil {
		mlScore = *ref.MLSeverityScore
	}
	score := (mlScore * 0.7) + 20

	now := time.Now()
	queue := &entity.TriageQueue{
		ReferralID:      referralID,
		HospitalID:      hospitalID,
		DepartmentID:    deptID,
		CompositeScore:  score,
		QueueStatus:     entity.QueueArrived,
		ArrivalStatus:   entity.ArrivalArrived,
		ArrivalBoost:    20,
		AppointmentDate: nil,
		ArrivedAt:       &now,
		MarkedBy:        &userID,
		AssignedAt:      now,
	}

	queue.DeptID = hospDept.ID

	err = u.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(queue).Error; err != nil {
			return err
		}

		u.auditRepo.LogWithContext(ctx, userID, entity.ActionWalkInRegistered, &referralID, nil, map[string]interface{}{
			"hospital_id": hospitalID,
			"dept_id":     deptID,
			"score":       score,
		})
		return nil
	})

	return queue, err
}

func (u *arrivalUseCase) MarkMissed(ctx context.Context, queueID uuid.UUID, missReason entity.MissReason, userID uuid.UUID) error {
	queue, err := u.triageRepo.FindByID(ctx, queueID)
	if err != nil {
		return err
	}

	if queue.ArrivalStatus == entity.ArrivalMissed {
		return errors.New("appointment already marked as missed")
	}
	if queue.ArrivalStatus != entity.ArrivalExpected && queue.ArrivalStatus != entity.ArrivalArrived {
		return errors.New("cannot mark missed: patient is not in expected or arrived state")
	}
	if queue.AppointmentDate != nil && queue.AppointmentDate.After(time.Now()) {
		return errors.New("cannot mark a future appointment as missed")
	}

	queue.ArrivalStatus = entity.ArrivalMissed
	queue.QueueStatus = entity.QueueMissed
	queue.MissReason = &missReason

	return u.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(queue).Error; err != nil {
			return err
		}

		update := &entity.ClinicalUpdate{
			ReferralID:     queue.ReferralID,
			UpdatedByID:    userID,
			UpdateReason:   "MISSED_APPOINTMENT_RE_EVALUATION",
			ClinicalNotes:  "Missed appointment – flagged for re-evaluation",
			RequiresReview: true,
		}
		if err := tx.Create(update).Error; err != nil {
			return err
		}

		u.auditRepo.LogWithContext(ctx, userID, entity.ActionMarkMissed, &queue.ReferralID, nil, map[string]interface{}{
			"queue_id":    queueID,
			"miss_reason": missReason,
		})

		_ = u.inAppNotifUC.CreateForEvent(ctx, "PATIENT_MISSED", queue.ReferralID, userID)

		return nil
	})
}
