package interfaces

import (
	"context"
	"time"

	"github.com/google/uuid"

	"Hospital-Referral-System/internal/delivery/http/dto"
	"Hospital-Referral-System/internal/domain/entity"
)

// CapacityManagementUseCase is the surface exposed to Department Head
// flows. Under the Schedule-on-Demand model overrides are IMMUTABLE - to
// change a future date, the existing override must be deleted and a new
// one created at least buffer_days+1 ahead of today.
type CapacityManagementUseCase interface {
	// GetSchedule returns the immutable DailySchedule history log for the
	// inclusive date range. Useful for a calendar / audit view; capacity
	// for the future is derived live elsewhere.
	GetSchedule(ctx context.Context, hospitalID, deptID uuid.UUID, startDate, endDate time.Time) ([]entity.DailySchedule, error)

	// GetOverrides lists every CapacityOverride for the department,
	// active and inactive (history).
	GetOverrides(ctx context.Context, hospitalID, deptID uuid.UUID) ([]entity.CapacityOverride, error)

	// ListOverridesByYearMonth filters the department's overrides by
	// target year (required) and optional month (1-12; 0 = any month).
	ListOverridesByYearMonth(ctx context.Context, hospitalID, deptID uuid.UUID, year, month int) ([]entity.CapacityOverride, error)

	// GetOverride returns one CapacityOverride by ID. Useful for the UI
	// edit-precheck (since updates are not allowed, the UI fetches the
	// row to decide whether to delete-and-recreate).
	GetOverride(ctx context.Context, overrideID uuid.UUID) (*entity.CapacityOverride, error)

	// CreateOverride inserts a new override row. The target date must be
	// at least system_configs.buffer_days + 1 days in the future and
	// must not collide with any other active override.
	CreateOverride(ctx context.Context, hospitalID, deptID uuid.UUID, date time.Time, newLimit int, reason string, userID uuid.UUID) error

	// DeleteOverride deactivates the override (IsActive = false). Live
	// capacity decisions then fall back to HospitalDepartment values.
	DeleteOverride(ctx context.Context, overrideID, userID uuid.UUID) error

	// GetCapacityDetail returns the rich daily view for the dept-head
	// dashboard: max, overbook, booked, staff hint, staff assigned,
	// available_slots, is_full. The staff_capacity and staff_assigned
	// fields are advisory; booking is not gated on them.
	GetCapacityDetail(ctx context.Context, hospitalID, deptID uuid.UUID, date time.Time) (*dto.CapacityDetailResponse, error)

	// GetScheduledPatientsForDate returns Expected/Arrived/Admitted
	// triage rows scheduled for the date with Referral + Patient
	// preloaded.
	GetScheduledPatientsForDate(ctx context.Context, hospitalID, deptID uuid.UUID, date time.Time) ([]entity.TriageQueue, error)

	// BuildCapacityCalendar produces a per-day rollup for the month,
	// suitable for a calendar widget. month must be in 1-12.
	BuildCapacityCalendar(ctx context.Context, hospitalID, deptID uuid.UUID, year, month int) ([]dto.CapacityCalendarDay, error)

	// UpdateStaffCapacity persists HospitalDepartment.MaxCapacityOfStaff.
	// The value is a soft hint exposed via GetCapacityDetail; no booking
	// rule depends on it.
	UpdateStaffCapacity(ctx context.Context, hospitalID, deptID uuid.UUID, value int, userID uuid.UUID) error

	// UpdateDailyCapacity persists the baseline daily capacity for a
	// (hospital, department). Both fields are written together:
	//   - standard_daily_limit replaces the prior baseline used by
	//     getEffectiveCapacity when no active CapacityOverride exists
	//     for the target date.
	//   - overbook_limit replaces the prior overbook ceiling used for
	//     emergency scheduling.
	// The change takes effect immediately for all future dates that do
	// not have an active CapacityOverride; existing CapacityOverride and
	// already-frozen DailySchedule rows are unaffected by design (logs
	// are immutable; overrides win).
	UpdateDailyCapacity(ctx context.Context, hospitalID, deptID uuid.UUID, standardDailyLimit, overbookLimit int, userID uuid.UUID) error

	// GetDailyCapacity returns the current baseline daily capacity
	// (standard_daily_limit, overbook_limit) for the (hospital,
	// department) pair so the dept-head UI can pre-fill the edit form
	// and reconcile its cache after a PUT round-trip.
	GetDailyCapacity(ctx context.Context, hospitalID, deptID uuid.UUID) (*dto.DeptHeadDailyCapacityResponse, error)
}
