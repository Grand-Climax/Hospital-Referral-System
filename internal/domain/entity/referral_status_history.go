package entity

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ReferralStatusHistory struct {
	ID         uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	ReferralID uuid.UUID      `gorm:"type:uuid;not null;index;index:idx_history_ref_created"`
	ChangedByID uuid.UUID      `gorm:"type:uuid;not null"`
	FromStatus *ReferralStatus `gorm:"type:varchar(50)"`
	ToStatus   ReferralStatus `gorm:"type:varchar(50);not null"`
	Reason     *string        `gorm:"type:text"`
	ChangedAt  time.Time      `gorm:"default:now();index;index:idx_history_ref_created"`
}

func (rsh *ReferralStatusHistory) BeforeCreate(tx *gorm.DB) (err error) {
	if rsh.ID == uuid.Nil {
		rsh.ID = uuid.New()
	}
	return
}
