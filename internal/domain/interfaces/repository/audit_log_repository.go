package interfaces

import (
	"context"

	"Hospital-Referral-System/internal/domain/entity"
)

type AuditLogRepository interface {
	Create(ctx context.Context, log *entity.AuditLog) error
}
