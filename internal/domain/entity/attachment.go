package entity

import (
	"time"

	"github.com/google/uuid"
)

type Attachment struct {
	ID          uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	ReferralID  uuid.UUID `gorm:"type:uuid;not null;index"`
	FileName    string    `gorm:"type:varchar(255);not null"`
	FileType    string    `gorm:"type:varchar(50);not null"`
	StoragePath string    `gorm:"type:text;not null"`
	UploadedAt  time.Time `gorm:"autoCreateTime"`

	// Relationships
	Referral *Referral `gorm:"foreignKey:ReferralID"`
}
