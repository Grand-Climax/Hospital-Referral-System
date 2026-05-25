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

type clinicalUseCase struct {
	db                *gorm.DB
	referralRepo      irepository.ReferralRepository
	clinicalRepo      irepository.ClinicalUpdateRepository
	outcomeRepo       irepository.ReferralOutcomeRepository
	referralAccessRepo irepository.ReferralAccessRepository
	auditLogRepo      irepository.AuditLogRepository
	inAppNotifUC      iusecase.InAppNotificationUseCase
}

func NewClinicalUseCase(
	db *gorm.DB,
	referralRepo irepository.ReferralRepository,
	clinicalRepo irepository.ClinicalUpdateRepository,
	outcomeRepo irepository.ReferralOutcomeRepository,
	referralAccessRepo irepository.ReferralAccessRepository,
	auditLogRepo irepository.AuditLogRepository,
	inAppNotifUC iusecase.InAppNotificationUseCase,
) iusecase.ClinicalUseCase {
	return &clinicalUseCase{
		db:                db,
		referralRepo:      referralRepo,
		clinicalRepo:      clinicalRepo,
		outcomeRepo:       outcomeRepo,
		referralAccessRepo: referralAccessRepo,
		auditLogRepo:      auditLogRepo,
		inAppNotifUC:       inAppNotifUC,
	}
}

func (u *clinicalUseCase) checkAccess(ctx context.Context, referralID, userID uuid.UUID) error {
	hasAccess, err := u.referralAccessRepo.CheckAccess(ctx, referralID, userID)
	if err != nil {
		return err
	}
	if !hasAccess {
		return errors.New("forbidden: you do not have permission to access this referral's clinical data")
	}
	return nil
}

func (u *clinicalUseCase) AddClinicalUpdate(ctx context.Context, referralID, userID uuid.UUID, req dto.AddClinicalUpdateRequest) error {
	if err := u.checkAccess(ctx, referralID, userID); err != nil {
		return err
	}

	requiresReview := false
	if req.UpdateReason == "CONDITION_CHANGE" || req.UpdateReason == "MISSED_APPOINTMENT_RE_EVALUATION" {
		requiresReview = true
	}

	update := &entity.ClinicalUpdate{
		ReferralID:     referralID,
		UpdatedByID:    userID,
		UpdateReason:   req.UpdateReason,
		ClinicalNotes:  req.ClinicalNotes,
		RequiresReview: requiresReview,
	}

	return u.db.Transaction(func(tx *gorm.DB) error {
		if err := u.clinicalRepo.Create(ctx, update); err != nil {
			return err
		}

		// Audit Log
		return u.auditLogRepo.Create(ctx, &entity.AuditLog{
			UserID:     userID,
			ReferralID: &referralID,
			ActionType: entity.ActionAddClinicalUpdate,
			Timestamp:  time.Now(),
		})
	})
}

func (u *clinicalUseCase) RecordOutcome(ctx context.Context, referralID, userID uuid.UUID, req dto.RecordOutcomeRequest) error {
	// Read the referral up-front so we can validate state BEFORE we open
	// the transaction. The old implementation did this inside the tx,
	// which meant a failed precondition still left a half-applied
	// outcome row behind on databases where the repo bypassed `tx`.
	referral, err := u.referralRepo.GetReferralByID(ctx, referralID)
	if err != nil {
		return err
	}
	if referral.Status != entity.StatusScheduled && referral.Status != entity.StatusAccepted {
		return errors.New("cannot record outcome: referral must be scheduled first")
	}

	// HARD GATE: only the assigned treating doctor (the one the
	// receptionist linked via /receptionist/.../assign-doctor) can
	// close the case. The sender, the receiving specialist, and any
	// consulting doctor are all rejected. This matches the FE
	// contract: the "Record outcome" CTA is only ever rendered for
	// the doctor whose ID equals triage_queue.assigned_doctor_id.
	//
	// We bypass the regular checkAccess() / ReferralAccess lookup
	// because that helper is too permissive for this specific
	// endpoint - it accepts senders, specialists, and consultants.
	var queue entity.TriageQueue
	if err := u.db.WithContext(ctx).
		Select("assigned_doctor_id").
		Where("referral_id = ?", referralID).
		First(&queue).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("forbidden: no triage entry exists for this referral")
		}
		return err
	}
	if queue.AssignedDoctorID == nil {
		return errors.New("forbidden: no treating doctor has been assigned to this referral yet")
	}
	if *queue.AssignedDoctorID != userID {
		return errors.New("forbidden: only the assigned treating doctor can record an outcome")
	}

	oldStatus := referral.Status
	newStatus := entity.StatusCompleted
	isDeceased := req.Outcome == "deceased"
	if isDeceased {
		newStatus = entity.StatusDeceased
	}

	if err := u.db.Transaction(func(tx *gorm.DB) error {
		// Write all five rows through `tx` so a failure in any one
		// step rolls the whole thing back. The previous version used
		// repos that hold their own `db` handle - the outcome,
		// status-history, and audit-log rows would commit
		// independently of the referral status update.
		outcome := &entity.ReferralOutcome{
			ReferralID:             referralID,
			Outcome:                req.Outcome,
			LengthOfStayDays:       req.LengthOfStayDays,
			WasReferralAppropriate: req.WasReferralAppropriate,
			OutcomeNotes:           &req.OutcomeNotes,
			RecordedByID:           userID,
		}
		if err := tx.Create(outcome).Error; err != nil {
			return err
		}

		updates := map[string]interface{}{"status": newStatus}
		if isDeceased {
			now := time.Now()
			updates["is_archived"] = true
			updates["archived_at"] = &now
		}
		if err := tx.Model(&entity.Referral{}).Where("id = ?", referralID).Updates(updates).Error; err != nil {
			return err
		}

		// Flip the triage row to ADMITTED for non-deceased outcomes.
		// The /deceased path deletes the triage row entirely, so this
		// branch only runs for the successful-completion paths
		// (discharged, improved, deteriorated, transferred).
		// ADMITTED here is used as the terminal "patient finished
		// treatment" marker so dashboards can filter out completed
		// referrals from active arrival lists.
		if !isDeceased {
			if err := tx.Model(&entity.TriageQueue{}).
				Where("referral_id = ?", referralID).
				Update("arrival_status", entity.ArrivalAdmitted).Error; err != nil {
				return err
			}
		}

		if err := tx.Create(&entity.ReferralStatusHistory{
			ReferralID:  referralID,
			ChangedByID: userID,
			FromStatus:  &oldStatus,
			ToStatus:    newStatus,
			Reason:      &req.Outcome,
		}).Error; err != nil {
			return err
		}

		return tx.Create(&entity.AuditLog{
			UserID:     userID,
			ReferralID: &referralID,
			ActionType: entity.ActionRecordOutcome,
			Timestamp:  time.Now(),
		}).Error
	}); err != nil {
		return err
	}

	// Post-commit side effects. Revocation + in-app notification are
	// best-effort - they must not roll back a successfully recorded
	// outcome if e.g. the in-app notifier hiccups.
	reason := "Referral completed"
	if isDeceased {
		reason = "Patient deceased"
	}
	_ = u.referralAccessRepo.RevokeAllByReferral(ctx, referralID, reason)
	_ = u.inAppNotifUC.CreateForEvent(ctx, "OUTCOME_RECORDED", referralID, userID)
	return nil
}

func (u *clinicalUseCase) GetClinicalHistory(ctx context.Context, referralID, userID uuid.UUID) ([]entity.ClinicalUpdate, error) {
	if err := u.checkAccess(ctx, referralID, userID); err != nil {
		return nil, err
	}

	return u.clinicalRepo.ListByReferralID(ctx, referralID)
}
