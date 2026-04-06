package repository

import (
	"context"
	"github.com/google/uuid"
	"Hospital-Referral-System/internal/domain/entity"
)

type AttachmentRepository interface {
	BaseRepository[entity.Attachment]
	CountByReferralID(ctx context.Context, referralID uuid.UUID) (int64, error)
	GetByReferralID(ctx context.Context, referralID uuid.UUID) ([]entity.Attachment, error)
	HardDelete(ctx context.Context, id uuid.UUID) error
}
