package interfaces

import (
	"context"

	"Hospital-Referral-System/internal/domain/entity"
	"github.com/google/uuid"
)

type AuditLogFilter struct {
	Page       int
	PageSize   int
	ActionType *entity.ActionType
	StartDate  *string
	EndDate    *string
}

type AuditLogRepository interface {
	Create(ctx context.Context, log *entity.AuditLog) error
	ListByHospital(ctx context.Context, hospitalID uuid.UUID, filter AuditLogFilter) ([]entity.AuditLog, int64, error)
}
