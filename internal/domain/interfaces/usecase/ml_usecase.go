package interfaces

import (
	"context"

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
}
