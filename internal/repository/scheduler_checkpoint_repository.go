package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"Hospital-Referral-System/internal/domain/entity"
	irepository "Hospital-Referral-System/internal/domain/interfaces/repository"
)

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

func (r *schedulerCheckpointRepository) GetNextEligibleDepartment(ctx context.Context, minAge time.Duration, leaseHolder string, leaseDuration time.Duration) (*entity.SchedulerCheckpoint, error) {
	var checkpoint entity.SchedulerCheckpoint

	// SQL for atomic lease acquisition with FOR UPDATE SKIP LOCKED
	// $1: minAge in seconds
	// $2: leaseHolder
	// $3: leaseDuration in seconds
	query := `
		WITH next_dept AS (
			SELECT hospital_id, department_id 
			FROM scheduler_checkpoints
			WHERE (lease_expires_at IS NULL OR lease_expires_at < NOW())
			  AND (last_processed_at IS NULL OR last_processed_at < NOW() - INTERVAL '1 second' * ?)
			ORDER BY last_processed_at ASC NULLS FIRST
			LIMIT 1
			FOR UPDATE SKIP LOCKED
		)
		UPDATE scheduler_checkpoints
		SET lease_holder = ?, lease_expires_at = NOW() + INTERVAL '1 second' * ?
		FROM next_dept
		WHERE scheduler_checkpoints.hospital_id = next_dept.hospital_id 
		  AND scheduler_checkpoints.department_id = next_dept.department_id
		RETURNING scheduler_checkpoints.*`

	result := r.db.WithContext(ctx).Raw(query, minAge.Seconds(), leaseHolder, leaseDuration.Seconds()).Scan(&checkpoint)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, nil
	}

	return &checkpoint, nil
}

func (r *schedulerCheckpointRepository) UpdateLastProcessed(ctx context.Context, hospitalID, departmentID uuid.UUID, leaseHolder string) error {
	return r.db.WithContext(ctx).Model(&entity.SchedulerCheckpoint{}).
		Where("hospital_id = ? AND department_id = ? AND lease_holder = ?", hospitalID, departmentID, leaseHolder).
		Updates(map[string]interface{}{
			"last_processed_at": time.Now(),
			"lease_holder":      nil,
			"lease_expires_at":  nil,
		}).Error
}

func (r *schedulerCheckpointRepository) ReleaseLease(ctx context.Context, hospitalID, departmentID uuid.UUID, leaseHolder string) error {
	return r.db.WithContext(ctx).Model(&entity.SchedulerCheckpoint{}).
		Where("hospital_id = ? AND department_id = ? AND lease_holder = ?", hospitalID, departmentID, leaseHolder).
		Updates(map[string]interface{}{
			"lease_holder":     nil,
			"lease_expires_at": nil,
		}).Error
}

func (r *schedulerCheckpointRepository) AcquireLease(ctx context.Context, hospitalID, departmentID uuid.UUID, leaseHolder string, leaseDuration time.Duration) (bool, error) {
	result := r.db.WithContext(ctx).Model(&entity.SchedulerCheckpoint{}).
		Where("hospital_id = ? AND department_id = ? AND (lease_expires_at IS NULL OR lease_expires_at < NOW())", hospitalID, departmentID).
		Updates(map[string]interface{}{
			"lease_holder":     leaseHolder,
			"lease_expires_at": time.Now().Add(leaseDuration),
		})
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected > 0, nil
}
