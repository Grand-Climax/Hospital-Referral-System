package entity

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UpdateReason string

const (
	ReasonMissedAppointmentReEvaluation UpdateReason = "MISSED_APPOINTMENT_RE_EVALUATION"
	ReasonConditionChange               UpdateReason = "CONDITION_CHANGE"
	ReasonSpecialistNote                UpdateReason = "SPECIALIST_NOTE"
)

type ClinicalUpdate struct {
	ID            uuid.UUID    `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	ReferralID    uuid.UUID    `gorm:"type:uuid;not null;index;index:idx_clinical_update_ref_created"`
	UpdatedBy     uuid.UUID    `gorm:"type:uuid;not null;index"`
	UpdateReason  UpdateReason `gorm:"type:varchar(50);not null"`
	ClinicalNotes string      `gorm:"type:text;not null"`
	CreatedAt     time.Time    `gorm:"default:now();index:idx_clinical_update_ref_created"`
	UpdatedAt     time.Time    `gorm:"default:now()"`

	// Relationships
	Referral Referral `gorm:"foreignKey:ReferralID"`
	Updater  User     `gorm:"foreignKey:UpdatedBy"`
}

func (cu *ClinicalUpdate) BeforeCreate(tx *gorm.DB) (err error) {
	if cu.ID == uuid.Nil {
		cu.ID = uuid.New()
	}
	return
}
