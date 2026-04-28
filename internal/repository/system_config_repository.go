package repository

import (
	"context"

	"gorm.io/gorm"

	"Hospital-Referral-System/internal/domain/entity"
	irepository "Hospital-Referral-System/internal/domain/interfaces/repository"
)

// systemConfigRepository manages system-wide key-value configuration.
type systemConfigRepository struct {
	*BaseRepository[entity.SystemConfig]
	db *gorm.DB
}

func NewSystemConfigRepository(db *gorm.DB) irepository.SystemConfigRepository {
	return &systemConfigRepository{
		BaseRepository: NewBaseRepository[entity.SystemConfig](db),
		db:             db,
	}
}

func (r *systemConfigRepository) GetByKey(ctx context.Context, key string) (*entity.SystemConfig, error) {
	var cfg entity.SystemConfig
	err := r.db.WithContext(ctx).Where("key = ?", key).First(&cfg).Error
	return &cfg, err
}

func (r *systemConfigRepository) GetAll(ctx context.Context) (map[string]string, error) {
	var cfgs []entity.SystemConfig
	if err := r.db.WithContext(ctx).Find(&cfgs).Error; err != nil {
		return nil, err
	}
	res := make(map[string]string)
	for _, c := range cfgs {
		res[c.Key] = c.Value
	}
	return res, nil
}

func (r *systemConfigRepository) Update(ctx context.Context, cfg *entity.SystemConfig) error {
	return r.db.WithContext(ctx).
		Model(&entity.SystemConfig{}).
		Where("key = ?", cfg.Key).
		Update("value", cfg.Value).Error
}

func (r *systemConfigRepository) BulkUpdate(ctx context.Context, updates map[string]string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for k, v := range updates {
			// Using INSERT ... ON CONFLICT (key) DO UPDATE SET value = ...
			if err := tx.Exec("INSERT INTO system_configs (key, value, updated_at) VALUES (?, ?, NOW()) ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_at = NOW()", k, v).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *systemConfigRepository) GetBool(ctx context.Context, key string, defaultValue bool) (bool, error) {
	cfg, err := r.GetByKey(ctx, key)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return defaultValue, nil
		}
		return defaultValue, err
	}
	return cfg.Value == "true" || cfg.Value == "1", nil
}
