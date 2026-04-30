package usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"Hospital-Referral-System/internal/delivery/http/dto"
	"Hospital-Referral-System/internal/domain/entity"
	irepository "Hospital-Referral-System/internal/domain/interfaces/repository"
	iusecase "Hospital-Referral-System/internal/domain/interfaces/usecase"
)

type referralUseCase struct {
	referralRepo      irepository.ReferralRepository
	clinicalRepo      irepository.ClinicalUpdateRepository
	outcomeRepo       irepository.ReferralOutcomeRepository
	networkRepo       irepository.NetworkRepository
	attachmentUseCase iusecase.AttachmentUseCase
	notifUC           iusecase.NotificationUseCase
}

func NewReferralUseCase(
	rRepo irepository.ReferralRepository,
	cRepo irepository.ClinicalUpdateRepository,
	oRepo irepository.ReferralOutcomeRepository,
	nRepo irepository.NetworkRepository,
	aUC iusecase.AttachmentUseCase,
	notifUC iusecase.NotificationUseCase,
) iusecase.ReferralUseCase {
	return &referralUseCase{
		referralRepo:      rRepo,
		clinicalRepo:      cRepo,
		outcomeRepo:       oRepo,
		networkRepo:       nRepo,
		attachmentUseCase: aUC,
		notifUC:           notifUC,
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
		entity.StatusAssigned:              true,
		entity.StatusCompleted:             true,
		entity.StatusNeedRevision:          true,
		entity.StatusCancelled:             true,
		entity.StatusRejectedByLiaison:     true,
		entity.StatusRejectedBySpecialist:  true,
		entity.StatusMissed:                true,
		entity.StatusRescheduled:           true,
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

func (u *referralUseCase) CreateDraftOrSubmit(ctx context.Context, doctorID uuid.UUID, senderHospitalID uuid.UUID, req dto.CreateReferralRequest) (*dto.ReferralCreationResponse, error) {
	// Verify Network
	isValidRoute, err := u.networkRepo.VerifyNetworkPathway(ctx, senderHospitalID, req.TargetHospitalID)
	if err != nil || !isValidRoute {
		return nil, errors.New("forbidden: no active referral network established between sender and target hospitals")
	}

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

	// Handle Pre-minted ID
	var refID uuid.UUID
	if req.ID != nil {
		refID = *req.ID
	} else {
		refID = uuid.New()
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

	// Bulk Attachments (Backward compatibility / Sync)
	if len(req.Attachments) > 10 {
		return nil, errors.New("cannot add more than 10 attachments")
	}
	for _, a := range req.Attachments {
		attachment := u.attachmentUseCase.PrepareAttachmentEntity(referral.ID, a.FileName, a.FileType, a.FileURL, a.PublicID, a.Category, a.FileSize)
		referral.Attachments = append(referral.Attachments, *attachment)
	}

	// Validation constraints
	if status == entity.StatusSubmitted {
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
	draftStatus := entity.StatusDraft
	if status == entity.StatusSubmitted {
		_ = u.logStatusChange(ctx, referral.ID, doctorID, &draftStatus, entity.StatusSubmitted, "")
	}

	return &dto.ReferralCreationResponse{
		Referral: referral,
		BaseResponse: dto.BaseResponse{
			Success: true,
			Message: "Referral created successfully",
		},
	}, nil
}

func (u *referralUseCase) ListForDoctor(ctx context.Context, doctorID uuid.UUID, filter irepository.ReferralFilter) ([]entity.Referral, int64, error) {
	if filter.Status != "" && !u.IsValidStatus(filter.Status) {
		return nil, 0, errors.New("forbidden: unknown or invalid referral status")
	}
	return u.referralRepo.ListForDoctor(ctx, doctorID, filter)
}

func (u *referralUseCase) GetDetailsForDoctor(ctx context.Context, id, doctorID uuid.UUID) (*entity.Referral, error) {
	ref, err := u.referralRepo.GetReferralByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if ref.ReferringDoctorID != doctorID {
		return nil, errors.New("unauthorized: can only view own referrals")
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

	// Bulk Append New Attachments
	if len(req.Attachments) > 0 {
		if len(existing.Attachments)+len(req.Attachments) > 10 {
			return nil, errors.New("total attachments cannot exceed 10")
		}
		for _, a := range req.Attachments {
			att := u.attachmentUseCase.PrepareAttachmentEntity(existing.ID, a.FileName, a.FileType, a.FileURL, a.PublicID, a.Category, a.FileSize)
			existing.Attachments = append(existing.Attachments, *att)
		}
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

	var responseData []dto.ListReferralResponse
	for _, r := range referrals {
		diag := ""
		icd := ""
		if len(r.Diagnoses) > 0 && r.Diagnoses[0].CodeInfo != nil {
			diag = r.Diagnoses[0].CodeInfo.Description
			icd = r.Diagnoses[0].ICDCode
		}

		patientNameFirst := ""
		patientNameMiddle := ""
		patientNameLast := ""
		patientRegion := ""
		if r.Patient != nil {
			patientNameFirst = r.Patient.FirstName
			patientNameMiddle = r.Patient.MiddleName
			patientNameLast = r.Patient.LastName
			if r.Patient.HomeRegion != nil {
				patientRegion = *r.Patient.HomeRegion
			}
		}

		condition := ""
		if r.ReferralForm != nil {
			condition = r.ReferralForm.ConditionAtReferral
		}

		responseData = append(responseData, dto.ListReferralResponse{
			ID:                  r.ID,
			PatientFirstName:    patientNameFirst,
			PatientMiddleName:   patientNameMiddle,
			PatientLastName:     patientNameLast,
			PatientRegion:       patientRegion,
			Department:          r.TargetDeptID.String(),
			Status:              string(r.Status),
			ICDCode:             icd,
			Diagnosis:           diag,
			ConditionAtReferral: condition,
			CreatedAt:           r.CreatedAt,
			UpdatedAt:           r.UpdatedAt,
		})
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
	return u.referralRepo.ListOutgoingForLiaison(ctx, hospID, filter)
}

func (u *referralUseCase) ListIncomingForLiaison(ctx context.Context, hospID uuid.UUID, filter irepository.ReferralFilter) ([]entity.Referral, int64, error) {
	if filter.Status != "" && !u.IsValidStatus(filter.Status) {
		return nil, 0, errors.New("forbidden: unknown or invalid referral status")
	}
	return u.referralRepo.ListIncomingForLiaison(ctx, hospID, filter)
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
	return u.referralRepo.CreateStatusHistory(ctx, history)
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
	return u.referralRepo.CreateStatusHistory(ctx, history)
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
	return u.referralRepo.CreateStatusHistory(ctx, history)
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
	ref.Status = entity.StatusForwarded
	ref.SpecialistID = nil

	if err := u.referralRepo.UpdateReferralTransaction(ctx, ref); err != nil {
		return err
	}

	history := &entity.ReferralStatusHistory{
		ReferralID:  id,
		ChangedByID: liaisonID,
		FromStatus:  &oldStatus,
		ToStatus:    entity.StatusForwarded,
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
	return u.referralRepo.ListForSpecialist(ctx, hospID, filter)
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
		entity.StatusAssigned:              true,
		entity.StatusCompleted:             true,
		entity.StatusRejectedBySpecialist:  true,
		entity.StatusMissed:                true,
		entity.StatusRescheduled:           true,
	}

	if !allowedStatuses[ref.Status] {
		return nil, errors.New("unauthorized: referral has not yet been forwarded to your hospital")
	}

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
	if ref.Status != entity.StatusForwarded {
		return errors.New("invalid status: must be FORWARDED")
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

	// Severity gate: a manual severity score must have been set before acceptance
	if severityScore == nil && ref.MLSeverityScore == nil {
		return errors.New("severity score must be set before accepting a referral; use the triage-severity endpoint first")
	}

	oldStatus := ref.Status
	ref.Status = entity.StatusAccepted
	if severityScore != nil {
		ref.MLSeverityScore = severityScore
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
	_ = u.notifUC.QueueNotification(ctx, id, entity.NotificationType("ACCEPTANCE"), message)

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
	return u.referralRepo.CreateStatusHistory(ctx, history)
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
	ref.Status = entity.StatusForwarded
	ref.SpecialistID = nil

	if err := u.referralRepo.UpdateReferralTransaction(ctx, ref); err != nil {
		return err
	}

	history := &entity.ReferralStatusHistory{
		ReferralID:  id,
		ChangedByID: specialistID,
		FromStatus:  &oldStatus,
		ToStatus:    entity.StatusForwarded,
		Reason:      &reason,
	}
	return u.referralRepo.CreateStatusHistory(ctx, history)
}

func (u *referralUseCase) SpecialistRerunML(ctx context.Context, id, specialistID, hospID uuid.UUID) error {
	// Dummy ML Rerun implementation
	return nil
}

// ---------------------------------------------------------
// Receptionist Actions
// ---------------------------------------------------------

func (u *referralUseCase) ListForReceptionist(ctx context.Context, hospID uuid.UUID, filter irepository.ReferralFilter) ([]entity.Referral, int64, error) {
	if filter.Status != "" && !u.IsValidStatus(filter.Status) {
		return nil, 0, errors.New("forbidden: unknown or invalid referral status")
	}
	return u.referralRepo.ListForReceptionist(ctx, hospID, filter)
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
		entity.StatusAssigned:    true,
		entity.StatusCompleted:   true,
		entity.StatusMissed:      true,
		entity.StatusRescheduled: true,
	}

	if !allowedStatuses[ref.Status] {
		return nil, errors.New("unauthorized: referral has not been accepted/scheduled yet")
	}

	return ref, nil
}

func (u *referralUseCase) ConfirmAttendance(ctx context.Context, id, receptionistID, hospID uuid.UUID, status string) error {
	ref, err := u.referralRepo.GetReferralByID(ctx, id)
	if err != nil {
		return err
	}
	if ref.TargetHospitalID != hospID {
		return errors.New("unauthorized")
	}

	// This mapping requires the Referral to be SCHEDULED prior, though some systems allow ACCEPTED -> ASSIGNED skipping SCHEDULED.
	// We'll enforce ACCEPTED or SCHEDULED or RESCHEDULED -> ASSIGNED, and SCHEDULED->MISSED, etc.
	oldStatus := ref.Status
	var newStatus entity.ReferralStatus

	switch status {
	case string(entity.StatusAssigned):
		if oldStatus == entity.StatusAssigned {
			return errors.New("referral is already assigned")
		}
		if oldStatus != entity.StatusAccepted && oldStatus != entity.StatusScheduled && oldStatus != entity.StatusRescheduled {
			return errors.New("invalid transition to ASSIGNED")
		}
		newStatus = entity.StatusAssigned
	case string(entity.StatusMissed):
		if oldStatus == entity.StatusMissed {
			return errors.New("referral is already marked as missed")
		}
		if oldStatus != entity.StatusScheduled && oldStatus != entity.StatusRescheduled {
			return errors.New("invalid transition to MISSED")
		}
		newStatus = entity.StatusMissed
	case string(entity.StatusCompleted):
		if oldStatus == entity.StatusCompleted {
			return errors.New("referral is already completed")
		}
		if oldStatus != entity.StatusAssigned {
			return errors.New("invalid transition: must be ASSIGNED to COMPLETE")
		}
		newStatus = entity.StatusCompleted
	case string(entity.StatusScheduled):
		if oldStatus == entity.StatusScheduled {
			return errors.New("referral is already scheduled")
		}
		if oldStatus != entity.StatusAccepted {
			return errors.New("invalid transition: must be ACCEPTED")
		}
		newStatus = entity.StatusScheduled
	default:
		return errors.New("invalid status selected")
	}

	ref.Status = newStatus

	if err := u.referralRepo.UpdateReferralTransaction(ctx, ref); err != nil {
		return err
	}
	return u.logStatusChange(ctx, id, receptionistID, &oldStatus, newStatus, "")
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
	return u.referralRepo.CountByStatusForHospitalAdmin(ctx, hospID)
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

