package interfaces

import (
	"context"
	"time"
)

type JobCheckpointRepository interface {
	GetLastRun(ctx context.Context, jobName string) (*time.Time, error)
	UpdateLastRun(ctx context.Context, jobName string, timestamp time.Time) error
}
