package repository

import (
	"context"

	"gorm.io/gorm"

	"Hospital-Referral-System/internal/domain/entity"
)

type UserListFilter struct {
	Page       int
	PageSize   int
	Role       *entity.UserRole
	HospitalID *string
	IsActive   *bool
	Search     *string // searches first_name, last_name, email
}

type UserRepository interface {
	BaseRepository[entity.User]
	FindByEmail(ctx context.Context, email string) (*entity.User, error)
	FindByNationalID(ctx context.Context, nationalID string) (*entity.User, error)
	ListUsers(ctx context.Context, filter UserListFilter) ([]entity.User, int64, error)
}

type userRepository struct {
	BaseRepository[entity.User]
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{
		BaseRepository: NewBaseRepository[entity.User](db),
		db:             db,
	}
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

func (r *userRepository) ListUsers(ctx context.Context, filter UserListFilter) ([]entity.User, int64, error) {
	var users []entity.User
	var total int64

	query := r.db.WithContext(ctx).Model(&entity.User{}).Where("is_deleted = false")

	if filter.Role != nil {
		query = query.Where("role = ?", *filter.Role)
	}
	if filter.HospitalID != nil {
		query = query.Where("hospital_id = ?", *filter.HospitalID)
	}
	if filter.IsActive != nil {
		query = query.Where("is_active = ?", *filter.IsActive)
	}
	if filter.Search != nil && *filter.Search != "" {
		search := "%" + *filter.Search + "%"
		query = query.Where("(first_name ILIKE ? OR last_name ILIKE ? OR email ILIKE ?)", search, search, search)
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
