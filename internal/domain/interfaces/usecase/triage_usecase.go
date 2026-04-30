package interfaces

import (
	"context"

	"time"

	"github.com/google/uuid"

	"Hospital-Referral-System/internal/delivery/http/dto"
	"Hospital-Referral-System/internal/domain/entity"
)

type TriageUseCase interface {
	LandInQueue(ctx context.Context, referralID uuid.UUID) error
	CalculateCompositeScore(ctx context.Context, referralID uuid.UUID) (float64, error)
	ListForTriage(ctx context.Context, hospitalID uuid.UUID, limit, offset int) ([]dto.TriageListResponse, int64, error)
	ReviewTriage(ctx context.Context, referralID, userID uuid.UUID, req dto.TriageReviewRequest) error
	SetManualSeverity(ctx context.Context, referralID, userID uuid.UUID, score float64, justification string) error
	ListScheduledInRange(ctx context.Context, hospitalID, deptID uuid.UUID, start, end time.Time) ([]entity.TriageQueue, error)
}
