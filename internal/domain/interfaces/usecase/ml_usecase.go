package interfaces

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
)

// MLUseCase orchestrates severity scoring and feedback with the external ML service.
type MLUseCase interface {
	ScheduleScore(referralID uuid.UUID)
	ScheduleScoreForce(referralID uuid.UUID)
	ScoreReferral(ctx context.Context, referralID uuid.UUID) error
	ScoreReferralForce(ctx context.Context, referralID uuid.UUID) error
	SendFeedbackAccept(ctx context.Context, referralID uuid.UUID) error
	SendFeedbackOverride(ctx context.Context, referralID uuid.UUID, correctedScore float64, doctorExplanation string) error
	MLSeverityOverride(ctx context.Context, referralID, userID uuid.UUID, score float64, justification string) error
	ProcessMLResult(ctx context.Context, referralID uuid.UUID, score float64, confidence float64, severityTier string, explanation json.RawMessage, modelVersion string, inputFeatures json.RawMessage, externalPredictionID *string, processingTimeMs *float64, triggerReason string) error
}
