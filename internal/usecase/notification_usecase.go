package usecase

import (
	"context"

	"github.com/google/uuid"

	"Hospital-Referral-System/internal/delivery/http/dto"
	"Hospital-Referral-System/internal/domain/entity"
	irepository "Hospital-Referral-System/internal/domain/interfaces/repository"
	iusecase "Hospital-Referral-System/internal/domain/interfaces/usecase"
	"Hospital-Referral-System/internal/infrastructure/sms"
)

type notificationUseCase struct {
	referralRepo     irepository.ReferralRepository
	notificationRepo irepository.NotificationRepository
	smsClient        sms.SMSClient
}

func NewNotificationUseCase(
	rRepo irepository.ReferralRepository,
	nRepo irepository.NotificationRepository,
	smsClient sms.SMSClient,
) iusecase.NotificationUseCase {
	return &notificationUseCase{
		referralRepo:     rRepo,
		notificationRepo: nRepo,
		smsClient:        smsClient,
	}
}

func (u *notificationUseCase) QueueNotification(ctx context.Context, referralID uuid.UUID, notifType entity.NotificationType, message string) error {
	ref, err := u.referralRepo.GetReferralByID(ctx, referralID)
	if err != nil {
		return err
	}

	recipient := ""
	if ref.Patient != nil && ref.Patient.PhoneNumber != nil {
		recipient = *ref.Patient.PhoneNumber
	}

	notif := &entity.Notification{
		ReferralID:       referralID,
		NotificationType: notifType,
		PhoneNumber:      recipient,
		Content:          message,
		DeliveryStatus:   entity.DeliveryQueued,
	}

	return u.notificationRepo.Create(ctx, notif)
}

func (u *notificationUseCase) TriggerManualSend(ctx context.Context) (*dto.NotificationSendSummary, error) {
	queued, err := u.notificationRepo.GetQueued(ctx, 50)
	if err != nil {
		return nil, err
	}

	summary := &dto.NotificationSendSummary{
		TotalProcessed: len(queued),
	}

	for _, n := range queued {
		resp, err := u.smsClient.Send(ctx, sms.SendRequest{
			To:      n.PhoneNumber,
			Message: n.Content,
		})

		if err != nil {
			_ = u.notificationRepo.UpdateDelivery(ctx, n.ID, entity.DeliveryFailed, nil)
			summary.FailedCount++
		} else {
			_ = u.notificationRepo.UpdateDelivery(ctx, n.ID, entity.DeliverySent, &resp.MessageID)
			summary.SentCount++
		}
	}

	return summary, nil
}

func (u *notificationUseCase) HandleSMSWebhook(ctx context.Context, messageID, status string) error {
	return nil
}
