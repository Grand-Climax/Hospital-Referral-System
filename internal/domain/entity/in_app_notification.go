package entity

import (
	"time"

	"github.com/google/uuid"
)

type InAppNotification struct {
	ID         uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	UserID     uuid.UUID  `gorm:"type:uuid;not null;index:idx_user_read_created,priority:1" json:"user_id"`
	ReferralID *uuid.UUID `gorm:"type:uuid;index" json:"referral_id,omitempty"`
	Title      string     `gorm:"type:varchar(255);not null" json:"title"`
	Message    string     `gorm:"type:text;not null" json:"message"`
	EventType  string     `gorm:"type:varchar(50);not null" json:"event_type"`
	IsRead     bool       `gorm:"default:false;index:idx_user_read_created,priority:2" json:"is_read"`
	CreatedAt  time.Time  `gorm:"default:now();index:idx_user_read_created,priority:3" json:"created_at"`

	// Relationships with Cascade Delete
	User     *User     `gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"user,omitempty"`
	Referral *Referral `gorm:"foreignKey:ReferralID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"referral,omitempty"`
}
