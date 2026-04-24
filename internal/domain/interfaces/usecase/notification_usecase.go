package interfaces

import (
	"context"

	"github.com/google/uuid"

	"Hospital-Referral-System/internal/domain/entity"
	"Hospital-Referral-System/internal/delivery/http/dto"
)

type NotificationUseCase interface {
	QueueNotification(ctx context.Context, referralID uuid.UUID, notifType entity.NotificationType, message string) error
	TriggerManualSend(ctx context.Context) (*dto.NotificationSendSummary, error)
	HandleSMSWebhook(ctx context.Context, messageID, status string) error
}
