package dto

import (
	"time"

	"Hospital-Referral-System/internal/domain/entity"
	"github.com/google/uuid"
)

type CreateReferralRequest struct {
	// Optional pre-minted ID
	ID *uuid.UUID `json:"id,omitempty" example:"550e8400-e29b-41d4-a716-446655440000"`

	// Patient Logic
	PatientID uuid.UUID `json:"patient_id" binding:"required" example:"e0000000-0000-0000-0000-000000000001"`

	// Routing Information
	TargetHospitalID uuid.UUID  `json:"target_hospital_id" binding:"required" example:"a3000000-0000-0000-0000-000000000003"`
	TargetDeptID     uuid.UUID  `json:"target_dept_id" binding:"required" example:"b5000000-0000-0000-0000-000000000005"`
	LiaisonOfficerID *uuid.UUID `json:"liaison_officer_id,omitempty" binding:"required" example:"d1000000-0000-0000-0000-000000000002"`

	// Annex IV Clinical Data
	ClinicalSummary              string  `json:"clinical_summary" example:"Patient complains of severe chest pain for 2 hours"`
	PatientHistory               string  `json:"patient_history" example:"Hypertension diagnosed 5 years ago"`
	PhysicalExaminationFindings  *string `json:"physical_examination_findings" example:"BP 180/110, HR 105"`
	InvestigationResults         *string `json:"investigation_results" example:"ECG shows ST elevation"`
	TreatmentGivenBeforeReferral *string `json:"treatment_given_before_referral" example:"Aspirin 300mg"`
	MedicationOnTransfer         *string `json:"medication_on_transfer" example:"IV Nitroglycerin"`
	ReasonOfReferral             string  `json:"reason_of_referral" example:"Requires immediate cardiological intervention"`
	ReasonForReferralCategory    *string `json:"reason_for_referral_category,omitempty" example:"EMERGENCY"`
	ConditionAtReferral          string  `json:"condition_at_referral" example:"UNSTABLE"`
	ModeOfTransport              *string `json:"mode_of_transport" example:"AMBULANCE"`
	AccompanyingPersonName       *string `json:"accompanying_person_name" example:"Sarah Kebede"`
	AccompanyingPersonPhone      *string `json:"accompanying_person_phone" binding:"omitempty,e164" example:"+251922334455"`

	// Diagnoses
	Diagnoses []DiagnosisDTO `json:"diagnoses" binding:"omitempty,dive"`

	// Status tracking
	Status string `json:"status" binding:"omitempty,oneof=DRAFT SUBMITTED" example:"SUBMITTED"`

	// Optional Extensions
	Vitals          *VitalsDTO                `json:"vitals"`
	EmergencyDetail *EmergencyDetailDTO       `json:"emergency_detail"`
}

type UpdateReferralRequest struct {
	// Patient ID cannot be changed in update usually, but we'll include it for consistency or restrict if needed.
	// For now, mirroring Create except Status.
	PatientID uuid.UUID `json:"patient_id" binding:"required" example:"e0000000-0000-0000-0000-000000000001"`

	// Routing Information
	TargetHospitalID uuid.UUID  `json:"target_hospital_id" binding:"required" example:"a3000000-0000-0000-0000-000000000003"`
	TargetDeptID     uuid.UUID  `json:"target_dept_id" binding:"required" example:"b5000000-0000-0000-0000-000000000005"`
	LiaisonOfficerID *uuid.UUID `json:"liaison_officer_id,omitempty" binding:"required" example:"d1000000-0000-0000-0000-000000000002"`

	// Annex IV Clinical Data
	ClinicalSummary              string  `json:"clinical_summary" example:"Patient complains of severe chest pain for 2 hours"`
	PatientHistory               string  `json:"patient_history" example:"Hypertension diagnosed 5 years ago"`
	PhysicalExaminationFindings  *string `json:"physical_examination_findings" example:"BP 180/110, HR 105"`
	InvestigationResults         *string `json:"investigation_results" example:"ECG shows ST elevation"`
	TreatmentGivenBeforeReferral *string `json:"treatment_given_before_referral" example:"Aspirin 300mg"`
	MedicationOnTransfer         *string `json:"medication_on_transfer" example:"IV Nitroglycerin"`
	ReasonOfReferral             string  `json:"reason_of_referral" example:"Requires immediate cardiological intervention"`
	ReasonForReferralCategory    *string `json:"reason_for_referral_category,omitempty" example:"EMERGENCY"`
	ConditionAtReferral          string  `json:"condition_at_referral" example:"UNSTABLE"`
	ModeOfTransport              *string `json:"mode_of_transport" example:"AMBULANCE"`
	AccompanyingPersonName       *string `json:"accompanying_person_name" example:"Sarah Kebede"`
	AccompanyingPersonPhone      *string `json:"accompanying_person_phone" binding:"omitempty,e164" example:"+251922334455"`

	// Diagnoses
	Diagnoses []DiagnosisDTO `json:"diagnoses" binding:"omitempty,dive"`

	// Optional Extensions
	Vitals          *VitalsDTO                `json:"vitals"`
	EmergencyDetail *EmergencyDetailDTO       `json:"emergency_detail"`
}

type DiagnosisDTO struct {
	ICDCode            string `json:"icd_code" binding:"required" example:"J18.9"`
	IsPrimary          bool   `json:"is_primary" example:"true"`
	DiagnosisCertainty string `json:"diagnosis_certainty" binding:"required,oneof=CONFIRMED SUSPECTED SYMPTOM_ONLY" example:"SUSPECTED"`
}

type VitalsDTO struct {
	SystolicBP      *int16   `json:"systolic_bp" binding:"omitempty,min=40,max=300"`
	DiastolicBP     *int16   `json:"diastolic_bp" binding:"omitempty,min=20,max=200"`
	HeartRate       *int16   `json:"heart_rate" binding:"omitempty,min=20,max=300"`
	SpO2            *float32 `json:"sp_o2" binding:"omitempty,min=0,max=100"`
	Temperature     *float32 `json:"temperature" binding:"omitempty,min=25,max=45"`
	RespiratoryRate *int16   `json:"respiratory_rate" binding:"omitempty,min=4,max=60"`
	GCSScore        *int16   `json:"gcs_score" binding:"omitempty,min=3,max=15"`
}

type EmergencyDetailDTO struct {
	EmergencyJustification string `json:"emergency_justification" binding:"required" example:"Patient requires immediate intubation and bypass surgery"`
}

// ListReferralResponse is the unified DTO for role-based list endpoints
type ListReferralResponse struct {
	ID                  uuid.UUID `json:"id"`
	PatientFirstName    string    `json:"patient_first_name"`
	PatientMiddleName   string    `json:"patient_middle_name"`
	PatientLastName     string    `json:"patient_last_name"`
	PatientRegion       string    `json:"patient_region"`
	Department          string    `json:"department"`
	Status              string    `json:"status"`
	ICDCode             string    `json:"icd_code"`
	Diagnosis           string    `json:"diagnosis"` // Typically the primary diagnosis CodeInfo name
	ConditionAtReferral string    `json:"condition_at_referral"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

type PaginatedReferralResponse struct {
	Data []ListReferralResponse `json:"data"`
	BaseResponse
	Total    int64 `json:"total"`
	Page     int   `json:"page"`
	PageSize int   `json:"page_size"`
}

type ReferralStatusCountResponse struct {
	Status string `json:"status"`
	Count  int64  `json:"count"`
}

type ReferralStatusCountListResponse struct {
	Data []ReferralStatusCountResponse `json:"data"`
	BaseResponse
}

type RejectDTO struct {
	Reason string `json:"reason" binding:"required,min=5"`
}

type ReviseDTO struct {
	Reason string `json:"reason" binding:"required,min=5"`
}

type CancelReferralRequest struct {
	Reason string `json:"reason" example:"Patient decided not to proceed with the referral"`
}

// LogResponseDTO is used by HospitalAdmins to view event log history without clinical details.
type LogResponseDTO struct {
	HistoryID   uuid.UUID `json:"history_id"`
	ReferralID  uuid.UUID `json:"referral_id"`
	ChangedByID uuid.UUID `json:"changed_by_id"`
	Role        string    `json:"role"`
	FromStatus  *string   `json:"from_status,omitempty"`
	ToStatus    string    `json:"to_status"`
	CreatedAt   string    `json:"created_at"`
}

type PaginatedLogResponse struct {
	Data []LogResponseDTO `json:"data"`
	BaseResponse
	Total    int64 `json:"total"`
	Page     int   `json:"page"`
	PageSize int   `json:"page_size"`
}

type ReferralDetailResponse struct {
	entity.Referral
	BaseResponse
	Redirections []RedirectionResponse `json:"redirections,omitempty"`
}

type ReferralCreationResponse struct {
	Referral *entity.Referral `json:"referral"`
	BaseResponse
}

type RedirectReferralRequest struct {
	TargetHospitalID uuid.UUID  `json:"target_hospital_id" binding:"required" example:"a3000000-0000-0000-0000-000000000003"`
	DepartmentID     *uuid.UUID `json:"department_id,omitempty" example:"b1000000-0000-0000-0000-000000000001"`
	Reason           string     `json:"reason" example:"Requires advanced cardiovascular intervention not available here"`
}

type RedirectionResponse struct {
	ID                         uuid.UUID `json:"id"`
	ReferralID                 uuid.UUID `json:"referral_id"`
	RedirectedFromHospitalID   uuid.UUID `json:"redirected_from_hospital_id"`
	RedirectedFromHospitalName string    `json:"redirected_from_hospital_name"`
	RedirectedToHospitalID     uuid.UUID `json:"redirected_to_hospital_id"`
	RedirectedToHospitalName   string    `json:"redirected_to_hospital_name"`
	RedirectedBySpecialistID   uuid.UUID `json:"redirected_by_specialist_id"`
	RedirectionReason          string    `json:"redirection_reason"`
	CreatedAt                  time.Time `json:"created_at"`
}

// ChangeDepartmentRequest is the body for PUT /specialist/referrals/:id/department
type ChangeDepartmentRequest struct {
	DepartmentID uuid.UUID `json:"department_id" binding:"required" example:"b1000000-0000-0000-0000-000000000001"`
}

type ReviewChecklistRequest struct {
	PatientIdentityVerified *bool `json:"patient_identity_verified,omitempty" example:"true"`
	ClinicalHistoryAttached *bool `json:"clinical_history_attached,omitempty" example:"true"`
	VitalsIncluded          *bool `json:"vitals_included,omitempty" example:"true"`
	AttachmentsIncluded     *bool `json:"attachments_included,omitempty" example:"true"`
}

type ReviewChecklistResponse struct {
	PatientIdentityVerified bool `json:"patient_identity_verified"`
	ClinicalHistoryAttached bool `json:"clinical_history_attached"`
	VitalsIncluded          bool `json:"vitals_included"`
	AttachmentsIncluded      bool `json:"attachments_included"`
}
