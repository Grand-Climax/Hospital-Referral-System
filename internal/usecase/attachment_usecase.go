package usecase

import (
	"context"

	"github.com/google/uuid"

	"Hospital-Referral-System/internal/domain/entity"
	irepository "Hospital-Referral-System/internal/domain/interfaces/repository"
	iusecase "Hospital-Referral-System/internal/domain/interfaces/usecase"
)

type attachmentUseCase struct {
	repo irepository.AttachmentRepository
}

func NewAttachmentUseCase(repo irepository.AttachmentRepository) iusecase.AttachmentUseCase {
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
