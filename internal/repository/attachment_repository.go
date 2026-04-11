package repository

import (
	"context"
	"time"

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

func (r *attachmentRepository) FindByPublicID(ctx context.Context, publicID string) (*entity.Attachment, error) {
	var attachment entity.Attachment
	err := r.db.WithContext(ctx).Where("public_id = ?", publicID).First(&attachment).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &attachment, nil
}

func (r *attachmentRepository) GetPendingAttachments(ctx context.Context) ([]entity.Attachment, error) {
	var attachments []entity.Attachment
	err := r.db.WithContext(ctx).Where("verification_status = ?", entity.VerificationPending).Find(&attachments).Error
	return attachments, err
}

func (r *attachmentRepository) GetPendingAttachmentsBatch(ctx context.Context, limit int) ([]entity.Attachment, error) {
	var attachments []entity.Attachment
	err := r.db.WithContext(ctx).Where("verification_status = ?", entity.VerificationPending).Limit(limit).Find(&attachments).Error
	return attachments, err
}

func (r *attachmentRepository) UpdateVerificationStatus(ctx context.Context, id uuid.UUID, status string, metadata map[string]interface{}, storagePath string, publicID string, rejectionReason string, rejectedAt *time.Time) error {
	updates := map[string]interface{}{
		"verification_status": status,
		"metadata":            metadata,
		"rejection_reason":    rejectionReason,
		"rejected_at":         rejectedAt,
	}
	if storagePath != "" {
		updates["storage_path"] = storagePath
	}
	if publicID != "" {
		updates["public_id"] = publicID
	}
	return r.db.WithContext(ctx).Model(&entity.Attachment{}).Where("id = ?", id).Updates(updates).Error
}

func (r *attachmentRepository) CountByPublicIDPrefix(ctx context.Context, prefix string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&entity.Attachment{}).Where("public_id LIKE ?", prefix+"%").Count(&count).Error
	return count, err
}
