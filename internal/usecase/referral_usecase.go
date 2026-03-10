package usecase

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"Hospital-Referral-System/internal/delivery/http/dto"
	"Hospital-Referral-System/internal/domain/entity"
	"Hospital-Referral-System/internal/repository"
)

var (
	ErrInvalidReferral = errors.New("invalid referral data")
)

type ReferralUseCase interface {
	CreateReferral(ctx context.Context, doctorID uuid.UUID, senderHospitalID uuid.UUID, req dto.CreateReferralRequest) (*entity.Referral, error)
	GetReferral(ctx context.Context, id uuid.UUID) (*entity.Referral, error)
	ListReferrals(ctx context.Context, userRole entity.UserRole, userHospitalID uuid.UUID, statusFilter, dateFrom, dateTo string) ([]entity.Referral, error)
	UpdateDraft(ctx context.Context, id uuid.UUID, req dto.CreateReferralRequest) (*entity.Referral, error)
	DeleteDraft(ctx context.Context, id uuid.UUID) error
	UpdateReferralStatus(ctx context.Context, id uuid.UUID, newStatus entity.ReferralStatus, userID uuid.UUID, reason string) error
}

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
	referralRepo repository.ReferralRepository
}

func NewReferralUseCase(repo repository.ReferralRepository) ReferralUseCase {
	return &referralUseCase{referralRepo: repo}
}

func (u *referralUseCase) CreateReferral(ctx context.Context, doctorID uuid.UUID, senderHospitalID uuid.UUID, req dto.CreateReferralRequest) (*entity.Referral, error) {
	// Construct the deeply nested entity
	
	// Map Diagnoses
	var diagnoses []entity.ReferralDiagnosis
	for _, d := range req.Diagnoses {
		diagnoses = append(diagnoses, entity.ReferralDiagnosis{
			ICDCode:            d.ICDCode,
			IsPrimary:          d.IsPrimary,
			DiagnosisCertainty: entity.DiagnosisCertainty(d.DiagnosisCertainty),
		})
	}

	// Map Form (Annex IV)
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

	patient := &entity.Patient{
		NationalIDEnc:  req.NationalIDEnc,
		NationalIDHash: req.NationalIDHash,
		PhoneNumber:    req.PhoneNumber,
		FirstName:      req.FirstName,
		MiddleName:     req.MiddleName,
		LastName:       req.LastName,
		Sex:            req.Sex,
		HomeRegion:     req.HomeRegion,
	}

	referral := &entity.Referral{
		ReferringDoctorID: doctorID,
		SenderHospitalID:  senderHospitalID,
		TargetHospitalID:  req.TargetHospitalID,
		TargetDeptID:      req.TargetDeptID,
		LiaisonOfficerID:  req.LiaisonOfficerID,
		Status:            entity.StatusDraft, // Initialize heavily normalized payload as DRAFT
		Patient:           patient,
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

	if err := u.referralRepo.CreateReferralTransaction(ctx, referral); err != nil {
		return nil, err
	}

	return referral, nil
}

func (u *referralUseCase) GetReferral(ctx context.Context, id uuid.UUID) (*entity.Referral, error) {
	return u.referralRepo.GetReferralByID(ctx, id)
}

func (u *referralUseCase) ListReferrals(ctx context.Context, userRole entity.UserRole, userHospitalID uuid.UUID, statusFilter, dateFrom, dateTo string) ([]entity.Referral, error) {
	filters := make(map[string]interface{})

	// Apply RBAC Scopes based on Role
	switch userRole {
	case entity.RoleReferringDoctor, entity.RoleReceptionist:
		filters["sender_hospital_id"] = userHospitalID
	case entity.RoleReceivingSpecialist, entity.RoleDeptHead:
		filters["target_hospital_id"] = userHospitalID
	case entity.RoleLiaisonOfficer:
		// Basic scope test setup, usually handles hospital bidirectional
		filters["sender_hospital_id"] = userHospitalID
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

func (u *referralUseCase) UpdateDraft(ctx context.Context, id uuid.UUID, req dto.CreateReferralRequest) (*entity.Referral, error) {
	// 1. Fetch existing
	existing, err := u.referralRepo.GetReferralByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// 2. Enforce DRAFT only status update
	if existing.Status != entity.StatusDraft {
		return nil, errors.New("only referrals in DRAFT status can be updated via this endpoint")
	}

	// 3. Map updates (simplified for brevity, normally you'd map fields selectively or fully recreate)
	existing.TargetHospitalID = req.TargetHospitalID
	existing.TargetDeptID = req.TargetDeptID
	existing.LiaisonOfficerID = req.LiaisonOfficerID
	
	if req.Status == "SUBMITTED" {
		existing.Status = entity.StatusSubmitted // mapped from DTO transition command
	}

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

func (u *referralUseCase) DeleteDraft(ctx context.Context, id uuid.UUID) error {
	existing, err := u.referralRepo.GetReferralByID(ctx, id)
	if err != nil {
		return err
	}

	if existing.Status != entity.StatusDraft {
		return errors.New("cannot delete a referral that has already been submitted")
	}

	return u.referralRepo.DeleteReferral(ctx, id)
}

func (u *referralUseCase) UpdateReferralStatus(ctx context.Context, id uuid.UUID, newStatus entity.ReferralStatus, userID uuid.UUID, reason string) error {
	existing, err := u.referralRepo.GetReferralByID(ctx, id)
	if err != nil {
		return err
	}

	// Validate Transition
	validNextStates, ok := validTransitions[existing.Status]
	if !ok {
		return errors.New("current status does not allow any transitions")
	}
	isValid := false
	for _, state := range validNextStates {
		if state == newStatus {
			isValid = true
			break
		}
	}
	if !isValid {
		return errors.New("invalid status transition from " + string(existing.Status) + " to " + string(newStatus))
	}

	// Persist rejection reason when sending back to doctor or final rejection
	if newStatus == entity.StatusNeedsRevision || newStatus == entity.StatusRejected {
		if reason == "" {
			return errors.New("a rejection reason is required when rejecting a referral")
		}
		existing.RejectionReason = &reason
	}

	// Clear rejection reason when doctor resubmits after fixing
	if newStatus == entity.StatusUnderLiaisonReview && existing.Status == entity.StatusNeedsRevision {
		existing.RejectionReason = nil
	}

	existing.Status = newStatus
	return u.referralRepo.UpdateReferralTransaction(ctx, existing)
}
