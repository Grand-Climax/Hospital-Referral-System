package usecase

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"

	"Hospital-Referral-System/internal/delivery/http/dto"
	"Hospital-Referral-System/internal/domain/entity"
	irepository "Hospital-Referral-System/internal/domain/interfaces/repository"
	iusecase "Hospital-Referral-System/internal/domain/interfaces/usecase"
	"Hospital-Referral-System/internal/infrastructure/crypto"
)

type referralUseCase struct {
	referralRepo      irepository.ReferralRepository
	clinicalRepo      irepository.ClinicalUpdateRepository
	outcomeRepo       irepository.ReferralOutcomeRepository
	networkRepo       irepository.NetworkRepository
	redirectionRepo   irepository.ReferralRedirectionRepository
	triageRepo        irepository.TriageQueueRepository
	deptRepo          irepository.DepartmentRepository
	attachmentUseCase iusecase.AttachmentUseCase
	notifUC           iusecase.NotificationUseCase
	inAppNotifUC      iusecase.InAppNotificationUseCase
	attachmentRepo    irepository.AttachmentRepository
	referralAccessRepo irepository.ReferralAccessRepository
	cryptoSvc         *crypto.PatientCryptoService
	mlUC              iusecase.MLUseCase
	mlRepo            irepository.MLPredictionRepository
}

// ReferralUseCaseFacade is an alias used in tests to reference the interface
type ReferralUseCaseFacade = iusecase.ReferralUseCase

func NewReferralUseCase(
	rRepo irepository.ReferralRepository,
	cRepo irepository.ClinicalUpdateRepository,
	oRepo irepository.ReferralOutcomeRepository,
	nRepo irepository.NetworkRepository,
	redirRepo irepository.ReferralRedirectionRepository,
	triageRepo irepository.TriageQueueRepository,
	aUC iusecase.AttachmentUseCase,
	notifUC iusecase.NotificationUseCase,
	inAppNotifUC iusecase.InAppNotificationUseCase,
	cryptoSvc *crypto.PatientCryptoService,
	deptRepo irepository.DepartmentRepository,
	attRepo irepository.AttachmentRepository,
	accessRepo irepository.ReferralAccessRepository,
	mlUC iusecase.MLUseCase,
	mlRepo irepository.MLPredictionRepository,
) iusecase.ReferralUseCase {
	return &referralUseCase{
		referralRepo:      rRepo,
		clinicalRepo:      cRepo,
		outcomeRepo:       oRepo,
		networkRepo:       nRepo,
		redirectionRepo:   redirRepo,
		triageRepo:        triageRepo,
		deptRepo:          deptRepo,
		attachmentUseCase: aUC,
		notifUC:           notifUC,
		inAppNotifUC:      inAppNotifUC,
		attachmentRepo:    attRepo,
		referralAccessRepo: accessRepo,
		cryptoSvc:         cryptoSvc,
		mlUC:              mlUC,
		mlRepo:            mlRepo,
	}
}

func (u *referralUseCase) triggerMLScore(referralID uuid.UUID) {
	if u.mlUC != nil {
		u.mlUC.ScheduleScore(referralID)
	}
}

// ---------------------------------------------------------
// Helper for History Log Generation & Status Validation
// ---------------------------------------------------------
func (u *referralUseCase) IsValidStatus(status string) bool {
	validStatuses := map[entity.ReferralStatus]bool{
		entity.StatusDraft:                 true,
		entity.StatusSubmitted:             true,
		entity.StatusUnderLiaisonReview:    true,
		entity.StatusForwarded:             true,
		entity.StatusUnderSpecialistReview: true,
		entity.StatusAccepted:              true,
		entity.StatusScheduled:             true,
		entity.StatusCompleted:             true,
		entity.StatusNeedRevision:          true,
		entity.StatusCancelled:             true,
		entity.StatusRejectedByLiaison:     true,
		entity.StatusRejectedBySpecialist:  true,
		entity.StatusRedirected:            true,
		entity.StatusDeceased:              true,
		entity.StatusRejectedAfterSend:     true,
	}
	return validStatuses[entity.ReferralStatus(status)]
}

func (u *referralUseCase) logStatusChange(ctx context.Context, id, userID uuid.UUID, from *entity.ReferralStatus, to entity.ReferralStatus, reason string) error {
	var reasonPtr *string
	if reason != "" {
		reasonPtr = &reason
	}
	return u.referralRepo.CreateStatusHistory(ctx, &entity.ReferralStatusHistory{
		ReferralID:  id,
		ChangedByID: userID,
		FromStatus:  from,
		ToStatus:    to,
		Reason:      reasonPtr,
	})
}

// ---------------------------------------------------------
// Doctor Actions
// ---------------------------------------------------------

func (u *referralUseCase) buildReferralEntity(doctorID, senderHospitalID uuid.UUID, req dto.CreateReferralRequest, refID uuid.UUID) (*entity.Referral, error) {
	var diagnoses []entity.ReferralDiagnosis
	for _, d := range req.Diagnoses {
		diagnoses = append(diagnoses, entity.ReferralDiagnosis{
			ICDCode:            d.ICDCode,
			IsPrimary:          d.IsPrimary,
			DiagnosisCertainty: entity.DiagnosisCertainty(d.DiagnosisCertainty),
		})
	}

	form := &entity.ReferralForm{
		ClinicalSummary:              req.ClinicalSummary,
		PatientHistory:               req.PatientHistory,
		PhysicalExaminationFindings:  req.PhysicalExaminationFindings,
		InvestigationResults:         req.InvestigationResults,
		TreatmentGivenBeforeReferral: req.TreatmentGivenBeforeReferral,
		MedicationOnTransfer:         req.MedicationOnTransfer,
		ReasonOfReferral:             req.ReasonOfReferral,
		ReasonForReferralCategory:    req.ReasonForReferralCategory,
		ConditionAtReferral:          req.ConditionAtReferral,
		ModeOfTransport:              req.ModeOfTransport,
		AccompanyingPersonName:       req.AccompanyingPersonName,
		AccompanyingPersonPhone:      req.AccompanyingPersonPhone,
	}

	status := entity.StatusDraft
	if req.Status == string(entity.StatusSubmitted) {
		status = entity.StatusSubmitted
	}

	referral := &entity.Referral{
		ID:                refID,
		ReferringDoctorID: doctorID,
		SenderHospitalID:  senderHospitalID,
		TargetHospitalID:  req.TargetHospitalID,
		TargetDeptID:      req.TargetDeptID,
		PatientID:         req.PatientID,
		LiaisonOfficerID:  req.LiaisonOfficerID,
		Status:            status,
		ReferralForm:      form,
		Diagnoses:         diagnoses,
	}

	if req.Vitals != nil {
		referral.Vitals = []entity.Vital{
			{
				ReferralID:      refID,
				SystolicBP:      req.Vitals.SystolicBP,
				DiastolicBP:     req.Vitals.DiastolicBP,
				HeartRate:       req.Vitals.HeartRate,
				SpO2:            req.Vitals.SpO2,
				Temperature:     req.Vitals.Temperature,
				RespiratoryRate: req.Vitals.RespiratoryRate,
				GCSScore:        req.Vitals.GCSScore,
			},
		}
	}

	if req.EmergencyDetail != nil {
		referral.EmergencyDetail = &entity.ReferralEmergencyDetail{
			EmergencyJustification: req.EmergencyDetail.EmergencyJustification,
		}
	}

	return referral, nil
}

func (u *referralUseCase) CreateDraftOrSubmit(ctx context.Context, doctorID uuid.UUID, senderHospitalID uuid.UUID, req dto.CreateReferralRequest) (*dto.ReferralCreationResponse, error) {
	// Verify Network
	isValidRoute, err := u.networkRepo.VerifyNetworkPathway(ctx, senderHospitalID, req.TargetHospitalID)
	if err != nil || !isValidRoute {
		return nil, errors.New("forbidden: no active referral network established between sender and target hospitals")
	}

	// Handle optional pre-minted ID
	refID := uuid.New()
	if req.ID != nil {
		refID = *req.ID
	}

	referral, err := u.buildReferralEntity(doctorID, senderHospitalID, req, refID)
	if err != nil {
		return nil, err
	}

	// Validation constraints
	if referral.Status == entity.StatusSubmitted {
		if len(referral.Diagnoses) == 0 {
			return nil, errors.New("cannot submit: at least one clinical diagnosis is required")
		}
		f := referral.ReferralForm
		if f.ClinicalSummary == "" || f.PatientHistory == "" || f.ReasonOfReferral == "" || f.ReasonForReferralCategory == nil || f.ConditionAtReferral == "" {
			return nil, errors.New("cannot submit: Clinical Summary, Patient History, Reason For Referral and Condition are structurally mandatory")
		}
	}

	if err := u.referralRepo.CreateReferralTransaction(ctx, referral); err != nil {
		return nil, err
	}

	// Log if submitted immediately
	if referral.Status == entity.StatusSubmitted {
		_ = u.logStatusChange(ctx, referral.ID, doctorID, nil, entity.StatusSubmitted, "Initial submission")
		_ = u.inAppNotifUC.CreateForEvent(ctx, "REFERRAL_SUBMITTED", referral.ID, doctorID)
		u.triggerMLScore(referral.ID)
	}

	// Load full referral for response
	ref, err := u.referralRepo.GetReferralByID(ctx, referral.ID)
	if err != nil {
		return nil, err
	}

	return &dto.ReferralCreationResponse{
		Referral: ref,
		BaseResponse: dto.BaseResponse{
			Success: true,
			Message: "Referral created successfully",
		},
	}, nil
}

func (u *referralUseCase) CreateReferralWithAttachments(ctx context.Context, doctorID, senderHospitalID uuid.UUID, req dto.CreateReferralRequest, refID uuid.UUID, uploads []iusecase.UploadedFileData) (*dto.ReferralCreationResponse, error) {
	// 1. Validate network pathway
	isValid, err := u.networkRepo.VerifyNetworkPathway(ctx, senderHospitalID, req.TargetHospitalID)
	if err != nil || !isValid {
		return nil, errors.New("forbidden: no active referral network established")
	}

	// 2. Build referral entity
	referral, err := u.buildReferralEntity(doctorID, senderHospitalID, req, refID)
	if err != nil {
		return nil, err
	}

	// 3. Start transaction
	err = u.referralRepo.CreateReferralTransaction(ctx, referral)
	if err != nil {
		return nil, err
	}

	// 4. Create attachment records
	for _, up := range uploads {
		att := &entity.Attachment{
			ReferralID:         refID,
			FileName:           up.FileName,
			FileType:           up.FileType,
			FileSize:           up.FileSize,
			Category:           up.Category,
			StoragePath:        up.URL,
			PublicID:           up.PublicID,
			VerificationStatus: up.Status,
			Metadata:           up.Metadata,
			RejectionReason:    up.Reason,
		}
		if err := u.attachmentRepo.Create(ctx, att); err != nil {
			return nil, fmt.Errorf("failed to save attachment: %w", err)
		}
		if up.Status == entity.VerificationRejected && referral.Status == entity.StatusSubmitted {
			referral.Status = entity.StatusNeedRevision
			msg := fmt.Sprintf("Attachment '%s' was rejected: %s", up.FileName, up.Reason)
			referral.RevisionReason = &msg
			u.referralRepo.UpdateReferralTransaction(ctx, referral)
		}
	}

	// 5. Reload referral with preloads
	ref, err := u.referralRepo.GetReferralByID(ctx, refID)
	if err != nil {
		return nil, err
	}

	if ref.Status == entity.StatusSubmitted {
		_ = u.logStatusChange(ctx, ref.ID, doctorID, nil, entity.StatusSubmitted, "Initial submission")
		_ = u.inAppNotifUC.CreateForEvent(ctx, "REFERRAL_SUBMITTED", ref.ID, doctorID)
		u.triggerMLScore(ref.ID)
	}

	return &dto.ReferralCreationResponse{
		Referral:     ref,
		BaseResponse: dto.BaseResponse{Success: true, Message: "Referral created successfully"},
	}, nil
}

func (u *referralUseCase) ListForDoctor(ctx context.Context, doctorID uuid.UUID, filter irepository.ReferralFilter) ([]entity.Referral, int64, error) {
	if filter.Status != "" && !u.IsValidStatus(filter.Status) {
		return nil, 0, errors.New("forbidden: unknown or invalid referral status")
	}
	referrals, count, err := u.referralRepo.ListForDoctor(ctx, doctorID, filter)
	if err != nil {
		return nil, 0, err
	}
	for i := range referrals {
		if referrals[i].Patient != nil {
			_ = referrals[i].Patient.DecryptFields(u.cryptoSvc)
		}
	}
	return referrals, count, nil
}

func (u *referralUseCase) ListAssignedReferrals(ctx context.Context, doctorID uuid.UUID, filter irepository.ReferralFilter, accessType string, includeRevoked bool) ([]entity.Referral, []entity.ReferralAccess, int64, error) {
	accesses, err := u.referralAccessRepo.ListByDoctor(ctx, doctorID)
	if err != nil {
		return nil, nil, 0, err
	}

	// Normalize the FE-friendly aliases ("treating", "consulting") to
	// the enum values stored in the DB ("TREATING_DOCTOR",
	// "CONSULTED_DOCTOR"). We still accept the raw enum form so older
	// callers don't break.
	switch strings.ToLower(strings.TrimSpace(accessType)) {
	case "treating", "treating_doctor":
		accessType = string(entity.AccessTreatingDoctor)
	case "consulting", "consulted", "consulted_doctor":
		accessType = string(entity.AccessConsultedDoctor)
	}

	filteredReferralIDs := make([]uuid.UUID, 0)
	accessTypeMap := make(map[uuid.UUID]*entity.ReferralAccess)

	for _, acc := range accesses {
		// Filter by access type
		if accessType != "" && !strings.EqualFold(string(acc.AccessType), accessType) {
			continue
		}
		// Filter by revocation
		if !includeRevoked && acc.RevokedAt != nil {
			continue
		}
		
		// If multiple accesses for same referral, prioritize non-revoked and latest
		if existing, ok := accessTypeMap[acc.ReferralID]; ok {
			if existing.RevokedAt != nil && acc.RevokedAt == nil {
				copyAcc := acc
				accessTypeMap[acc.ReferralID] = &copyAcc
			} else if (existing.RevokedAt == nil && acc.RevokedAt == nil) || (existing.RevokedAt != nil && acc.RevokedAt != nil) {
				if acc.GrantedAt.After(existing.GrantedAt) {
					copyAcc := acc
					accessTypeMap[acc.ReferralID] = &copyAcc
				}
			}
		} else {
			copyAcc := acc
			accessTypeMap[acc.ReferralID] = &copyAcc
			filteredReferralIDs = append(filteredReferralIDs, acc.ReferralID)
		}
	}

	if len(filteredReferralIDs) == 0 {
		return []entity.Referral{}, []entity.ReferralAccess{}, 0, nil
	}

	// Fetch referrals by IDs with filtering/pagination
	filter.ReferralIDs = filteredReferralIDs
	referrals, count, err := u.referralRepo.ListForDoctor(ctx, uuid.Nil, filter) // Use uuid.Nil to bypass creator check if IDs are provided
	if err != nil {
		return nil, nil, 0, err
	}

	// Build matching access slice in same order as referrals
	matchedAccesses := make([]entity.ReferralAccess, 0, len(referrals))
	for i := range referrals {
		if referrals[i].Patient != nil {
			_ = referrals[i].Patient.DecryptFields(u.cryptoSvc)
		}
		if acc, ok := accessTypeMap[referrals[i].ID]; ok {
			matchedAccesses = append(matchedAccesses, *acc)
		}
	}

	return referrals, matchedAccesses, count, nil
}

func (u *referralUseCase) GetDetailsForDoctor(ctx context.Context, id, doctorID uuid.UUID) (*entity.Referral, error) {
	ref, err := u.referralRepo.GetReferralByID(ctx, id)
	if err != nil {
		return nil, err
	}
	// Authorize either as the sender OR as an active grantee
	// (TREATING_DOCTOR / CONSULTED_DOCTOR). Without the second branch,
	// treating doctors who see the row via /doctor/referrals/assigned
	// cannot open the detail page that lists vitals, ICD codes, and
	// clinical history.
	if ref.ReferringDoctorID != doctorID {
		hasAccess, accessErr := u.referralAccessRepo.CheckAccess(ctx, id, doctorID)
		if accessErr != nil {
			return nil, accessErr
		}
		if !hasAccess {
			return nil, errors.New("unauthorized: not the sender and no active access grant")
		}
	}
	if ref.Patient != nil {
		_ = ref.Patient.DecryptFields(u.cryptoSvc)
	}
	return ref, nil
}

func (u *referralUseCase) UpdateAndResubmit(ctx context.Context, id, doctorID uuid.UUID, req dto.UpdateReferralRequest, submit bool) (*dto.ReferralCreationResponse, error) {
	existing, err := u.referralRepo.GetReferralByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if existing.ReferringDoctorID != doctorID {
		return nil, errors.New("unauthorized: only the creator can update this draft/referral")
	}

	// Redundancy Check
	if existing.Status == entity.StatusSubmitted || existing.Status == entity.StatusForwarded ||
		existing.Status == entity.StatusUnderLiaisonReview || existing.Status == entity.StatusUnderSpecialistReview {
		return nil, errors.New("referral is already submitted; please wait for review")
	}

	// Immutability Check
	if existing.Status != entity.StatusDraft && existing.Status != entity.StatusNeedRevision {
		return nil, errors.New("forbidden: referral is immutable in current state")
	}

	oldStatus := existing.Status
	// status fork: if submit is true, we move to SUBMITTED. Otherwise, we keep existing status (Draft/NeedRevision).
	nextStatus := existing.Status
	if submit {
		nextStatus = entity.StatusSubmitted
	}

	// Map
	existing.TargetHospitalID = req.TargetHospitalID
	existing.TargetDeptID = req.TargetDeptID
	existing.LiaisonOfficerID = req.LiaisonOfficerID

	existing.ReferralForm.ClinicalSummary = req.ClinicalSummary
	existing.ReferralForm.PatientHistory = req.PatientHistory
	existing.ReferralForm.PhysicalExaminationFindings = req.PhysicalExaminationFindings
	existing.ReferralForm.InvestigationResults = req.InvestigationResults
	existing.ReferralForm.TreatmentGivenBeforeReferral = req.TreatmentGivenBeforeReferral
	existing.ReferralForm.MedicationOnTransfer = req.MedicationOnTransfer
	existing.ReferralForm.ReasonOfReferral = req.ReasonOfReferral
	existing.ReferralForm.ReasonForReferralCategory = req.ReasonForReferralCategory
	existing.ReferralForm.ConditionAtReferral = req.ConditionAtReferral
	existing.ReferralForm.ModeOfTransport = req.ModeOfTransport

	var diagnoses []entity.ReferralDiagnosis
	for _, d := range req.Diagnoses {
		diagnoses = append(diagnoses, entity.ReferralDiagnosis{
			ReferralID:         existing.ID,
			ICDCode:            d.ICDCode,
			IsPrimary:          d.IsPrimary,
			DiagnosisCertainty: entity.DiagnosisCertainty(d.DiagnosisCertainty),
		})
	}
	existing.Diagnoses = diagnoses

	if req.Vitals != nil {
		existing.Vitals = []entity.Vital{{
			ReferralID:      existing.ID,
			SystolicBP:      req.Vitals.SystolicBP,
			DiastolicBP:     req.Vitals.DiastolicBP,
			HeartRate:       req.Vitals.HeartRate,
			SpO2:            req.Vitals.SpO2,
			Temperature:     req.Vitals.Temperature,
			RespiratoryRate: req.Vitals.RespiratoryRate,
			GCSScore:        req.Vitals.GCSScore,
		}}
	} else {
		existing.Vitals = nil
	}

	if nextStatus == entity.StatusSubmitted {
		if len(existing.Diagnoses) == 0 {
			return nil, errors.New("cannot submit: at least one diagnosis is required")
		}
		// Clear revision/rejection reasons as they are now addressed
		existing.RevisionReason = nil
		existing.RejectionReason = nil
	}

	existing.Status = nextStatus
	if err := u.referralRepo.UpdateReferralTransaction(ctx, existing); err != nil {
		return nil, err
	}

	if oldStatus != nextStatus {
		_ = u.logStatusChange(ctx, existing.ID, doctorID, &oldStatus, nextStatus, "")
		if nextStatus == entity.StatusSubmitted {
			_ = u.inAppNotifUC.CreateForEvent(ctx, "REFERRAL_SUBMITTED", existing.ID, doctorID)
			u.triggerMLScore(existing.ID)
		}
	}

	return &dto.ReferralCreationResponse{
		Referral: existing,
		BaseResponse: dto.BaseResponse{
			Success: true,
			Message: "Referral updated successfully",
		},
	}, nil
}

func (u *referralUseCase) CancelReferral(ctx context.Context, id, doctorID uuid.UUID, reason string) error {
	existing, err := u.referralRepo.GetReferralByID(ctx, id)
	if err != nil {
		return err
	}
	if existing.ReferringDoctorID != doctorID {
		return errors.New("unauthorized")
	}
	if existing.Status == entity.StatusCancelled {
		return errors.New("referral is already cancelled")
	}
	if existing.Status != entity.StatusDraft && existing.Status != entity.StatusNeedRevision {
		return errors.New("forbidden: cannot cancel once in active review pipeline")
	}

	oldStatus := existing.Status
	existing.Status = entity.StatusCancelled

	if err := u.referralRepo.UpdateReferralTransaction(ctx, existing); err != nil {
		return err
	}

	return u.logStatusChange(ctx, id, doctorID, &oldStatus, entity.StatusCancelled, reason)
}

func (u *referralUseCase) MarkDeceased(ctx context.Context, referralID, userID uuid.UUID, role entity.UserRole, hospID uuid.UUID, reason string) error {
	// Authorization: only referring doctor, liaison of sender hospital, specialist of target hospital, or system admin
	switch role {
	case entity.RoleReferringDoctor, entity.RoleLiaisonOfficer, entity.RoleReceivingSpecialist, entity.RoleSystemSuperAdmin:
		// allowed
	default:
		return errors.New("unauthorized: only clinical staff can mark a referral as deceased")
	}

	ref, err := u.referralRepo.GetReferralByID(ctx, referralID)
	if err != nil {
		return err
	}

	// Verify the user has a legitimate connection to this referral
	if role == entity.RoleLiaisonOfficer && ref.SenderHospitalID != hospID {
		return errors.New("unauthorized: liaison belongs to a different hospital")
	}
	if role == entity.RoleReceivingSpecialist && ref.TargetHospitalID != hospID {
		return errors.New("unauthorized: specialist belongs to a different hospital")
	}
	if role == entity.RoleReferringDoctor && ref.ReferringDoctorID != userID {
		return errors.New("unauthorized: you are not the referring doctor")
	}

	oldStatus := ref.Status
	ref.Status = entity.StatusDeceased
	ref.IsArchived = true
	now := time.Now()
	ref.ArchivedAt = &now

	if err := u.referralRepo.UpdateReferralTransaction(ctx, ref); err != nil {
		return err
	}

	_ = u.triageRepo.DeleteByReferralID(ctx, referralID)
	_ = u.logStatusChange(ctx, referralID, userID, &oldStatus, entity.StatusDeceased, reason)
	_ = u.inAppNotifUC.CreateForEvent(ctx, "PATIENT_DECEASED", referralID, userID)

	return nil
}

func (u *referralUseCase) RejectAfterSend(ctx context.Context, referralID, userID, hospID uuid.UUID, role entity.UserRole, reason string) error {
	ref, err := u.referralRepo.GetReferralByID(ctx, referralID)
	if err != nil {
		return err
	}

	// Authorization: only referring doctor or liaison of sender hospital
	isDoctor := ref.ReferringDoctorID == userID
	isLiaison := role == entity.RoleLiaisonOfficer && ref.SenderHospitalID == hospID

	if !isDoctor && !isLiaison {
		return errors.New("unauthorized: only the referring doctor or a liaison of the sender hospital can reject after send")
	}

	// Allowed statuses
	allowed := map[entity.ReferralStatus]bool{
		entity.StatusSubmitted:             true,
		entity.StatusUnderLiaisonReview:    true,
		entity.StatusForwarded:             true,
		entity.StatusUnderSpecialistReview: true,
		entity.StatusAccepted:              true,
	}
	if !allowed[ref.Status] {
		return fmt.Errorf("cannot reject after send: referral is already %s", ref.Status)
	}

	oldStatus := ref.Status
	ref.Status = entity.StatusRejectedAfterSend
	ref.RejectionReason = &reason

	if err := u.referralRepo.UpdateReferralTransaction(ctx, ref); err != nil {
		return err
	}

	// Remove from triage queue if present
	_ = u.triageRepo.DeleteByReferralID(ctx, referralID)

	// Log status change
	_ = u.logStatusChange(ctx, referralID, userID, &oldStatus, entity.StatusRejectedAfterSend, reason)

	// Notification
	if isDoctor {
		_ = u.inAppNotifUC.CreateForEvent(ctx, "REFERRAL_REJECTED_AFTER_SEND", referralID, userID)
	} else {
		_ = u.inAppNotifUC.CreateForEvent(ctx, "REFERRAL_REJECTED_AFTER_SEND_BY_LIAISON", referralID, userID)
	}

	return nil
}

func (u *referralUseCase) DeleteAttachmentsByReferralID(ctx context.Context, id, doctorID uuid.UUID) error {
	existing, err := u.referralRepo.GetReferralByID(ctx, id)
	if err != nil {
		return err
	}

	if existing.ReferringDoctorID != doctorID {
		return errors.New("unauthorized: only the referring doctor can manage attachments")
	}

	if existing.Status != entity.StatusDraft && existing.Status != entity.StatusNeedRevision {
		return errors.New("forbidden: attachments can only be removed in DRAFT or NEED_REVISION status")
	}

	if len(existing.Attachments) == 0 {
		return nil
	}

	for _, att := range existing.Attachments {
		if err := u.attachmentUseCase.DeleteAttachment(ctx, att.ID); err != nil {
			return err
		}
	}

	return nil
}

func (u *referralUseCase) GetDoctorDashboardStats(ctx context.Context, doctorID uuid.UUID) (*dto.DoctorDashboardStats, error) {
	total, pending, accepted, critical, err := u.referralRepo.GetDoctorStats(ctx, doctorID)
	if err != nil {
		return nil, err
	}
	return &dto.DoctorDashboardStats{
		TotalReferrals: total,
		Pending:        pending,
		Accepted:       accepted,
		Critical:       critical,
	}, nil
}

func (u *referralUseCase) GetLatestPendingReferrals(ctx context.Context, doctorID uuid.UUID, limit int) ([]dto.ListReferralResponse, error) {
	referrals, err := u.referralRepo.GetLatestPendingForDoctor(ctx, doctorID, limit)
	if err != nil {
		return nil, err
	}

	responseData := make([]dto.ListReferralResponse, 0, len(referrals))
	for i := range referrals {
		if referrals[i].Patient != nil {
			_ = referrals[i].Patient.DecryptFields(u.cryptoSvc)
		}
		responseData = append(responseData, dto.MapListReferralResponse(referrals[i]))
	}
	return responseData, nil
}

// ---------------------------------------------------------
// Liaison Actions
// ---------------------------------------------------------

func (u *referralUseCase) ListOutgoingForLiaison(ctx context.Context, hospID uuid.UUID, filter irepository.ReferralFilter) ([]entity.Referral, int64, error) {
	if filter.Status != "" && !u.IsValidStatus(filter.Status) {
		return nil, 0, errors.New("forbidden: unknown or invalid referral status")
	}
	referrals, count, err := u.referralRepo.ListOutgoingForLiaison(ctx, hospID, filter)
	if err != nil {
		return nil, 0, err
	}
	for i := range referrals {
		if referrals[i].Patient != nil {
			_ = referrals[i].Patient.DecryptFields(u.cryptoSvc)
		}
	}
	return referrals, count, nil
}

func (u *referralUseCase) ListIncomingForLiaison(ctx context.Context, hospID uuid.UUID, filter irepository.ReferralFilter) ([]entity.Referral, int64, error) {
	if filter.Status != "" && !u.IsValidStatus(filter.Status) {
		return nil, 0, errors.New("forbidden: unknown or invalid referral status")
	}
	referrals, count, err := u.referralRepo.ListIncomingForLiaison(ctx, hospID, filter)
	if err != nil {
		return nil, 0, err
	}
	for i := range referrals {
		if referrals[i].Patient != nil {
			_ = referrals[i].Patient.DecryptFields(u.cryptoSvc)
		}
	}
	return referrals, count, nil
}

func (u *referralUseCase) GetLiaisonDashboardStats(ctx context.Context, hospID uuid.UUID) (*iusecase.LiaisonDashboardStats, error) {
	now := time.Now()
	currentStart := now.AddDate(0, 0, -30)
	previousStart := now.AddDate(0, 0, -60)

	// --- Total Referrals ---
	currentTotal, _ := u.referralRepo.CountBySenderHospitalAndStatuses(ctx, hospID, nil, true, &currentStart, nil)
	previousTotal, _ := u.referralRepo.CountBySenderHospitalAndStatuses(ctx, hospID, nil, true, &previousStart, &currentStart)

	// --- Pending Review ---
	pendingStatuses := []entity.ReferralStatus{
		entity.StatusSubmitted,
		entity.StatusUnderLiaisonReview,
	}
	currentPending, _ := u.referralRepo.CountBySenderHospitalAndStatuses(ctx, hospID, pendingStatuses, false, &currentStart, nil)
	previousPending, _ := u.referralRepo.CountBySenderHospitalAndStatuses(ctx, hospID, pendingStatuses, false, &previousStart, &currentStart)

	// --- Approved Today ---
	currentApprovedToday, _ := u.referralRepo.CountAcceptedOrCompletedToday(ctx, hospID)

	// --- Rejected ---
	rejectedStatuses := []entity.ReferralStatus{
		entity.StatusRejectedByLiaison,
		entity.StatusRejectedBySpecialist,
		entity.StatusRejectedAfterSend,
	}
	currentRejected, _ := u.referralRepo.CountBySenderHospitalAndStatuses(ctx, hospID, rejectedStatuses, false, &currentStart, nil)
	previousRejected, _ := u.referralRepo.CountBySenderHospitalAndStatuses(ctx, hospID, rejectedStatuses, false, &previousStart, &currentStart)

	// Helper: calculate percentage change
	calcChange := func(current, previous int64) float64 {
		if previous == 0 {
			if current == 0 {
				return 0
			}
			return 100.0 // first time data, treat as 100% increase
		}
		return float64(current-previous) / float64(previous) * 100.0
	}

	return &iusecase.LiaisonDashboardStats{
		TotalReferrals: iusecase.StatItem{
			Count:  currentTotal,
			Change: calcChange(currentTotal, previousTotal),
		},
		PendingReview: iusecase.StatItem{
			Count:  currentPending,
			Change: calcChange(currentPending, previousPending),
		},
		ApprovedToday: iusecase.StatItem{
			Count:  currentApprovedToday,
			Change: 0, // daily metric, no historical comparison for now
		},
		Rejected: iusecase.StatItem{
			Count:  currentRejected,
			Change: calcChange(currentRejected, previousRejected),
		},
	}, nil
}

func (u *referralUseCase) GetDetailsForLiaison(ctx context.Context, id, hospID uuid.UUID) (*entity.Referral, error) {
	ref, err := u.referralRepo.GetReferralByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if ref.SenderHospitalID != hospID {
		return nil, errors.New("unauthorized: referral does not originate from your hospital")
	}

	// Status constraint: Liaisons cannot view drafts (consistent with list)
	if ref.Status == entity.StatusDraft {
		return nil, errors.New("forbidden: cannot view drafts")
	}

	if ref.Patient != nil {
		_ = ref.Patient.DecryptFields(u.cryptoSvc)
	}
	return ref, nil
}

func (u *referralUseCase) LiaisonRead(ctx context.Context, id, liaisonID, hospID uuid.UUID) error {
	ref, err := u.referralRepo.GetReferralByID(ctx, id)
	if err != nil {
		return err
	}
	if ref.SenderHospitalID != hospID {
		return errors.New("unauthorized")
	}
	if ref.Status == entity.StatusUnderLiaisonReview {
		return errors.New("referral is already under liaison review")
	}
	if ref.Status != entity.StatusSubmitted {
		return errors.New("invalid status transition: must be SUBMITTED")
	}

	oldStatus := ref.Status
	ref.Status = entity.StatusUnderLiaisonReview

	if err := u.referralRepo.UpdateReferralTransaction(ctx, ref); err != nil {
		return err
	}
	return u.logStatusChange(ctx, id, liaisonID, &oldStatus, entity.StatusUnderLiaisonReview, "")
}

func (u *referralUseCase) UpdateReviewChecklist(ctx context.Context, referralID, liaisonID, hospID uuid.UUID, req dto.ReviewChecklistRequest) error {
	ref, err := u.referralRepo.GetReferralByID(ctx, referralID)
	if err != nil {
		return err
	}
	if ref.SenderHospitalID != hospID {
		return errors.New("unauthorized: liaison belongs to a different hospital")
	}

	// Status Restriction: Only allow updates in SUBMITTED or UNDER_LIAISON_REVIEW
	if ref.Status != entity.StatusSubmitted && ref.Status != entity.StatusUnderLiaisonReview {
		return fmt.Errorf("forbidden: checklist can only be updated when referral is SUBMITTED or UNDER_LIAISON_REVIEW, current status: %s", ref.Status)
	}

	updates := irepository.ReferralUpdateFields{
		PatientIdentityVerified: req.PatientIdentityVerified,
		ClinicalHistoryAttached: req.ClinicalHistoryAttached,
		VitalsIncluded:          req.VitalsIncluded,
		AttachmentsIncluded:     req.AttachmentsIncluded,
	}

	return u.referralRepo.UpdateFields(ctx, referralID, updates)
}

func (u *referralUseCase) GetReviewChecklist(ctx context.Context, referralID, hospID uuid.UUID) (*dto.ReviewChecklistResponse, error) {
	ref, err := u.referralRepo.GetReferralByID(ctx, referralID)
	if err != nil {
		return nil, err
	}
	if ref.SenderHospitalID != hospID {
		return nil, errors.New("unauthorized")
	}

	return &dto.ReviewChecklistResponse{
		PatientIdentityVerified: ref.PatientIdentityVerified,
		ClinicalHistoryAttached: ref.ClinicalHistoryAttached,
		VitalsIncluded:          ref.VitalsIncluded,
		AttachmentsIncluded:     ref.AttachmentsIncluded,
	}, nil
}

func (u *referralUseCase) LiaisonForward(ctx context.Context, id, liaisonID, hospID uuid.UUID, comment string) error {
	ref, err := u.referralRepo.GetReferralByID(ctx, id)
	if err != nil {
		return err
	}

	if ref.SenderHospitalID != hospID {
		return errors.New("unauthorized hospital access")
	}

	if ref.Status != entity.StatusSubmitted && ref.Status != entity.StatusUnderLiaisonReview {
		return errors.New("invalid referral status for forwarding")
	}

	// Gatekeeper: Ensure all attachments are confirmed and none are rejected
	for _, att := range ref.Attachments {
		if att.VerificationStatus == entity.VerificationPending {
			return errors.New("referral attachments need to be confirmed before liaison action")
		}
		if att.VerificationStatus == entity.VerificationRejected {
			return errors.New("referral has rejected attachments and requires revision by the doctor")
		}
	}

	// Checklist Enforcement Logic
	condition := ""
	if ref.ReferralForm != nil {
		condition = strings.ToLower(ref.ReferralForm.ConditionAtReferral)
	}

	switch condition {
	case "stable":
		if !ref.PatientIdentityVerified || !ref.ClinicalHistoryAttached || !ref.VitalsIncluded || !ref.AttachmentsIncluded {
			return errors.New("all 4 review checklist items must be verified before forwarding a stable referral: Patient Identity, Clinical History, Vitals, Attachments")
		}
	case "emergency", "critical":
		if !ref.PatientIdentityVerified {
			return errors.New("patient identity must be verified before forwarding an emergency/critical referral")
		}
	}

	oldStatus := ref.Status
	ref.Status = entity.StatusForwarded
	if err := u.referralRepo.UpdateReferralTransaction(ctx, ref); err != nil {
		return err
	}

	history := &entity.ReferralStatusHistory{
		ReferralID:  id,
		ChangedByID: liaisonID,
		FromStatus:  &oldStatus,
		ToStatus:    entity.StatusForwarded,
		Reason:      &comment,
	}
	if err := u.referralRepo.CreateStatusHistory(ctx, history); err != nil {
		return err
	}

	_ = u.inAppNotifUC.CreateForEvent(ctx, "REFERRAL_FORWARDED", id, liaisonID)

	return nil
}

func (u *referralUseCase) LiaisonReject(ctx context.Context, id, liaisonID, hospID uuid.UUID, reason string) error {
	ref, err := u.referralRepo.GetReferralByID(ctx, id)
	if err != nil {
		return err
	}

	if ref.SenderHospitalID != hospID {
		return errors.New("unauthorized: your hospital did not initiate this referral")
	}

	if ref.Status != entity.StatusSubmitted && ref.Status != entity.StatusUnderLiaisonReview {
		return errors.New("invalid referral status for rejection; it must be in SUBMITTED or UNDER_LIAISON_REVIEW")
	}

	// Gatekeeper: Ensure all attachments are confirmed and none are rejected
	for _, att := range ref.Attachments {
		if att.VerificationStatus == entity.VerificationPending {
			return errors.New("referral attachments need to be confirmed before liaison action")
		}
		if att.VerificationStatus == entity.VerificationRejected {
			return errors.New("referral has rejected attachments and requires revision by the doctor")
		}
	}

	oldStatus := ref.Status
	ref.Status = entity.StatusRejectedByLiaison
	ref.RejectionReason = &reason // Persist reason to entity

	if err := u.referralRepo.UpdateReferralTransaction(ctx, ref); err != nil {
		return err
	}

	history := &entity.ReferralStatusHistory{
		ReferralID:  id,
		ChangedByID: liaisonID,
		FromStatus:  &oldStatus,
		ToStatus:    entity.StatusRejectedByLiaison,
		Reason:      &reason,
	}
	if err := u.referralRepo.CreateStatusHistory(ctx, history); err != nil {
		return err
	}

	_ = u.inAppNotifUC.CreateForEvent(ctx, "REFERRAL_REJECTED_BY_LIAISON", id, liaisonID)

	return nil
}

func (u *referralUseCase) LiaisonRevise(ctx context.Context, id, liaisonID, hospID uuid.UUID, reason string) error {
	ref, err := u.referralRepo.GetReferralByID(ctx, id)
	if err != nil {
		return err
	}

	if ref.SenderHospitalID != hospID {
		return errors.New("unauthorized: your hospital did not initiate this referral")
	}

	if ref.Status != entity.StatusSubmitted && ref.Status != entity.StatusUnderLiaisonReview {
		return errors.New("invalid referral status for revision; it must be in SUBMITTED or UNDER_LIAISON_REVIEW")
	}

	// Gatekeeper: Ensure all attachments are confirmed and none are rejected
	for _, att := range ref.Attachments {
		if att.VerificationStatus == entity.VerificationPending {
			return errors.New("referral attachments need to be confirmed before liaison action")
		}
		if att.VerificationStatus == entity.VerificationRejected {
			return errors.New("referral has rejected attachments and requires revision by the doctor")
		}
	}

	oldStatus := ref.Status
	ref.Status = entity.StatusNeedRevision
	ref.RevisionReason = &reason // Persist reason to entity

	if err := u.referralRepo.UpdateReferralTransaction(ctx, ref); err != nil {
		return err
	}

	history := &entity.ReferralStatusHistory{
		ReferralID:  id,
		ChangedByID: liaisonID,
		FromStatus:  &oldStatus,
		ToStatus:    entity.StatusNeedRevision,
		Reason:      &reason,
	}

	if err := u.referralRepo.CreateStatusHistory(ctx, history); err != nil {
		return err
	}

	_ = u.inAppNotifUC.CreateForEvent(ctx, "REFERRAL_NEEDS_REVISION", id, liaisonID)

	return nil
}

func (u *referralUseCase) LiaisonUnassignSpecialist(ctx context.Context, id, liaisonID, hospID uuid.UUID, reason string) error {
	ref, err := u.referralRepo.GetReferralByID(ctx, id)
	if err != nil {
		return err
	}

	if ref.TargetHospitalID != hospID {
		return errors.New("unauthorized hospital access")
	}

	if ref.Status != entity.StatusUnderSpecialistReview {
		return errors.New("referral is not currently under specialist review")
	}

	oldStatus := ref.Status
	// Revert status to REDIRECTED if there is any redirection history, otherwise FORWARDED
	var targetStatus entity.ReferralStatus
	redirections, err := u.redirectionRepo.ListByReferralID(ctx, id)
	if err == nil && len(redirections) > 0 {
		targetStatus = entity.StatusRedirected
	} else {
		targetStatus = entity.StatusForwarded
	}

	ref.Status = targetStatus
	ref.SpecialistID = nil

	if err := u.referralRepo.UpdateReferralTransaction(ctx, ref); err != nil {
		return err
	}

	history := &entity.ReferralStatusHistory{
		ReferralID:  id,
		ChangedByID: liaisonID,
		FromStatus:  &oldStatus,
		ToStatus:    targetStatus,
		Reason:      &reason,
	}
	return u.referralRepo.CreateStatusHistory(ctx, history)
}

// ---------------------------------------------------------
// Specialist Actions
// ---------------------------------------------------------

func (u *referralUseCase) ListForSpecialist(ctx context.Context, hospID, specialistID uuid.UUID, filter irepository.ReferralFilter) ([]entity.Referral, int64, error) {
	if filter.Status != "" && !u.IsValidStatus(filter.Status) {
		return nil, 0, errors.New("forbidden: unknown or invalid referral status")
	}
	referrals, count, err := u.referralRepo.ListForSpecialist(ctx, hospID, filter)
	if err != nil {
		return nil, 0, err
	}
	for i := range referrals {
		if referrals[i].Patient != nil {
			_ = referrals[i].Patient.DecryptFields(u.cryptoSvc)
		}
	}
	u.enrichReferralsMLBatch(ctx, referrals)
	return referrals, count, nil
}

func (u *referralUseCase) GetDetailsForSpecialist(ctx context.Context, id, hospID uuid.UUID) (*entity.Referral, error) {
	ref, err := u.referralRepo.GetReferralByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if ref.TargetHospitalID != hospID {
		return nil, errors.New("unauthorized: this referral is targeted to another hospital")
	}

	// Status constraint: same as ListForSpecialist
	allowedStatuses := map[entity.ReferralStatus]bool{
		entity.StatusForwarded:             true,
		entity.StatusUnderSpecialistReview: true,
		entity.StatusAccepted:              true,
		entity.StatusScheduled:             true,
		entity.StatusCompleted:             true,
		entity.StatusRejectedBySpecialist:  true,
		entity.StatusRedirected:            true,
		entity.StatusRejectedAfterSend:     true,
	}

	if !allowedStatuses[ref.Status] {
		return nil, errors.New("unauthorized: referral has not yet been forwarded to your hospital")
	}

	if ref.Patient != nil {
		_ = ref.Patient.DecryptFields(u.cryptoSvc)
	}
	u.enrichReferralML(ctx, ref)
	return ref, nil
}

func (u *referralUseCase) SpecialistRead(ctx context.Context, id, specialistID, hospID uuid.UUID) error {
	ref, err := u.referralRepo.GetReferralByID(ctx, id)
	if err != nil {
		return err
	}
	if ref.TargetHospitalID != hospID {
		return errors.New("unauthorized")
	}
	if ref.Status == entity.StatusUnderSpecialistReview {
		return errors.New("referral is already under specialist review")
	}
	if ref.Status != entity.StatusForwarded && ref.Status != entity.StatusRedirected {
		return errors.New("invalid status: must be FORWARDED or REDIRECTED")
	}

	oldStatus := ref.Status
	ref.Status = entity.StatusUnderSpecialistReview
	ref.SpecialistID = &specialistID

	if err := u.referralRepo.UpdateReferralTransaction(ctx, ref); err != nil {
		return err
	}
	return u.logStatusChange(ctx, id, specialistID, &oldStatus, entity.StatusUnderSpecialistReview, "")
}

func (u *referralUseCase) SpecialistAccept(ctx context.Context, id, specialistID, hospID uuid.UUID, severityScore *float64) error {
	ref, err := u.referralRepo.GetReferralByID(ctx, id)
	if err != nil {
		return err
	}

	if ref.TargetHospitalID != hospID {
		return errors.New("unauthorized hospital access")
	}

	if ref.Status != entity.StatusUnderSpecialistReview {
		return errors.New("referral must be under specialist review to be accepted")
	}

	// Ownership check: only the assigned specialist can accept
	if ref.SpecialistID != nil && *ref.SpecialistID != specialistID {
		return errors.New("referral is claimed by another specialist")
	}

	if ref.MLStatus == entity.MLStatusPending {
		// Self-healing: if stuck PENDING for more than 2.5 minutes, auto-fail it to prevent blocking the clinical flow indefinitely
		if ref.MLRunStartedAt != nil && time.Since(*ref.MLRunStartedAt) > 150*time.Second {
			log.Printf("ML status was stuck PENDING for referral %s for over 2.5 minutes. Auto-marking as FAILED to unblock clinical flow.", ref.ID)
			ref.MLStatus = entity.MLStatusFailed
			now := time.Now()
			ref.MLLastFailedAt = &now
			ref.MLRunStartedAt = nil
			errMsg := "ML scoring timed out / stuck in PENDING for more than 2.5 minutes"
			ref.MLLastError = &errMsg
			_ = u.referralRepo.Update(ctx, ref)
		} else {
			return errors.New("ML severity scoring is in progress; wait for scoring to finish or set manual severity override via the ML severity override endpoint")
		}
	}

	// Severity gate: ML success, manual override, or explicit score on accept
	if severityScore == nil && ref.MLSeverityScore == nil {
		return errors.New("severity score must be set before accepting a referral; wait for ML scoring or use the ml-severity-override endpoint")
	}

	oldStatus := ref.Status
	ref.Status = entity.StatusAccepted
	if severityScore != nil {
		ref.MLSeverityScore = severityScore
	}

	if ref.TriageStatus != entity.TriageOverridden && u.mlUC != nil {
		if err := u.mlUC.SendFeedbackAccept(ctx, id); err != nil {
			log.Printf("ml feedback accept referral %s: %v", id, err)
		} else {
			ref.TriageStatus = entity.TriageReviewed
		}
	}

	if err := u.referralRepo.UpdateReferralTransaction(ctx, ref); err != nil {
		return err
	}

	reason := "Accepted by specialist"
	history := &entity.ReferralStatusHistory{
		ReferralID:  id,
		ChangedByID: specialistID,
		FromStatus:  &oldStatus,
		ToStatus:    entity.StatusAccepted,
		Reason:      &reason,
	}

	if err := u.referralRepo.CreateStatusHistory(ctx, history); err != nil {
		return err
	}

	_ = u.inAppNotifUC.CreateForEvent(ctx, "REFERRAL_ACCEPTED", id, specialistID)

	// Queue Notification
	hospitalName := "the hospital"
	deptName := "the department"
	if ref.ReceiverHospital != nil {
		hospitalName = ref.ReceiverHospital.Name
	}
	if ref.TargetDepartment != nil {
		deptName = ref.TargetDepartment.Name
	}
	message := fmt.Sprintf("Your referral to %s, %s has been accepted. Please wait for your appointment date.", hospitalName, deptName)
	_ = u.notifUC.QueueNotification(ctx, id, entity.NotifyAcceptance, message)

	return nil
}

func (u *referralUseCase) SpecialistReject(ctx context.Context, id, specialistID, hospID uuid.UUID, reason string) error {
	ref, err := u.referralRepo.GetReferralByID(ctx, id)
	if err != nil {
		return err
	}

	if ref.TargetHospitalID != hospID {
		return errors.New("unauthorized: this referral is targeted to another hospital")
	}

	if ref.Status != entity.StatusUnderSpecialistReview {
		return errors.New("invalid status: referral must be in UNDER_SPECIALIST_REVIEW to be rejected by you")
	}

	// Ownership check: only the assigned specialist can reject
	if ref.SpecialistID != nil && *ref.SpecialistID != specialistID {
		return errors.New("forbidden: this referral is already claimed and being reviewed by another specialist")
	}

	oldStatus := ref.Status
	ref.Status = entity.StatusRejectedBySpecialist
	ref.RejectionReason = &reason // Persist reason to entity

	if err := u.referralRepo.UpdateReferralTransaction(ctx, ref); err != nil {
		return err
	}

	history := &entity.ReferralStatusHistory{
		ReferralID:  id,
		ChangedByID: specialistID,
		FromStatus:  &oldStatus,
		ToStatus:    entity.StatusRejectedBySpecialist,
		Reason:      &reason,
	}
	if err := u.referralRepo.CreateStatusHistory(ctx, history); err != nil {
		return err
	}

	_ = u.inAppNotifUC.CreateForEvent(ctx, "REFERRAL_REJECTED_BY_SPECIALIST", id, specialistID)

	return nil
}

func (u *referralUseCase) SpecialistRelease(ctx context.Context, id, specialistID, hospID uuid.UUID, reason string) error {
	ref, err := u.referralRepo.GetReferralByID(ctx, id)
	if err != nil {
		return err
	}

	if ref.TargetHospitalID != hospID {
		return errors.New("unauthorized hospital access")
	}

	// Can only release if currently assigned to this specialist
	if ref.SpecialistID == nil || *ref.SpecialistID != specialistID {
		return errors.New("you are not the specialist assigned to this referral")
	}

	if ref.Status != entity.StatusUnderSpecialistReview {
		return errors.New("invalid status for release")
	}

	oldStatus := ref.Status
	// Revert status to REDIRECTED if there is any redirection history, otherwise FORWARDED
	var targetStatus entity.ReferralStatus
	redirections, err := u.redirectionRepo.ListByReferralID(ctx, id)
	if err == nil && len(redirections) > 0 {
		targetStatus = entity.StatusRedirected
	} else {
		targetStatus = entity.StatusForwarded
	}

	ref.Status = targetStatus
	ref.SpecialistID = nil

	if err := u.referralRepo.UpdateReferralTransaction(ctx, ref); err != nil {
		return err
	}

	history := &entity.ReferralStatusHistory{
		ReferralID:  id,
		ChangedByID: specialistID,
		FromStatus:  &oldStatus,
		ToStatus:    targetStatus,
		Reason:      &reason,
	}
	return u.referralRepo.CreateStatusHistory(ctx, history)
}

func (u *referralUseCase) SpecialistRerunML(ctx context.Context, id, specialistID, hospID uuid.UUID) error {
	if _, err := u.GetDetailsForSpecialist(ctx, id, hospID); err != nil {
		return err
	}
	ref, err := u.referralRepo.GetReferralByID(ctx, id)
	if err != nil {
		return err
	}
	if ref.SpecialistID == nil || *ref.SpecialistID != specialistID {
		return errors.New("ML rerun is only allowed for the specialist assigned to this referral")
	}

	// If stuck PENDING for > 2.5 minutes, or FAILED, bypass constraints to allow recovery
	isStuckOrFailed := ref.MLStatus == entity.MLStatusFailed || 
		(ref.MLStatus == entity.MLStatusPending && ref.MLRunStartedAt != nil && time.Since(*ref.MLRunStartedAt) > 150*time.Second)

	if !isStuckOrFailed {
		if ref.Status != entity.StatusUnderSpecialistReview {
			return errors.New("ML can only be rerun while the referral is under specialist review")
		}
	}

	if u.mlUC == nil {
		return errors.New("ML service is not configured")
	}
	u.mlUC.ScheduleScoreForce(id)
	return nil
}

// ---------------------------------------------------------
// Receptionist Actions
// ---------------------------------------------------------

func (u *referralUseCase) ListForReceptionist(ctx context.Context, hospID uuid.UUID, filter irepository.ReferralFilter) ([]entity.Referral, int64, error) {
	if filter.Status != "" && !u.IsValidStatus(filter.Status) {
		return nil, 0, errors.New("forbidden: unknown or invalid referral status")
	}
	referrals, count, err := u.referralRepo.ListForReceptionist(ctx, hospID, filter)
	if err != nil {
		return nil, 0, err
	}
	for i := range referrals {
		if referrals[i].Patient != nil {
			_ = referrals[i].Patient.DecryptFields(u.cryptoSvc)
		}
	}
	u.enrichReferralsMLBatch(ctx, referrals)
	return referrals, count, nil
}

func (u *referralUseCase) GetDetailsForReceptionist(ctx context.Context, id, hospID uuid.UUID) (*entity.Referral, error) {
	ref, err := u.referralRepo.GetReferralByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if ref.TargetHospitalID != hospID {
		return nil, errors.New("unauthorized: this referral is targeted to another hospital")
	}

	// Status constraint: same as ListForReceptionist
	allowedStatuses := map[entity.ReferralStatus]bool{
		entity.StatusAccepted:    true,
		entity.StatusScheduled:   true,
		entity.StatusCompleted:   true,
	}

	if !allowedStatuses[ref.Status] {
		return nil, errors.New("unauthorized: referral has not been accepted/scheduled yet")
	}

	if ref.Patient != nil {
		_ = ref.Patient.DecryptFields(u.cryptoSvc)
	}
	u.enrichReferralML(ctx, ref)
	return ref, nil
}


// ---------------------------------------------------------
// Admin Actions
// ---------------------------------------------------------

func (u *referralUseCase) ListForSystemAdmin(ctx context.Context, filter irepository.ReferralFilter) ([]entity.Referral, int64, error) {
	if filter.Status != "" && !u.IsValidStatus(filter.Status) {
		return nil, 0, errors.New("forbidden: unknown or invalid referral status")
	}
	return u.referralRepo.ListForSystemAdmin(ctx, filter)
}

func (u *referralUseCase) ListInboundForHospitalAdmin(ctx context.Context, hospID uuid.UUID, filter irepository.ReferralFilter) ([]entity.Referral, int64, error) {
	if filter.Status != "" && !u.IsValidStatus(filter.Status) {
		return nil, 0, errors.New("forbidden: unknown or invalid referral status")
	}
	return u.referralRepo.ListInboundForHospitalAdmin(ctx, hospID, filter)
}

func (u *referralUseCase) ListOutboundForHospitalAdmin(ctx context.Context, hospID uuid.UUID, filter irepository.ReferralFilter) ([]entity.Referral, int64, error) {
	if filter.Status != "" && !u.IsValidStatus(filter.Status) {
		return nil, 0, errors.New("forbidden: unknown or invalid referral status")
	}
	return u.referralRepo.ListOutboundForHospitalAdmin(ctx, hospID, filter)
}

func (u *referralUseCase) ListPendingApprovalsForHospitalAdmin(ctx context.Context, hospID uuid.UUID, filter irepository.ReferralFilter) ([]entity.Referral, int64, error) {
	pendingStatuses := []entity.ReferralStatus{
		entity.StatusSubmitted,
		entity.StatusUnderLiaisonReview,
		entity.StatusForwarded,
		entity.StatusUnderSpecialistReview,
	}
	return u.referralRepo.ListByStatusesForHospitalAdmin(ctx, hospID, filter, pendingStatuses)
}

func (u *referralUseCase) ListRejectedRedirectedForHospitalAdmin(ctx context.Context, hospID uuid.UUID, filter irepository.ReferralFilter) ([]entity.Referral, int64, error) {
	rejectedRedirected := []entity.ReferralStatus{
		entity.StatusRejectedByLiaison,
		entity.StatusRejectedBySpecialist,
		entity.StatusNeedRevision,
	}
	return u.referralRepo.ListByStatusesForHospitalAdmin(ctx, hospID, filter, rejectedRedirected)
}

func (u *referralUseCase) GetDetailsForHospitalAdmin(ctx context.Context, hospID, referralID uuid.UUID) (*entity.Referral, error) {
	return u.referralRepo.GetDetailsForHospitalAdmin(ctx, hospID, referralID)
}

func (u *referralUseCase) GetReferralStatusCountsForHospitalAdmin(ctx context.Context, hospID uuid.UUID) ([]irepository.ReferralStatusCount, error) {
	return u.referralRepo.GetReferralStatusCounts(ctx, hospID)
}

func (u *referralUseCase) GetMonthlyReferralTotalsForHospitalAdmin(ctx context.Context, hospID uuid.UUID, months int) ([]irepository.MonthlyReferralTotal, error) {
	return u.referralRepo.GetMonthlyReferralTotalsForHospitalAdmin(ctx, hospID, months)
}

func (u *referralUseCase) GetAcceptanceRejectionRateForHospitalAdmin(ctx context.Context, hospID uuid.UUID) (float64, float64, error) {
	return u.referralRepo.GetAcceptanceRejectionRateForHospitalAdmin(ctx, hospID)
}

func (u *referralUseCase) GetMissedAppointmentRateForHospitalAdmin(ctx context.Context, hospID uuid.UUID) (float64, error) {
	return u.referralRepo.GetMissedAppointmentRateForHospitalAdmin(ctx, hospID)
}

func (u *referralUseCase) GetBusiestDepartmentsForHospitalAdmin(ctx context.Context, hospID uuid.UUID, limit int) ([]irepository.DepartmentReferralLoad, error) {
	return u.referralRepo.GetBusiestDepartmentsForHospitalAdmin(ctx, hospID, limit)
}

func (u *referralUseCase) GetAverageWaitTimeForHospitalAdmin(ctx context.Context, hospID uuid.UUID) (float64, error) {
	return u.referralRepo.GetAverageWaitTimeForHospitalAdmin(ctx, hospID)
}

func (u *referralUseCase) GetTopReferringHospitalsForHospitalAdmin(ctx context.Context, hospID uuid.UUID, limit int) ([]irepository.ReferringHospitalCount, error) {
	return u.referralRepo.GetTopReferringHospitalsForHospitalAdmin(ctx, hospID, limit)
}

func (u *referralUseCase) GetHospitalLogsForAdmin(ctx context.Context, hospID uuid.UUID, limit, page int) ([]entity.ReferralStatusHistory, int64, error) {
	return u.referralRepo.GetHospitalLogsForAdmin(ctx, hospID, limit, page)
}

func (u *referralUseCase) GetReferralStatusHistoryForHospitalAdmin(ctx context.Context, hospID, referralID uuid.UUID, limit, page int) ([]entity.ReferralStatusHistory, int64, error) {
	return u.referralRepo.GetReferralStatusHistoryForHospital(ctx, hospID, referralID, limit, page)
}

func (u *referralUseCase) GetMohDashboardSummary(ctx context.Context, filter irepository.MohAnalyticsFilter) (*irepository.MohDashboardSummary, error) {
	return u.referralRepo.GetMohDashboardSummary(ctx, filter)
}

func (u *referralUseCase) GetMohReferralTrends(ctx context.Context, filter irepository.MohAnalyticsFilter, granularity string) ([]irepository.MohReferralTrendPoint, error) {
	return u.referralRepo.GetMohReferralTrends(ctx, filter, granularity)
}

func (u *referralUseCase) GetMohHospitalLoad(ctx context.Context, filter irepository.MohAnalyticsFilter) ([]irepository.MohHospitalLoadMetric, error) {
	return u.referralRepo.GetMohHospitalLoad(ctx, filter)
}

func (u *referralUseCase) GetMohDiseaseHotspots(ctx context.Context, filter irepository.MohAnalyticsFilter) ([]irepository.MohDiseaseHotspot, error) {
	return u.referralRepo.GetMohDiseaseHotspots(ctx, filter)
}

func (u *referralUseCase) GetMohSeverityDistribution(ctx context.Context, filter irepository.MohAnalyticsFilter) ([]irepository.MohSeverityDistribution, error) {
	return u.referralRepo.GetMohSeverityDistribution(ctx, filter)
}

func (u *referralUseCase) RedirectReferral(ctx context.Context, id, specialistID, hospID, targetHospitalID uuid.UUID, reason string, newDeptID *uuid.UUID) error {
	ref, err := u.referralRepo.GetReferralByID(ctx, id)
	if err != nil {
		return err
	}
	if ref.TargetHospitalID != hospID {
		return errors.New("unauthorized: referral is not at your hospital")
	}

	// Only UNDER_SPECIALIST_REVIEW or ACCEPTED statuses are redirectable
	if ref.Status != entity.StatusUnderSpecialistReview && ref.Status != entity.StatusAccepted {
		return fmt.Errorf("invalid referral status for redirection: %s", ref.Status)
	}

	// Check forbidden chain
	forbiddenHospitals := make(map[uuid.UUID]bool)
	forbiddenHospitals[ref.SenderHospitalID] = true
	forbiddenHospitals[ref.TargetHospitalID] = true // current hospital

	redirections, _ := u.redirectionRepo.ListByReferralID(ctx, id)
	for _, r := range redirections {
		forbiddenHospitals[r.RedirectedToHospitalID] = true
	}

	if forbiddenHospitals[targetHospitalID] {
		return errors.New("circular redirection detected: hospital already involved in this referral chain")
	}

	// Verify target exists in outgoing network
	ok, err := u.networkRepo.VerifyNetworkPathway(ctx, hospID, targetHospitalID)
	if err != nil || !ok {
		return errors.New("target hospital is not in your referral network")
	}

	// Department-lock: verify target hospital has the required department
	effectiveDeptID := ref.TargetDeptID
	if newDeptID != nil {
		effectiveDeptID = *newDeptID
	}

	if u.deptRepo != nil {
		_, deptErr := u.deptRepo.FindHospitalDepartment(ctx, targetHospitalID, effectiveDeptID)
		if deptErr != nil {
			return errors.New("target hospital does not have the required department")
		}
	}

	// Update referral
	err = u.referralRepo.UpdateTargetDeptAndStatus(ctx, id, targetHospitalID, effectiveDeptID, entity.StatusRedirected)
	if err != nil {
		return err
	}

	// Create redirection record
	redirection := &entity.ReferralRedirection{
		ReferralID:               id,
		RedirectedFromHospitalID: hospID,
		RedirectedToHospitalID:   targetHospitalID,
		RedirectedBySpecialistID: specialistID,
		RedirectionReason:        &reason,
	}
	if err := u.redirectionRepo.Create(ctx, redirection); err != nil {
		return err
	}

	// Delete triage queue entry
	_ = u.triageRepo.DeleteByReferralID(ctx, id)

	_ = u.inAppNotifUC.CreateForEvent(ctx, "REFERRAL_REDIRECTED", id, specialistID)

	return nil
}

func (u *referralUseCase) ListRedirectionOptions(ctx context.Context, id, specialistID, hospID uuid.UUID, filterDeptID *uuid.UUID) ([]entity.Hospital, error) {
	ref, err := u.referralRepo.GetReferralByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Check forbidden chain
	forbiddenHospitals := make(map[uuid.UUID]bool)
	forbiddenHospitals[ref.SenderHospitalID] = true
	forbiddenHospitals[ref.TargetHospitalID] = true

	redirections, _ := u.redirectionRepo.ListByReferralID(ctx, id)
	for _, r := range redirections {
		forbiddenHospitals[r.RedirectedToHospitalID] = true
	}

	networkHospitals, err := u.networkRepo.GetOutgoingNetworkHospitals(ctx, hospID)
	if err != nil {
		return nil, err
	}

	effectiveDeptID := ref.TargetDeptID
	if filterDeptID != nil {
		effectiveDeptID = *filterDeptID
	}

	var options []entity.Hospital
	for _, h := range networkHospitals {
		if forbiddenHospitals[h.ID] {
			continue
		}
		// Department-lock: only include hospitals that have the required department
		if u.deptRepo != nil {
			_, deptErr := u.deptRepo.FindHospitalDepartment(ctx, h.ID, effectiveDeptID)
			if deptErr != nil {
				continue // Hospital doesn't have the required department
			}
		}
		options = append(options, h)
	}

	return options, nil
}

func (u *referralUseCase) GetRedirectionHistory(ctx context.Context, referralID, userID uuid.UUID, role string, hospID uuid.UUID) ([]entity.ReferralRedirection, error) {
	// Re-use existing auth logic
	var err error
	switch role {
	case string(entity.RoleReferringDoctor):
		_, err = u.GetDetailsForDoctor(ctx, referralID, userID)
	case string(entity.RoleLiaisonOfficer):
		_, err = u.GetDetailsForLiaison(ctx, referralID, hospID)
	case string(entity.RoleReceivingSpecialist):
		_, err = u.GetDetailsForSpecialist(ctx, referralID, hospID)
	case string(entity.RoleSystemSuperAdmin):
		_, err = u.referralRepo.GetReferralByID(ctx, referralID)
	default:
		return nil, errors.New("unauthorized role")
	}

	if err != nil {
		return nil, err
	}

	return u.redirectionRepo.ListByReferralID(ctx, referralID)
}

// ChangeDepartment allows a specialist at the target hospital to update the referral's target department,
// as long as the referral is in ACCEPTED or UNDER_SPECIALIST_REVIEW status and the new department
// exists at the current hospital.
func (u *referralUseCase) ChangeDepartment(ctx context.Context, referralID, specialistID, hospID, newDeptID uuid.UUID) error {
	ref, err := u.referralRepo.GetReferralByID(ctx, referralID)
	if err != nil {
		return err
	}

	if ref.TargetHospitalID != hospID {
		return errors.New("unauthorized: referral is not at your hospital")
	}

	if ref.Status != entity.StatusUnderSpecialistReview && ref.Status != entity.StatusAccepted {
		return fmt.Errorf("cannot change department: referral must be ACCEPTED or UNDER_SPECIALIST_REVIEW, current status: %s", ref.Status)
	}

	// Verify the new department exists at the current hospital
	if u.deptRepo != nil {
		_, deptErr := u.deptRepo.FindHospitalDepartment(ctx, hospID, newDeptID)
		if deptErr != nil {
			return errors.New("the specified department does not exist at your hospital")
		}
	}

	oldDeptID := ref.TargetDeptID
	ref.TargetDeptID = newDeptID

	if err := u.referralRepo.UpdateReferralTransaction(ctx, ref); err != nil {
		return err
	}

	// Audit log the department change
	reason := fmt.Sprintf("Department changed from %s to %s by specialist", oldDeptID, newDeptID)
	return u.logStatusChange(ctx, referralID, specialistID, &ref.Status, ref.Status, reason)
}

func (u *referralUseCase) GetMLPredictionForSpecialist(ctx context.Context, referralID, hospID uuid.UUID) (*entity.MLPrediction, error) {
	ref, err := u.referralRepo.GetReferralByID(ctx, referralID)
	if err != nil {
		return nil, err
	}
	if ref.TargetHospitalID != hospID {
		return nil, errors.New("unauthorized: this referral is targeted to another hospital")
	}
	pred, err := u.mlRepo.GetByReferralID(ctx, referralID)
	if err != nil {
		return nil, err
	}
	return pred, nil
}

