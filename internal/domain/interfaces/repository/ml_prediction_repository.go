package interfaces

import (
	"context"

	"github.com/google/uuid"

	"Hospital-Referral-System/internal/domain/entity"
)

// MLPredictionRepository stores and retrieves ML triage severity predictions.
type MLPredictionRepository interface {
	BaseRepository[entity.MLPrediction]
	GetLatestByReferralID(ctx context.Context, referralID uuid.UUID) (*entity.MLPrediction, error)
	GetActiveByReferralID(ctx context.Context, referralID uuid.UUID) (*entity.MLPrediction, error)
	GetPendingFeedbackByReferralID(ctx context.Context, referralID uuid.UUID) (*entity.MLPrediction, error)
	MapActiveByReferralIDs(ctx context.Context, referralIDs []uuid.UUID) (map[uuid.UUID]*entity.MLPrediction, error)
}
