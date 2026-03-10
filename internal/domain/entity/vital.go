package entity

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Vital struct {
	ID              uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	ReferralID      uuid.UUID  `gorm:"type:uuid;not null;index"`
	RecordedAt      time.Time  `gorm:"default:CURRENT_TIMESTAMP;index"`
	SystolicBP      *int16     `gorm:"type:smallint"`
	DiastolicBP     *int16     `gorm:"type:smallint"`
	HeartRate       *int16     `gorm:"type:smallint"`
	SpO2            *float32   `gorm:"type:numeric(5,2)"`
	Temperature     *float32   `gorm:"type:numeric(4,1)"`
	RespiratoryRate *int16     `gorm:"type:smallint"`
	GCSScore        *int16     `gorm:"type:smallint"`
}

func (v *Vital) BeforeCreate(tx *gorm.DB) (err error) {
	if v.ID == uuid.Nil {
		v.ID = uuid.New()
	}
	return
}
