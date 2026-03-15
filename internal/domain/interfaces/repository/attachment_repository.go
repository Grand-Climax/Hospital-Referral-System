package repository

import "Hospital-Referral-System/internal/domain/entity"

type AttachmentRepository interface {
	BaseRepository[entity.Attachment]
}
