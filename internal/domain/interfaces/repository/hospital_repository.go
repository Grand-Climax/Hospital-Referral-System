package interfaces

import (
	"context"
	"time"

	"github.com/google/uuid"

	"Hospital-Referral-System/internal/domain/entity"
)

type HospitalListFilter struct {
	Page     int
	PageSize int
	Tier     *entity.HospitalTier
	Region   *string
	IsActive *bool
	Search   *string
}

type HospitalRepository interface {
	BaseRepository[entity.Hospital]
	ListHospitals(ctx context.Context, filter HospitalListFilter) ([]entity.Hospital, int64, error)
}

// SystemConfigRepository manages system-wide key-value configuration (e.g., aging_factor, max_capacity).
type SystemConfigRepository interface {
	BaseRepository[entity.SystemConfig]
	GetByKey(ctx context.Context, key string) (*entity.SystemConfig, error)
	GetAll(ctx context.Context) (map[string]string, error)
	Update(ctx context.Context, cfg *entity.SystemConfig) error
	BulkUpdate(ctx context.Context, updates map[string]string) error
	GetBool(ctx context.Context, key string, defaultValue bool) (bool, error)
}

// CapacityOverrideRepository manages temporary capacity override records
// set by hospital/department admins to allow overbooking on specific dates.
type CapacityOverrideRepository interface {
	BaseRepository[entity.CapacityOverride]
	GetActive(ctx context.Context, hospitalID, deptID uuid.UUID, date time.Time) (*entity.CapacityOverride, error)
	ListByDept(ctx context.Context, hospitalID, deptID uuid.UUID) ([]entity.CapacityOverride, error)

	// ListByDeptAndYearMonth returns overrides whose target_date falls in
	// the given year (required, > 0) and optional month (1-12; 0 = any
	// month). Active and inactive rows are included, ordered ascending by
	// target_date so a calendar UI can render them top-down.
	ListByDeptAndYearMonth(ctx context.Context, hospitalID, deptID uuid.UUID, year, month int) ([]entity.CapacityOverride, error)
}
