package entity

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ReferralNetwork struct {
	ID                    uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	SenderHospitalID      uuid.UUID `gorm:"type:uuid;not null;index;uniqueIndex:idx_sender_receiver"`
	ReceiverHospitalID    uuid.UUID `gorm:"type:uuid;not null;index;uniqueIndex:idx_sender_receiver"`
	ReferralType          string    `gorm:"type:varchar(50);default:'routine'"`
	RequiresAdminApproval bool      `gorm:"default:false"`

	// Relationships
	SenderHospital   *Hospital `gorm:"foreignKey:SenderHospitalID"`
	ReceiverHospital *Hospital `gorm:"foreignKey:ReceiverHospitalID"`
}

func (r *ReferralNetwork) BeforeCreate(tx *gorm.DB) (err error) {
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	return
}
