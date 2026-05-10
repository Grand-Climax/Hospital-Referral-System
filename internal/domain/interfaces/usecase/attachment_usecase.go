package interfaces

import (
	"context"

	"github.com/google/uuid"

	"Hospital-Referral-System/internal/domain/entity"
	iinfra "Hospital-Referral-System/internal/domain/interfaces/infrastructure"
)

type AttachmentUseCase interface {
	PrepareAttachmentEntity(referralID uuid.UUID, fileName, fileType, storagePath, publicID, category string, fileSize int64) *entity.Attachment
	GetAttachment(ctx context.Context, id uuid.UUID) (*entity.Attachment, error)
	GetAttachmentsByReferralID(ctx context.Context, referralID uuid.UUID) ([]entity.Attachment, error)
	DeleteAttachment(ctx context.Context, id uuid.UUID) error
	DeleteAttachmentFromReferral(ctx context.Context, referralID, attachmentID, doctorID uuid.UUID) error
	UploadAndAddAttachment(ctx context.Context, referralID, doctorID uuid.UUID, fileBytes []byte, fileName, fileType, category string, fileSize int64) (*entity.Attachment, error)
	VerifyAttachmentBytes(att *entity.Attachment, data []byte) (status string, metadata map[string]interface{}, reason string)
	Storage() iinfra.StorageService
}
