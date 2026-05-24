package dto

import (
	"time"

	"github.com/google/uuid"
)

// This file defines the role-aware triage-queue detail response DTOs.
// All three share the queue-level facts (composite score, statuses,
// timestamps); they differ in clinical depth and the action set the
// caller is allowed to perform.
//
// Wire shapes are tracked in Swagger via the Go struct tags below.

// ── Shared sub-DTOs ───────────────────────────────────────────────────────────

// TriageDetailPatient is the patient block exposed to specialists and
// (in a redacted form) other roles. Sensitive PII (national_id, phone)
// is only included when the role projection populates it.
type TriageDetailPatient struct {
	ID          uuid.UUID `json:"id"`
	FullName    string    `json:"full_name" example:"Hanan Tadesse"`
	FirstName   string    `json:"first_name,omitempty" example:"Hanan"`
	MiddleName  string    `json:"middle_name,omitempty" example:""`
	LastName    string    `json:"last_name,omitempty" example:"Tadesse"`
	Sex         string    `json:"sex,omitempty" example:"female"`
	AgeYears    *int      `json:"age_years,omitempty" example:"34"`
	HomeRegion  string    `json:"home_region,omitempty" example:"Oromia"`
	NationalID  string    `json:"national_id,omitempty"`
	PhoneNumber string    `json:"phone_number,omitempty"`
}

// TriageDetailVitals is the latest vitals snapshot for the referral.
type TriageDetailVitals struct {
	RecordedAt      *time.Time `json:"recorded_at,omitempty"`
	SystolicBP      *int16     `json:"systolic_bp,omitempty"`
	DiastolicBP     *int16     `json:"diastolic_bp,omitempty"`
	HeartRate       *int16     `json:"heart_rate,omitempty"`
	SpO2            *float32   `json:"sp_o2,omitempty"`
	Temperature     *float32   `json:"temperature,omitempty"`
	RespiratoryRate *int16     `json:"respiratory_rate,omitempty"`
	GCSScore        *int16     `json:"gcs_score,omitempty"`
}

// TriageDetailDiagnosis is one ICD-10 entry on the referral.
type TriageDetailDiagnosis struct {
	ICDCode            string `json:"icd_code" example:"I21.9"`
	Description        string `json:"description,omitempty" example:"Acute myocardial infarction, unspecified"`
	IsPrimary          bool   `json:"is_primary" example:"true"`
	DiagnosisCertainty string `json:"diagnosis_certainty" example:"CONFIRMED"`
}

// TriageDetailDoctor is a minimal doctor card returned for the
// treating-doctor pointer and each consulting doctor.
type TriageDetailDoctor struct {
	UserID       uuid.UUID  `json:"user_id"`
	FullName     string     `json:"full_name" example:"Dr. Selamawit Bekele"`
	Email        string     `json:"email,omitempty"`
	AccessType   string     `json:"access_type,omitempty" example:"TREATING_DOCTOR"`
	GrantedAt    *time.Time `json:"granted_at,omitempty"`
	GrantedBy    *uuid.UUID `json:"granted_by,omitempty"`
	RevokedAt    *time.Time `json:"revoked_at,omitempty"`
	RevokeReason string     `json:"revoke_reason,omitempty"`
}

// TriageDetailTimelineEvent is one tick on the arrival_history timeline.
// Source is the audit_logs table, filtered to a curated action set.
type TriageDetailTimelineEvent struct {
	At          time.Time `json:"at"`
	Event       string    `json:"event" example:"CONFIRM_ARRIVAL"`
	Description string    `json:"description,omitempty" example:"Patient arrived and was checked in"`
	ActorID     uuid.UUID `json:"actor_id,omitempty"`
	ActorName   string    `json:"actor_name,omitempty"`
}

// TriageDetailAvailableActions encodes the role-aware action bitmap the
// FE uses to enable/disable buttons. Server is the source of truth so
// guards never live only in the UI.
type TriageDetailAvailableActions struct {
	Schedule          bool `json:"schedule"`
	EmergencySchedule bool `json:"emergency_schedule"`
	ReturnToTriage    bool `json:"return_to_triage"`
	MarkArrived       bool `json:"mark_arrived"`
	MarkMissed        bool `json:"mark_missed"`
	AssignDoctor      bool `json:"assign_doctor"`
	RevokeDoctor      bool `json:"revoke_doctor"`
}

// ── Specialist projection ─────────────────────────────────────────────────────

// TriageDetailSpecialistResponse is the rich detail view a Receiving
// Specialist sees: full PII, clinical text, diagnoses, vitals, ML
// severity, access list, arrival timeline.
type TriageDetailSpecialistResponse struct {
	Success bool `json:"success" example:"true"`
	Data    struct {
		QueueID             uuid.UUID                    `json:"queue_id"`
		ReferralID          uuid.UUID                    `json:"referral_id"`
		ArrivalStatus       string                       `json:"arrival_status" example:"EXPECTED"`
		ReferralStatus      string                       `json:"referral_status" example:"ACCEPTED"`
		ConditionAtReferral string                       `json:"condition_at_referral" example:"critical"`
		CompositeScore      float64                      `json:"composite_score" example:"82.5"`
		MLSeverityScore     *float64                     `json:"ml_severity_score,omitempty" example:"0.73"`
		TriageStatus        string                       `json:"triage_status" example:"AUTO_SCORED"`
		AppointmentDate     *time.Time                   `json:"appointment_date,omitempty"`
		DepartmentID        uuid.UUID                    `json:"department_id"`
		DepartmentName      string                       `json:"department_name" example:"Cardiology"`
		CreatedAt           time.Time                    `json:"created_at"`
		Patient             TriageDetailPatient          `json:"patient"`
		Vitals              *TriageDetailVitals          `json:"vitals,omitempty"`
		Diagnoses           []TriageDetailDiagnosis      `json:"diagnoses"`
		ClinicalSummary     string                       `json:"clinical_summary"`
		ReasonOfReferral    string                       `json:"reason_of_referral"`
		InvestigationResult string                       `json:"investigation_results,omitempty"`
		TreatingDoctor      *TriageDetailDoctor          `json:"treating_doctor,omitempty"`
		ConsultingDoctors   []TriageDetailDoctor         `json:"consulting_doctors"`
		ReferralAccessList  []TriageDetailDoctor         `json:"referral_access_list"`
		ArrivalHistory      []TriageDetailTimelineEvent  `json:"arrival_history"`
		AvailableActions    TriageDetailAvailableActions `json:"available_actions"`
	} `json:"data"`
}

// ── Receptionist projection (redacted) ───────────────────────────────────────

// TriageDetailReceptionistResponse strips clinical fields, severity
// scores, and the access list. Receptionist sees patient name, time,
// arrival status, and assigned doctor only.
type TriageDetailReceptionistResponse struct {
	Success bool `json:"success" example:"true"`
	Data    struct {
		QueueID            uuid.UUID                    `json:"queue_id"`
		ReferralID         uuid.UUID                    `json:"referral_id"`
		ArrivalStatus      string                       `json:"arrival_status" example:"EXPECTED"`
		ReferralStatus     string                       `json:"referral_status" example:"SCHEDULED"`
		AppointmentDate    *time.Time                   `json:"appointment_date,omitempty"`
		DepartmentID       uuid.UUID                    `json:"department_id"`
		DepartmentName     string                       `json:"department_name" example:"Cardiology"`
		Patient            TriageDetailPatient          `json:"patient"`
		AssignedDoctor     *TriageDetailDoctor          `json:"assigned_doctor,omitempty"`
		ArrivedAt          *time.Time                   `json:"arrived_at,omitempty"`
		MissReason         string                       `json:"miss_reason,omitempty"`
		ArrivalHistory     []TriageDetailTimelineEvent  `json:"arrival_history"`
		AvailableActions   TriageDetailAvailableActions `json:"available_actions"`
	} `json:"data"`
}

// ── Department Head projection ───────────────────────────────────────────────

// TriageDetailDeptHeadResponse is the operations-oriented view a Dept
// Head sees: capacity-relevant fields (score, statuses, assignment,
// condition), no clinical text or ML internals.
type TriageDetailDeptHeadResponse struct {
	Success bool `json:"success" example:"true"`
	Data    struct {
		QueueID             uuid.UUID                    `json:"queue_id"`
		ReferralID          uuid.UUID                    `json:"referral_id"`
		ArrivalStatus       string                       `json:"arrival_status" example:"EXPECTED"`
		ReferralStatus      string                       `json:"referral_status" example:"SCHEDULED"`
		ConditionAtReferral string                       `json:"condition_at_referral" example:"urgent"`
		CompositeScore      float64                      `json:"composite_score" example:"61.2"`
		AppointmentDate     *time.Time                   `json:"appointment_date,omitempty"`
		DepartmentID        uuid.UUID                    `json:"department_id"`
		DepartmentName      string                       `json:"department_name" example:"Cardiology"`
		HasDoctorAssigned   bool                         `json:"has_doctor_assigned"`
		AssignedDoctor      *TriageDetailDoctor          `json:"assigned_doctor,omitempty"`
		Patient             TriageDetailPatient          `json:"patient"`
		ArrivalHistory      []TriageDetailTimelineEvent  `json:"arrival_history"`
		AvailableActions    TriageDetailAvailableActions `json:"available_actions"`
		CreatedAt           time.Time                    `json:"created_at"`
	} `json:"data"`
}
