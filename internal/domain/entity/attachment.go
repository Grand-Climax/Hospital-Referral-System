package entity

import (
	"time"

	"github.com/google/uuid"
)

type Attachment struct {
	ID          uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	ReferralID  uuid.UUID `gorm:"type:uuid;not null;index" json:"referral_id"`
	FileName    string    `gorm:"type:varchar(255);not null" json:"file_name"`
	FileType    string    `gorm:"type:varchar(50);not null" json:"file_type"`
	StoragePath string    `gorm:"type:text;not null" json:"storage_path"`
	UploadedAt  time.Time `gorm:"autoCreateTime" json:"uploaded_at"`

	// Relationships
	Referral *Referral `gorm:"foreignKey:ReferralID" json:"referral,omitempty"`
}
