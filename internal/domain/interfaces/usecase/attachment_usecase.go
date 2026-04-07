package usecase

import (
	"context"

	"github.com/google/uuid"

	"Hospital-Referral-System/internal/delivery/http/dto"
	"Hospital-Referral-System/internal/domain/entity"
)

type AttachmentUseCase interface {
	SaveAttachment(ctx context.Context, referralID uuid.UUID, fileName, fileType, storagePath, publicID, category string, fileSize int64) (*entity.Attachment, error)
	PrepareAttachmentEntity(referralID uuid.UUID, fileName, fileType, storagePath, publicID, category string, fileSize int64) *entity.Attachment
	GetAttachment(ctx context.Context, id uuid.UUID) (*entity.Attachment, error)
	GetAttachmentsByReferralID(ctx context.Context, referralID uuid.UUID) ([]entity.Attachment, error)
	DeleteAttachment(ctx context.Context, id uuid.UUID) error
	GenerateSignature(hospitalID uuid.UUID) (map[string]interface{}, error)

	// New secure methods
	AddAttachmentsToReferral(ctx context.Context, referralID, doctorID uuid.UUID, reqs []dto.CreateAttachmentRequest) ([]entity.Attachment, error)
	DeleteAttachmentFromReferral(ctx context.Context, referralID, attachmentID, doctorID uuid.UUID) error
	ProcessWebhookAttachment(ctx context.Context, payload map[string]interface{}) error
	UploadAndAddAttachment(ctx context.Context, referralID, doctorID uuid.UUID, file interface{}, fileName, fileType, category string, fileSize int64) (*entity.Attachment, error)
}
