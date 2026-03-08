package entity

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Vital struct {
	ID              uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	ReferralID      uuid.UUID  `gorm:"type:uuid;not null;index"`
	RecordedAt      time.Time  `gorm:"default:now();index"`
	SystolicBP      *int16     `gorm:"type:smallint"`
	DiastolicBP     *int16     `gorm:"type:smallint"`
	HeartRate       *int16     `gorm:"type:smallint"`
	SpO2            *float64   `gorm:"type:numeric(5,2)"`
	Temperature     *float64   `gorm:"type:numeric(4,1)"`
	RespiratoryRate *int16     `gorm:"type:smallint"`
	GCSScore        *int16     `gorm:"type:smallint"`

	// Timestamps
	CreatedAt time.Time `gorm:"default:now()"`
	UpdatedAt time.Time `gorm:"default:now()"`

	// Relationship
	Referral Referral `gorm:"foreignKey:ReferralID"`
}

func (v *Vital) BeforeCreate(tx *gorm.DB) (err error) {
	if v.ID == uuid.Nil {
		v.ID = uuid.New()
	}
	return
}
