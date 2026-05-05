package entity

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ReferralStatus string

const (
	StatusDraft                 ReferralStatus = "DRAFT"
	StatusSubmitted             ReferralStatus = "SUBMITTED"
	StatusUnderLiaisonReview    ReferralStatus = "UNDER_LIAISON_REVIEW"
	StatusForwarded             ReferralStatus = "FORWARDED"
	StatusUnderSpecialistReview ReferralStatus = "UNDER_SPECIALIST_REVIEW"
	StatusAccepted              ReferralStatus = "ACCEPTED"
	StatusScheduled             ReferralStatus = "SCHEDULED"
	StatusAssigned              ReferralStatus = "ASSIGNED"
	StatusCompleted             ReferralStatus = "COMPLETED"
	
	// Interruption Statuses
	StatusNeedRevision         ReferralStatus = "NEED_REVISION"
	StatusCancelled            ReferralStatus = "CANCELLED"
	StatusRejectedByLiaison    ReferralStatus = "REJECTED_BY_LIAISON"
	StatusRejectedBySpecialist ReferralStatus = "REJECTED_BY_SPECIALIST"
	StatusMissed               ReferralStatus = "MISSED"
	StatusRescheduled          ReferralStatus = "RESCHEDULED"
	StatusRedirected           ReferralStatus = "REDIRECTED"
	StatusAdmitted             ReferralStatus = "ADMITTED"
	StatusRejectedAfterSend    ReferralStatus = "REJECTED_AFTER_SEND"
	StatusDeceased             ReferralStatus = "DECEASED"
)

type TriageStatus string

const (
	TriageAutoScored TriageStatus = "AUTO_SCORED"
	TriageReviewed   TriageStatus = "REVIEWED"
	TriageOverridden TriageStatus = "OVERRIDDEN"
)

type Referral struct {
	ID                  uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	PatientID           uuid.UUID `gorm:"type:uuid;not null;index" json:"patient_id"`
	ReferringDoctorID   uuid.UUID `gorm:"type:uuid;not null;index" json:"referring_doctor_id"`
	SenderHospitalID    uuid.UUID `gorm:"type:uuid;not null;index:idx_sender_created" json:"sender_hospital_id"`
	TargetHospitalID    uuid.UUID `gorm:"type:uuid;not null;index:idx_target_status" json:"target_hospital_id"`
	LiaisonOfficerID    *uuid.UUID `gorm:"type:uuid" json:"liaison_officer_id,omitempty"`
	TargetDeptID        uuid.UUID `gorm:"type:uuid;not null;index:idx_dept_status_created" json:"target_dept_id"`
	Status              ReferralStatus `gorm:"type:referralstatus;not null;default:'DRAFT';index;index:idx_target_status;index:idx_dept_status_created" json:"status"`
	TriageStatus        TriageStatus   `gorm:"type:triagestatus;not null;default:'AUTO_SCORED'" json:"triage_status"`

	// Triage priority
	ActiveMLPredictionID *uuid.UUID `gorm:"type:uuid;index" json:"active_ml_prediction_id,omitempty"`
	MLSeverityScore      *float64   `gorm:"type:numeric(5,2);index" json:"ml_severity_score,omitempty"`
	WaitingHoursWeight   float64    `gorm:"type:numeric(5,2);default:0.00" json:"waiting_hours_weight"`

	// Handle ML Failure
	MLStatus       string `gorm:"type:varchar(50);default:'PENDING'" json:"ml_status"`
	MLRetryCount   int    `gorm:"default:0" json:"ml_retry_count"`
	MLLastError    *string `gorm:"type:text" json:"ml_last_error,omitempty"`

	// Rejection and Revision
	RejectionReason *string `gorm:"type:text" json:"rejection_reason,omitempty"`
	RevisionReason  *string `gorm:"type:text" json:"revision_reason,omitempty"`

	// Specialist Assignment
	SpecialistID    *uuid.UUID `gorm:"type:uuid;index" json:"specialist_id,omitempty"`

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
	ReferralForm    *ReferralForm            `gorm:"foreignKey:ReferralID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"referral_form,omitempty"`
	Diagnoses       []ReferralDiagnosis      `gorm:"foreignKey:ReferralID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"diagnoses,omitempty"`
	Vitals          []Vital                  `gorm:"foreignKey:ReferralID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"vitals,omitempty"`
	EmergencyDetail *ReferralEmergencyDetail `gorm:"foreignKey:ReferralID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"emergency_detail,omitempty"`
	Attachments     []Attachment            `gorm:"foreignKey:ReferralID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"attachments,omitempty"`
	Redirections    []ReferralRedirection   `gorm:"foreignKey:ReferralID" json:"redirections,omitempty"`
	
	ReceiverHospital *Hospital `gorm:"foreignKey:TargetHospitalID" json:"receiver_hospital,omitempty"`
	SenderHospital   *Hospital `gorm:"foreignKey:SenderHospitalID" json:"sender_hospital,omitempty"`
	TargetDepartment *Department `gorm:"foreignKey:TargetDeptID" json:"target_department,omitempty"`
}

func (r *Referral) BeforeCreate(tx *gorm.DB) (err error) {
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	return
}
