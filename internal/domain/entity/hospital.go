package entity

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type HospitalTier string

const (
	PrimaryHosp     HospitalTier = "PRIMARY"
	GeneralHosp     HospitalTier = "GENERAL"
	SpecializedHosp HospitalTier = "SPECIALIZED"
	TertiaryHosp    HospitalTier = "TERTIARY"
)

type Hospital struct {
	ID           uuid.UUID    `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	Name         string       `gorm:"type:varchar(255);not null"`
	TierLevel    HospitalTier `gorm:"type:varchar(50);not null"`
	Region       string       `gorm:"type:varchar(100);not null;index"`
	Address      *string      `gorm:"type:text"`
	ContactPhone *string      `gorm:"type:varchar(20)"`
	IsActive     bool         `gorm:"default:true"`
	IsDeleted    bool         `gorm:"default:false;index"`
	DeletedAt    gorm.DeletedAt
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (h *Hospital) BeforeCreate(tx *gorm.DB) (err error) {
	if h.ID == uuid.Nil {
		h.ID = uuid.New()
	}
	return
}
