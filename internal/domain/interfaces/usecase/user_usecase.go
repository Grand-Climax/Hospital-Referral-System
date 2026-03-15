package usecase

import (
	"context"

	"github.com/google/uuid"

	"Hospital-Referral-System/internal/domain/entity"
	irepository "Hospital-Referral-System/internal/domain/interfaces/repository"
)

type UserUseCase interface {
	CreateUser(ctx context.Context, user *entity.User, rawPassword string) error
	GetUserByID(ctx context.Context, id uuid.UUID) (*entity.User, error)
	GetMyProfile(ctx context.Context, userID uuid.UUID) (*entity.User, error)
	UpdateUser(ctx context.Context, user *entity.User) error
	DeleteUser(ctx context.Context, id uuid.UUID) error
	ListUsers(ctx context.Context, filter irepository.UserListFilter) ([]entity.User, int64, error)
	AssignRole(ctx context.Context, userID uuid.UUID, role entity.UserRole) error
}
