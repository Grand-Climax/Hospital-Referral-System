package entity

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ArrivalStatus string

const (
	ArrivalExpected ArrivalStatus = "EXPECTED"
	ArrivalArrived  ArrivalStatus = "ARRIVED"
	ArrivalAdmitted ArrivalStatus = "ADMITTED"
	ArrivalMissed   ArrivalStatus = "MISSED"
)

type MissReason string

const (
	MissPatientNoShow             MissReason = "PATIENT_NO_SHOW"
	MissPatientContactedResched   MissReason = "PATIENT_CONTACTED_RESCHEDULE"
	MissHospitalCancelled         MissReason = "HOSPITAL_CANCELLED"
	MissHospitalCapacityIssue      MissReason = "HOSPITAL_CAPACITY_ISSUE"
)

type QueueStatus string

const (
	QueueWaiting   QueueStatus = "WAITING"
	QueueScheduled QueueStatus = "SCHEDULED"
	QueueArrived   QueueStatus = "ARRIVED"
	QueueMissed     QueueStatus = "MISSED"
	QueueCancelled QueueStatus = "CANCELLED"
)

type TriageQueue struct {
	ID                uuid.UUID     `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	ReferralID        uuid.UUID     `gorm:"type:uuid;not null;uniqueIndex:idx_ref_date" json:"referral_id"`
	DeptID            uuid.UUID     `gorm:"type:uuid;not null;index:idx_triage_dept_score" json:"dept_id"`
	AppointmentDate   *time.Time     `gorm:"type:date;uniqueIndex:idx_ref_date;index:idx_triage_dept_score" json:"appointment_date,omitempty"`
	CompositeScore    float64       `gorm:"type:numeric(5,2);not null;index:idx_triage_dept_score,priority:desc" json:"composite_score"`
	AssignedAt        time.Time     `gorm:"default:now()" json:"assigned_at"`
	ArrivalStatus     ArrivalStatus `gorm:"type:arrivalstatus;not null;default:'EXPECTED';index:idx_triage_date_status" json:"arrival_status"`
	ArrivedAt         *time.Time    `json:"arrived_at,omitempty"`
	MarkedBy          *uuid.UUID    `gorm:"type:uuid" json:"marked_by,omitempty"`
	MissReason        *MissReason   `gorm:"type:missreason" json:"miss_reason,omitempty"`
	AssignedDoctorID  *uuid.UUID    `gorm:"type:uuid;index" json:"assigned_doctor_id,omitempty"`
	DoctorAssignedAt  *time.Time    `json:"doctor_assigned_at,omitempty"`
	QueueStatus       QueueStatus   `gorm:"type:queuestatus;default:'WAITING'" json:"queue_status"`
	ArrivalBoost      int           `gorm:"default:0" json:"arrival_boost"`
	RescheduleReason  *string       `gorm:"type:varchar(50)" json:"reschedule_reason,omitempty"`
}

func (tq *TriageQueue) BeforeCreate(tx *gorm.DB) (err error) {
	if tq.ID == uuid.Nil {
		tq.ID = uuid.New()
	}
	return
}
