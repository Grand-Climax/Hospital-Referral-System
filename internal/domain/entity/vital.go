package entity

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Vital struct {
	ID              uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	ReferralID      uuid.UUID  `gorm:"type:uuid;not null;index" json:"referral_id"`
	RecordedAt      time.Time  `gorm:"default:CURRENT_TIMESTAMP;index" json:"recorded_at"`
	SystolicBP      *int16     `gorm:"type:smallint" json:"systolic_bp,omitempty"`
	DiastolicBP     *int16     `gorm:"type:smallint" json:"diastolic_bp,omitempty"`
	HeartRate       *int16     `gorm:"type:smallint" json:"heart_rate,omitempty"`
	SpO2            *float32   `gorm:"type:numeric(5,2)" json:"sp_o2,omitempty"`
	Temperature     *float32   `gorm:"type:numeric(4,1)" json:"temperature,omitempty"`
	RespiratoryRate *int16     `gorm:"type:smallint" json:"respiratory_rate,omitempty"`
	GCSScore        *int16     `gorm:"type:smallint" json:"gcs_score,omitempty"`

	// Relationships
	Referral *Referral `gorm:"foreignKey:ReferralID" json:"referral,omitempty"`
}

func (v *Vital) BeforeCreate(tx *gorm.DB) (err error) {
	if v.ID == uuid.Nil {
		v.ID = uuid.New()
	}
	return
}
