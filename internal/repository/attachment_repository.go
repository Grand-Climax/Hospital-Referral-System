package repository

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"Hospital-Referral-System/internal/domain/entity"
	irepository "Hospital-Referral-System/internal/domain/interfaces/repository"
)

type attachmentRepository struct {
	*BaseRepository[entity.Attachment]
	db *gorm.DB
}

func NewAttachmentRepository(db *gorm.DB) irepository.AttachmentRepository {
	return &attachmentRepository{
		BaseRepository: NewBaseRepository[entity.Attachment](db),
		db:             db,
	}
}

func (r *attachmentRepository) CountByReferralID(ctx context.Context, referralID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&entity.Attachment{}).Where("referral_id = ?", referralID).Count(&count).Error
	return count, err
}

func (r *attachmentRepository) GetByReferralID(ctx context.Context, referralID uuid.UUID) ([]entity.Attachment, error) {
	var attachments []entity.Attachment
	err := r.db.WithContext(ctx).Where("referral_id = ?", referralID).Find(&attachments).Error
	return attachments, err
}

func (r *attachmentRepository) HardDelete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Unscoped().Delete(&entity.Attachment{}, "id = ?", id).Error
}
