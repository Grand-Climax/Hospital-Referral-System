package usecase

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"Hospital-Referral-System/internal/domain/entity"
	iinfra "Hospital-Referral-System/internal/domain/interfaces/infrastructure"
	irepository "Hospital-Referral-System/internal/domain/interfaces/repository"
	iusecase "Hospital-Referral-System/internal/domain/interfaces/usecase"
	"Hospital-Referral-System/internal/pkg/auth"
)

var (
	ErrUserNotFound     = errors.New("user not found")
	ErrEmailExists      = errors.New("a user with this email already exists")
	ErrNationalIDExists = errors.New("a user with this national ID already exists")
	ErrInvalidRole      = errors.New("invalid user role")
)

type userUseCase struct {
	repo    irepository.UserRepository
	storage iinfra.StorageService
}

func NewUserUseCase(repo irepository.UserRepository, storage iinfra.StorageService) iusecase.UserUseCase {
	return &userUseCase{repo: repo, storage: storage}
}

var validRoles = map[entity.UserRole]bool{
	entity.RoleReferringDoctor:     true,
	entity.RoleLiaisonOfficer:      true,
	entity.RoleReceivingSpecialist: true,
	entity.RoleReceptionist:        true,
	entity.RoleMohAnalyst:          true,
	entity.RoleDeptHead:            true,
	entity.RoleSystemSuperAdmin:    true,
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

func (u *userUseCase) GetUserByID(ctx context.Context, id, requesterID uuid.UUID) (*entity.User, error) {
	requester, err := u.repo.FindByID(ctx, requesterID)
	if err != nil {
		return nil, errors.New("unauthorized: requester not found")
	}

	user, err := u.repo.FindByID(ctx, id)
	if err != nil {
		return nil, ErrUserNotFound
	}

	if user.IsDeleted {
		return nil, ErrUserNotFound
	}

	// Visibility Logic (GetUserByID)
	if !u.canSeeTarget(requester, user) {
		return nil, errors.New("forbidden: access to this profile is restricted")
	}

	return user, nil
}

func (u *userUseCase) GetMyProfile(ctx context.Context, userID uuid.UUID) (*entity.User, error) {
	user, err := u.repo.FindByID(ctx, userID)
	if err != nil || user.IsDeleted {
		return nil, ErrUserNotFound
	}
	return user, nil
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
	// Note: ProfileImage fields can be updated here or via specialized method

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

func (u *userUseCase) ListUsers(ctx context.Context, filter irepository.UserListFilter, requesterID uuid.UUID) ([]entity.User, int64, error) {
	requester, err := u.repo.FindByID(ctx, requesterID)
	if err != nil {
		return nil, 0, errors.New("unauthorized: requester not found")
	}

	// Apply Exclusion Matrix to Filter
	if requester.Role == entity.RoleMohAnalyst {
		// MoH Analyst cannot list users at all
		return []entity.User{}, 0, nil
	}

	if requester.Role == entity.RoleReceptionist {
		// Strictly localized
		hospStr := ""
		if requester.HospitalID != nil {
			hospStr = requester.HospitalID.String()
		}
		filter.HospitalID = &hospStr
	} else if requester.Role != entity.RoleSystemSuperAdmin {
		// Global roles (Doctor, Specialist, etc) see only their hospital staff by default
		// but can view specific profiles across hospitals via GetUserByID
		if requester.HospitalID != nil {
			hospStr := requester.HospitalID.String()
			filter.HospitalID = &hospStr
		}
		filter.ExcludeRoles = []entity.UserRole{entity.RoleSystemSuperAdmin}
	}

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

func (u *userUseCase) DeleteProfileImage(ctx context.Context, userID uuid.UUID) error {
	user, err := u.repo.FindByID(ctx, userID)
	if err != nil || user.IsDeleted {
		return ErrUserNotFound
	}

	// Delete from Cloudinary if PublicID exists
	if user.ProfileImagePublicID != "" {
		_ = u.storage.DeleteFile(ctx, user.ProfileImagePublicID)
	}

	user.ProfileImageURL = ""
	user.ProfileImagePublicID = ""
	return u.repo.Update(ctx, user)
}

func (u *userUseCase) ModerateProfileImage(ctx context.Context, userID, moderatorID uuid.UUID) error {
	moderator, err := u.repo.FindByID(ctx, moderatorID)
	if err != nil || moderator.IsDeleted {
		return errors.New("invalid moderator")
	}

	target, err := u.repo.FindByID(ctx, userID)
	if err != nil || target.IsDeleted {
		return ErrUserNotFound
	}

	// Scoping check for HospitalAdmin
	if moderator.Role == entity.RoleHospitalAdmin {
		if moderator.HospitalID == nil || target.HospitalID == nil || *moderator.HospitalID != *target.HospitalID {
			return errors.New("forbidden: hospital admin can only moderate their own hospital's users")
		}
	}

	return u.DeleteProfileImage(ctx, userID)
}

// canSeeTarget implements the row-level visibility matrix
func (u *userUseCase) canSeeTarget(requester, target *entity.User) bool {
	if requester.ID == target.ID {
		return true // Self access
	}

	if requester.Role == entity.RoleSystemSuperAdmin {
		return true // Global access
	}

	// MOH_ANALYST cannot view anyone else
	if requester.Role == entity.RoleMohAnalyst {
		return false
	}

	// Target is SystemAdmin: hidden from everyone except other SystemAdmins
	if target.Role == entity.RoleSystemSuperAdmin {
		return false
	}

	// Target is Receptionist: hidden from external hospitals
	if target.Role == entity.RoleReceptionist {
		if requester.HospitalID == nil || target.HospitalID == nil || *requester.HospitalID != *target.HospitalID {
			return false
		}
	}

	// Receptionist requester: strictly localized
	if requester.Role == entity.RoleReceptionist {
		if requester.HospitalID == nil || target.HospitalID == nil || *requester.HospitalID != *target.HospitalID {
			return false
		}
	}

	return true
}
