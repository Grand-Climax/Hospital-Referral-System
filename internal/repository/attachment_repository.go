package repository

import (
	"gorm.io/gorm"

	"Hospital-Referral-System/internal/domain/entity"
)

type AttachmentRepository interface {
	BaseRepository[entity.Attachment]
}

type attachmentRepository struct {
	BaseRepository[entity.Attachment]
	db *gorm.DB
}

func NewAttachmentRepository(db *gorm.DB) AttachmentRepository {
	return &attachmentRepository{
		BaseRepository: NewBaseRepository[entity.Attachment](db),
		db:             db,
	}
}
