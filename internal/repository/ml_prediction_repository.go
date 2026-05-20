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

func (r *mlPredictionRepository) GetActiveByReferralID(ctx context.Context, referralID uuid.UUID) (*entity.MLPrediction, error) {
	var prediction entity.MLPrediction
	err := r.db.WithContext(ctx).
		Where("referral_id = ? AND is_active = ?", referralID, true).
		Order("predicted_at desc").
		First(&prediction).Error
	return &prediction, err
}

func (r *mlPredictionRepository) GetPendingFeedbackByReferralID(ctx context.Context, referralID uuid.UUID) (*entity.MLPrediction, error) {
	var prediction entity.MLPrediction
	err := r.db.WithContext(ctx).
		Where("referral_id = ? AND external_prediction_id IS NOT NULL AND external_prediction_id != '' AND feedback_sent_at IS NULL", referralID).
		Order("predicted_at desc").
		First(&prediction).Error
	return &prediction, err
}

func (r *mlPredictionRepository) MapActiveByReferralIDs(ctx context.Context, referralIDs []uuid.UUID) (map[uuid.UUID]*entity.MLPrediction, error) {
	out := make(map[uuid.UUID]*entity.MLPrediction)
	if len(referralIDs) == 0 {
		return out, nil
	}
	var preds []entity.MLPrediction
	err := r.db.WithContext(ctx).
		Where("referral_id IN ? AND is_active = ?", referralIDs, true).
		Order("predicted_at desc").
		Find(&preds).Error
	if err != nil {
		return nil, err
	}
	for i := range preds {
		id := preds[i].ReferralID
		if _, exists := out[id]; !exists {
			out[id] = &preds[i]
		}
	}
	return out, nil
}
