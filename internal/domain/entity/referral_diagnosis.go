package entity

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type DiagnosisCertainty string

const (
	CertaintyConfirmed   DiagnosisCertainty = "CONFIRMED"
	CertaintySuspected   DiagnosisCertainty = "SUSPECTED"
	CertaintySymptomOnly DiagnosisCertainty = "SYMPTOM_ONLY"
)

type ReferralDiagnosis struct {
	ID                 uuid.UUID          `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	ReferralID         uuid.UUID          `gorm:"type:uuid;not null;index;uniqueIndex:idx_ref_diag"`
	ICDCode            string             `gorm:"type:varchar(10);not null;index;uniqueIndex:idx_ref_diag"`
	IsPrimary          bool               `gorm:"default:true"`
	DiagnosisCertainty DiagnosisCertainty `gorm:"type:varchar(20);not null;default:'CONFIRMED'"`

	// Relationship
	CodeInfo *ICDCode `gorm:"foreignKey:ICDCode;references:Code"`
}

func (r *ReferralDiagnosis) BeforeCreate(tx *gorm.DB) (err error) {
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	return
}
