package interfaces

import (
	"context"
	"time"

	"github.com/google/uuid"

	"Hospital-Referral-System/internal/domain/entity"
)

type SchedulerCheckpointRepository interface {
	BaseRepository[entity.SchedulerCheckpoint]

	// GetNextEligibleDepartment finds one department that hasn't been processed recently
	// and is not currently leased, acquires a lease for it, and returns the checkpoint.
	GetNextEligibleDepartment(ctx context.Context, minAge time.Duration, leaseHolder string, leaseDuration time.Duration) (*entity.SchedulerCheckpoint, error)

	// UpdateLastProcessed updates the last_processed_at timestamp and releases the lease.
	UpdateLastProcessed(ctx context.Context, hospitalID, departmentID uuid.UUID, leaseHolder string) error

	// ReleaseLease removes the lease for the given department if held by the given holder.
	ReleaseLease(ctx context.Context, hospitalID, departmentID uuid.UUID, leaseHolder string) error

	// AcquireLease attempts to acquire a lease for the given department.
	AcquireLease(ctx context.Context, hospitalID, departmentID uuid.UUID, leaseHolder string, leaseDuration time.Duration) (bool, error)
}
