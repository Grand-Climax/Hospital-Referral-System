package repository

import (
	"context"

	"gorm.io/gorm"

	"Hospital-Referral-System/internal/domain/entity"
	irepository "Hospital-Referral-System/internal/domain/interfaces/repository"
)

type hospitalRepository struct {
	*BaseRepository[entity.Hospital]
	db *gorm.DB
}

func NewHospitalRepository(db *gorm.DB) irepository.HospitalRepository {
	return &hospitalRepository{
		BaseRepository: NewBaseRepository[entity.Hospital](db),
		db:             db,
	}
}

func (r *hospitalRepository) ListHospitals(ctx context.Context, filter irepository.HospitalListFilter) ([]entity.Hospital, int64, error) {
	var hospitals []entity.Hospital
	var total int64

	query := r.db.WithContext(ctx).Model(&entity.Hospital{}).Where("is_deleted = false")

	if filter.Tier != nil {
		query = query.Where("tier_level = ?", *filter.Tier)
	}
	if filter.Region != nil && *filter.Region != "" {
		query = query.Where("region = ?", *filter.Region)
	}
	if filter.IsActive != nil {
		query = query.Where("is_active = ?", *filter.IsActive)
	}
	if filter.Search != nil && *filter.Search != "" {
		search := "%" + *filter.Search + "%"
		query = query.Where("name ILIKE ?", search)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	page := filter.Page
	if page < 1 {
		page = 1
	}
	pageSize := filter.PageSize
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	offset := (page - 1) * pageSize
	if err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&hospitals).Error; err != nil {
		return nil, 0, err
	}

	return hospitals, total, nil
}
