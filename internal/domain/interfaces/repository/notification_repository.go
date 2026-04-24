package interfaces

import (
	"context"

	"github.com/google/uuid"

	"Hospital-Referral-System/internal/domain/entity"
)

// NotificationRepository manages SMS/notification delivery queuing and status tracking.
type NotificationRepository interface {
	Create(ctx context.Context, n *entity.Notification) error
	GetQueued(ctx context.Context, limit int) ([]entity.Notification, error)
	UpdateDelivery(ctx context.Context, id uuid.UUID, status entity.DeliveryStatus, messageID *string) error
}
