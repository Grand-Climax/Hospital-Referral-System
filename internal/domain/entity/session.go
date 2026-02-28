package entity

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Session struct {
	ID               uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	UserID           uuid.UUID `gorm:"type:uuid;not null;index"`
	RefreshTokenHash string    `gorm:"type:varchar(255);not null;index"`
	IPAddress        *string   `gorm:"type:varchar(45)"`
	UserAgent        *string   `gorm:"type:text"`
	ExpiresAt        time.Time `gorm:"not null;index"`
	CreatedAt        time.Time `gorm:"default:now()"`
	RevokedAt        *time.Time
}

func (s *Session) BeforeCreate(tx *gorm.DB) (err error) {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	return
}
