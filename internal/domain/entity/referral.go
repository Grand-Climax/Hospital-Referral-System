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
	StatusForwarded          ReferralStatus = "FORWARDED"
	StatusReceived           ReferralStatus = "RECEIVED"
	StatusSpecialistAssigned ReferralStatus = "SPECIALIST_ASSIGNED"
	StatusScheduled          ReferralStatus = "SCHEDULED"
	StatusCompleted          ReferralStatus = "COMPLETED"
	StatusRejected           ReferralStatus = "REJECTED"
	StatusCancelled          ReferralStatus = "CANCELLED"
	StatusMissed             ReferralStatus = "MISSED" // Keep missed for appointment analytics
)

type Referral struct {
	ID                  uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	PatientID           uuid.UUID `gorm:"type:uuid;not null;index"`
	ReferringDoctorID   uuid.UUID `gorm:"type:uuid;not null;index"`
	SenderHospitalID    uuid.UUID `gorm:"type:uuid;not null;index:idx_sender_created"`
	TargetHospitalID    uuid.UUID `gorm:"type:uuid;not null;index:idx_target_status"`
	LiaisonOfficerID    *uuid.UUID `gorm:"type:uuid"`
	TargetDeptID        uuid.UUID `gorm:"type:uuid;not null;index:idx_dept_status_created"`
	Status              ReferralStatus `gorm:"type:varchar(50);not null;default:'DRAFT';index;index:idx_target_status;index:idx_dept_status_created"`

	// Triage priority
	ActiveMLPredictionID *uuid.UUID `gorm:"type:uuid;index"`
	MLSeverityScore      *float64   `gorm:"type:numeric(5,2);index"`
	WaitingHoursWeight   float64    `gorm:"type:numeric(5,2);default:0.00"`

	// Handle ML Failure
	MLStatus       string `gorm:"type:varchar(50);default:'PENDING'"`
	MLRetryCount   int    `gorm:"default:0"`
	MLLastError    *string `gorm:"type:text"`

	// Rejection
	RejectionReason *string `gorm:"type:text"`

	// Timestamps
	CreatedAt time.Time `gorm:"default:now();index:idx_sender_created;index:idx_dept_status_created;index:idx_patient_created"`
	UpdatedAt time.Time `gorm:"default:now()"`

	// Idempotency
	IdempotencyKey *string `gorm:"type:varchar(255);unique"`

	// Lifecycle
	IsArchived bool       `gorm:"default:false;index"`
	ArchivedAt *time.Time
	IsDeleted  bool       `gorm:"default:false"`
	DeletedAt  gorm.DeletedAt

	// Relationships
	Patient         *Patient                 `gorm:"foreignKey:PatientID"`
	ReferralForm    *ReferralForm            `gorm:"foreignKey:ReferralID"`
	Diagnoses       []ReferralDiagnosis      `gorm:"foreignKey:ReferralID"`
	Vitals          []Vital                  `gorm:"foreignKey:ReferralID"`
	EmergencyDetail *ReferralEmergencyDetail `gorm:"foreignKey:ReferralID"`
}

func (r *Referral) BeforeCreate(tx *gorm.DB) (err error) {
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	return
}
