package interfaces

import (
	"context"
	"time"

	"github.com/google/uuid"

	"Hospital-Referral-System/internal/domain/entity"
)

// DailyScheduleRepository persists the "Schedule-on-Demand" history log.
// Rows are created lazily the first time a patient is booked for a given
// (hospital, department, date) tuple, then BookedSlots is updated as a
// snapshot on every subsequent booking. MaxSlots and OverbookLimit are
// frozen at creation time. Live capacity is computed elsewhere from the
// TriageQueue + HospitalDepartment + CapacityOverride.
type DailyScheduleRepository interface {
	BaseRepository[entity.DailySchedule]

	// FindByDeptAndDate returns the existing log row or gorm.ErrRecordNotFound.
	FindByDeptAndDate(ctx context.Context, hospitalID, deptID uuid.UUID, date time.Time) (*entity.DailySchedule, error)

	// FindByDeptAndDateRange returns log rows in [start, end] (inclusive)
	// for calendar / reporting views.
	FindByDeptAndDateRange(ctx context.Context, hospitalID, deptID uuid.UUID, start, end time.Time) ([]entity.DailySchedule, error)

	// CreateLog inserts a new log row. Used on the first booking for a date.
	CreateLog(ctx context.Context, log *entity.DailySchedule) error

	// UpdateBookedSlots updates the BookedSlots snapshot on an existing log row.
	UpdateBookedSlots(ctx context.Context, id uuid.UUID, count int) error
}
