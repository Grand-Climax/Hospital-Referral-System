package entity

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type HospitalTier string

const (
	PrimaryHosp     HospitalTier = "PRIMARY"
	SecondaryHosp   HospitalTier = "SECONDARY"
	SpecializedHosp HospitalTier = "SPECIALIZED"
	TertiaryHosp    HospitalTier = "TERTIARY"
)

type Hospital struct {
	ID           uuid.UUID    `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	Name         string       `gorm:"type:varchar(255);not null" json:"name"`
	TierLevel    HospitalTier `gorm:"type:varchar(50);not null" json:"tier_level"`
	Region       string       `gorm:"type:varchar(100);not null;index" json:"region"`
	Address      *string      `gorm:"type:text" json:"address,omitempty"`
	ContactPhone *string      `gorm:"type:varchar(20)" json:"contact_phone,omitempty"`
	IsActive     bool         `gorm:"default:true" json:"is_active"`
	IsDeleted    bool         `gorm:"default:false;index" json:"-"`
	DeletedAt    gorm.DeletedAt `json:"-"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
}

func (h *Hospital) BeforeCreate(tx *gorm.DB) (err error) {
	if h.ID == uuid.Nil {
		h.ID = uuid.New()
	}
	return
}
