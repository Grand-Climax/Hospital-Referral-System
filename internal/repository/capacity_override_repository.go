package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"Hospital-Referral-System/internal/domain/entity"
	irepository "Hospital-Referral-System/internal/domain/interfaces/repository"
)

type capacityOverrideRepository struct {
	*BaseRepository[entity.CapacityOverride]
	db *gorm.DB
}

func NewCapacityOverrideRepository(db *gorm.DB) irepository.CapacityOverrideRepository {
	return &capacityOverrideRepository{
		BaseRepository: NewBaseRepository[entity.CapacityOverride](db),
		db:             db,
	}
}

func (r *capacityOverrideRepository) GetActive(ctx context.Context, hospitalID, deptID uuid.UUID, date time.Time) (*entity.CapacityOverride, error) {
	var override entity.CapacityOverride
	err := r.db.WithContext(ctx).
		Where("hospital_id = ? AND department_id = ? AND target_date = ? AND is_active = ?", hospitalID, deptID, date.Format("2006-01-02"), true).
		First(&override).Error
	if err != nil {
		return nil, err
	}
	return &override, nil
}

func (r *capacityOverrideRepository) ListByDept(ctx context.Context, hospitalID, deptID uuid.UUID) ([]entity.CapacityOverride, error) {
	var overrides []entity.CapacityOverride
	err := r.db.WithContext(ctx).
		Where("hospital_id = ? AND department_id = ? AND is_active = ?", hospitalID, deptID, true).
		Order("target_date DESC").
		Find(&overrides).Error
	return overrides, err
}

func (r *capacityOverrideRepository) ListByDeptAndYearMonth(ctx context.Context, hospitalID, deptID uuid.UUID, year, month int) ([]entity.CapacityOverride, error) {
	var overrides []entity.CapacityOverride
	query := r.db.WithContext(ctx).
		Where("hospital_id = ? AND department_id = ?", hospitalID, deptID)
	if year > 0 {
		query = query.Where("EXTRACT(YEAR FROM target_date) = ?", year)
	}
	if month > 0 {
		query = query.Where("EXTRACT(MONTH FROM target_date) = ?", month)
	}
	err := query.Order("target_date ASC").Find(&overrides).Error
	return overrides, err
}
