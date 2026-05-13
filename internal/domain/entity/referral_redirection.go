package entity

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ReferralRedirection struct {
	ID                       uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	ReferralID               uuid.UUID      `gorm:"type:uuid;not null;index" json:"referral_id"`
	RedirectedFromHospitalID uuid.UUID      `gorm:"type:uuid;not null" json:"redirected_from_hospital_id"`
	RedirectedToHospitalID   uuid.UUID      `gorm:"type:uuid;not null" json:"redirected_to_hospital_id"`
	RedirectedBySpecialistID uuid.UUID      `gorm:"type:uuid;not null" json:"redirected_by_specialist_id"`
	RedirectionReason        *string        `gorm:"type:text" json:"redirection_reason,omitempty"`
	CreatedAt                time.Time      `json:"created_at"`
	UpdatedAt                time.Time      `json:"updated_at"`
	DeletedAt                gorm.DeletedAt `gorm:"index" json:"-"`

	RedirectedFromHospital *Hospital `gorm:"foreignKey:RedirectedFromHospitalID" json:"redirected_from_hospital"`
	RedirectedToHospital   *Hospital `gorm:"foreignKey:RedirectedToHospitalID" json:"redirected_to_hospital"`
	Referral               *Referral `gorm:"foreignKey:ReferralID" json:"referral,omitempty"`
	RedirectedBy           *User     `gorm:"foreignKey:RedirectedBySpecialistID" json:"redirected_by,omitempty"`
}

func (rr *ReferralRedirection) BeforeCreate(tx *gorm.DB) (err error) {
	if rr.ID == uuid.Nil {
		rr.ID = uuid.New()
	}
	return
}
