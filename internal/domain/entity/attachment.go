package entity

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type FileType string

const (
	FileTypePDF   FileType = "PDF"
	FileTypeImage FileType = "IMAGE"
	FileTypeDICOM FileType = "DICOM"
)

type Attachment struct {
	ID          uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	ReferralID  uuid.UUID `gorm:"type:uuid;not null;index"`
	FileName    string    `gorm:"type:varchar(255);not null"`
	FileType    FileType  `gorm:"type:varchar(50);not null"`
	StoragePath string    `gorm:"type:text;not null"`
	UploadedAt  time.Time `gorm:"default:now()"`

	// Timestamps
	CreatedAt time.Time `gorm:"default:now()"`
	UpdatedAt time.Time `gorm:"default:now()"`

	// Relationship
	Referral Referral `gorm:"foreignKey:ReferralID"`
}

func (a *Attachment) BeforeCreate(tx *gorm.DB) (err error) {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	return
}
