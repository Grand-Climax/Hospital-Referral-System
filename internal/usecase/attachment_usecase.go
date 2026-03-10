package usecase

import (
	"context"

	"github.com/google/uuid"

	"Hospital-Referral-System/internal/domain/entity"
	"Hospital-Referral-System/internal/repository"
)

type AttachmentUseCase interface {
	SaveAttachment(ctx context.Context, referralID uuid.UUID, fileName, fileType, storagePath string) (*entity.Attachment, error)
	GetAttachment(ctx context.Context, id uuid.UUID) (*entity.Attachment, error)
}

type attachmentUseCase struct {
	repo repository.AttachmentRepository
}

func NewAttachmentUseCase(repo repository.AttachmentRepository) AttachmentUseCase {
	return &attachmentUseCase{repo: repo}
}

func (u *attachmentUseCase) SaveAttachment(ctx context.Context, referralID uuid.UUID, fileName, fileType, storagePath string) (*entity.Attachment, error) {
	attachment := &entity.Attachment{
		ReferralID:  referralID,
		FileName:    fileName,
		FileType:    fileType,
		StoragePath: storagePath,
	}
	if err := u.repo.Create(ctx, attachment); err != nil {
		return nil, err
	}
	return attachment, nil
}

func (u *attachmentUseCase) GetAttachment(ctx context.Context, id uuid.UUID) (*entity.Attachment, error) {
	return u.repo.FindByID(ctx, id)
}
