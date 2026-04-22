package entity

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// StaffReplacementLog tracks in-place replacement updates for hospital staff.
// Replacement keeps the same user_id and updates identity/credential attributes.
type StaffReplacementLog struct {
	ID                uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	HospitalID        uuid.UUID `gorm:"type:uuid;not null;index"`
	UserID            uuid.UUID `gorm:"type:uuid;not null;index"`
	RoleAtReplacement UserRole  `gorm:"type:varchar(50);not null;index"`
	OldEmail          string    `gorm:"type:varchar(255);not null"`
	NewEmail          string    `gorm:"type:varchar(255);not null"`
	Reason            string    `gorm:"type:text;not null"`
	ReplacedByAdminID uuid.UUID `gorm:"type:uuid;not null;index"`
	CreatedAt         time.Time `gorm:"default:now();index"`
}

func (s *StaffReplacementLog) BeforeCreate(tx *gorm.DB) (err error) {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	return
}
