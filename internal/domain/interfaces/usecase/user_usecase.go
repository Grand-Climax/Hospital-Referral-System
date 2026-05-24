package interfaces

import (
	"context"

	"github.com/google/uuid"

	"Hospital-Referral-System/internal/delivery/http/dto"
	"Hospital-Referral-System/internal/domain/entity"
	irepository "Hospital-Referral-System/internal/domain/interfaces/repository"
)

type HospitalAdminReplacementInput struct {
	FirstName  string
	MiddleName string
	LastName   string
	Email      string
	Password   string
	Reason     string
}

type HospitalAdminSessionFilter struct {
	StaffID  *uuid.UUID
	Page     int
	PageSize int
}

type UserUseCase interface {
	CreateUser(ctx context.Context, user *entity.User, rawPassword string) error
	GetUserByID(ctx context.Context, id, requesterID uuid.UUID) (*entity.User, error)
	GetMyProfile(ctx context.Context, userID uuid.UUID) (*entity.User, error)
	UpdateUser(ctx context.Context, user *entity.User) error
	DeleteUser(ctx context.Context, id uuid.UUID) error
	ListUsers(ctx context.Context, filter irepository.UserListFilter, requesterID uuid.UUID) ([]entity.User, int64, error)

	// ListDepartmentStaff returns the REFERRING_DOCTOR and RECEPTIONIST
	// users assigned to the given (hospital, department). Used by the
	// dept-scoped UIs to populate staff dropdowns.
	ListDepartmentStaff(ctx context.Context, hospitalID, deptID uuid.UUID) ([]entity.User, error)
	AssignRole(ctx context.Context, userID uuid.UUID, role entity.UserRole) error
	DeleteProfileImage(ctx context.Context, userID uuid.UUID) error
	ModerateProfileImage(ctx context.Context, userID, moderatorID uuid.UUID) error
	UpdateProfileImage(ctx context.Context, userID uuid.UUID, file interface{}) error

	// Hospital admin staff management
	HospitalAdminCreateStaff(ctx context.Context, adminID uuid.UUID, user *entity.User, rawPassword string) error
	HospitalAdminListStaff(ctx context.Context, adminID uuid.UUID, filter irepository.UserListFilter) ([]entity.User, int64, error)
	HospitalAdminGetStaffByID(ctx context.Context, adminID, staffID uuid.UUID) (*entity.User, error)
	HospitalAdminChangeStaffRole(ctx context.Context, adminID, staffID uuid.UUID, role entity.UserRole) error
	HospitalAdminSoftDeleteStaff(ctx context.Context, adminID, staffID uuid.UUID) error
	HospitalAdminReplaceStaff(ctx context.Context, adminID, staffID uuid.UUID, input HospitalAdminReplacementInput) error
	HospitalAdminSetStaffActive(ctx context.Context, adminID, staffID uuid.UUID, isActive bool) error
	HospitalAdminReassignStaffDepartment(ctx context.Context, adminID, staffID uuid.UUID, departmentID *uuid.UUID) error
	HospitalAdminListActiveStaffSessions(ctx context.Context, adminID uuid.UUID, filter HospitalAdminSessionFilter) ([]entity.Session, int64, error)
	HospitalAdminForceLogoutStaff(ctx context.Context, adminID, staffID uuid.UUID) (int64, error)
	GetPersonnelWidgetStats(ctx context.Context, adminID uuid.UUID) (*dto.HospitalAdminPersonnelWidgetResponse, error)
}
