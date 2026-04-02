package usecase

import (
	"context"

	"github.com/google/uuid"
)

type SpecialistUseCase interface {
	AcceptReferral(ctx context.Context, referralID, specialistID, destHospitalID uuid.UUID) error
	RejectReferral(ctx context.Context, referralID, specialistID, destHospitalID uuid.UUID, reason string) error
	// Could also list incoming referrals here later
}
