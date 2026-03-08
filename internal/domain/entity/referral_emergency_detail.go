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
	AdmittedAt             *time.Time

	// Timestamps
	CreatedAt time.Time `gorm:"default:now()"`
	UpdatedAt time.Time `gorm:"default:now()"`

	// Relationship
	Referral Referral `gorm:"foreignKey:ReferralID"`
}

func (red *ReferralEmergencyDetail) BeforeCreate(tx *gorm.DB) (err error) {
	if red.ID == uuid.Nil {
		red.ID = uuid.New()
	}
	return
}
