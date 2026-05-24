package repository

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"Hospital-Referral-System/internal/domain/entity"
	irepository "Hospital-Referral-System/internal/domain/interfaces/repository"
)

type departmentRepository struct {
	*BaseRepository[entity.Department]
	db *gorm.DB
}

func NewDepartmentRepository(db *gorm.DB) irepository.DepartmentRepository {
	return &departmentRepository{
		BaseRepository: NewBaseRepository[entity.Department](db),
		db:             db,
	}
}

func (r *departmentRepository) ListDepartments(ctx context.Context, filter irepository.DepartmentListFilter) ([]entity.Department, int64, error) {
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

func (r *departmentRepository) UpdateHospitalDepartment(ctx context.Context, link *entity.HospitalDepartment) error {
	return r.db.WithContext(ctx).Save(link).Error
}

// UpdateStaffCapacity is a narrow update path used by the soft-hint
// PUT /staff-capacity endpoint. It avoids loading + Save (which would
// touch every column) so concurrent capacity-related writes do not
// stomp each other.
func (r *departmentRepository) UpdateStaffCapacity(ctx context.Context, hospitalID, departmentID uuid.UUID, value int) error {
	return r.db.WithContext(ctx).
		Model(&entity.HospitalDepartment{}).
		Where("hospital_id = ? AND department_id = ?", hospitalID, departmentID).
		Update("max_capacity_of_staff", value).Error
}
