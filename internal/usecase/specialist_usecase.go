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

type specialistUseCase struct {
	referralRepo irepository.ReferralRepository
}

func NewSpecialistUseCase(repo irepository.ReferralRepository) iusecase.SpecialistUseCase {
	return &specialistUseCase{referralRepo: repo}
}

func (u *specialistUseCase) AcceptReferral(ctx context.Context, referralID, specialistID, destHospitalID uuid.UUID) error {
	referral, err := u.referralRepo.GetReferralByID(ctx, referralID)
	if err != nil {
		return err
	}

	// Must be targeted to this hospital
	if referral.TargetHospitalID != destHospitalID {
		return errors.New("unauthorized: referral is not directed to your hospital")
	}

	// Check if it's in a valid state to be accepted (FORWARDED or SPECIALIST_REVIEW)
	if referral.Status != entity.StatusForwarded && referral.Status != entity.StatusSpecialistReview {
		return errors.New("invalid status: referral must be forwarded or received before a specialist can accept it")
	}

	oldStatus := referral.Status
	referral.Status = entity.StatusSpecialistAssigned

	if err := u.referralRepo.UpdateReferralTransaction(ctx, referral); err != nil {
		return err
	}

	return u.referralRepo.CreateStatusHistory(ctx, &entity.ReferralStatusHistory{
		ReferralID:  referralID,
		ChangedByID: specialistID,
		FromStatus:  &oldStatus,
		ToStatus:    entity.StatusSpecialistAssigned,
		ChangedAt:   time.Now(),
	})
}

func (u *specialistUseCase) RejectReferral(ctx context.Context, referralID, specialistID, destHospitalID uuid.UUID, reason string) error {
	referral, err := u.referralRepo.GetReferralByID(ctx, referralID)
	if err != nil {
		return err
	}

	if referral.TargetHospitalID != destHospitalID {
		return errors.New("unauthorized: referral is not directed to your hospital")
	}

	if referral.Status != entity.StatusForwarded && referral.Status != entity.StatusSpecialistReview {
		return errors.New("invalid status: referral cannot be rejected from current state")
	}

	if reason == "" {
		return errors.New("reason is required when a specialist rejects a referral")
	}

	oldStatus := referral.Status
	
	// Bounce back to the sender hospital's liaison
	referral.Status = entity.StatusUnderLiaisonReview
	referral.RejectionReason = &reason
	referral.TargetHospitalID = uuid.Nil // Optional: clear the target, or leave it to show history? Better to leave it or clear?
	// The problem dictates changing status to UNDER_LIAISON_REVIEW, so the liaison has to choose another hospital.
	// We'll leave TargetHospitalID intact so they know it was rejected by them, but maybe they update it upon forwarding.

	if err := u.referralRepo.UpdateReferralTransaction(ctx, referral); err != nil {
		return err
	}

	return u.referralRepo.CreateStatusHistory(ctx, &entity.ReferralStatusHistory{
		ReferralID:  referralID,
		ChangedByID: specialistID,
		FromStatus:  &oldStatus,
		ToStatus:    entity.StatusUnderLiaisonReview,
		Reason:      &reason,
		ChangedAt:   time.Now(),
	})
}
