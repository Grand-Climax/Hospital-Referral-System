package repository

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"Hospital-Referral-System/internal/domain/entity"
	irepository "Hospital-Referral-System/internal/domain/interfaces/repository"
)

// clinicalUpdateRepository persists clinical progress notes added to a referral episode.
// Embeds BaseRepository for standard CRUD.
type clinicalUpdateRepository struct {
	*BaseRepository[entity.ClinicalUpdate]
	db *gorm.DB
}

func NewClinicalUpdateRepository(db *gorm.DB) irepository.ClinicalUpdateRepository {
	return &clinicalUpdateRepository{
		BaseRepository: NewBaseRepository[entity.ClinicalUpdate](db),
		db:             db,
	}
}

func (r *clinicalUpdateRepository) ListByReferralID(ctx context.Context, referralID uuid.UUID) ([]entity.ClinicalUpdate, error) {
	var updates []entity.ClinicalUpdate
	err := r.db.WithContext(ctx).
		Where("referral_id = ?", referralID).
		Order("created_at asc").
		Find(&updates).Error
	return updates, err
}
