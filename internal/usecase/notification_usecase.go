package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"strings"

	"Hospital-Referral-System/internal/delivery/http/dto"
	"Hospital-Referral-System/internal/domain/entity"
	irepository "Hospital-Referral-System/internal/domain/interfaces/repository"
	iusecase "Hospital-Referral-System/internal/domain/interfaces/usecase"
	"Hospital-Referral-System/internal/infrastructure/crypto"
	"Hospital-Referral-System/internal/infrastructure/sms"
	"Hospital-Referral-System/pkg/utils"
)

type notificationUseCase struct {
	referralRepo     irepository.ReferralRepository
	notificationRepo irepository.NotificationRepository
	triageRepo        irepository.TriageQueueRepository
	configRepo        irepository.SystemConfigRepository
	jobCheckpointRepo irepository.JobCheckpointRepository
	smsClient        sms.SMSClient
	cryptoSvc        *crypto.PatientCryptoService
}

func NewNotificationUseCase(
	rRepo irepository.ReferralRepository,
	nRepo irepository.NotificationRepository,
	tRepo irepository.TriageQueueRepository,
	cRepo irepository.SystemConfigRepository,
	jRepo irepository.JobCheckpointRepository,
	smsClient sms.SMSClient,
	cryptoSvc *crypto.PatientCryptoService,
) iusecase.NotificationUseCase {
	return &notificationUseCase{
		referralRepo:     rRepo,
		notificationRepo: nRepo,
		triageRepo:        tRepo,
		configRepo:        cRepo,
		jobCheckpointRepo: jRepo,
		smsClient:         smsClient,
		cryptoSvc:        cryptoSvc,
	}
}

func (u *notificationUseCase) QueueNotification(ctx context.Context, referralID uuid.UUID, notifType entity.NotificationType, _ string) error {
	ref, err := u.referralRepo.GetReferralByID(ctx, referralID)
	if err != nil {
		return err
	}
	if ref.Patient == nil {
		return fmt.Errorf("referral has no patient")
	}
	patient := ref.Patient

	if err := patient.DecryptFields(u.cryptoSvc); err != nil {
		return err
	}

	if !patient.AllowSMS {
		return nil
	}

	deliveryStatus := entity.DeliveryQueued
	if !strings.HasPrefix(patient.PhonePlain, "+251") {
		deliveryStatus = entity.DeliveryManualRequired
	}

	lang := ""
	if patient.HomeRegion != nil {
		lang = string(*patient.HomeRegion)
	}

	placeholders := map[string]string{
		"ReferralID": referralID.String(),
	}
	if ref.ReceiverHospital != nil {
		placeholders["Hospital"] = ref.ReceiverHospital.Name
	}
	if ref.TargetDepartment != nil {
		placeholders["Department"] = ref.TargetDepartment.Name
	}
	if notifType == entity.NotifyScheduling || notifType == entity.NotifyReschedule {
		queue, _ := u.triageRepo.GetByReferralID(ctx, referralID)
		if queue != nil && queue.AppointmentDate != nil {
			placeholders["Date"] = queue.AppointmentDate.Format("2006-01-02")
			placeholders["NewDate"] = queue.AppointmentDate.Format("2006-01-02")
			placeholders["Time"] = queue.AppointmentDate.Format("15:04")
		}
	}

	templateKey := mapTypeToKey(notifType)
	content := utils.GetLocalizedSMS(templateKey, lang, placeholders)

	autoNotify := false
	if u.configRepo != nil {
		val, err := u.configRepo.GetBool(ctx, "auto_notify", false)
		if err == nil {
			autoNotify = val
		}
	}

	if !autoNotify && deliveryStatus != entity.DeliveryManualRequired {
		deliveryStatus = entity.DeliveryManualRequired
	}

	notif := &entity.Notification{
		ReferralID:       referralID,
		NotificationType: notifType,
		PhoneNumber:      patient.PhonePlain,
		Content:          content,
		DeliveryStatus:   deliveryStatus,
	}

	return u.notificationRepo.Create(ctx, notif)
}

func mapTypeToKey(notifType entity.NotificationType) string {
	switch notifType {
	case entity.NotifyAcceptance:
		return "accepted"
	case entity.NotifyScheduling:
		return "scheduled"
	case entity.NotifyReschedule:
		return "rescheduled"
	case entity.NotifyReminder:
		return "reminder"
	default:
		return "scheduled"
	}
}

func (u *notificationUseCase) TriggerManualSend(ctx context.Context, hospitalID, deptID *uuid.UUID) (*dto.NotificationSendSummary, error) {
	pending, err := u.notificationRepo.GetPendingByFilter(ctx, hospitalID, deptID, 
		[]entity.DeliveryStatus{entity.DeliveryQueued, entity.DeliveryManualRequired}, 50)
	if err != nil {
		return nil, err
	}

	summary := &dto.NotificationSendSummary{
		TotalProcessed: len(pending),
	}

	for _, n := range pending {
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

func (u *notificationUseCase) ProcessPendingSMS(ctx context.Context, limit int) (*dto.NotificationSendSummary, error) {
	enabled, err := u.configRepo.GetBool(ctx, "enable_cron_jobs", false)
	if err != nil || !enabled {
		return &dto.NotificationSendSummary{TotalProcessed: 0}, nil
	}

	pending, err := u.notificationRepo.GetPendingByFilter(ctx, nil, nil, 
		[]entity.DeliveryStatus{entity.DeliveryQueued, entity.DeliveryManualRequired}, limit)
	if err != nil {
		return nil, err
	}
	if len(pending) == 0 {
		return &dto.NotificationSendSummary{TotalProcessed: 0}, nil
	}

	// Prepare bulk payload
	var recipients []sms.BulkRecipient
	for _, n := range pending {
		recipients = append(recipients, sms.BulkRecipient{
			To:      n.PhoneNumber,
			Message: n.Content,
		})
	}

	bulkReq := sms.BulkSendRequest{
		To:       recipients,
		Campaign: fmt.Sprintf("ReferralHub-%s", time.Now().Format("20060102-150405")),
	}

	summary := &dto.NotificationSendSummary{TotalProcessed: len(pending)}

	// Attempt bulk send
	bulkResp, err := u.smsClient.SendBulk(ctx, bulkReq)
	
	if err != nil || bulkResp == nil || bulkResp.Acknowledge != "success" {
		// Bulk failed completely – fall back to individual sends
		return u.sendIndividualWithFallback(ctx, pending, summary)
	}

	// Bulk succeeded – map responses to notifications
	// Create a map of phone -> message_id
	msgMap := make(map[string]string)
	for _, msg := range bulkResp.Response.Messages {
		msgMap[msg.To] = msg.MessageID
	}

	for _, n := range pending {
		if msgID, ok := msgMap[n.PhoneNumber]; ok {
			_ = u.notificationRepo.UpdateDelivery(ctx, n.ID, entity.DeliverySent, &msgID)
			summary.SentCount++
		} else {
			// This recipient wasn't in success list – mark as failed
			_ = u.notificationRepo.UpdateDelivery(ctx, n.ID, entity.DeliveryFailed, nil)
			summary.FailedCount++
		}
	}

	// Update checkpoint
	if u.jobCheckpointRepo != nil {
		_ = u.jobCheckpointRepo.UpdateLastRun(ctx, "sms_processing", time.Now())
	}

	return summary, nil
}

// Helper: send individual notifications when bulk fails
func (u *notificationUseCase) sendIndividualWithFallback(ctx context.Context, notifications []entity.Notification, summary *dto.NotificationSendSummary) (*dto.NotificationSendSummary, error) {
	for _, n := range notifications {
		resp, err := u.smsClient.Send(ctx, sms.SendRequest{
			To:      n.PhoneNumber,
			Message: n.Content,
		})
		if err != nil || resp == nil || resp.Acknowledge != "success" {
			_ = u.notificationRepo.UpdateDelivery(ctx, n.ID, entity.DeliveryFailed, nil)
			summary.FailedCount++
		} else {
			_ = u.notificationRepo.UpdateDelivery(ctx, n.ID, entity.DeliverySent, &resp.MessageID)
			summary.SentCount++
		}
	}

	// Update checkpoint even if fallback happened
	if u.jobCheckpointRepo != nil {
		_ = u.jobCheckpointRepo.UpdateLastRun(ctx, "sms_processing", time.Now())
	}

	return summary, nil
}

// Deprecated: AfroMessage does not provide delivery status webhooks.
func (u *notificationUseCase) UpdateStatus(ctx context.Context) (*dto.NotificationStatusSummary, error) {
	return &dto.NotificationStatusSummary{}, nil
}

func (u *notificationUseCase) ResendNotification(ctx context.Context, id uuid.UUID) (*entity.Notification, error) {
	n, err := u.notificationRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if n.DeliveryStatus != entity.DeliveryFailed && 
	   n.DeliveryStatus != entity.DeliveryQueued &&
	   n.DeliveryStatus != entity.DeliveryManualRequired {
		return nil, fmt.Errorf("only failed, queued, or manual_required notifications can be resent (current status: %s)", n.DeliveryStatus)
	}

	resp, err := u.smsClient.Send(ctx, sms.SendRequest{
		To:      n.PhoneNumber,
		Message: n.Content,
	})

	if err != nil {
		_ = u.notificationRepo.UpdateDelivery(ctx, n.ID, entity.DeliveryFailed, nil)
		return nil, fmt.Errorf("resend failed: %w", err)
	}

	err = u.notificationRepo.UpdateDelivery(ctx, n.ID, entity.DeliverySent, &resp.MessageID)
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
		err := u.QueueNotification(ctx, app.ReferralID, entity.NotifyReminder, message)
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
