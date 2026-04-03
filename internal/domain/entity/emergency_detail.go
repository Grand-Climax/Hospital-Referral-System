package entity

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ReferralEmergencyDetail struct {
	ID                     uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	ReferralID             uuid.UUID  `gorm:"type:uuid;not null;uniqueIndex" json:"referral_id"`
	EmergencyJustification string     `gorm:"type:text;not null" json:"emergency_justification"`
	AdmittedAt             *time.Time `gorm:"type:timestamp" json:"admitted_at,omitempty"`
}

func (r *ReferralEmergencyDetail) BeforeCreate(tx *gorm.DB) (err error) {
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	return
}
