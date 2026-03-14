package usecase

import (
	"context"

	"github.com/google/uuid"

	"Hospital-Referral-System/internal/domain/entity"
)

type LiaisonUseCase interface {
	ListSubmittedReferrals(ctx context.Context, hospitalID uuid.UUID) ([]entity.Referral, error)
	ApproveReferral(ctx context.Context, referralID, userID, hospitalID uuid.UUID) error
	RejectReferral(ctx context.Context, referralID, userID, hospitalID uuid.UUID, reason string) error
	ForwardReferral(ctx context.Context, referralID, userID, hospitalID, targetHospitalID uuid.UUID, comment string) error
}
