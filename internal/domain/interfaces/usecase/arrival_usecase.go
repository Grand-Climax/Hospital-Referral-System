package interfaces

import (
	"context"

	"github.com/google/uuid"

	"Hospital-Referral-System/internal/domain/entity"
)

type ArrivalUseCase interface {
	MarkExpected(ctx context.Context, referralID uuid.UUID) error
	ConfirmArrival(ctx context.Context, referralID, receptionistID uuid.UUID) error
	UpdateArrivalStatus(ctx context.Context, referralID uuid.UUID, status entity.ArrivalStatus) error
}
