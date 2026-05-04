package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"Hospital-Referral-System/internal/delivery/http/dto"
	"Hospital-Referral-System/internal/domain/entity"
	irepository "Hospital-Referral-System/internal/domain/interfaces/repository"
	iusecase "Hospital-Referral-System/internal/domain/interfaces/usecase"
	"Hospital-Referral-System/internal/infrastructure/crypto"
	"Hospital-Referral-System/internal/infrastructure/sms"
)

type notificationUseCase struct {
	referralRepo     irepository.ReferralRepository
	notificationRepo irepository.NotificationRepository
	triageRepo       irepository.TriageQueueRepository
	smsClient        sms.SMSClient
	cryptoSvc        *crypto.PatientCryptoService
}

func NewNotificationUseCase(
	rRepo irepository.ReferralRepository,
	nRepo irepository.NotificationRepository,
	tRepo irepository.TriageQueueRepository,
	smsClient sms.SMSClient,
	cryptoSvc *crypto.PatientCryptoService,
) iusecase.NotificationUseCase {
	return &notificationUseCase{
		referralRepo:     rRepo,
		notificationRepo: nRepo,
		triageRepo:       tRepo,
		smsClient:        smsClient,
		cryptoSvc:        cryptoSvc,
	}
}

func (u *notificationUseCase) QueueNotification(ctx context.Context, referralID uuid.UUID, notifType entity.NotificationType, message string) error {
	// Preload everything needed for message construction and verification
	ref, err := u.referralRepo.GetReferralByID(ctx, referralID)
	if err != nil {
		return err
	}

	if ref.Patient == nil {
		return fmt.Errorf("referral has no associated patient")
	}

	// Respect allow_sms flag
	if !ref.Patient.AllowSMS {
		return nil
	}

	recipient := ""
	if ref.Patient != nil {
		_ = ref.Patient.DecryptFields(u.cryptoSvc)
		recipient = ref.Patient.PhonePlain
	}

	if recipient == "" {
		return fmt.Errorf("patient has no phone number")
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

func (u *notificationUseCase) TriggerManualSend(ctx context.Context, hospitalID, deptID *uuid.UUID) (*dto.NotificationSendSummary, error) {
	queued, err := u.notificationRepo.GetQueuedByFilter(ctx, hospitalID, deptID, 50)
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

func (u *notificationUseCase) UpdateStatus(ctx context.Context) (*dto.NotificationStatusSummary, error) {
	// Fetch up to 20 "SENT" or "RESEND" notifications to avoid rate limit issues (2s delay per check)
	sent, err := u.notificationRepo.GetSent(ctx, 20)
	if err != nil {
		return nil, err
	}

	summary := &dto.NotificationStatusSummary{
		TotalChecked: len(sent),
	}

	for _, n := range sent {
		statusData, err := u.smsClient.GetStatus(ctx, n.ProviderMessageID)
		if err != nil {
			// Log error but continue
			summary.StillProcessing++
			continue
		}

		switch statusData.Status {
		case "DELIVRD", "DELIVERED":
			_ = u.notificationRepo.UpdateDelivery(ctx, n.ID, entity.DeliveryDelivered, nil)
			summary.DeliveredCount++
		case "UNDELIV", "FAILED", "REJECTD":
			_ = u.notificationRepo.UpdateDelivery(ctx, n.ID, entity.DeliveryFailed, nil)
			summary.FailedCount++
		default:
			summary.StillProcessing++
		}
	}

	return summary, nil
}

func (u *notificationUseCase) ResendNotification(ctx context.Context, id uuid.UUID) (*entity.Notification, error) {
	n, err := u.notificationRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if n.DeliveryStatus != entity.DeliveryFailed && n.DeliveryStatus != entity.DeliveryQueued {
		return nil, fmt.Errorf("only failed or queued notifications can be resent (current status: %s)", n.DeliveryStatus)
	}

	resp, err := u.smsClient.Send(ctx, sms.SendRequest{
		To:      n.PhoneNumber,
		Message: n.Content,
	})

	if err != nil {
		_ = u.notificationRepo.UpdateDelivery(ctx, n.ID, entity.DeliveryFailed, nil)
		return nil, fmt.Errorf("resend failed: %w", err)
	}

	err = u.notificationRepo.UpdateDelivery(ctx, n.ID, entity.DeliveryResend, &resp.MessageID)
	if err != nil {
		return nil, err
	}

	return u.notificationRepo.FindByID(ctx, id)
}

func (u *notificationUseCase) QueueReminders(ctx context.Context) (int, error) {
	tomorrow := time.Now().AddDate(0, 0, 1)
	appointments, err := u.triageRepo.FindAppointmentsForReminders(ctx, tomorrow)
	if err != nil {
		return 0, err
	}

	queuedCount := 0
	for _, app := range appointments {
		hospitalName := "the hospital"
		if app.Referral != nil && app.Referral.ReceiverHospital != nil {
			hospitalName = app.Referral.ReceiverHospital.Name
		}

		message := fmt.Sprintf("Reminder: You have an appointment at %s tomorrow. Please arrive on time.", hospitalName)
		err := u.QueueNotification(ctx, app.ReferralID, entity.NotificationType("REMINDER"), message)
		if err == nil {
			queuedCount++
		}
	}

	return queuedCount, nil
}

func (u *notificationUseCase) HandleSMSWebhook(ctx context.Context, messageID, status string) error {
	// Webhook logic if needed, but manual update is preferred for now
	return nil
}

func (u *notificationUseCase) ListNotifications(ctx context.Context, filter irepository.NotificationListFilter) ([]entity.Notification, int64, error) {
	return u.notificationRepo.ListWithFilter(ctx, filter)
}
