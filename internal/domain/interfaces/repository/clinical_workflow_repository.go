package interfaces

import (
	"context"
	"time"

	"github.com/google/uuid"

	"Hospital-Referral-System/internal/domain/entity"
)

// ClinicalUpdateRepository persists clinical progress notes added during a referral episode.
type ClinicalUpdateRepository interface {
	BaseRepository[entity.ClinicalUpdate]
	ListByReferralID(ctx context.Context, referralID uuid.UUID) ([]entity.ClinicalUpdate, error)
	ExistsForReferralAndDate(ctx context.Context, referralID uuid.UUID, reason string, date time.Time) (bool, error)
}
