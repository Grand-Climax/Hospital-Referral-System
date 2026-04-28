package entity

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ReferralForm struct {
	ID                           uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	ReferralID                   uuid.UUID `gorm:"type:uuid;not null;uniqueIndex" json:"referral_id"`
	ClinicalSummary              string    `gorm:"type:text;not null" json:"clinical_summary"`
	PatientHistory               string    `gorm:"type:text;not null" json:"patient_history"`
	PhysicalExaminationFindings  *string   `gorm:"type:text" json:"physical_examination_findings,omitempty"`
	InvestigationResults         *string   `gorm:"type:text" json:"investigation_results,omitempty"`
	TreatmentGivenBeforeReferral *string   `gorm:"type:text" json:"treatment_given_before_referral,omitempty"`
	MedicationOnTransfer         *string   `gorm:"type:text" json:"medication_on_transfer,omitempty"`
	ReasonOfReferral             string    `gorm:"type:text;not null" json:"reason_of_referral"`
	ReasonForReferralCategory    *string   `gorm:"type:varchar(50)" json:"reason_for_referral_category,omitempty"`
	ConditionAtReferral          string    `gorm:"type:varchar(50);not null" json:"condition_at_referral"`
	ModeOfTransport              *string   `gorm:"type:varchar(50)" json:"mode_of_transport,omitempty"`
	AccompanyingPersonName       *string   `gorm:"type:varchar(100)" json:"accompanying_person_name,omitempty"`
	AccompanyingPersonPhone      *string   `gorm:"type:varchar(20)" json:"accompanying_person_phone,omitempty"`
}

func (r *ReferralForm) BeforeCreate(tx *gorm.DB) (err error) {
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	return
}
