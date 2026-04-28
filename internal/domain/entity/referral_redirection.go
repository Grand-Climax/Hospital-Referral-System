package entity

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ReferralRedirection struct {
	ID                     uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	ReferralID             uuid.UUID `gorm:"type:uuid;not null;unique;index" json:"referral_id"`
	RedirectedToHospitalID uuid.UUID `gorm:"type:uuid;not null" json:"redirected_to_hospital_id"`
	RedirectionReason      *string   `gorm:"type:text" json:"redirection_reason,omitempty"`
}

func (rr *ReferralRedirection) BeforeCreate(tx *gorm.DB) (err error) {
	if rr.ID == uuid.Nil {
		rr.ID = uuid.New()
	}
	return
}
