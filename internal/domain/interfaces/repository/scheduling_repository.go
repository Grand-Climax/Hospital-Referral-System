package interfaces

import (
	"context"
	"time"

	"github.com/google/uuid"

	"Hospital-Referral-System/internal/domain/entity"
)

// DailyScheduleRepository manages per-hospital per-day appointment slot capacities.
type DailyScheduleRepository interface {
	BaseRepository[entity.DailySchedule]
	GetOrCreate(ctx context.Context, hospitalID, deptID uuid.UUID, date time.Time, defaultMaxSlots int) (*entity.DailySchedule, error)
	GetByDeptAndDate(ctx context.Context, hospitalID, deptID uuid.UUID, date time.Time) (*entity.DailySchedule, error)
	FindByDeptAndDateRange(ctx context.Context, deptID uuid.UUID, start, end time.Time) ([]entity.DailySchedule, error)
	// IncrementBookedSlots uses optimistic locking (version) to prevent double-booking.
	IncrementBookedSlots(ctx context.Context, id uuid.UUID, version int) error
}

// SchedulerCheckpointRepository tracks the last time the scheduler processed batch jobs (e.g., missed-arrival detection).
type SchedulerCheckpointRepository interface {
	BaseRepository[entity.SchedulerCheckpoint]
	GetLastCheckpoint(ctx context.Context) (time.Time, error)
	UpdateCheckpoint(ctx context.Context, timestamp time.Time) error
}



