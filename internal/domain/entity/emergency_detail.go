package entity

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ReferralEmergencyDetail struct {
	ID                     uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	ReferralID             uuid.UUID  `gorm:"type:uuid;not null;uniqueIndex"`
	EmergencyJustification string     `gorm:"type:text;not null"`
	AdmittedAt             *time.Time `gorm:"type:timestamp"`
}

func (r *ReferralEmergencyDetail) BeforeCreate(tx *gorm.DB) (err error) {
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	return
}
