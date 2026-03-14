package repository

import (
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
