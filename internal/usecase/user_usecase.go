package usecase

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"Hospital-Referral-System/internal/domain/entity"
	"Hospital-Referral-System/internal/pkg/auth"
	"Hospital-Referral-System/internal/repository"
)

var (
	ErrUserNotFound     = errors.New("user not found")
	ErrEmailExists      = errors.New("a user with this email already exists")
	ErrNationalIDExists = errors.New("a user with this national ID already exists")
	ErrInvalidRole      = errors.New("invalid user role")
)

type UserUseCase interface {
	CreateUser(ctx context.Context, user *entity.User, rawPassword string) error
	GetUserByID(ctx context.Context, id uuid.UUID) (*entity.User, error)
	GetMyProfile(ctx context.Context, userID uuid.UUID) (*entity.User, error)
	UpdateUser(ctx context.Context, user *entity.User) error
	DeleteUser(ctx context.Context, id uuid.UUID) error
	ListUsers(ctx context.Context, filter repository.UserListFilter) ([]entity.User, int64, error)
	AssignRole(ctx context.Context, userID uuid.UUID, role entity.UserRole) error
}

type userUseCase struct {
	repo repository.UserRepository
}

func NewUserUseCase(repo repository.UserRepository) UserUseCase {
	return &userUseCase{repo: repo}
}

var validRoles = map[entity.UserRole]bool{
	entity.RoleReferringDoctor:     true,
	entity.RoleLiaisonOfficer:      true,
	entity.RoleReceivingSpecialist: true,
	entity.RoleReceptionist:        true,
	entity.RoleMohAnalyst:          true,
	entity.RoleDeptHead:            true,
	entity.RoleSystemAdmin:         true,
}

func (u *userUseCase) CreateUser(ctx context.Context, user *entity.User, rawPassword string) error {
	// Validate role
	if !validRoles[user.Role] {
		return ErrInvalidRole
	}

	// Check email uniqueness
	existing, _ := u.repo.FindByEmail(ctx, user.Email)
	if existing != nil {
		return ErrEmailExists
	}

	// Check national ID uniqueness if provided
	if user.NationalID != "" {
		existingNID, _ := u.repo.FindByNationalID(ctx, user.NationalID)
		if existingNID != nil {
			return ErrNationalIDExists
		}
	}

	// Hash password
	hash, err := auth.HashPassword(rawPassword)
	if err != nil {
		return err
	}
	user.PasswordHash = hash
	user.IsActive = true
	user.IsDeleted = false

	return u.repo.Create(ctx, user)
}

func (u *userUseCase) GetUserByID(ctx context.Context, id uuid.UUID) (*entity.User, error) {
	user, err := u.repo.FindByID(ctx, id)
	if err != nil {
		return nil, ErrUserNotFound
	}
	if user.IsDeleted {
		return nil, ErrUserNotFound
	}
	return user, nil
}

func (u *userUseCase) GetMyProfile(ctx context.Context, userID uuid.UUID) (*entity.User, error) {
	return u.GetUserByID(ctx, userID)
}

func (u *userUseCase) UpdateUser(ctx context.Context, user *entity.User) error {
	existing, err := u.repo.FindByID(ctx, user.ID)
	if err != nil || existing.IsDeleted {
		return ErrUserNotFound
	}

	// Preserve immutable fields
	user.PasswordHash = existing.PasswordHash
	user.CreatedAt = existing.CreatedAt
	user.IsDeleted = existing.IsDeleted

	return u.repo.Update(ctx, user)
}

func (u *userUseCase) DeleteUser(ctx context.Context, id uuid.UUID) error {
	user, err := u.repo.FindByID(ctx, id)
	if err != nil {
		return ErrUserNotFound
	}
	user.IsDeleted = true
	user.IsActive = false
	return u.repo.Update(ctx, user)
}

func (u *userUseCase) ListUsers(ctx context.Context, filter repository.UserListFilter) ([]entity.User, int64, error) {
	return u.repo.ListUsers(ctx, filter)
}

func (u *userUseCase) AssignRole(ctx context.Context, userID uuid.UUID, role entity.UserRole) error {
	if !validRoles[role] {
		return ErrInvalidRole
	}

	user, err := u.repo.FindByID(ctx, userID)
	if err != nil || user.IsDeleted {
		return ErrUserNotFound
	}

	user.Role = role
	return u.repo.Update(ctx, user)
}
