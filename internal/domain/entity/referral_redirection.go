package entity

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ReferralRedirection struct {
	ID                      uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	ReferralID              uuid.UUID  `gorm:"type:uuid;not null;uniqueIndex"`
	RedirectedToHospitalID  *uuid.UUID `gorm:"type:uuid;index"`
	RedirectionReason       *string    `gorm:"type:text"`

	// Timestamps
	CreatedAt time.Time `gorm:"default:now()"`
	UpdatedAt time.Time `gorm:"default:now()"`

	// Relationships
	Referral              Referral  `gorm:"foreignKey:ReferralID"`
	RedirectedToHospital  *Hospital `gorm:"foreignKey:RedirectedToHospitalID"`
}

func (rr *ReferralRedirection) BeforeCreate(tx *gorm.DB) (err error) {
	if rr.ID == uuid.Nil {
		rr.ID = uuid.New()
	}
	return
}
