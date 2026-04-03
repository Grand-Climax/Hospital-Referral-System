package entity

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ReferralStatus string

const (
	StatusDraft              ReferralStatus = "DRAFT"
	StatusSubmitted          ReferralStatus = "SUBMITTED"
	StatusUnderLiaisonReview ReferralStatus = "UNDER_LIAISON_REVIEW"
	StatusNeedsRevision      ReferralStatus = "NEEDS_REVISION"      // Liaison rejected → doctor must fix
	StatusForwarded          ReferralStatus = "FORWARDED"
	StatusSpecialistReview   ReferralStatus = "SPECIALIST_REVIEW"   // Received → awaiting specialist decision
	StatusAccepted           ReferralStatus = "ACCEPTED"
	StatusRejected           ReferralStatus = "REJECTED"            // Terminal rejection by specialist/admin
	StatusSpecialistAssigned ReferralStatus = "SPECIALIST_ASSIGNED"
	StatusScheduled          ReferralStatus = "SCHEDULED"
	StatusCompleted          ReferralStatus = "COMPLETED"
	StatusCancelled          ReferralStatus = "CANCELLED"
	StatusMissed             ReferralStatus = "MISSED"
)

type Referral struct {
	ID                  uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	PatientID           uuid.UUID `gorm:"type:uuid;not null;index" json:"patient_id"`
	ReferringDoctorID   uuid.UUID `gorm:"type:uuid;not null;index" json:"referring_doctor_id"`
	SenderHospitalID    uuid.UUID `gorm:"type:uuid;not null;index:idx_sender_created" json:"sender_hospital_id"`
	TargetHospitalID    uuid.UUID `gorm:"type:uuid;not null;index:idx_target_status" json:"target_hospital_id"`
	LiaisonOfficerID    *uuid.UUID `gorm:"type:uuid" json:"liaison_officer_id,omitempty"`
	TargetDeptID        uuid.UUID `gorm:"type:uuid;not null;index:idx_dept_status_created" json:"target_dept_id"`
	Status              ReferralStatus `gorm:"type:varchar(50);not null;default:'DRAFT';index;index:idx_target_status;index:idx_dept_status_created" json:"status"`

	// Triage priority
	ActiveMLPredictionID *uuid.UUID `gorm:"type:uuid;index" json:"active_ml_prediction_id,omitempty"`
	MLSeverityScore      *float64   `gorm:"type:numeric(5,2);index" json:"ml_severity_score,omitempty"`
	WaitingHoursWeight   float64    `gorm:"type:numeric(5,2);default:0.00" json:"waiting_hours_weight"`

	// Handle ML Failure
	MLStatus       string `gorm:"type:varchar(50);default:'PENDING'" json:"ml_status"`
	MLRetryCount   int    `gorm:"default:0" json:"ml_retry_count"`
	MLLastError    *string `gorm:"type:text" json:"ml_last_error,omitempty"`

	// Rejection
	RejectionReason *string `gorm:"type:text" json:"rejection_reason,omitempty"`

	// Timestamps
	CreatedAt time.Time `gorm:"default:now();index:idx_sender_created;index:idx_dept_status_created;index:idx_patient_created" json:"created_at"`
	UpdatedAt time.Time `gorm:"default:now()" json:"updated_at"`

	// Idempotency
	IdempotencyKey *string `gorm:"type:varchar(255);unique" json:"-"`

	// Lifecycle
	IsArchived bool       `gorm:"default:false;index" json:"is_archived"`
	ArchivedAt *time.Time `json:"archived_at,omitempty"`
	IsDeleted  bool       `gorm:"default:false" json:"-"`
	DeletedAt  gorm.DeletedAt `json:"-"`

	// Relationships
	Patient         *Patient                 `gorm:"foreignKey:PatientID" json:"patient,omitempty"`
	ReferralForm    *ReferralForm            `gorm:"foreignKey:ReferralID" json:"referral_form,omitempty"`
	Diagnoses       []ReferralDiagnosis      `gorm:"foreignKey:ReferralID" json:"diagnoses,omitempty"`
	Vitals          []Vital                  `gorm:"foreignKey:ReferralID" json:"vitals,omitempty"`
	EmergencyDetail *ReferralEmergencyDetail `gorm:"foreignKey:ReferralID" json:"emergency_detail,omitempty"`
}

func (r *Referral) BeforeCreate(tx *gorm.DB) (err error) {
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	return
}
