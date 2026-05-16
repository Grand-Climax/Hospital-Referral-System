package interfaces

import (
	"context"

	"github.com/google/uuid"

	"Hospital-Referral-System/internal/domain/entity"
	irepository "Hospital-Referral-System/internal/domain/interfaces/repository"
	"Hospital-Referral-System/internal/delivery/http/dto"
)

type NotificationUseCase interface {
	QueueNotification(ctx context.Context, referralID uuid.UUID, notifType entity.NotificationType, message string) error
	TriggerManualSend(ctx context.Context, hospitalID, deptID *uuid.UUID) (*dto.NotificationSendSummary, error)
	ProcessPendingSMS(ctx context.Context, limit int) (*dto.NotificationSendSummary, error)
	UpdateStatus(ctx context.Context) (*dto.NotificationStatusSummary, error)
	ResendNotification(ctx context.Context, id uuid.UUID) (*entity.Notification, error)
	QueueReminders(ctx context.Context) (int, error)
	HandleSMSWebhook(ctx context.Context, messageID, status string) error
	ListNotifications(ctx context.Context, filter irepository.NotificationListFilter) ([]entity.Notification, int64, error)
}
