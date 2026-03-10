package dto

import "github.com/google/uuid"

type CreateReferralRequest struct {
	// Patient Demographics
	NationalIDEnc  *string `json:"national_id_enc" example:"NAT-12345"`
	NationalIDHash *string `json:"national_id_hash"`
	PhoneNumber    *string `json:"phone_number" binding:"omitempty,e164" example:"+251911223344"`
	FirstName      string  `json:"first_name" example:"Abebe"`
	MiddleName     string  `json:"middle_name" example:"Kebede"`
	LastName       string  `json:"last_name" example:"Tadesse"`
	Sex            string  `json:"sex" binding:"omitempty,oneof=male female unknown" example:"male"`
	DateOfBirth    string  `json:"date_of_birth" binding:"omitempty,datetime=2006-01-02" example:"1985-05-18"`
	HomeRegion     *string `json:"home_region" example:"Oromia"`

	// Routing Information
	TargetHospitalID uuid.UUID  `json:"target_hospital_id" binding:"required" example:"0f74f069-d52d-4482-9ba5-41b007fdc1e5"`
	TargetDeptID     uuid.UUID  `json:"target_dept_id" binding:"required" example:"dfc2b777-a5d5-424b-911a-976b2e8d8614"`
	LiaisonOfficerID *uuid.UUID `json:"liaison_officer_id"`

	// Annex IV Clinical Data
	ClinicalSummary              string  `json:"clinical_summary" example:"Patient complains of severe chest pain for 2 hours"`
	PatientHistory               string  `json:"patient_history" example:"Hypertension diagnosed 5 years ago"`
	PhysicalExaminationFindings  *string `json:"physical_examination_findings" example:"BP 180/110, HR 105"`
	InvestigationResults         *string `json:"investigation_results" example:"ECG shows ST elevation"`
	TreatmentGivenBeforeReferral *string `json:"treatment_given_before_referral" example:"Aspirin 300mg"`
	MedicationOnTransfer         *string `json:"medication_on_transfer" example:"IV Nitroglycerin"`
	ReasonOfReferral             string  `json:"reason_of_referral" example:"Requires immediate cardiological intervention"`
	ReasonForReferralCategory    string  `json:"reason_for_referral_category" example:"EMERGENCY"`
	ConditionAtReferral          string  `json:"condition_at_referral" example:"UNSTABLE"`
	ModeOfTransport              *string `json:"mode_of_transport" example:"AMBULANCE"`
	AccompanyingPersonName       *string `json:"accompanying_person_name" example:"Sarah Kebede"`
	AccompanyingPersonPhone      *string `json:"accompanying_person_phone" binding:"omitempty,e164" example:"+251922334455"`

	// Diagnoses
	Diagnoses []DiagnosisDTO `json:"diagnoses" binding:"omitempty,dive"`

	// Status tracking
	Status string `json:"status" binding:"omitempty,oneof=DRAFT SUBMITTED" example:"SUBMITTED"`

	// Optional Extensions
	Vitals          *VitalsDTO          `json:"vitals"`
	EmergencyDetail *EmergencyDetailDTO `json:"emergency_detail"`
}

type DiagnosisDTO struct {
	ICDCode            string `json:"icd_code" binding:"required" example:"I21.9"`
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
	EmergencyJustification string `json:"emergency_justification" binding:"required"`
}
