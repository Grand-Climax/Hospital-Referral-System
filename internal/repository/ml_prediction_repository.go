package repository

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"Hospital-Referral-System/internal/domain/entity"
	irepository "Hospital-Referral-System/internal/domain/interfaces/repository"
)

// mlPredictionRepository stores triage severity scores returned by the ML service.
// Embeds BaseRepository for standard CRUD.
type mlPredictionRepository struct {
	*BaseRepository[entity.MLPrediction]
	db *gorm.DB
}

func NewMLPredictionRepository(db *gorm.DB) irepository.MLPredictionRepository {
	return &mlPredictionRepository{
		BaseRepository: NewBaseRepository[entity.MLPrediction](db),
		db:             db,
	}
}

func (r *mlPredictionRepository) Create(ctx context.Context, prediction *entity.MLPrediction) error {
	return r.db.WithContext(ctx).Create(prediction).Error
}

func (r *mlPredictionRepository) GetLatestByReferralID(ctx context.Context, referralID uuid.UUID) (*entity.MLPrediction, error) {
	var prediction entity.MLPrediction
	err := r.db.WithContext(ctx).
		Where("referral_id = ?", referralID).
		Order("predicted_at desc").
		First(&prediction).Error
	return &prediction, err
}
