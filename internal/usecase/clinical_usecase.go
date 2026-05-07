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
	if err := u.checkAccess(ctx, referralID, userID); err != nil {
		return err
	}

	outcome := &entity.ReferralOutcome{
		ReferralID:             referralID,
		Outcome:                req.Outcome,
		LengthOfStayDays:       req.LengthOfStayDays,
		WasReferralAppropriate: req.WasReferralAppropriate,
		OutcomeNotes:           &req.OutcomeNotes,
		RecordedByID:           userID,
	}

	return u.db.Transaction(func(tx *gorm.DB) error {
		if err := u.outcomeRepo.Create(ctx, outcome); err != nil {
			return err
		}

		referral, err := u.referralRepo.GetReferralByID(ctx, referralID)
		if err != nil {
			return err
		}

		oldStatus := referral.Status
		newStatus := entity.StatusCompleted
		if req.Outcome == "deceased" {
			newStatus = entity.StatusDeceased
			referral.IsArchived = true
			now := time.Now()
			referral.ArchivedAt = &now
		}
		referral.Status = newStatus

		updates := map[string]interface{}{
			"status":      newStatus,
			"is_archived": referral.IsArchived,
			"archived_at": referral.ArchivedAt,
		}
		if err := tx.Model(&entity.Referral{}).Where("id = ?", referralID).Updates(updates).Error; err != nil {
			return err
		}

		// Status History
		_ = u.referralRepo.CreateStatusHistory(ctx, &entity.ReferralStatusHistory{
			ReferralID:  referralID,
			ChangedByID: userID,
			FromStatus:  &oldStatus,
			ToStatus:    newStatus,
			Reason:      &req.Outcome,
		})

		// Audit Log
		// Audit Log
		err = u.auditLogRepo.Create(ctx, &entity.AuditLog{
			UserID:     userID,
			ReferralID: &referralID,
			ActionType: entity.ActionRecordOutcome,
			Timestamp:  time.Now(),
		})
		if err != nil {
			return err
		}

		_ = u.inAppNotifUC.CreateForEvent(ctx, "OUTCOME_RECORDED", referralID, userID)

		return nil
	})
}

func (u *clinicalUseCase) GetClinicalHistory(ctx context.Context, referralID, userID uuid.UUID) ([]entity.ClinicalUpdate, error) {
	if err := u.checkAccess(ctx, referralID, userID); err != nil {
		return nil, err
	}

	return u.clinicalRepo.ListByReferralID(ctx, referralID)
}
