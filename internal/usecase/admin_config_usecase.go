package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/google/uuid"

	"Hospital-Referral-System/internal/domain/entity"
	irepository "Hospital-Referral-System/internal/domain/interfaces/repository"
	iusecase "Hospital-Referral-System/internal/domain/interfaces/usecase"
)

type adminConfigUseCase struct {
	configRepo   irepository.SystemConfigRepository
	auditLogRepo irepository.AuditLogRepository
}

func NewAdminConfigUseCase(configRepo irepository.SystemConfigRepository, auditLogRepo irepository.AuditLogRepository) iusecase.AdminConfigUseCase {
	return &adminConfigUseCase{
		configRepo:   configRepo,
		auditLogRepo: auditLogRepo,
	}
}

func (u *adminConfigUseCase) GetConfig(ctx context.Context) (map[string]string, error) {
	return u.configRepo.GetAll(ctx)
}

func (u *adminConfigUseCase) UpdateConfig(ctx context.Context, updates map[string]string, userID uuid.UUID) error {
	// 1. Validation
	for k, v := range updates {
		if err := validateConfig(k, v); err != nil {
			return err
		}
	}

	// 2. Retrieve current config for audit
	oldConfig, err := u.configRepo.GetAll(ctx)
	if err != nil {
		return err
	}

	// 3. Perform bulk update
	if err := u.configRepo.BulkUpdate(ctx, updates); err != nil {
		return err
	}

	// 4. Construct new config for audit
	newConfig := make(map[string]string)
	for k, v := range oldConfig {
		newConfig[k] = v
	}
	for k, v := range updates {
		newConfig[k] = v
	}

	// 5. Audit Log
	oldJSON, _ := json.Marshal(oldConfig)
	newJSON, _ := json.Marshal(newConfig)
	oldStr := string(oldJSON)
	newStr := string(newJSON)

	auditLog := &entity.AuditLog{
		UserID:     userID,
		ActionType: entity.ActionUpdateSystemConfig,
		OldValue:   &oldStr,
		NewValue:   &newStr,
	}

	return u.auditLogRepo.Create(ctx, auditLog)
}

func validateConfig(key, value string) error {
	switch key {
	case "buffer_days":
		v, err := strconv.Atoi(value)
		if err != nil || v < 1 {
			return fmt.Errorf("buffer_days must be a positive integer (>= 1)")
		}
	case "aging_factor":
		v, err := strconv.ParseFloat(value, 64)
		if err != nil || v <= 0 {
			return fmt.Errorf("aging_factor must be a positive float (> 0)")
		}
	case "max_horizon_days":
		v, err := strconv.Atoi(value)
		if err != nil || v < 1 {
			return fmt.Errorf("max_horizon_days must be a positive integer (>= 1)")
		}
	case "overbook_limit_default":
		v, err := strconv.Atoi(value)
		if err != nil || v < 0 {
			return fmt.Errorf("overbook_limit_default must be a non-negative integer (>= 0)")
		}
	}
	return nil
}
