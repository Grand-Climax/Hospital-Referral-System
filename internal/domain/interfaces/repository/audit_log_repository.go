package interfaces

import (
	"context"

	"Hospital-Referral-System/internal/domain/entity"
	"github.com/google/uuid"
)

type AuditLogFilter struct {
	Page        int
	PageSize    int
	ActionType  *entity.ActionType
	ActionTypes []entity.ActionType // OR-match across multiple actions when non-empty (takes precedence over ActionType)
	StartDate   *string
	EndDate     *string
}

type AuditLogRepository interface {
	Create(ctx context.Context, log *entity.AuditLog) error
	LogWithContext(ctx context.Context, userID uuid.UUID, action entity.ActionType, referralID *uuid.UUID, oldValue, newValue interface{}) error
	ListByHospital(ctx context.Context, hospitalID uuid.UUID, filter AuditLogFilter) ([]entity.AuditLog, int64, error)

	// ListByReferralAndActions returns audit_log rows for a specific
	// referral filtered to the requested action types, ordered by
	// timestamp ASC so the caller can build a chronological timeline
	// (e.g. the arrival_history in the triage-detail endpoint).
	// limit <= 0 means "no limit"; the chronological order is preserved.
	ListByReferralAndActions(ctx context.Context, referralID uuid.UUID, actions []entity.ActionType, limit int) ([]entity.AuditLog, error)
}
