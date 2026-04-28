package usecase

import (
	"context"
	"time"

	"github.com/google/uuid"

	"Hospital-Referral-System/internal/delivery/http/dto"
	irepository "Hospital-Referral-System/internal/domain/interfaces/repository"
	iusecase "Hospital-Referral-System/internal/domain/interfaces/usecase"
)

type schedulerServiceUseCase struct {
	checkpointRepo irepository.SchedulerCheckpointRepository
	configRepo     irepository.SystemConfigRepository
	schedUC        iusecase.SchedulingUseCase
}

func NewSchedulerServiceUseCase(
	checkpointRepo irepository.SchedulerCheckpointRepository,
	configRepo irepository.SystemConfigRepository,
	schedUC iusecase.SchedulingUseCase,
) iusecase.SchedulerServiceUseCase {
	return &schedulerServiceUseCase{
		checkpointRepo: checkpointRepo,
		configRepo:     configRepo,
		schedUC:        schedUC,
	}
}

func (u *schedulerServiceUseCase) RunSchedulerCycle(ctx context.Context, leaseHolder string) (*dto.BatchScheduleResult, error) {
	// 1. Read auto_notify config
	autoNotify, err := u.configRepo.GetBool(ctx, "auto_notify", false)
	if err != nil {
		// Log error but continue with default? 
		// For safety, let's treat any error as "false" but log it if we had a logger.
		autoNotify = false
	}

	// 2. Acquire lease for the next eligible department
	// minAge: 1 hour (as suggested)
	// leaseDuration: 5 minutes (enough for one department's batch)
	minAge := 1 * time.Hour
	leaseDuration := 5 * time.Minute

	checkpoint, err := u.checkpointRepo.GetNextEligibleDepartment(ctx, minAge, leaseHolder, leaseDuration)
	if err != nil {
		return nil, err
	}

	if checkpoint == nil {
		return &dto.BatchScheduleResult{
			Message: "No eligible departments for processing at this time",
		}, nil
	}

	// 3. Run Batch Scheduling
	// Use uuid.Nil as userID to represent System
	result, err := u.schedUC.BatchSchedule(ctx, checkpoint.HospitalID, checkpoint.DeptID, uuid.Nil, autoNotify)
	if err != nil {
		// Release lease on failure so it can be retried later
		_ = u.checkpointRepo.ReleaseLease(ctx, checkpoint.HospitalID, checkpoint.DeptID, leaseHolder)
		return nil, err
	}

	// 4. Update last processed timestamp and release lease
	err = u.checkpointRepo.UpdateLastProcessed(ctx, checkpoint.HospitalID, checkpoint.DeptID, leaseHolder)
	if err != nil {
		return nil, err
	}

	return result, nil
}
