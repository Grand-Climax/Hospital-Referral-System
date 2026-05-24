package interfaces

import (
	"context"

	"github.com/google/uuid"

	"Hospital-Referral-System/internal/domain/entity"
)

type DepartmentListFilter struct {
	Page     int
	PageSize int
	Search   *string
}

type DepartmentRepository interface {
	BaseRepository[entity.Department]
	ListDepartments(ctx context.Context, filter DepartmentListFilter) ([]entity.Department, int64, error)
	LinkToHospital(ctx context.Context, link *entity.HospitalDepartment) error
	UnlinkFromHospital(ctx context.Context, hospitalID, departmentID uuid.UUID) error
	ListHospitalDepartments(ctx context.Context, hospitalID uuid.UUID) ([]entity.HospitalDepartment, error)
	FindHospitalDepartment(ctx context.Context, hospitalID, departmentID uuid.UUID) (*entity.HospitalDepartment, error)
	UpdateHospitalDepartment(ctx context.Context, link *entity.HospitalDepartment) error

	// UpdateStaffCapacity sets HospitalDepartment.MaxCapacityOfStaff to
	// the given value. This is a soft hint surfaced through the capacity
	// detail view; no booking rule depends on it.
	UpdateStaffCapacity(ctx context.Context, hospitalID, departmentID uuid.UUID, value int) error

	// UpdateDailyCapacity persists the new baseline daily capacity for
	// the (hospital, department) pair. It writes standard_daily_limit and
	// overbook_limit together so the engine never sees a half-applied
	// update where one column is at the new value and the other is stale.
	// Existing CapacityOverride and DailySchedule rows are not touched -
	// the change only affects future getEffectiveCapacity calls for dates
	// without an active override.
	UpdateDailyCapacity(ctx context.Context, hospitalID, departmentID uuid.UUID, standardDailyLimit, overbookLimit int) error
}
