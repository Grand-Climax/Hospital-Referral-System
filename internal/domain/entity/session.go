package entity

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Session struct {
	ID               uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	UserID           uuid.UUID `gorm:"type:uuid;not null;index" json:"user_id"`
	RefreshTokenHash string    `gorm:"type:varchar(255);not null;index" json:"-"`
	IPAddress        *string   `gorm:"type:varchar(45)" json:"ip_address,omitempty"`
	UserAgent        *string   `gorm:"type:text" json:"user_agent,omitempty"`
	ExpiresAt        time.Time `gorm:"not null;index" json:"expires_at"`
	CreatedAt        time.Time `gorm:"default:now()" json:"created_at"`
	RevokedAt        *time.Time `json:"revoked_at,omitempty"`

	// Relationships
	User *User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

func (s *Session) BeforeCreate(tx *gorm.DB) (err error) {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	return
}
