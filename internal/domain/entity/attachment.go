package entity

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

const (
	VerificationPending  = "PENDING"
	VerificationVerified = "VERIFIED"
	VerificationRejected = "REJECTED"
)

type Attachment struct {
	ID                 uuid.UUID         `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	ReferralID         uuid.UUID         `gorm:"type:uuid;not null;index" json:"referral_id"`
	FileName           string            `gorm:"type:varchar(255);default:''" json:"file_name"`
	FileType           string            `gorm:"type:varchar(50);default:''" json:"file_type"`
	FileSize           int64             `gorm:"default:0" json:"file_size"`
	Category           string            `gorm:"type:varchar(100);default:'GENERAL_CLINICAL'" json:"category"`
	StoragePath        string            `gorm:"type:text;default:''" json:"storage_path"`
	PublicID           string            `gorm:"type:varchar(255)" json:"-"`
	Metadata           datatypes.JSONMap `gorm:"type:jsonb" json:"metadata,omitempty" swaggertype:"object"`
	VerificationStatus string            `gorm:"type:varchar(20);default:'PENDING';index" json:"verification" example:"VERIFIED"`
	RejectionReason    string            `gorm:"type:text;default:''" json:"rejection_message,omitempty" example:"Metadata extraction failed"`
	RejectedAt         *time.Time        `gorm:"type:timestamp" json:"rejected_at,omitempty" example:"2026-04-11T19:55:00Z"`
	UploadedAt         time.Time         `gorm:"autoCreateTime" json:"uploaded_at"`

	// Relationships
	Referral *Referral `gorm:"foreignKey:ReferralID" json:"referral,omitempty"`
}
