package repository

import (
	"context"
	"time"

	"gorm.io/gorm"

	"Hospital-Referral-System/internal/domain/entity"
	irepository "Hospital-Referral-System/internal/domain/interfaces/repository"
)

type jobCheckpointRepository struct {
	db *gorm.DB
}

func NewJobCheckpointRepository(db *gorm.DB) irepository.JobCheckpointRepository {
	return &jobCheckpointRepository{db: db}
}

func (r *jobCheckpointRepository) GetLastRun(ctx context.Context, jobName string) (*time.Time, error) {
	var cp entity.JobCheckpoint
	if err := r.db.WithContext(ctx).Where("job_name = ?", jobName).First(&cp).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return cp.LastRunAt, nil
}

func (r *jobCheckpointRepository) UpdateLastRun(ctx context.Context, jobName string, timestamp time.Time) error {
	return r.db.WithContext(ctx).Model(&entity.JobCheckpoint{}).
		Where("job_name = ?", jobName).
		Update("last_run_at", timestamp).Error
}
