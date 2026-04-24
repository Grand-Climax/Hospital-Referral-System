package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"Hospital-Referral-System/internal/domain/entity"
	irepository "Hospital-Referral-System/internal/domain/interfaces/repository"
)



// schedulerCheckpointRepository tracks the last successful run of background scheduling jobs.
// Placed here because it is a scheduling-domain concern, not an admin concern.
type schedulerCheckpointRepository struct {
	*BaseRepository[entity.SchedulerCheckpoint]
	db *gorm.DB
}

func NewSchedulerCheckpointRepository(db *gorm.DB) irepository.SchedulerCheckpointRepository {
	return &schedulerCheckpointRepository{
		BaseRepository: NewBaseRepository[entity.SchedulerCheckpoint](db),
		db:             db,
	}
}

func (r *schedulerCheckpointRepository) GetLastCheckpoint(ctx context.Context) (time.Time, error) {
	var cp entity.SchedulerCheckpoint
	err := r.db.WithContext(ctx).Order("last_processed_at desc").First(&cp).Error
	if err != nil {
		return time.Time{}, err
	}
	if cp.LastProcessedAt == nil {
		return time.Time{}, nil
	}
	return *cp.LastProcessedAt, nil
}

func (r *schedulerCheckpointRepository) UpdateCheckpoint(ctx context.Context, ts time.Time) error {
	return r.db.WithContext(ctx).Create(&entity.SchedulerCheckpoint{
		LastProcessedAt: &ts,
		HospitalID:      uuid.Nil,
		DeptID:          uuid.Nil,
	}).Error
}
