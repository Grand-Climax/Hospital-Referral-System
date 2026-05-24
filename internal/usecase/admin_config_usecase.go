package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"

	"Hospital-Referral-System/internal/domain/entity"
	irepository "Hospital-Referral-System/internal/domain/interfaces/repository"
	iusecase "Hospital-Referral-System/internal/domain/interfaces/usecase"
)

type adminConfigUseCase struct {
	configRepo   irepository.SystemConfigRepository
	auditLogRepo irepository.AuditLogRepository
	inAppNotifUC iusecase.InAppNotificationUseCase
}

// readOnlyConfigKeys are returned by GET /admin/config but must not be updated via PUT
// (e.g. migration-managed schema_version).
var readOnlyConfigKeys = map[string]struct{}{
	"schema_version": {},
}

func NewAdminConfigUseCase(configRepo irepository.SystemConfigRepository,
	auditLogRepo irepository.AuditLogRepository,
	inAppNotifUC iusecase.InAppNotificationUseCase,
) iusecase.AdminConfigUseCase {
	return &adminConfigUseCase{
		configRepo:   configRepo,
		auditLogRepo: auditLogRepo,
		inAppNotifUC: inAppNotifUC,
	}
}

func (u *adminConfigUseCase) GetConfig(ctx context.Context) (map[string]string, error) {
	return u.configRepo.GetAll(ctx)
}

func (u *adminConfigUseCase) UpdateConfig(ctx context.Context, updates map[string]string, userID uuid.UUID) error {
	// 0. Strip read-only keys (clients often round-trip the full GET payload).
	for k := range readOnlyConfigKeys {
		delete(updates, k)
	}
	if len(updates) == 0 {
		return nil
	}

	// 1. Retrieve current config (needed for MFA/SMS coupling and audit).
	oldConfig, err := u.configRepo.GetAll(ctx)
	if err != nil {
		return err
	}

	// sms_otp_enabled only applies when MFA is on; disable SMS when MFA is turned off.
	if v, ok := updates["mfa_enabled"]; ok && isConfigFalse(v) {
		updates["sms_otp_enabled"] = "false"
	}
	if v, ok := updates["sms_otp_enabled"]; ok && isConfigTrue(v) {
		mfaVal := oldConfig["mfa_enabled"]
		if v2, inUpdate := updates["mfa_enabled"]; inUpdate {
			mfaVal = v2
		}
		if isConfigFalse(mfaVal) {
			return fmt.Errorf("sms_otp_enabled requires mfa_enabled to be true")
		}
	}

	// 2. Validation
	for k, v := range updates {
		if err := validateConfig(k, v); err != nil {
			return err
		}
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

	if err := u.auditLogRepo.Create(ctx, auditLog); err != nil {
		return err
	}

	_ = u.inAppNotifUC.CreateForEvent(ctx, "SYSTEM_CONFIG_UPDATED", uuid.Nil, userID)

	return nil
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
	case "auto_notify":
		if value != "true" && value != "false" {
			return fmt.Errorf("auto_notify must be 'true' or 'false'")
		}
	case "sms_otp_enabled", "mfa_enabled", "mfa_sms_fallback_email":
		if value != "true" && value != "false" && value != "1" && value != "0" {
			return fmt.Errorf("%s must be true/false (or 1/0)", key)
		}
	case "mfa_otp_ttl_seconds":
		v, err := strconv.Atoi(value)
		if err != nil || v < 60 {
			return fmt.Errorf("mfa_otp_ttl_seconds must be an integer >= 60")
		}
	case "mfa_otp_max_attempts":
		v, err := strconv.Atoi(value)
		if err != nil || v < 1 {
			return fmt.Errorf("mfa_otp_max_attempts must be an integer >= 1")
		}
	case "mfa_otp_resend_cooldown_seconds":
		v, err := strconv.Atoi(value)
		if err != nil || v < 1 {
			return fmt.Errorf("mfa_otp_resend_cooldown_seconds must be an integer >= 1")
		}
	case "last_waiting_weight_update":
		// Can be empty or a valid RFC3339 timestamp
		if value != "" {
			_, err := time.Parse(time.RFC3339, value)
			if err != nil {
				return fmt.Errorf("last_waiting_weight_update must be empty or a valid RFC3339 timestamp")
			}
		}
	case "enable_cron_jobs":
		if value != "true" && value != "false" {
			return fmt.Errorf("enable_cron_jobs must be 'true' or 'false'")
		}
	default:
		return fmt.Errorf("invalid config key: %s", key)
	}
	return nil
}

func isConfigTrue(value string) bool {
	return value == "true" || value == "1"
}

func isConfigFalse(value string) bool {
	return value == "false" || value == "0" || value == ""
}
