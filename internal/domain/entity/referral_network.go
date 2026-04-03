package entity

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ReferralNetwork struct {
	ID                    uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	SenderHospitalID      uuid.UUID `gorm:"type:uuid;not null;index;uniqueIndex:idx_sender_receiver" json:"sender_hospital_id"`
	ReceiverHospitalID    uuid.UUID `gorm:"type:uuid;not null;index;uniqueIndex:idx_sender_receiver" json:"receiver_hospital_id"`
	ReferralType          string    `gorm:"type:varchar(50);default:'routine'" json:"referral_type"`
	RequiresAdminApproval bool      `gorm:"default:false" json:"requires_admin_approval"`

	// Relationships
	SenderHospital   *Hospital `gorm:"foreignKey:SenderHospitalID" json:"sender_hospital,omitempty"`
	ReceiverHospital *Hospital `gorm:"foreignKey:ReceiverHospitalID" json:"receiver_hospital,omitempty"`
}

func (r *ReferralNetwork) BeforeCreate(tx *gorm.DB) (err error) {
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	return
}
