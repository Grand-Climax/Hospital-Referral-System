package entity

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ReferralForm struct {
	ID                           uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	ReferralID                   uuid.UUID `gorm:"type:uuid;not null;uniqueIndex"`
	ClinicalSummary              string    `gorm:"type:text;not null"`
	PatientHistory               string    `gorm:"type:text;not null"`
	PhysicalExaminationFindings  *string   `gorm:"type:text"`
	InvestigationResults         *string   `gorm:"type:text"`
	TreatmentGivenBeforeReferral *string   `gorm:"type:text"`
	MedicationOnTransfer         *string   `gorm:"type:text"`
	ReasonOfReferral             string    `gorm:"type:text;not null"`
	ReasonForReferralCategory    string    `gorm:"type:varchar(50);not null"`
	ConditionAtReferral          string    `gorm:"type:varchar(50);not null"`
	ModeOfTransport              *string   `gorm:"type:varchar(50)"`
	AccompanyingPersonName       *string   `gorm:"type:varchar(100)"`
	AccompanyingPersonPhone      *string   `gorm:"type:varchar(20)"`
}

func (r *ReferralForm) BeforeCreate(tx *gorm.DB) (err error) {
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	return
}
