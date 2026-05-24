package repository

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"Hospital-Referral-System/internal/domain/entity"
	irepository "Hospital-Referral-System/internal/domain/interfaces/repository"
)

type userRepository struct {
	*BaseRepository[entity.User]
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) irepository.UserRepository {
	return &userRepository{
		BaseRepository: NewBaseRepository[entity.User](db),
		db:             db,
	}
}

func (r *userRepository) FindByID(ctx context.Context, id interface{}) (*entity.User, error) {
	var user entity.User
	err := r.db.WithContext(ctx).
		Preload("Hospital").
		Preload("Department").
		First(&user, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *userRepository) FindByEmail(ctx context.Context, email string) (*entity.User, error) {
	var user entity.User
	err := r.db.WithContext(ctx).Where("email = ? AND is_deleted = false", email).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *userRepository) FindByNationalID(ctx context.Context, nationalID string) (*entity.User, error) {
	var user entity.User
	err := r.db.WithContext(ctx).Where("national_id = ? AND is_deleted = false", nationalID).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *userRepository) ListUsers(ctx context.Context, filter irepository.UserListFilter) ([]entity.User, int64, error) {
	var users []entity.User
	var total int64

	query := r.db.WithContext(ctx).Model(&entity.User{}).Where("is_deleted = false")

	if len(filter.Roles) > 0 {
		query = query.Where("role IN ?", filter.Roles)
	} else if filter.Role != nil {
		query = query.Where("role = ?", *filter.Role)
	}
	if filter.HospitalID != nil {
		query = query.Where("hospital_id = ?", *filter.HospitalID)
	}
	if filter.DepartmentID != nil {
		query = query.Where("department_id = ?", *filter.DepartmentID)
	}
	if filter.Email != nil {
		query = query.Where("email ILIKE ?", "%"+*filter.Email+"%")
	}
	if filter.IsActive != nil {
		query = query.Where("is_active = ?", *filter.IsActive)
	}
	if len(filter.ExcludeRoles) > 0 {
		query = query.Where("role NOT IN ?", filter.ExcludeRoles)
	}
	if filter.ExcludeOtherReceptionists != nil {
		query = query.Where("(role != ? OR hospital_id = ?)", entity.RoleReceptionist, *filter.ExcludeOtherReceptionists)
	}
	if filter.Search != nil && *filter.Search != "" {
		search := "%" + *filter.Search + "%"
		query = query.Where("(first_name ILIKE ? OR last_name ILIKE ? OR email ILIKE ?)", search, search, search)
	}
	if filter.Name != nil && *filter.Name != "" {
		tokens := strings.Fields(*filter.Name)
		for _, token := range tokens {
			t := "%" + token + "%"
			query = query.Where("(first_name ILIKE ? OR middle_name ILIKE ? OR last_name ILIKE ?)", t, t, t)
		}
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
	if err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&users).Error; err != nil {
		return nil, 0, err
	}

	return users, total, nil
}

func (r *userRepository) Create(ctx context.Context, user *entity.User) error {
	err := r.db.WithContext(ctx).Create(user).Error
	if err != nil {
		if strings.Contains(err.Error(), "unique constraint") || strings.Contains(err.Error(), "duplicate key") {
			if strings.Contains(err.Error(), "users_email_key") || strings.Contains(err.Error(), "uni_users_email") {
				return entity.ErrEmailAlreadyExists
			}
			if strings.Contains(err.Error(), "users_national_id_key") || strings.Contains(err.Error(), "uni_users_national_id") {
				return entity.ErrNationalIDAlreadyExists
			}
		}
		return err
	}
	return nil
}

func (r *userRepository) CreateStaffReplacementLog(ctx context.Context, log *entity.StaffReplacementLog) error {
	return r.db.WithContext(ctx).Create(log).Error
}

func (r *userRepository) Update(ctx context.Context, user *entity.User) error {
	user.Hospital = nil
	user.Department = nil
	return r.db.WithContext(ctx).Omit("Hospital", "Department").Save(user).Error
}

func (r *userRepository) CountHospitalStaffByStatus(ctx context.Context, hospitalID uuid.UUID) (total int64, active int64, inactive int64, err error) {
	err = r.db.WithContext(ctx).Model(&entity.User{}).
		Where("hospital_id = ? AND is_deleted = false", hospitalID).
		Count(&total).Error
	if err != nil {
		return 0, 0, 0, err
	}

	err = r.db.WithContext(ctx).Model(&entity.User{}).
		Where("hospital_id = ? AND is_deleted = false AND is_active = true", hospitalID).
		Count(&active).Error
	if err != nil {
		return 0, 0, 0, err
	}

	err = r.db.WithContext(ctx).Model(&entity.User{}).
		Where("hospital_id = ? AND is_deleted = false AND is_active = false", hospitalID).
		Count(&inactive).Error
	if err != nil {
		return 0, 0, 0, err
	}

	return total, active, inactive, nil
}

