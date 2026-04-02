package usecase

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"Hospital-Referral-System/internal/delivery/http/dto"
	"Hospital-Referral-System/internal/domain/entity"
	irepository "Hospital-Referral-System/internal/domain/interfaces/repository"
	iusecase "Hospital-Referral-System/internal/domain/interfaces/usecase"
)

var (
	ErrInvalidReferral = errors.New("invalid referral data")
)

var validTransitions = map[entity.ReferralStatus][]entity.ReferralStatus{
	// Doctor workflow
	entity.StatusDraft:    {entity.StatusSubmitted},
	entity.StatusSubmitted: {entity.StatusUnderLiaisonReview, entity.StatusCancelled},

	// Liaison review: accept (forward) or reject (send back to doctor for revision)
	entity.StatusUnderLiaisonReview: {entity.StatusForwarded, entity.StatusNeedsRevision},

	// Doctor fixes form and resubmits after liaison rejection
	entity.StatusNeedsRevision: {entity.StatusUnderLiaisonReview, entity.StatusCancelled},

	// Forwarded → target hospital marks received → enters specialist review queue
	entity.StatusForwarded: {entity.StatusReceived},
	entity.StatusReceived:  {entity.StatusSpecialistReview},

	// Specialist review: accept (assign) or reject (back to liaison, who notifies doctor)
	entity.StatusSpecialistReview: {entity.StatusSpecialistAssigned, entity.StatusUnderLiaisonReview},

	// Accepted by specialist → scheduling pipeline
	entity.StatusSpecialistAssigned: {entity.StatusScheduled, entity.StatusRejected},
	entity.StatusScheduled:          {entity.StatusCompleted, entity.StatusCancelled, entity.StatusMissed},
}

type referralUseCase struct {
	referralRepo irepository.ReferralRepository
	networkRepo  irepository.NetworkRepository
}

func NewReferralUseCase(rRepo irepository.ReferralRepository, nRepo irepository.NetworkRepository) iusecase.ReferralUseCase {
	return &referralUseCase{referralRepo: rRepo, networkRepo: nRepo}
}

func (u *referralUseCase) CreateReferral(ctx context.Context, doctorID uuid.UUID, senderHospitalID uuid.UUID, req dto.CreateReferralRequest) (*entity.Referral, error) {
	// 1. Verify Network Pathway
	isValidRoute, err := u.networkRepo.VerifyNetworkPathway(ctx, senderHospitalID, req.TargetHospitalID)
	if err != nil {
		return nil, errors.New("failed to verify network routing")
	}
	if !isValidRoute {
		return nil, errors.New("forbidden: no active referral network established between sender and target hospitals")
	}

	// 2. Map Diagnoses
	var diagnoses []entity.ReferralDiagnosis
	for _, d := range req.Diagnoses {
		diagnoses = append(diagnoses, entity.ReferralDiagnosis{
			ICDCode:            d.ICDCode,
			IsPrimary:          d.IsPrimary,
			DiagnosisCertainty: entity.DiagnosisCertainty(d.DiagnosisCertainty),
		})
	}

	// 3. Map Form (Annex IV)
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

	// Determine Initial Request Status enforcing gating limits
	requestStatus := entity.StatusDraft

	// 4. Construct envelope mapping existing Patient explicitly
	referral := &entity.Referral{
		ReferringDoctorID: doctorID,
		SenderHospitalID:  senderHospitalID,
		TargetHospitalID:  req.TargetHospitalID,
		TargetDeptID:      req.TargetDeptID,
		PatientID:         req.PatientID,
		LiaisonOfficerID:  req.LiaisonOfficerID,
		Status:            requestStatus,
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

	if err := u.verifySubmissionRequirements(referral); err != nil {
		return nil, err
	}

	if err := u.referralRepo.CreateReferralTransaction(ctx, referral); err != nil {
		return nil, err
	}

	return referral, nil
}

func (u *referralUseCase) GetReferral(ctx context.Context, id, userID, hospID, deptID uuid.UUID, userRole entity.UserRole) (*entity.Referral, error) {
	referral, err := u.referralRepo.GetReferralByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// RBAC Validation for GetByID
	switch userRole {
	case entity.RoleSystemSuperAdmin:
		// Allowed unconditionally
	case entity.RoleReferringDoctor:
		if referral.ReferringDoctorID != userID {
			return nil, errors.New("unauthorized: can only view own referrals")
		}
	case entity.RoleLiaisonOfficer:
		if referral.SenderHospitalID != hospID {
			return nil, errors.New("unauthorized: referral does not belong to your facility")
		}
	case entity.RoleReceivingSpecialist, entity.RoleDeptHead:
		if referral.TargetHospitalID != hospID || (deptID != uuid.Nil && referral.TargetDeptID != deptID) {
			return nil, errors.New("unauthorized: referral target mismatch for your department/hospital")
		}
	case entity.RoleReceptionist:
		if referral.TargetHospitalID != hospID {
			return nil, errors.New("unauthorized: patient not assigned to your hospital")
		}
	default:
		return nil, errors.New("unauthorized role")
	}

	return referral, nil
}

func (u *referralUseCase) ListReferrals(ctx context.Context, userID, hospID, deptID uuid.UUID, userRole entity.UserRole, statusFilter, dateFrom, dateTo string) ([]entity.Referral, error) {
	filters := make(map[string]interface{})

	// Apply RBAC Scopes based on Role
	switch userRole {
	case entity.RoleSystemSuperAdmin:
		// No restrictions, sees all nationwide.
	case entity.RoleReferringDoctor:
		filters["referring_doctor_id"] = userID // Strictly personal creations
	case entity.RoleLiaisonOfficer:
		filters["sender_hospital_id"] = hospID  // Outgoing gatekeeper
	case entity.RoleReceivingSpecialist, entity.RoleDeptHead:
		filters["target_hospital_id"] = hospID
		if deptID != uuid.Nil {
			filters["target_dept_id"] = deptID // Incoming queue
		}
	case entity.RoleReceptionist:
		filters["target_hospital_id"] = hospID // Usually filtered further by front desk for "ACCEPTED" only.
	default:
		// Unknown or restricted roles return empty to be safe
		return []entity.Referral{}, errors.New("unauthorized role")
	}

	if statusFilter != "" {
		filters["status"] = entity.ReferralStatus(statusFilter)
	}

	if dateFrom != "" {
		filters["start_date"] = dateFrom
	}

	if dateTo != "" {
		filters["end_date"] = dateTo
	}

	return u.referralRepo.ListReferrals(ctx, filters)
}

func (u *referralUseCase) UpdateDraft(ctx context.Context, id, userID uuid.UUID, req dto.CreateReferralRequest) (*entity.Referral, error) {
	// 1. Fetch existing
	existing, err := u.referralRepo.GetReferralByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Enforce creator constraint
	if existing.ReferringDoctorID != userID {
		return nil, errors.New("unauthorized: only the creator can update this draft")
	}

	// 2. Enforce DRAFT or NEEDS_REVISION only status update
	if existing.Status != entity.StatusDraft && existing.Status != entity.StatusNeedsRevision {
		return nil, errors.New("forbidden: only referrals actively in DRAFT or NEEDS_REVISION status can be updated via this endpoint")
	}

	// 3. Map updates
	existing.TargetHospitalID = req.TargetHospitalID
	existing.TargetDeptID = req.TargetDeptID
	existing.LiaisonOfficerID = req.LiaisonOfficerID

	// Update Form
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

	// Re-map Diagnoses
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

	// Re-map Vitals
	if req.Vitals != nil {
		existing.Vitals = []entity.Vital{
			{
				ReferralID:      existing.ID,
				SystolicBP:      req.Vitals.SystolicBP,
				DiastolicBP:     req.Vitals.DiastolicBP,
				HeartRate:       req.Vitals.HeartRate,
				SpO2:            req.Vitals.SpO2,
				Temperature:     req.Vitals.Temperature,
				RespiratoryRate: req.Vitals.RespiratoryRate,
				GCSScore:        req.Vitals.GCSScore,
			},
		}
	} else {
		existing.Vitals = nil
	}

	if err := u.referralRepo.UpdateReferralTransaction(ctx, existing); err != nil {
		return nil, err
	}

	return existing, nil
}

func (u *referralUseCase) DeleteDraft(ctx context.Context, id, userID uuid.UUID) error {
	existing, err := u.referralRepo.GetReferralByID(ctx, id)
	if err != nil {
		return err
	}

	if existing.ReferringDoctorID != userID {
		return errors.New("unauthorized: only the creator can delete this draft")
	}

	if existing.Status != entity.StatusDraft && existing.Status != entity.StatusNeedsRevision {
		return errors.New("cannot delete a referral that has already been submitted and accepted for review")
	}

	return u.referralRepo.DeleteReferral(ctx, id)
}

func (u *referralUseCase) SubmitReferral(ctx context.Context, id, userID uuid.UUID) error {
	existing, err := u.referralRepo.GetReferralByID(ctx, id)
	if err != nil {
		return err
	}

	if existing.ReferringDoctorID != userID {
		return errors.New("unauthorized: only the creator can submit this referral")
	}

	if existing.Status != entity.StatusDraft {
		return errors.New("invalid status: can only submit a referral in DRAFT status")
	}

	// Fake the status temporarily for validation check
	existing.Status = entity.StatusSubmitted
	if err := u.verifySubmissionRequirements(existing); err != nil {
		return err
	}

	oldStatus := entity.StatusDraft

	if err := u.referralRepo.UpdateReferralTransaction(ctx, existing); err != nil {
		return err
	}

	return u.referralRepo.CreateStatusHistory(ctx, &entity.ReferralStatusHistory{
		ReferralID:  id,
		ChangedByID: userID,
		FromStatus:  &oldStatus,
		ToStatus:    entity.StatusSubmitted,
		ChangedAt:   time.Now(),
	})
}

func (u *referralUseCase) ResubmitReferral(ctx context.Context, id, userID uuid.UUID) error {
	existing, err := u.referralRepo.GetReferralByID(ctx, id)
	if err != nil {
		return err
	}

	if existing.ReferringDoctorID != userID {
		return errors.New("unauthorized: only the creator can resubmit this referral")
	}

	if existing.Status != entity.StatusNeedsRevision {
		return errors.New("invalid status: can only resubmit a referral in NEEDS_REVISION status")
	}

	existing.Status = entity.StatusUnderLiaisonReview
	existing.RejectionReason = nil // Clear previous rejection

	if err := u.referralRepo.UpdateReferralTransaction(ctx, existing); err != nil {
		return err
	}

	oldStatus := entity.StatusNeedsRevision
	return u.referralRepo.CreateStatusHistory(ctx, &entity.ReferralStatusHistory{
		ReferralID:  id,
		ChangedByID: userID,
		FromStatus:  &oldStatus,
		ToStatus:    entity.StatusUnderLiaisonReview,
		ChangedAt:   time.Now(),
	})
}

func (u *referralUseCase) CancelReferral(ctx context.Context, id, userID uuid.UUID, reason string) error {
	existing, err := u.referralRepo.GetReferralByID(ctx, id)
	if err != nil {
		return err
	}

	if existing.ReferringDoctorID != userID {
		return errors.New("unauthorized: only the creator can cancel this referral")
	}

	if existing.Status != entity.StatusDraft && existing.Status != entity.StatusNeedsRevision {
		return errors.New("cannot cancel a referral that has already been submitted and accepted for review")
	}

	oldStatus := existing.Status
	existing.Status = entity.StatusCancelled

	var reasonPtr *string
	if reason != "" {
		reasonPtr = &reason
	}

	if err := u.referralRepo.UpdateReferralTransaction(ctx, existing); err != nil {
		return err
	}

	return u.referralRepo.CreateStatusHistory(ctx, &entity.ReferralStatusHistory{
		ReferralID:  id,
		ChangedByID: userID,
		FromStatus:  &oldStatus,
		ToStatus:    entity.StatusCancelled,
		Reason:      reasonPtr,
		ChangedAt:   time.Now(),
	})
}

func (u *referralUseCase) verifySubmissionRequirements(ref *entity.Referral) error {
	if ref.Status != entity.StatusSubmitted {
		return nil // No severe restrictions exist for piecemeal DRAFTS
	}

	// Rule 1: Diagnoses
	if len(ref.Diagnoses) == 0 {
		return errors.New("cannot submit: at least one clinical diagnosis is required")
	}

	// Rule 2: Referral Form Fields
	if ref.ReferralForm == nil {
		return errors.New("cannot submit: referral form annex is missing")
	}

	f := ref.ReferralForm
	if f.ClinicalSummary == "" || f.PatientHistory == "" || f.ReasonOfReferral == "" || f.ReasonForReferralCategory == "" || f.ConditionAtReferral == "" {
		return errors.New("cannot submit: Clinical Summary, Patient History, Reason For Referral and Condition are structurally mandatory")
	}

	return nil
}
