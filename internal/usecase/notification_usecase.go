package usecase

import (
	"context"
	"fmt"
	"log"
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

// smsFailureReason flattens an AfroMessage send error into a stable,
// human-readable string we can persist in notifications.failure_reason
// AND emit via log.Printf so Cloud Logging picks it up. Without this
// the only signal was a `delivery_status=FAILED` row with no breadcrumb,
// which forced operators to guess between "bad phone number", "expired
// JWT", "carrier outage", and "bulk endpoint 500".
func smsFailureReason(err error, resp *sms.SendResponse) *string {
	switch {
	case err != nil:
		s := err.Error()
		return &s
	case resp == nil:
		s := "AfroMessage returned a nil response without an error"
		return &s
	default:
		return nil
	}
}

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

// QueueNotification builds the localized SMS for `notifType`, persists a
// Notification row, and (on Cloud Run, which has no cron) attempts an
// immediate synchronous Send so the row ends up SENT or FAILED before
// the function returns.
//
// Skip rules:
//   - patient.AllowSMS = false                -> drop silently
//   - phone is not a +251 number              -> save row as MANUAL_REQUIRED,
//                                                do NOT call the provider
//                                                (foreign-number policy)
//   - auto_notify system config is OFF        -> save row as MANUAL_REQUIRED,
//                                                wait for an operator to
//                                                trigger TriggerManualSend
//
// Otherwise: persist as QUEUED, call smsClient.Send, then update to
// SENT (with message_id) or FAILED. Any provider failure is swallowed
// here - the row is still in the DB for inspection / resend, and the
// rest of the lifecycle step must not roll back just because an SMS
// failed.
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

	lang := ""
	if patient.HomeRegion != nil {
		lang = string(*patient.HomeRegion)
	}

	placeholders := map[string]string{
		"ReferralID": referralID.String(),
		// Sensible defaults so we never leak a literal {{Hospital}} /
		// {{Department}} into a real SMS even if a relationship was
		// somehow not preloaded.
		"Hospital":   "the referral hospital",
		"Department": "the receiving department",
	}
	if ref.ReceiverHospital != nil && ref.ReceiverHospital.Name != "" {
		placeholders["Hospital"] = ref.ReceiverHospital.Name
	}
	if ref.TargetDepartment != nil && ref.TargetDepartment.Name != "" {
		placeholders["Department"] = ref.TargetDepartment.Name
	}
	if notifType == entity.NotifyScheduling || notifType == entity.NotifyReschedule || notifType == entity.NotifyMissedReschedule || notifType == entity.NotifyReminder || notifType == entity.NotifyMissed {
		queue, _ := u.triageRepo.GetByReferralID(ctx, referralID)
		if queue != nil && queue.AppointmentDate != nil {
			// triage_queues.appointment_date is stored as a DATE column
			// (no time), so Postgres always hands us back 00:00. The
			// system is day-slot based - patients are booked for the day,
			// not a specific time - so we render the SMS with the
			// clinic's default opening time (08:00) instead of the
			// stored midnight, which would otherwise read "at 00:00" and
			// confuse patients into thinking the appointment is at
			// midnight.
			appt := *queue.AppointmentDate
			if appt.Hour() == 0 && appt.Minute() == 0 {
				appt = time.Date(appt.Year(), appt.Month(), appt.Day(), 8, 0, 0, 0, appt.Location())
			}
			// Human-friendly format: e.g. "Mon, 25 May 2026 at 08:00".
			// SMS-length friendly (~25 chars) and unambiguous across
			// regions - no MM/DD vs DD/MM confusion.
			human := appt.Format("Mon, 02 Jan 2006 at 15:04")
			placeholders["Date"] = human
			placeholders["NewDate"] = human
			placeholders["Time"] = appt.Format("15:04")
		}
	}

	templateKey := mapTypeToKey(notifType)
	content := utils.GetLocalizedSMS(templateKey, lang, placeholders)

	// Decide initial status based on auto_notify + phone country.
	autoNotify := false
	if u.configRepo != nil {
		if val, cfgErr := u.configRepo.GetBool(ctx, "auto_notify", false); cfgErr == nil {
			autoNotify = val
		}
	}

	canSendNow := autoNotify && strings.HasPrefix(patient.PhonePlain, "+251") && u.smsClient != nil
	initialStatus := entity.DeliveryQueued
	if !canSendNow {
		initialStatus = entity.DeliveryManualRequired
	}

	notif := &entity.Notification{
		ReferralID:       referralID,
		NotificationType: notifType,
		PhoneNumber:      patient.PhonePlain,
		Content:          content,
		DeliveryStatus:   initialStatus,
	}
	if err := u.notificationRepo.Create(ctx, notif); err != nil {
		return err
	}

	if !canSendNow {
		return nil
	}

	// Synchronous send. We deliberately don't return provider errors:
	// the lifecycle event has already happened, and the Notification
	// row is the audit trail (FAILED rows can be resent later).
	//
	// IMPORTANT: AfroMessage's Send() already gates on its own
	// `acknowledge == "success"` check internally and returns an error
	// when the provider rejected the message; the returned SendResponse
	// does NOT carry the Acknowledge field through (it only sets
	// MessageID + Status="Sent"). So a nil err + non-nil resp is the
	// only correct success signal here. Checking resp.Acknowledge
	// would mark every successful SMS as FAILED.
	resp, sendErr := u.smsClient.Send(ctx, sms.SendRequest{
		To:      patient.PhonePlain,
		Message: content,
	})
	if sendErr != nil || resp == nil {
		reason := smsFailureReason(sendErr, resp)
		log.Printf("sms send FAILED referral=%s notif=%s to=%s reason=%v",
			ref.ID, notif.ID, patient.PhonePlain, derefStr(reason))
		_ = u.notificationRepo.UpdateDelivery(ctx, notif.ID, entity.DeliveryFailed, nil, reason)
		return nil
	}
	_ = u.notificationRepo.UpdateDelivery(ctx, notif.ID, entity.DeliverySent, &resp.MessageID, nil)
	return nil
}

// derefStr safely formats a *string for log lines.
func derefStr(s *string) string {
	if s == nil {
		return "<nil>"
	}
	return *s
}

func mapTypeToKey(notifType entity.NotificationType) string {
	switch notifType {
	case entity.NotifyAcceptance:
		return "accepted"
	case entity.NotifyScheduling:
		return "scheduled"
	case entity.NotifyReschedule:
		return "rescheduled"
	case entity.NotifyMissed:
		return "missed"
	case entity.NotifyMissedReschedule:
		return "missed_rescheduled"
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

		if err != nil || resp == nil {
			reason := smsFailureReason(err, resp)
			log.Printf("sms manual-send FAILED notif=%s to=%s reason=%v", n.ID, n.PhoneNumber, derefStr(reason))
			_ = u.notificationRepo.UpdateDelivery(ctx, n.ID, entity.DeliveryFailed, nil, reason)
			summary.FailedCount++
		} else {
			_ = u.notificationRepo.UpdateDelivery(ctx, n.ID, entity.DeliverySent, &resp.MessageID, nil)
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
			_ = u.notificationRepo.UpdateDelivery(ctx, n.ID, entity.DeliverySent, &msgID, nil)
			summary.SentCount++
		} else {
			reason := "bulk send did not return a message_id for this recipient (likely rejected by AfroMessage)"
			log.Printf("sms bulk-send FAILED notif=%s to=%s reason=%s", n.ID, n.PhoneNumber, reason)
			_ = u.notificationRepo.UpdateDelivery(ctx, n.ID, entity.DeliveryFailed, nil, &reason)
			summary.FailedCount++
		}
	}

	// Update checkpoint
	if u.jobCheckpointRepo != nil {
		_ = u.jobCheckpointRepo.UpdateLastRun(ctx, "sms_processing", time.Now())
	}

	return summary, nil
}

// Helper: send individual notifications when bulk fails.
//
// AfroMessage's Send() returns a non-nil error whenever the provider
// did not acknowledge success, so a nil err + non-nil resp is the
// authoritative success signal. The returned SendResponse never
// carries the Acknowledge field, so checking it here would mark every
// successful SMS as FAILED.
func (u *notificationUseCase) sendIndividualWithFallback(ctx context.Context, notifications []entity.Notification, summary *dto.NotificationSendSummary) (*dto.NotificationSendSummary, error) {
	for _, n := range notifications {
		resp, err := u.smsClient.Send(ctx, sms.SendRequest{
			To:      n.PhoneNumber,
			Message: n.Content,
		})
		if err != nil || resp == nil {
			reason := smsFailureReason(err, resp)
			log.Printf("sms fallback FAILED notif=%s to=%s reason=%v", n.ID, n.PhoneNumber, derefStr(reason))
			_ = u.notificationRepo.UpdateDelivery(ctx, n.ID, entity.DeliveryFailed, nil, reason)
			summary.FailedCount++
		} else {
			_ = u.notificationRepo.UpdateDelivery(ctx, n.ID, entity.DeliverySent, &resp.MessageID, nil)
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

	if err != nil || resp == nil {
		reason := smsFailureReason(err, resp)
		log.Printf("sms resend FAILED notif=%s to=%s reason=%v", n.ID, n.PhoneNumber, derefStr(reason))
		_ = u.notificationRepo.UpdateDelivery(ctx, n.ID, entity.DeliveryFailed, nil, reason)
		if err != nil {
			return nil, fmt.Errorf("resend failed: %w", err)
		}
		return nil, fmt.Errorf("resend failed: nil response from provider")
	}

	err = u.notificationRepo.UpdateDelivery(ctx, n.ID, entity.DeliverySent, &resp.MessageID, nil)
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
