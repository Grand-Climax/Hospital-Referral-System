package entity

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ReferralOutcome struct {
	ID                     uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	ReferralID             uuid.UUID `gorm:"type:uuid;not null;unique;index" json:"referral_id"`
	Outcome                string    `gorm:"type:varchar(50);not null" json:"outcome"` // e.g., 'improved','deteriorated','deceased','transferred','discharged'
	LengthOfStayDays       *int      `json:"length_of_stay_days,omitempty"`
	WasReferralAppropriate *bool     `json:"was_referral_appropriate,omitempty"`
	OutcomeNotes           *string   `gorm:"type:text" json:"outcome_notes,omitempty"`
	RecordedByID           uuid.UUID `gorm:"type:uuid;not null" json:"recorded_by_id"`
	RecordedAt             time.Time `gorm:"default:now()" json:"recorded_at"`
}

func (ro *ReferralOutcome) BeforeCreate(tx *gorm.DB) (err error) {
	if ro.ID == uuid.Nil {
		ro.ID = uuid.New()
	}
	return
}
