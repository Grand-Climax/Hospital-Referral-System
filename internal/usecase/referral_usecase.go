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

func (u *referralUseCase) ListForDoctor(ctx context.Context, doctorID uuid.UUID, limit, offset int, statusFilter string) ([]entity.Referral, int64, error) {
	return u.referralRepo.ListForDoctor(ctx, doctorID, limit, offset, statusFilter)
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

func (u *referralUseCase) UpdateAndResubmit(ctx context.Context, id, doctorID uuid.UUID, req dto.CreateReferralRequest) (*entity.Referral, error) {
	existing, err := u.referralRepo.GetReferralByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if existing.ReferringDoctorID != doctorID {
		return nil, errors.New("unauthorized: only the creator can update this draft/referral")
	}

	// Immutability Check
	if existing.Status != entity.StatusDraft && existing.Status != entity.StatusNeedRevision {
		return nil, errors.New("forbidden: referral is immutable in current state")
	}

	oldStatus := existing.Status
	// If the user wants to jump strictly to submitted
	nextStatus := entity.StatusDraft
	if req.Status == string(entity.StatusSubmitted) {
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

// ---------------------------------------------------------
// Liaison Actions
// ---------------------------------------------------------

func (u *referralUseCase) ListForLiaison(ctx context.Context, hospID uuid.UUID, limit, offset int, statusFilter string) ([]entity.Referral, int64, error) {
	return u.referralRepo.ListForLiaison(ctx, hospID, limit, offset, statusFilter)
}

func (u *referralUseCase) GetDetailsForLiaison(ctx context.Context, id, hospID uuid.UUID) (*entity.Referral, error) {
	ref, err := u.referralRepo.GetReferralByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if ref.SenderHospitalID != hospID {
		return nil, errors.New("unauthorized: referral does not originate from your hospital")
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
		return errors.New("unauthorized")
	}
	if ref.Status != entity.StatusUnderLiaisonReview {
		return errors.New("invalid status transition: must be UNDER_LIAISON_REVIEW")
	}

	oldStatus := ref.Status
	ref.Status = entity.StatusForwarded

	if err := u.referralRepo.UpdateReferralTransaction(ctx, ref); err != nil {
		return err
	}
	return u.logStatusChange(ctx, id, liaisonID, &oldStatus, entity.StatusForwarded, comment)
}

func (u *referralUseCase) LiaisonReject(ctx context.Context, id, liaisonID, hospID uuid.UUID, reason string) error {
	ref, err := u.referralRepo.GetReferralByID(ctx, id)
	if err != nil {
		return err
	}
	if ref.SenderHospitalID != hospID {
		return errors.New("unauthorized")
	}
	if ref.Status != entity.StatusUnderLiaisonReview {
		return errors.New("invalid status: must be UNDER_LIAISON_REVIEW")
	}
	if reason == "" {
		return errors.New("rejection reason is required")
	}

	oldStatus := ref.Status
	ref.Status = entity.StatusRejectedByLiaison
	ref.RejectionReason = &reason

	if err := u.referralRepo.UpdateReferralTransaction(ctx, ref); err != nil {
		return err
	}
	return u.logStatusChange(ctx, id, liaisonID, &oldStatus, entity.StatusRejectedByLiaison, reason)
}

func (u *referralUseCase) LiaisonRevise(ctx context.Context, id, liaisonID, hospID uuid.UUID, reason string) error {
	ref, err := u.referralRepo.GetReferralByID(ctx, id)
	if err != nil {
		return err
	}
	if ref.SenderHospitalID != hospID {
		return errors.New("unauthorized")
	}
	if ref.Status != entity.StatusUnderLiaisonReview {
		return errors.New("invalid status: must be UNDER_LIAISON_REVIEW")
	}
	if reason == "" {
		return errors.New("revision reason is required")
	}

	oldStatus := ref.Status
	ref.Status = entity.StatusNeedRevision
	ref.RevisionReason = &reason

	if err := u.referralRepo.UpdateReferralTransaction(ctx, ref); err != nil {
		return err
	}
	return u.logStatusChange(ctx, id, liaisonID, &oldStatus, entity.StatusNeedRevision, reason)
}

// ---------------------------------------------------------
// Specialist Actions
// ---------------------------------------------------------

func (u *referralUseCase) ListForSpecialist(ctx context.Context, hospID, specialistID uuid.UUID, limit, offset int, statusFilter string) ([]entity.Referral, int64, error) {
	return u.referralRepo.ListForSpecialist(ctx, hospID, specialistID, limit, offset, statusFilter)
}

func (u *referralUseCase) GetDetailsForSpecialist(ctx context.Context, id, hospID uuid.UUID) (*entity.Referral, error) {
	ref, err := u.referralRepo.GetReferralByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if ref.TargetHospitalID != hospID {
		return nil, errors.New("unauthorized")
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
		return errors.New("unauthorized")
	}
	if ref.Status != entity.StatusUnderSpecialistReview {
		return errors.New("invalid status: must be UNDER_SPECIALIST_REVIEW")
	}

	oldStatus := ref.Status
	ref.Status = entity.StatusAccepted
	if severityScore != nil {
		ref.MLSeverityScore = severityScore
	}

	if err := u.referralRepo.UpdateReferralTransaction(ctx, ref); err != nil {
		return err
	}
	return u.logStatusChange(ctx, id, specialistID, &oldStatus, entity.StatusAccepted, "")
}

func (u *referralUseCase) SpecialistReject(ctx context.Context, id, specialistID, hospID uuid.UUID, reason string) error {
	ref, err := u.referralRepo.GetReferralByID(ctx, id)
	if err != nil {
		return err
	}
	if ref.TargetHospitalID != hospID {
		return errors.New("unauthorized")
	}
	if ref.Status != entity.StatusForwarded && ref.Status != entity.StatusUnderSpecialistReview {
		return errors.New("invalid status: must be FORWARDED or UNDER_SPECIALIST_REVIEW")
	}
	if reason == "" {
		return errors.New("rejection reason is required")
	}

	oldStatus := ref.Status
	ref.Status = entity.StatusRejectedBySpecialist
	ref.RejectionReason = &reason
	ref.SpecialistID = &specialistID

	if err := u.referralRepo.UpdateReferralTransaction(ctx, ref); err != nil {
		return err
	}
	return u.logStatusChange(ctx, id, specialistID, &oldStatus, entity.StatusRejectedBySpecialist, reason)
}

func (u *referralUseCase) SpecialistRerunML(ctx context.Context, id, specialistID, hospID uuid.UUID) error {
	// Dummy ML Rerun implementation
	return nil
}

// ---------------------------------------------------------
// Receptionist Actions
// ---------------------------------------------------------

func (u *referralUseCase) ListForReceptionist(ctx context.Context, hospID uuid.UUID, limit, offset int, statusFilter string) ([]entity.Referral, int64, error) {
	return u.referralRepo.ListForReceptionist(ctx, hospID, limit, offset, statusFilter)
}

func (u *referralUseCase) GetDetailsForReceptionist(ctx context.Context, id, hospID uuid.UUID) (*entity.Referral, error) {
	ref, err := u.referralRepo.GetReferralByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if ref.TargetHospitalID != hospID {
		return nil, errors.New("unauthorized")
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
		if oldStatus != entity.StatusAccepted && oldStatus != entity.StatusScheduled && oldStatus != entity.StatusRescheduled {
			return errors.New("invalid transition to ASSIGNED")
		}
		newStatus = entity.StatusAssigned
	case string(entity.StatusMissed):
		if oldStatus != entity.StatusScheduled && oldStatus != entity.StatusRescheduled {
			return errors.New("invalid transition to MISSED")
		}
		newStatus = entity.StatusMissed
	case string(entity.StatusCompleted):
		if oldStatus != entity.StatusAssigned {
			return errors.New("invalid transition: must be ASSIGNED to COMPLETE")
		}
		newStatus = entity.StatusCompleted
	case string(entity.StatusScheduled):
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

func (u *referralUseCase) ListForSystemAdmin(ctx context.Context, limit, offset int, statusFilter string) ([]entity.Referral, int64, error) {
	return u.referralRepo.ListForSystemAdmin(ctx, limit, offset, statusFilter)
}

func (u *referralUseCase) GetHospitalLogsForAdmin(ctx context.Context, hospID uuid.UUID, limit, offset int) ([]entity.ReferralStatusHistory, int64, error) {
	return u.referralRepo.GetHospitalLogsForAdmin(ctx, hospID, limit, offset)
}
