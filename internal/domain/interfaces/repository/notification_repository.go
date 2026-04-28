package interfaces

import (
	"context"

	"github.com/google/uuid"

	"Hospital-Referral-System/internal/domain/entity"
)

type NotificationListFilter struct {
	ReferralID       *uuid.UUID
	NotificationType *entity.NotificationType
	DeliveryStatus   *entity.DeliveryStatus
	HospitalID       *uuid.UUID
	DepartmentID     *uuid.UUID
	Page             int
	PageSize         int
}

// NotificationRepository manages SMS/notification delivery queuing and status tracking.
type NotificationRepository interface {
	Create(ctx context.Context, n *entity.Notification) error
	FindByID(ctx context.Context, id uuid.UUID) (*entity.Notification, error)
	GetQueued(ctx context.Context, limit int) ([]entity.Notification, error)
	GetQueuedByFilter(ctx context.Context, hospitalID, deptID *uuid.UUID, limit int) ([]entity.Notification, error)
	GetSent(ctx context.Context, limit int) ([]entity.Notification, error)
	UpdateDelivery(ctx context.Context, id uuid.UUID, status entity.DeliveryStatus, messageID *string) error
	ListWithFilter(ctx context.Context, filter NotificationListFilter) ([]entity.Notification, int64, error)
}
