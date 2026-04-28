package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"Hospital-Referral-System/internal/domain/entity"
	irepository "Hospital-Referral-System/internal/domain/interfaces/repository"
	iusecase "Hospital-Referral-System/internal/domain/interfaces/usecase"
)

type dailyWeightUseCase struct {
	configRepo irepository.SystemConfigRepository
	triageRepo irepository.TriageQueueRepository
	auditRepo  irepository.AuditLogRepository
}

func NewDailyWeightUseCase(
	cfgRepo irepository.SystemConfigRepository,
	tRepo irepository.TriageQueueRepository,
	auditRepo irepository.AuditLogRepository,
) iusecase.DailyWeightUseCase {
	return &dailyWeightUseCase{
		configRepo: cfgRepo,
		triageRepo: tRepo,
		auditRepo:  auditRepo,
	}
}

func (u *dailyWeightUseCase) Execute(ctx context.Context, userID uuid.UUID) (string, error) {
	const configKey = "last_waiting_weight_update"
	todayStr := time.Now().Format("2006-01-02")

	// 1. Idempotency Check
	cfg, err := u.configRepo.GetByKey(ctx, configKey)
	if err == nil && cfg.Value == todayStr {
		return "Already updated today", nil
	}

	// 2. Perform Update
	rowsAffected, err := u.triageRepo.IncrementWaitingWeights(ctx)
	if err != nil {
		return "", err
	}

	// 3. Update Last Execution Date
	// BulkUpdate handles upsert (insert on conflict update)
	_ = u.configRepo.BulkUpdate(ctx, map[string]string{configKey: todayStr})

	// 4. Audit Log
	rowsStr := fmt.Sprintf("%d", rowsAffected)
	_ = u.auditRepo.LogWithContext(ctx, userID, entity.ActionDailyWeightUpdate, nil, nil, map[string]string{"rows_affected": rowsStr})

	return fmt.Sprintf("Successfully updated %d records", rowsAffected), nil
}
