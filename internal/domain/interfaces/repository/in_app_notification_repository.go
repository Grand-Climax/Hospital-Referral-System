package interfaces

import (
	"context"
	"time"

	"github.com/google/uuid"

	"Hospital-Referral-System/internal/domain/entity"
)

type InAppNotificationFilter struct {
	EventType  string
	IsRead     *bool
	ReferralID *uuid.UUID
	StartDate  *time.Time
	EndDate    *time.Time
	Search     string // search in title and message
}

type InAppNotificationRepository interface {
	BaseRepository[entity.InAppNotification]
	ListByUser(ctx context.Context, userID uuid.UUID, filter InAppNotificationFilter, limit, offset int) ([]entity.InAppNotification, int64, error)
	MarkRead(ctx context.Context, id, userID uuid.UUID) error
	MarkAllRead(ctx context.Context, userID uuid.UUID) error
	CountUnread(ctx context.Context, userID uuid.UUID) (int64, error)
}
