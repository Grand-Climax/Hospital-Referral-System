package interfaces

import (
	"context"

	"github.com/google/uuid"

	"Hospital-Referral-System/internal/domain/entity"
)

type UserListFilter struct {
	Page                      int
	PageSize                  int
	Role                      *entity.UserRole
	HospitalID                *string
	DepartmentID              *string
	Email                     *string
	IsActive                  *bool
	Search                    *string // searches first_name, last_name, email (legacy)
	Name                      *string // specific tokenized name search
	ExcludeRoles              []entity.UserRole
	ExcludeOtherReceptionists *string // HospitalID to keep (others are excluded)
}

type UserRepository interface {
	BaseRepository[entity.User]
	FindByEmail(ctx context.Context, email string) (*entity.User, error)
	FindByNationalID(ctx context.Context, nationalID string) (*entity.User, error)
	ListUsers(ctx context.Context, filter UserListFilter) ([]entity.User, int64, error)
	CreateStaffReplacementLog(ctx context.Context, log *entity.StaffReplacementLog) error
	CountHospitalStaffByStatus(ctx context.Context, hospitalID uuid.UUID) (total int64, active int64, inactive int64, err error)
}
