package dto

import "github.com/google/uuid"

type CreateReferralRequest struct {
	// Patient Demographics
	NationalIDEnc  *string `json:"national_id_enc"`
	NationalIDHash *string `json:"national_id_hash"`
	PhoneNumber    *string `json:"phone_number" binding:"omitempty,e164"`
	FirstName      string  `json:"first_name"`
	MiddleName     string  `json:"middle_name"`
	LastName       string  `json:"last_name"`
	Sex            string  `json:"sex" binding:"omitempty,oneof=male female unknown"`
	DateOfBirth    string  `json:"date_of_birth" binding:"omitempty,datetime=2006-01-02"`
	HomeRegion     *string `json:"home_region"`

	// Routing Information
	TargetHospitalID uuid.UUID  `json:"target_hospital_id" binding:"required"`
	TargetDeptID     uuid.UUID  `json:"target_dept_id" binding:"required"`
	LiaisonOfficerID *uuid.UUID `json:"liaison_officer_id"`

	// Annex IV Clinical Data
	ClinicalSummary              string  `json:"clinical_summary"`
	PatientHistory               string  `json:"patient_history"`
	PhysicalExaminationFindings  *string `json:"physical_examination_findings"`
	InvestigationResults         *string `json:"investigation_results"`
	TreatmentGivenBeforeReferral *string `json:"treatment_given_before_referral"`
	MedicationOnTransfer         *string `json:"medication_on_transfer"`
	ReasonOfReferral             string  `json:"reason_of_referral"`
	ReasonForReferralCategory    string  `json:"reason_for_referral_category"`
	ConditionAtReferral          string  `json:"condition_at_referral"`
	ModeOfTransport              *string `json:"mode_of_transport"`
	AccompanyingPersonName       *string `json:"accompanying_person_name"`
	AccompanyingPersonPhone      *string `json:"accompanying_person_phone" binding:"omitempty,e164"`

	// Diagnoses
	Diagnoses []DiagnosisDTO `json:"diagnoses" binding:"omitempty,dive"`

	// Status tracking
	Status string `json:"status" binding:"omitempty,oneof=DRAFT SUBMITTED"`

	// Optional Extensions
	Vitals          *VitalsDTO          `json:"vitals"`
	EmergencyDetail *EmergencyDetailDTO `json:"emergency_detail"`
}

type DiagnosisDTO struct {
	ICDCode            string `json:"icd_code" binding:"required"`
	IsPrimary          bool   `json:"is_primary"`
	DiagnosisCertainty string `json:"diagnosis_certainty" binding:"required,oneof=CONFIRMED SUSPECTED SYMPTOM_ONLY"`
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
	EmergencyJustification string `json:"emergency_justification" binding:"required"`
}
