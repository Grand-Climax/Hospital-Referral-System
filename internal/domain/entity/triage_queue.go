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

type TriageQueue struct {
	ID               uuid.UUID     `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	ReferralID       uuid.UUID     `gorm:"type:uuid;not null;uniqueIndex:idx_ref_date" json:"referral_id"`
	HospitalID       uuid.UUID     `gorm:"type:uuid;not null;index:idx_triage_hosp_dept,priority:1" json:"hospital_id"`
	DepartmentID     uuid.UUID     `gorm:"type:uuid;not null;index:idx_triage_hosp_dept,priority:2" json:"department_id"`
	AppointmentDate  *time.Time     `gorm:"type:date;uniqueIndex:idx_ref_date;index:idx_triage_dept_score" json:"appointment_date,omitempty"`
	CompositeScore   float64       `gorm:"type:numeric(5,2);not null;index:idx_triage_dept_score,priority:desc" json:"composite_score"`
	AssignedAt       time.Time     `gorm:"default:now()" json:"assigned_at"`
	ArrivalStatus    ArrivalStatus `gorm:"type:arrivalstatus;not null;default:'EXPECTED';index:idx_triage_date_status" json:"arrival_status"`
	ArrivedAt        *time.Time    `json:"arrived_at,omitempty"`
	MarkedBy         *uuid.UUID    `gorm:"type:uuid" json:"marked_by,omitempty"`
	MissReason       *MissReason   `gorm:"type:missreason" json:"miss_reason,omitempty"`
	AssignedDoctorID *uuid.UUID    `gorm:"type:uuid;index" json:"assigned_doctor_id,omitempty"`
	DoctorAssignedAt *time.Time    `json:"doctor_assigned_at,omitempty"`
	WaitingHoursWeight float64     `gorm:"type:numeric(5,2);default:0.00" json:"waiting_hours_weight"`

	Hospital       *Hospital   `gorm:"foreignKey:HospitalID" json:"hospital,omitempty"`
	Department     *Department `gorm:"foreignKey:DepartmentID" json:"department,omitempty"`
	Referral       *Referral   `gorm:"foreignKey:ReferralID" json:"referral,omitempty"`
	MarkedByAdmin  *User       `gorm:"foreignKey:MarkedBy" json:"marked_by_admin,omitempty"`
	AssignedDoctor *User       `gorm:"foreignKey:AssignedDoctorID" json:"assigned_doctor,omitempty"`
}

func (tq *TriageQueue) BeforeCreate(tx *gorm.DB) (err error) {
	if tq.ID == uuid.Nil {
		tq.ID = uuid.New()
	}
	return
}
