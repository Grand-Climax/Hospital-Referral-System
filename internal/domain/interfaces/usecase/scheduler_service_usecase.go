package interfaces

import (
	"context"

	"Hospital-Referral-System/internal/delivery/http/dto"
)

type SchedulerServiceUseCase interface {
	RunSchedulerCycle(ctx context.Context, leaseHolder string) (*dto.BatchScheduleResult, error)
}
