package entity

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ClinicalUpdate struct {
	ID             uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	ReferralID     uuid.UUID `gorm:"type:uuid;not null;index" json:"referral_id"`
	UpdatedByID    uuid.UUID `gorm:"column:updated_by;type:uuid;not null" json:"updated_by_id"`
	UpdateReason   string    `gorm:"type:varchar(100);not null" json:"update_reason"` // e.g., 'MISSED_APPOINTMENT_RE_EVALUATION','CONDITION_CHANGE','SPECIALIST_NOTE'
	ClinicalNotes  string    `gorm:"type:text;not null" json:"clinical_notes"`
	CreatedAt      time.Time `gorm:"default:now();index" json:"created_at"`
	RequiresReview bool      `gorm:"default:false" json:"requires_review"`
}

func (cu *ClinicalUpdate) BeforeCreate(tx *gorm.DB) (err error) {
	if cu.ID == uuid.Nil {
		cu.ID = uuid.New()
	}
	return
}
