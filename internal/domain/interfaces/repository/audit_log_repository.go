package interfaces

import (
	"context"

	"github.com/google/uuid"

	"Hospital-Referral-System/internal/domain/entity"
)

type AuditLogRepository interface {
	Create(ctx context.Context, log *entity.AuditLog) error
	LogWithContext(ctx context.Context, userID uuid.UUID, action entity.ActionType, referralID *uuid.UUID, oldValue, newValue interface{}) error
}
