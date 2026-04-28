package interfaces

import (
	"context"

	"github.com/google/uuid"

	"Hospital-Referral-System/internal/delivery/http/dto"
	"Hospital-Referral-System/internal/domain/entity"
)

type ClinicalUseCase interface {
	AddClinicalUpdate(ctx context.Context, referralID, userID uuid.UUID, req dto.AddClinicalUpdateRequest) error
	RecordOutcome(ctx context.Context, referralID, userID uuid.UUID, req dto.RecordOutcomeRequest) error
	GetClinicalHistory(ctx context.Context, referralID, userID uuid.UUID) ([]entity.ClinicalUpdate, error)
}
