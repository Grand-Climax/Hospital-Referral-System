package usecase

import (
	"context"
	"fmt"
	"log"

	"github.com/google/uuid"

	"Hospital-Referral-System/internal/domain/entity"
	irepository "Hospital-Referral-System/internal/domain/interfaces/repository"
	iusecase "Hospital-Referral-System/internal/domain/interfaces/usecase"
)

type inAppNotificationUseCase struct {
	notifRepo    irepository.InAppNotificationRepository
	userRepo     irepository.UserRepository
	referralRepo irepository.ReferralRepository
}

func NewInAppNotificationUseCase(
	notifRepo irepository.InAppNotificationRepository,
	userRepo irepository.UserRepository,
	referralRepo irepository.ReferralRepository,
) iusecase.InAppNotificationUseCase {
	return &inAppNotificationUseCase{
		notifRepo:    notifRepo,
		userRepo:     userRepo,
		referralRepo: referralRepo,
	}
}

func (u *inAppNotificationUseCase) CreateForEvent(ctx context.Context, eventType string, referralID uuid.UUID, actorID uuid.UUID) error {
	// Fetch referral with related data
	referral, err := u.referralRepo.GetReferralByID(ctx, referralID)
	if err != nil {
		return fmt.Errorf("failed to fetch referral: %v", err)
	}

	var patientName string
	if referral.Patient != nil {
		patientName = fmt.Sprintf("%s %s", referral.Patient.FirstNamePlain, referral.Patient.LastNamePlain)
		if patientName == " " {
			patientName = "Patient"
		}
	} else {
		patientName = "Patient"
	}

	var senderHospitalName string
	if referral.SenderHospital != nil {
		senderHospitalName = referral.SenderHospital.Name
	} else {
		senderHospitalName = "Sending Hospital"
	}

	var targetHospitalName string
	if referral.ReceiverHospital != nil {
		targetHospitalName = referral.ReceiverHospital.Name
	} else {
		targetHospitalName = "Target Hospital"
	}

	title := ""
	message := ""
	var recipientIDs []uuid.UUID

	switch eventType {
	case "REFERRAL_SUBMITTED":
		title = "New Referral Submitted"
		message = fmt.Sprintf("A new referral for patient %s has been submitted from %s.", patientName, senderHospitalName)
		// Notify all liaisons of the sender hospital
		liaisons, _, _ := u.userRepo.ListUsers(ctx, irepository.UserListFilter{
			Role:       ptrRole(entity.RoleLiaisonOfficer),
			HospitalID: ptrStr(referral.SenderHospitalID.String()),
			PageSize:   100,
		})
		for _, l := range liaisons {
			recipientIDs = append(recipientIDs, l.ID)
		}

	case "REFERRAL_FORWARDED":
		title = "Referral Forwarded"
		message = fmt.Sprintf("A referral for patient %s has been forwarded to your department in %s.", patientName, targetHospitalName)
		// Notify specialists in target hospital + department
		specialists, _, _ := u.userRepo.ListUsers(ctx, irepository.UserListFilter{
			Role:         ptrRole(entity.RoleReceivingSpecialist),
			HospitalID:   ptrStr(referral.TargetHospitalID.String()),
			DepartmentID: ptrStr(referral.TargetDeptID.String()),
			PageSize:     100,
		})
		for _, s := range specialists {
			recipientIDs = append(recipientIDs, s.ID)
		}

	case "REFERRAL_ACCEPTED":
		title = "Referral Accepted"
		message = fmt.Sprintf("Your referral for patient %s has been accepted by %s.", patientName, targetHospitalName)
		recipientIDs = append(recipientIDs, referral.ReferringDoctorID)

	case "REFERRAL_REJECTED_BY_LIAISON":
		title = "Referral Rejected by Liaison"
		reason := "No reason provided"
		if referral.RejectionReason != nil {
			reason = *referral.RejectionReason
		}
		message = fmt.Sprintf("Your referral for patient %s was rejected by the liaison. Reason: %s", patientName, reason)
		recipientIDs = append(recipientIDs, referral.ReferringDoctorID)

	case "REFERRAL_REJECTED_BY_SPECIALIST":
		title = "Referral Rejected by Specialist"
		reason := "No reason provided"
		if referral.RejectionReason != nil {
			reason = *referral.RejectionReason
		}
		message = fmt.Sprintf("Your referral for patient %s was rejected by the specialist at %s. Reason: %s", patientName, targetHospitalName, reason)
		recipientIDs = append(recipientIDs, referral.ReferringDoctorID)

	case "REFERRAL_REDIRECTED":
		title = "Referral Redirected"
		message = fmt.Sprintf("Referral for patient %s has been redirected to %s.", patientName, targetHospitalName)
		// Notify referring doctor
		recipientIDs = append(recipientIDs, referral.ReferringDoctorID)
		// Notify original sender liaisons
		liaisons, _, _ := u.userRepo.ListUsers(ctx, irepository.UserListFilter{
			Role:       ptrRole(entity.RoleLiaisonOfficer),
			HospitalID: ptrStr(referral.SenderHospitalID.String()),
			PageSize:   100,
		})
		for _, l := range liaisons {
			recipientIDs = append(recipientIDs, l.ID)
		}
		// Notify specialists at the NEW target hospital's matching department
		specialists, _, _ := u.userRepo.ListUsers(ctx, irepository.UserListFilter{
			Role:         ptrRole(entity.RoleReceivingSpecialist),
			HospitalID:   ptrStr(referral.TargetHospitalID.String()),
			DepartmentID: ptrStr(referral.TargetDeptID.String()),
			PageSize:     100,
		})
		for _, s := range specialists {
			recipientIDs = append(recipientIDs, s.ID)
		}

	case "APPOINTMENT_SCHEDULED":
		title = "Appointment Scheduled"
		message = fmt.Sprintf("An appointment has been scheduled for your patient %s at %s.", patientName, targetHospitalName)
		recipientIDs = append(recipientIDs, referral.ReferringDoctorID)

	case "APPOINTMENT_RESCHEDULED":
		title = "Appointment Rescheduled"
		message = fmt.Sprintf("The appointment for patient %s at %s has been rescheduled.", patientName, targetHospitalName)
		recipientIDs = append(recipientIDs, referral.ReferringDoctorID)

	case "PATIENT_ARRIVED":
		title = "Patient Arrived"
		message = fmt.Sprintf("Your patient %s has arrived at %s for their appointment.", patientName, targetHospitalName)
		recipientIDs = append(recipientIDs, referral.ReferringDoctorID)

	case "PATIENT_MISSED":
		title = "Appointment Missed"
		message = fmt.Sprintf("Your patient %s missed their scheduled appointment at %s.", patientName, targetHospitalName)
		recipientIDs = append(recipientIDs, referral.ReferringDoctorID)

	case "OUTCOME_RECORDED":
		title = "Referral Outcome Recorded"
		message = fmt.Sprintf("A final clinical outcome has been recorded for patient %s at %s.", patientName, targetHospitalName)
		recipientIDs = append(recipientIDs, referral.ReferringDoctorID)

	case "DOCTOR_ASSIGNED":
		title = "Patient Assigned to You"
		message = fmt.Sprintf("Patient %s has been assigned to you for clinical review at %s.", patientName, targetHospitalName)
		if referral.SpecialistID != nil {
			recipientIDs = append(recipientIDs, *referral.SpecialistID)
		}

	case "BATCH_SCHEDULE_COMPLETED":
		title = "Batch Scheduling Completed"
		message = "The automated batch scheduling for your department has been successfully completed."
		// Notify Dept Head of the target department
		heads, _, _ := u.userRepo.ListUsers(ctx, irepository.UserListFilter{
			Role:         ptrRole(entity.RoleDeptHead),
			HospitalID:   ptrStr(referral.TargetHospitalID.String()),
			DepartmentID: ptrStr(referral.TargetDeptID.String()),
			PageSize:     10,
		})
		for _, h := range heads {
			recipientIDs = append(recipientIDs, h.ID)
		}

	case "CAPACITY_OVERRIDE_CREATED":
		title = "Capacity Override Created"
		message = "A new capacity override has been created for your department."
		heads, _, _ := u.userRepo.ListUsers(ctx, irepository.UserListFilter{
			Role:         ptrRole(entity.RoleDeptHead),
			HospitalID:   ptrStr(referral.TargetHospitalID.String()),
			DepartmentID: ptrStr(referral.TargetDeptID.String()),
			PageSize:     10,
		})
		for _, h := range heads {
			recipientIDs = append(recipientIDs, h.ID)
		}

	case "STAFF_ADDED":
		title = "New Staff Added"
		message = "A new staff member has been registered at your hospital."
		// Notify Hospital Admins
		admins, _, _ := u.userRepo.ListUsers(ctx, irepository.UserListFilter{
			Role:       ptrRole(entity.RoleHospitalAdmin),
			HospitalID: ptrStr(referral.TargetHospitalID.String()),
			PageSize:   10,
		})
		for _, a := range admins {
			recipientIDs = append(recipientIDs, a.ID)
		}

	case "SYSTEM_CONFIG_UPDATED":
		title = "System Configuration Updated"
		message = "A system-wide configuration parameter has been updated."
		// Notify all Super Admins
		supers, _, _ := u.userRepo.ListUsers(ctx, irepository.UserListFilter{
			Role:     ptrRole(entity.RoleSystemSuperAdmin),
			PageSize: 20,
		})
		for _, s := range supers {
			recipientIDs = append(recipientIDs, s.ID)
		}

	default:
		log.Printf("Unknown notification event type: %s", eventType)
		return nil
	}

	// Dedup and filter out actor
	uniqueRecipients := make(map[uuid.UUID]bool)
	for _, id := range recipientIDs {
		if id != actorID && id != uuid.Nil {
			uniqueRecipients[id] = true
		}
	}

	// Create notifications in background or sequence
	for id := range uniqueRecipients {
		notif := &entity.InAppNotification{
			UserID:     id,
			ReferralID: &referralID,
			Title:      title,
			Message:    message,
			EventType:  eventType,
			IsRead:     false,
		}
		if err := u.notifRepo.Create(ctx, notif); err != nil {
			log.Printf("Failed to create in-app notification for user %s: %v", id, err)
		}
	}

	return nil
}

func (u *inAppNotificationUseCase) ListForUser(ctx context.Context, userID uuid.UUID, filter irepository.InAppNotificationFilter, limit, page int) ([]entity.InAppNotification, int64, int64, error) {
	offset := (page - 1) * limit
	notifications, total, err := u.notifRepo.ListByUser(ctx, userID, filter, limit, offset)
	if err != nil {
		return nil, 0, 0, err
	}

	unreadCount, _ := u.notifRepo.CountUnread(ctx, userID)

	return notifications, total, unreadCount, nil
}

func (u *inAppNotificationUseCase) MarkRead(ctx context.Context, id, userID uuid.UUID) error {
	return u.notifRepo.MarkRead(ctx, id, userID)
}

func (u *inAppNotificationUseCase) MarkAllRead(ctx context.Context, userID uuid.UUID) error {
	return u.notifRepo.MarkAllRead(ctx, userID)
}

func (u *inAppNotificationUseCase) GetUnreadCount(ctx context.Context, userID uuid.UUID) (int64, error) {
	return u.notifRepo.CountUnread(ctx, userID)
}

// Helpers
func ptrRole(r entity.UserRole) *entity.UserRole { return &r }
func ptrStr(s string) *string             { return &s }
