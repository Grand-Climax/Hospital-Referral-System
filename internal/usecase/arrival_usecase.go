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

	if queue.AppointmentDate == nil {
		return errors.New("no scheduled appointment")
	}

	if queue.AppointmentDate.Truncate(24 * time.Hour).Unix() != time.Now().Truncate(24 * time.Hour).Unix() {
		return errors.New("can only arrive on scheduled date")
	}

	queue.ArrivalStatus = entity.ArrivalArrived
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

	if queue.ArrivalStatus != entity.ArrivalArrived && queue.ArrivalStatus != entity.ArrivalAdmitted {
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


func (u *arrivalUseCase) MarkMissed(ctx context.Context, queueID uuid.UUID, missReason entity.MissReason, userID uuid.UUID) error {
	queue, err := u.triageRepo.FindByID(ctx, queueID)
	if err != nil {
		return err
	}

	if queue.ArrivalStatus == entity.ArrivalMissed {
		return errors.New("appointment already marked as missed")
	}
	if queue.ArrivalStatus != entity.ArrivalExpected {
		return errors.New("cannot mark missed: patient is not in expected state")
	}
	if queue.AppointmentDate != nil && queue.AppointmentDate.After(time.Now()) {
		return errors.New("cannot mark a future appointment as missed")
	}

	queue.ArrivalStatus = entity.ArrivalMissed
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
