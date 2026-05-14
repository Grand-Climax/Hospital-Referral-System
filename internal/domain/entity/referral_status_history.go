package entity

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ReferralStatusHistory struct {
	ID          uuid.UUID       `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	ReferralID  uuid.UUID       `gorm:"type:uuid;not null;index;index:idx_history_ref_created" json:"referral_id"`
	ChangedByID uuid.UUID       `gorm:"type:uuid;not null" json:"changed_by_id"`
	FromStatus  *ReferralStatus `gorm:"type:varchar(50)" json:"from_status,omitempty"`
	ToStatus    ReferralStatus  `gorm:"type:varchar(50);not null" json:"to_status"`
	Reason      *string         `gorm:"type:text" json:"reason,omitempty"`
	ChangedAt   time.Time       `gorm:"default:now();index;index:idx_history_ref_created" json:"changed_at"`

	// Relationships
	Referral  *Referral `gorm:"foreignKey:ReferralID" json:"referral,omitempty"`
	ChangedBy *User     `gorm:"foreignKey:ChangedByID" json:"changed_by,omitempty"`
}

func (rsh *ReferralStatusHistory) BeforeCreate(tx *gorm.DB) (err error) {
	if rsh.ID == uuid.Nil {
		rsh.ID = uuid.New()
	}
	return
}
