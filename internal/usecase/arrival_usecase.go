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
	// notifUC is used to dispatch the patient-facing MISSED SMS when a
	// receptionist marks a no-show. We deliberately call it post-commit
	// so the SMS isn't sent for transactions that roll back.
	notifUC iusecase.NotificationUseCase
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
	notifUC iusecase.NotificationUseCase,
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
		notifUC:            notifUC,
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

func (u *arrivalUseCase) AssignDoctor(ctx context.Context, queueID uuid.UUID, doctorID uuid.UUID, userID uuid.UUID, reason string) error {
	queue, err := u.triageRepo.FindByID(ctx, queueID)
	if err != nil {
		return err
	}

	if queue.AssignedDoctorID != nil {
		cleanupReason := reason
		if cleanupReason == "" {
			cleanupReason = "Doctor reassigned by receptionist"
		}
		if err := u.referralAccessRepo.RevokeAllByReferral(ctx, queue.ReferralID, cleanupReason); err != nil {
			return err
		}
		// Clear assignment temporarily
		queue.AssignedDoctorID = nil
		queue.DoctorAssignedAt = nil
	}

	doctor, err := u.userRepo.FindByID(ctx, doctorID)
	if err != nil {
		return errors.New("doctor not found")
	}

	if doctor.Role != entity.RoleReferringDoctor {
		return errors.New("selected user is not a referring doctor")
	}

	if doctor.HospitalID == nil || *doctor.HospitalID != queue.HospitalID {
		return errors.New("doctor does not belong to the same hospital")
	}

	// Receptionists can only assign doctors from the same department as the triage queue.
	// This rule is universal for all callers of this method.
	if doctor.DepartmentID == nil || *doctor.DepartmentID != queue.DepartmentID {
		return errors.New("doctor does not belong to the required department")
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


func (u *arrivalUseCase) RevokeDoctorAssignment(ctx context.Context, queueID, userID uuid.UUID, reason string) error {
	queue, err := u.triageRepo.FindByID(ctx, queueID)
	if err != nil {
		return err
	}
	if queue.AssignedDoctorID == nil {
		return errors.New("no doctor is currently assigned")
	}

	// Revoke ALL active ReferralAccess grants (treating and consulting)
	if err := u.referralAccessRepo.RevokeAllByReferral(ctx, queue.ReferralID, reason); err != nil {
		return err
	}

	// Clear the assignment on the queue
	queue.AssignedDoctorID = nil
	queue.DoctorAssignedAt = nil
	if err := u.triageRepo.Update(ctx, queue); err != nil {
		return err
	}

	u.auditRepo.LogWithContext(ctx, userID, entity.ActionUnassignDoctor, &queue.ReferralID, nil, map[string]interface{}{
		"queue_id": queueID,
		"reason":   reason,
	})
	return nil
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

	if err := u.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(queue).Error; err != nil {
			return err
		}

		u.auditRepo.LogWithContext(ctx, userID, entity.ActionMarkMissed, &queue.ReferralID, nil, map[string]interface{}{
			"queue_id":    queueID,
			"miss_reason": missReason,
		})

		_ = u.inAppNotifUC.CreateForEvent(ctx, "PATIENT_MISSED", queue.ReferralID, userID)

		return nil
	}); err != nil {
		return err
	}

	// Fire the patient-facing MISSED SMS AFTER commit so we don't text
	// somebody about a no-show that ultimately rolled back. The SMS use
	// case is best-effort - a provider failure must not invalidate a
	// receptionist's recorded action.
	if u.notifUC != nil {
		_ = u.notifUC.QueueNotification(ctx, queue.ReferralID, entity.NotifyMissed, "")
	}

	return nil
}

func (u *arrivalUseCase) ReturnToTriage(ctx context.Context, queueID, userID uuid.UUID) error {
	queue, err := u.triageRepo.FindByID(ctx, queueID)
	if err != nil {
		return err
	}

	if queue.ArrivalStatus != entity.ArrivalMissed {
		return errors.New("only missed appointments can be returned to triage")
	}

	queue.ArrivalStatus = entity.ArrivalExpected
	queue.AppointmentDate = nil
	queue.MissReason = nil
	queue.AssignedDoctorID = nil
	queue.DoctorAssignedAt = nil

	return u.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(queue).Error; err != nil {
			return err
		}

		if err := tx.Model(&entity.Referral{}).
			Where("id = ?", queue.ReferralID).
			Update("status", entity.StatusAccepted).Error; err != nil {
			return err
		}

		u.auditRepo.LogWithContext(ctx, userID, "RETURN_TO_TRIAGE", &queue.ReferralID, nil, map[string]interface{}{
			"queue_id": queueID,
		})

		return nil
	})
}

func (u *arrivalUseCase) ListMissedByHospital(ctx context.Context, hospitalID uuid.UUID, limit, offset int) ([]*entity.TriageQueue, int64, error) {
	return u.triageRepo.ListMissedByHospital(ctx, hospitalID, limit, offset)
}

func (u *arrivalUseCase) GetTriageQueueByReferralID(ctx context.Context, referralID uuid.UUID) (*entity.TriageQueue, error) {
	return u.triageRepo.GetByReferralID(ctx, referralID)
}


func (u *arrivalUseCase) GrantConsultAccess(ctx context.Context, referralID, granterID, doctorID uuid.UUID) error {
	if granterID == doctorID {
		return errors.New("cannot grant consult access to yourself")
	}

	queue, err := u.triageRepo.GetByReferralID(ctx, referralID)
	if err != nil {
		return errors.New("referral record not found in triage")
	}

	// 1. Verify granter is the assigned treating doctor
	if queue.AssignedDoctorID == nil || *queue.AssignedDoctorID != granterID {
		return errors.New("only the assigned treating doctor can grant consulting access")
	}

	// 2. Verify granter has active TREATING_DOCTOR access
	access, err := u.referralAccessRepo.GetActiveAccess(ctx, referralID, granterID)
	if err != nil || access == nil || access.AccessType != "TREATING_DOCTOR" {
		return errors.New("granter does not have active treating access")
	}

	// 3. Verify target is a Referring Doctor in the same hospital
	doctor, err := u.userRepo.FindByID(ctx, doctorID)
	if err != nil || doctor.Role != entity.RoleReferringDoctor || doctor.HospitalID == nil || *doctor.HospitalID != queue.HospitalID {
		return errors.New("target doctor is invalid or belongs to a different hospital")
	}

	// 4. Check if active access already exists
	existing, err := u.referralAccessRepo.GetActiveAccess(ctx, referralID, doctorID)
	if err != nil {
		return err
	}
	if existing != nil {
		return errors.New("doctor already has active access to this referral")
	}

	// 5. Create grant
	newGrant := &entity.ReferralAccess{
		ReferralID: referralID,
		UserID:     doctorID,
		AccessType: "CONSULTED_DOCTOR",
		GrantedBy:  granterID,
	}
	if err := u.referralAccessRepo.Create(ctx, newGrant); err != nil {
		return err
	}

	u.auditRepo.LogWithContext(ctx, granterID, entity.ActionGrantConsultAccess, &referralID, nil, map[string]interface{}{
		"doctor_id": doctorID,
	})

	_ = u.inAppNotifUC.CreateForEvent(ctx, "CONSULTANT_ADDED", referralID, granterID)

	return nil
}

func (u *arrivalUseCase) RevokeConsultAccess(ctx context.Context, referralID, granterID, doctorID uuid.UUID, reason string) error {
	// 1. Verify granter is the treating doctor
	queue, err := u.triageRepo.GetByReferralID(ctx, referralID)
	if err != nil || queue.AssignedDoctorID == nil || *queue.AssignedDoctorID != granterID {
		return errors.New("unauthorized: only the assigned treating doctor can revoke consulting access")
	}

	// 2. Find active CONSULTED_DOCTOR access
	access, err := u.referralAccessRepo.GetActiveAccess(ctx, referralID, doctorID)
	if err != nil || access == nil || access.AccessType != "CONSULTED_DOCTOR" {
		return errors.New("no active consulting access found for this doctor")
	}

	// 3. Grantor Check: Only original grantor can revoke
	if access.GrantedBy != granterID {
		return errors.New("you can only revoke access grants you created")
	}

	// 4. Revoke
	now := time.Now()
	access.RevokedAt = &now
	access.RevokeReason = &reason
	if err := u.referralAccessRepo.Update(ctx, access); err != nil {
		return err
	}

	u.auditRepo.LogWithContext(ctx, granterID, entity.ActionRevokeConsultAccess, &referralID, nil, map[string]interface{}{
		"doctor_id": doctorID,
		"reason":    reason,
	})

	return nil
}


