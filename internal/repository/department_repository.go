package repository

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"Hospital-Referral-System/internal/domain/entity"
)

type DepartmentListFilter struct {
	Page     int
	PageSize int
	Search   *string
}

type DepartmentRepository interface {
	BaseRepository[entity.Department]
	ListDepartments(ctx context.Context, filter DepartmentListFilter) ([]entity.Department, int64, error)
	LinkToHospital(ctx context.Context, link *entity.HospitalDepartment) error
	UnlinkFromHospital(ctx context.Context, hospitalID, departmentID uuid.UUID) error
	ListHospitalDepartments(ctx context.Context, hospitalID uuid.UUID) ([]entity.HospitalDepartment, error)
	FindHospitalDepartment(ctx context.Context, hospitalID, departmentID uuid.UUID) (*entity.HospitalDepartment, error)
}

type departmentRepository struct {
	BaseRepository[entity.Department]
	db *gorm.DB
}

func NewDepartmentRepository(db *gorm.DB) DepartmentRepository {
	return &departmentRepository{
		BaseRepository: NewBaseRepository[entity.Department](db),
		db:             db,
	}
}

func (r *departmentRepository) ListDepartments(ctx context.Context, filter DepartmentListFilter) ([]entity.Department, int64, error) {
	var departments []entity.Department
	var total int64

	query := r.db.WithContext(ctx).Model(&entity.Department{})

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
	if err := query.Order("name ASC").Offset(offset).Limit(pageSize).Find(&departments).Error; err != nil {
		return nil, 0, err
	}

	return departments, total, nil
}

func (r *departmentRepository) LinkToHospital(ctx context.Context, link *entity.HospitalDepartment) error {
	return r.db.WithContext(ctx).Create(link).Error
}

func (r *departmentRepository) UnlinkFromHospital(ctx context.Context, hospitalID, departmentID uuid.UUID) error {
	return r.db.WithContext(ctx).
		Where("hospital_id = ? AND department_id = ?", hospitalID, departmentID).
		Delete(&entity.HospitalDepartment{}).Error
}

func (r *departmentRepository) ListHospitalDepartments(ctx context.Context, hospitalID uuid.UUID) ([]entity.HospitalDepartment, error) {
	var links []entity.HospitalDepartment
	err := r.db.WithContext(ctx).
		Preload("Department").
		Where("hospital_id = ?", hospitalID).
		Find(&links).Error
	return links, err
}

func (r *departmentRepository) FindHospitalDepartment(ctx context.Context, hospitalID, departmentID uuid.UUID) (*entity.HospitalDepartment, error) {
	var link entity.HospitalDepartment
	err := r.db.WithContext(ctx).
		Where("hospital_id = ? AND department_id = ?", hospitalID, departmentID).
		First(&link).Error
	if err != nil {
		return nil, err
	}
	return &link, nil
}
