package interfaces

import (
	"context"

	"github.com/google/uuid"

	"Hospital-Referral-System/internal/domain/entity"
	irepository "Hospital-Referral-System/internal/domain/interfaces/repository"
)

type InAppNotificationUseCase interface {
	CreateForEvent(ctx context.Context, eventType string, referralID uuid.UUID, actorID uuid.UUID) error
	ListForUser(ctx context.Context, userID uuid.UUID, filter irepository.InAppNotificationFilter, limit, page int) ([]entity.InAppNotification, int64, int64, error)
	MarkRead(ctx context.Context, id, userID uuid.UUID) error
	MarkAllRead(ctx context.Context, userID uuid.UUID) error
	GetUnreadCount(ctx context.Context, userID uuid.UUID) (int64, error)
}
