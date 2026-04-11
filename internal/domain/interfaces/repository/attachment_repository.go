package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"Hospital-Referral-System/internal/domain/entity"
)

type AttachmentRepository interface {
	BaseRepository[entity.Attachment]
	CountByReferralID(ctx context.Context, referralID uuid.UUID) (int64, error)
	GetByReferralID(ctx context.Context, referralID uuid.UUID) ([]entity.Attachment, error)
	HardDelete(ctx context.Context, id uuid.UUID) error
	FindByPublicID(ctx context.Context, publicID string) (*entity.Attachment, error)
	GetPendingAttachments(ctx context.Context) ([]entity.Attachment, error)
	GetPendingAttachmentsBatch(ctx context.Context, limit int) ([]entity.Attachment, error)
	UpdateVerificationStatus(ctx context.Context, id uuid.UUID, status string, metadata map[string]interface{}, storagePath string, publicID string, rejectionReason string, rejectedAt *time.Time) error
	CountByPublicIDPrefix(ctx context.Context, prefix string) (int64, error)
}
