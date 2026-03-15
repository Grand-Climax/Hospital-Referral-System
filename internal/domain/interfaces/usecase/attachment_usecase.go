package usecase

import (
	"context"

	"github.com/google/uuid"

	"Hospital-Referral-System/internal/domain/entity"
)

type AttachmentUseCase interface {
	SaveAttachment(ctx context.Context, referralID uuid.UUID, fileName, fileType, storagePath string) (*entity.Attachment, error)
	GetAttachment(ctx context.Context, id uuid.UUID) (*entity.Attachment, error)
}
