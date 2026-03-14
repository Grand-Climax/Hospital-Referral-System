package usecase

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"Hospital-Referral-System/internal/domain/entity"
	irepository "Hospital-Referral-System/internal/domain/interfaces/repository"
	iusecase "Hospital-Referral-System/internal/domain/interfaces/usecase"
)

var (
	ErrReferralNotFound       = errors.New("referral not found")
	ErrNotYourHospital        = errors.New("referral does not belong to your hospital")
	ErrInvalidStatusForAction = errors.New("referral is not in the required status for this action")
)

type liaisonUseCase struct {
	referralRepo irepository.ReferralRepository
}

func NewLiaisonUseCase(referralRepo irepository.ReferralRepository) iusecase.LiaisonUseCase {
	return &liaisonUseCase{referralRepo: referralRepo}
}

func (u *liaisonUseCase) ListSubmittedReferrals(ctx context.Context, hospitalID uuid.UUID) ([]entity.Referral, error) {
	filters := map[string]interface{}{
		"sender_hospital_id": hospitalID,
		"status":             entity.StatusSubmitted,
	}
	return u.referralRepo.ListReferrals(ctx, filters)
}

func (u *liaisonUseCase) ApproveReferral(ctx context.Context, referralID, userID, hospitalID uuid.UUID) error {
	referral, err := u.referralRepo.GetReferralByID(ctx, referralID)
	if err != nil {
		return ErrReferralNotFound
	}

	if referral.SenderHospitalID != hospitalID {
		return ErrNotYourHospital
	}

	if referral.Status != entity.StatusSubmitted {
		return ErrInvalidStatusForAction
	}

	oldStatus := referral.Status
	referral.Status = entity.StatusUnderLiaisonReview
	referral.LiaisonOfficerID = &userID

	if err := u.referralRepo.UpdateReferralTransaction(ctx, referral); err != nil {
		return err
	}

	return u.referralRepo.CreateStatusHistory(ctx, &entity.ReferralStatusHistory{
		ReferralID:  referralID,
		ChangedByID: userID,
		FromStatus:  &oldStatus,
		ToStatus:    entity.StatusUnderLiaisonReview,
		ChangedAt:   time.Now(),
	})
}

func (u *liaisonUseCase) RejectReferral(ctx context.Context, referralID, userID, hospitalID uuid.UUID, reason string) error {
	referral, err := u.referralRepo.GetReferralByID(ctx, referralID)
	if err != nil {
		return ErrReferralNotFound
	}

	if referral.SenderHospitalID != hospitalID {
		return ErrNotYourHospital
	}

	if referral.Status != entity.StatusUnderLiaisonReview {
		return ErrInvalidStatusForAction
	}

	if reason == "" {
		return errors.New("rejection reason is required")
	}

	oldStatus := referral.Status
	referral.Status = entity.StatusNeedsRevision
	referral.RejectionReason = &reason

	if err := u.referralRepo.UpdateReferralTransaction(ctx, referral); err != nil {
		return err
	}

	return u.referralRepo.CreateStatusHistory(ctx, &entity.ReferralStatusHistory{
		ReferralID:  referralID,
		ChangedByID: userID,
		FromStatus:  &oldStatus,
		ToStatus:    entity.StatusNeedsRevision,
		Reason:      &reason,
		ChangedAt:   time.Now(),
	})
}

func (u *liaisonUseCase) ForwardReferral(ctx context.Context, referralID, userID, hospitalID, targetHospitalID uuid.UUID, comment string) error {
	referral, err := u.referralRepo.GetReferralByID(ctx, referralID)
	if err != nil {
		return ErrReferralNotFound
	}

	if referral.SenderHospitalID != hospitalID {
		return ErrNotYourHospital
	}

	if referral.Status != entity.StatusUnderLiaisonReview {
		return ErrInvalidStatusForAction
	}

	oldStatus := referral.Status
	referral.Status = entity.StatusForwarded
	referral.TargetHospitalID = targetHospitalID

	if err := u.referralRepo.UpdateReferralTransaction(ctx, referral); err != nil {
		return err
	}

	var reasonPtr *string
	if comment != "" {
		reasonPtr = &comment
	}

	return u.referralRepo.CreateStatusHistory(ctx, &entity.ReferralStatusHistory{
		ReferralID:  referralID,
		ChangedByID: userID,
		FromStatus:  &oldStatus,
		ToStatus:    entity.StatusForwarded,
		Reason:      reasonPtr,
		ChangedAt:   time.Now(),
	})
}
