package repository

import (
	"context"

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
