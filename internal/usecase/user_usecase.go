package usecase

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"

	"Hospital-Referral-System/internal/domain/entity"
	iinfra "Hospital-Referral-System/internal/domain/interfaces/infrastructure"
	irepository "Hospital-Referral-System/internal/domain/interfaces/repository"
	iusecase "Hospital-Referral-System/internal/domain/interfaces/usecase"
	"Hospital-Referral-System/internal/infrastructure/cache"
	"Hospital-Referral-System/internal/pkg/auth"
)

var (
	ErrUserNotFound        = errors.New("user not found")
	ErrEmailExists         = errors.New("a user with this email already exists")
	ErrNationalIDExists    = errors.New("a user with this national ID already exists")
	ErrInvalidRole         = errors.New("invalid user role")
	ErrInvalidDepartment   = errors.New("invalid department_id")
	ErrForbiddenStaffScope = errors.New("forbidden: hospital admin can only manage staff in their own hospital")
	ErrInvalidAdminScope   = errors.New("forbidden: hospital admin must belong to a hospital")
	ErrCannotManageUser    = errors.New("forbidden: this user cannot be managed by hospital admin")
	ErrCannotManageSelf    = errors.New("forbidden: hospital admin cannot perform this operation on their own account")
	ErrSecurityUnavailable = errors.New("security subsystem is unavailable")
)

type userUseCase struct {
	repo     irepository.UserRepository
	storage  iinfra.StorageService
	authRepo     irepository.AuthRepository
	sessions     cache.SessionStore
	inAppNotifUC iusecase.InAppNotificationUseCase
}

func NewUserUseCase(repo irepository.UserRepository, storage iinfra.StorageService, inAppNotifUC iusecase.InAppNotificationUseCase) iusecase.UserUseCase {
	return &userUseCase{repo: repo, storage: storage, inAppNotifUC: inAppNotifUC}
}

func NewUserUseCaseWithSecurity(repo irepository.UserRepository, storage iinfra.StorageService, authRepo irepository.AuthRepository, sessions cache.SessionStore, inAppNotifUC iusecase.InAppNotificationUseCase) iusecase.UserUseCase {
	return &userUseCase{
		repo:     repo,
		storage:  storage,
		authRepo:     authRepo,
		sessions:     sessions,
		inAppNotifUC: inAppNotifUC,
	}
}

var validRoles = map[entity.UserRole]bool{
	entity.RoleReferringDoctor:     true,
	entity.RoleLiaisonOfficer:      true,
	entity.RoleReceivingSpecialist: true,
	entity.RoleReceptionist:        true,
	entity.RoleMohAnalyst:          true,
	entity.RoleDeptHead:            true,
	entity.RoleHospitalAdmin:       true,
	entity.RoleSystemSuperAdmin:    true,
}

var hospitalAdminManageableRoles = map[entity.UserRole]bool{
	entity.RoleReferringDoctor:     true,
	entity.RoleLiaisonOfficer:      true,
	entity.RoleReceivingSpecialist: true,
	entity.RoleReceptionist:        true,
	entity.RoleDeptHead:            true,
	entity.RoleHospitalAdmin:       true,
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

	if err := u.repo.Create(ctx, user); err != nil {
		// Normalize DB FK errors into a client-facing validation error.
		if strings.Contains(err.Error(), "fk_users_department") {
			return ErrInvalidDepartment
		}
		return err
	}

	_ = u.inAppNotifUC.CreateForEvent(ctx, "STAFF_ADDED", uuid.Nil, uuid.Nil)

	return nil
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

	// Email uniqueness check if changed
	if user.Email != existing.Email {
		existingEmail, _ := u.repo.FindByEmail(ctx, user.Email)
		if existingEmail != nil && existingEmail.ID != user.ID {
			return ErrEmailExists
		}
	}

	// National ID uniqueness check if changed
	if user.NationalID != "" && user.NationalID != existing.NationalID {
		existingNID, _ := u.repo.FindByNationalID(ctx, user.NationalID)
		if existingNID != nil && existingNID.ID != user.ID {
			return ErrNationalIDExists
		}
	}

	// Preserve immutable fields
	if user.PasswordHash == "" {
		user.PasswordHash = existing.PasswordHash
	}
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
	err = u.repo.Update(ctx, user)
	if err == nil {
		_ = u.inAppNotifUC.CreateForEvent(ctx, "ROLE_CHANGED", uuid.Nil, uuid.Nil)
	}
	return err
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

func (u *userUseCase) UpdateProfileImage(ctx context.Context, userID uuid.UUID, file interface{}) error {
	user, err := u.repo.FindByID(ctx, userID)
	if err != nil || user.IsDeleted {
		return ErrUserNotFound
	}

	// 1. Upload new image to "profile_images" folder
	url, publicID, err := u.storage.UploadFile(ctx, file, "profile_images")
	if err != nil {
		return err
	}

	// 2. Delete old image if a PublicID exists
	if user.ProfileImagePublicID != "" {
		_ = u.storage.DeleteFile(ctx, user.ProfileImagePublicID)
	}

	// 3. Update user record
	user.ProfileImageURL = url
	user.ProfileImagePublicID = publicID

	return u.repo.Update(ctx, user)
}

func (u *userUseCase) hospitalAdminContext(ctx context.Context, adminID uuid.UUID) (*entity.User, uuid.UUID, error) {
	admin, err := u.repo.FindByID(ctx, adminID)
	if err != nil || admin.IsDeleted {
		return nil, uuid.Nil, ErrUserNotFound
	}
	if admin.Role != entity.RoleHospitalAdmin {
		return nil, uuid.Nil, ErrForbiddenStaffScope
	}
	if admin.HospitalID == nil {
		return nil, uuid.Nil, ErrInvalidAdminScope
	}
	return admin, *admin.HospitalID, nil
}

func (u *userUseCase) validateHospitalAdminTarget(admin *entity.User, target *entity.User) error {
	if target.IsDeleted {
		return ErrUserNotFound
	}
	if admin.ID == target.ID {
		return ErrCannotManageSelf
	}
	if target.HospitalID == nil || admin.HospitalID == nil || *target.HospitalID != *admin.HospitalID {
		return ErrForbiddenStaffScope
	}
	if !hospitalAdminManageableRoles[target.Role] {
		return ErrCannotManageUser
	}
	return nil
}

func (u *userUseCase) HospitalAdminCreateStaff(ctx context.Context, adminID uuid.UUID, user *entity.User, rawPassword string) error {
	_, adminHospID, err := u.hospitalAdminContext(ctx, adminID)
	if err != nil {
		return err
	}
	if !hospitalAdminManageableRoles[user.Role] {
		return ErrInvalidRole
	}

	// Enforce strict same-hospital creation.
	user.HospitalID = &adminHospID
	return u.CreateUser(ctx, user, rawPassword)
}

func (u *userUseCase) HospitalAdminListStaff(ctx context.Context, adminID uuid.UUID, filter irepository.UserListFilter) ([]entity.User, int64, error) {
	_, adminHospID, err := u.hospitalAdminContext(ctx, adminID)
	if err != nil {
		return nil, 0, err
	}

	hospStr := adminHospID.String()
	filter.HospitalID = &hospStr
	filter.ExcludeRoles = []entity.UserRole{entity.RoleSystemSuperAdmin, entity.RoleMohAnalyst}
	users, _, err := u.repo.ListUsers(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	// Defensive filtering in case repository exclusion rules change.
	filtered := make([]entity.User, 0, len(users))
	for _, usr := range users {
		if usr.HospitalID == nil || *usr.HospitalID != adminHospID {
			continue
		}
		if !hospitalAdminManageableRoles[usr.Role] {
			continue
		}
		filtered = append(filtered, usr)
	}
	return filtered, int64(len(filtered)), nil
}

func (u *userUseCase) HospitalAdminGetStaffByID(ctx context.Context, adminID, staffID uuid.UUID) (*entity.User, error) {
	admin, _, err := u.hospitalAdminContext(ctx, adminID)
	if err != nil {
		return nil, err
	}
	target, err := u.repo.FindByID(ctx, staffID)
	if err != nil {
		return nil, ErrUserNotFound
	}
	if err := u.validateHospitalAdminTarget(admin, target); err != nil {
		return nil, err
	}
	return target, nil
}

func (u *userUseCase) HospitalAdminChangeStaffRole(ctx context.Context, adminID, staffID uuid.UUID, role entity.UserRole) error {
	if !hospitalAdminManageableRoles[role] {
		return ErrInvalidRole
	}

	admin, _, err := u.hospitalAdminContext(ctx, adminID)
	if err != nil {
		return err
	}
	target, err := u.repo.FindByID(ctx, staffID)
	if err != nil {
		return ErrUserNotFound
	}
	if err := u.validateHospitalAdminTarget(admin, target); err != nil {
		return err
	}

	target.Role = role
	err = u.repo.Update(ctx, target)
	if err == nil {
		_ = u.inAppNotifUC.CreateForEvent(ctx, "STAFF_ROLE_CHANGED", uuid.Nil, adminID)
	}
	return err
}

func (u *userUseCase) HospitalAdminSoftDeleteStaff(ctx context.Context, adminID, staffID uuid.UUID) error {
	admin, _, err := u.hospitalAdminContext(ctx, adminID)
	if err != nil {
		return err
	}
	target, err := u.repo.FindByID(ctx, staffID)
	if err != nil {
		return ErrUserNotFound
	}
	if err := u.validateHospitalAdminTarget(admin, target); err != nil {
		return err
	}

	target.IsDeleted = true
	target.IsActive = false
	if err := u.repo.Update(ctx, target); err != nil {
		return err
	}
	_, err = u.revokeUserSessions(ctx, target.ID)
	return err
}

func (u *userUseCase) HospitalAdminSetStaffActive(ctx context.Context, adminID, staffID uuid.UUID, isActive bool) error {
	admin, _, err := u.hospitalAdminContext(ctx, adminID)
	if err != nil {
		return err
	}

	target, err := u.repo.FindByID(ctx, staffID)
	if err != nil {
		return ErrUserNotFound
	}
	if err := u.validateHospitalAdminTarget(admin, target); err != nil {
		return err
	}

	target.IsActive = isActive
	if err := u.repo.Update(ctx, target); err != nil {
		return err
	}

	if !isActive {
		_, err = u.revokeUserSessions(ctx, target.ID)
		if err != nil {
			return err
		}
	}
	return nil
}

func (u *userUseCase) HospitalAdminReassignStaffDepartment(ctx context.Context, adminID, staffID uuid.UUID, departmentID *uuid.UUID) error {
	admin, _, err := u.hospitalAdminContext(ctx, adminID)
	if err != nil {
		return err
	}
	target, err := u.repo.FindByID(ctx, staffID)
	if err != nil {
		return ErrUserNotFound
	}
	if err := u.validateHospitalAdminTarget(admin, target); err != nil {
		return err
	}

	target.DepartmentID = departmentID
	if err := u.repo.Update(ctx, target); err != nil {
		if strings.Contains(err.Error(), "fk_users_department") {
			return ErrInvalidDepartment
		}
		return err
	}
	return nil
}

func (u *userUseCase) HospitalAdminListActiveStaffSessions(ctx context.Context, adminID uuid.UUID, filter iusecase.HospitalAdminSessionFilter) ([]entity.Session, int64, error) {
	_, adminHospID, err := u.hospitalAdminContext(ctx, adminID)
	if err != nil {
		return nil, 0, err
	}
	if u.authRepo == nil {
		return nil, 0, ErrSecurityUnavailable
	}
	return u.authRepo.ListActiveSessionsByHospital(ctx, adminHospID, filter.StaffID, filter.Page, filter.PageSize)
}

func (u *userUseCase) HospitalAdminForceLogoutStaff(ctx context.Context, adminID, staffID uuid.UUID) (int64, error) {
	admin, _, err := u.hospitalAdminContext(ctx, adminID)
	if err != nil {
		return 0, err
	}
	target, err := u.repo.FindByID(ctx, staffID)
	if err != nil {
		return 0, ErrUserNotFound
	}
	if err := u.validateHospitalAdminTarget(admin, target); err != nil {
		return 0, err
	}
	return u.revokeUserSessions(ctx, target.ID)
}

func (u *userUseCase) revokeUserSessions(ctx context.Context, userID uuid.UUID) (int64, error) {
	if u.authRepo == nil {
		return 0, nil
	}

	activeSessions, err := u.authRepo.ListActiveSessionsByUser(ctx, userID)
	if err != nil {
		return 0, err
	}

	revoked, err := u.authRepo.RevokeActiveSessionsByUser(ctx, userID)
	if err != nil {
		return 0, err
	}

	if u.sessions != nil {
		for _, sess := range activeSessions {
			_ = u.sessions.DeleteSession(ctx, sess.RefreshTokenHash)
		}
	}
	return revoked, nil
}

func (u *userUseCase) HospitalAdminReplaceStaff(ctx context.Context, adminID, staffID uuid.UUID, input iusecase.HospitalAdminReplacementInput) error {
	admin, adminHospID, err := u.hospitalAdminContext(ctx, adminID)
	if err != nil {
		return err
	}
	target, err := u.repo.FindByID(ctx, staffID)
	if err != nil {
		return ErrUserNotFound
	}
	if err := u.validateHospitalAdminTarget(admin, target); err != nil {
		return err
	}

	if existing, err := u.repo.FindByEmail(ctx, input.Email); err == nil && existing != nil && existing.ID != target.ID {
		return ErrEmailExists
	}

	hash, err := auth.HashPassword(input.Password)
	if err != nil {
		return err
	}

	oldEmail := target.Email
	target.FirstName = input.FirstName
	target.MiddleName = input.MiddleName
	target.LastName = input.LastName
	target.Email = input.Email
	target.PasswordHash = hash

	if err := u.repo.Update(ctx, target); err != nil {
		return err
	}

	return u.repo.CreateStaffReplacementLog(ctx, &entity.StaffReplacementLog{
		HospitalID:        adminHospID,
		UserID:            target.ID,
		RoleAtReplacement: target.Role,
		OldEmail:          oldEmail,
		NewEmail:          target.Email,
		Reason:            input.Reason,
		ReplacedByAdminID: admin.ID,
	})
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
