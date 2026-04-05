package usecase

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"Hospital-Referral-System/internal/delivery/http/dto"
	"Hospital-Referral-System/internal/domain/entity"
	irepository "Hospital-Referral-System/internal/domain/interfaces/repository"
	iusecase "Hospital-Referral-System/internal/domain/interfaces/usecase"
)

type referralUseCase struct {
	referralRepo irepository.ReferralRepository
	networkRepo  irepository.NetworkRepository
}

func NewReferralUseCase(rRepo irepository.ReferralRepository, nRepo irepository.NetworkRepository) iusecase.ReferralUseCase {
	return &referralUseCase{referralRepo: rRepo, networkRepo: nRepo}
}

// ---------------------------------------------------------
// Helper for History Log Generation
// ---------------------------------------------------------
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

func (u *referralUseCase) CreateDraftOrSubmit(ctx context.Context, doctorID uuid.UUID, senderHospitalID uuid.UUID, req dto.CreateReferralRequest) (*entity.Referral, error) {
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

	referral := &entity.Referral{
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

	return referral, nil
}

func (u *referralUseCase) ListForDoctor(ctx context.Context, doctorID uuid.UUID, limit, page int, statusFilter string) ([]entity.Referral, int64, error) {
	return u.referralRepo.ListForDoctor(ctx, doctorID, limit, page, statusFilter)
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

func (u *referralUseCase) UpdateAndResubmit(ctx context.Context, id, doctorID uuid.UUID, req dto.UpdateReferralRequest, submit bool) (*entity.Referral, error) {
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
	}

	existing.Status = nextStatus
	if err := u.referralRepo.UpdateReferralTransaction(ctx, existing); err != nil {
		return nil, err
	}

	if oldStatus != nextStatus {
		_ = u.logStatusChange(ctx, existing.ID, doctorID, &oldStatus, nextStatus, "")
	}

	return existing, nil
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
		if r.Patient != nil {
			patientNameFirst = r.Patient.FirstName
			patientNameMiddle = r.Patient.MiddleName
			patientNameLast = r.Patient.LastName
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
			Department:          r.TargetDeptID.String(),
			Date:                r.CreatedAt.Format("2006-01-02"),
			Status:              string(r.Status),
			ICDCode:             icd,
			Diagnosis:           diag,
			ConditionAtReferral: condition,
		})
	}
	return responseData, nil
}

// ---------------------------------------------------------
// Liaison Actions
// ---------------------------------------------------------

func (u *referralUseCase) ListOutgoingForLiaison(ctx context.Context, hospID uuid.UUID, limit, page int, statusFilter string) ([]entity.Referral, int64, error) {
	return u.referralRepo.ListOutgoingForLiaison(ctx, hospID, limit, page, statusFilter)
}

func (u *referralUseCase) ListIncomingForLiaison(ctx context.Context, hospID uuid.UUID, limit, page int, statusFilter string) ([]entity.Referral, int64, error) {
	return u.referralRepo.ListIncomingForLiaison(ctx, hospID, limit, page, statusFilter)
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

func (u *referralUseCase) ListForSpecialist(ctx context.Context, hospID, specialistID uuid.UUID, limit, page int, statusFilter string) ([]entity.Referral, int64, error) {
	return u.referralRepo.ListForSpecialist(ctx, hospID, limit, page, statusFilter)
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
		entity.StatusForwarded:              true,
		entity.StatusUnderSpecialistReview:  true,
		entity.StatusAccepted:               true,
		entity.StatusScheduled:              true,
		entity.StatusAssigned:               true,
		entity.StatusCompleted:              true,
		entity.StatusRejectedBySpecialist:   true,
		entity.StatusMissed:                 true,
		entity.StatusRescheduled:            true,
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
	return u.referralRepo.CreateStatusHistory(ctx, history)
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

func (u *referralUseCase) ListForReceptionist(ctx context.Context, hospID uuid.UUID, limit, page int, statusFilter string) ([]entity.Referral, int64, error) {
	return u.referralRepo.ListForReceptionist(ctx, hospID, limit, page, statusFilter)
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

func (u *referralUseCase) ListForSystemAdmin(ctx context.Context, limit, page int, statusFilter string) ([]entity.Referral, int64, error) {
	return u.referralRepo.ListForSystemAdmin(ctx, limit, page, statusFilter)
}

func (u *referralUseCase) GetHospitalLogsForAdmin(ctx context.Context, hospID uuid.UUID, limit, page int) ([]entity.ReferralStatusHistory, int64, error) {
	return u.referralRepo.GetHospitalLogsForAdmin(ctx, hospID, limit, page)
}
