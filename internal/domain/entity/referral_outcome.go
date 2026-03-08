package entity

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type OutcomeType string

const (
	OutcomeImproved     OutcomeType = "IMPROVED"
	OutcomeDeteriorated OutcomeType = "DETERIORATED"
	OutcomeDeceased     OutcomeType = "DECEASED"
	OutcomeTransferred  OutcomeType = "TRANSFERRED"
	OutcomeDischarged   OutcomeType = "DISCHARGED"
)

type ReferralOutcome struct {
	ID                     uuid.UUID   `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	ReferralID             uuid.UUID   `gorm:"type:uuid;not null;uniqueIndex"`
	Outcome                OutcomeType `gorm:"type:varchar(50);not null"`
	LengthOfStayDays       *int        `gorm:"type:int"`
	WasReferralAppropriate *bool
	OutcomeNotes           *string     `gorm:"type:text"`
	RecordedBy             uuid.UUID   `gorm:"type:uuid;not null;index"`
	RecordedAt             time.Time   `gorm:"default:now()"`

	// Timestamps
	CreatedAt time.Time `gorm:"default:now()"`
	UpdatedAt time.Time `gorm:"default:now()"`

	// Relationships
	Referral Referral `gorm:"foreignKey:ReferralID"`
	Recorder User     `gorm:"foreignKey:RecordedBy"`
}

func (ro *ReferralOutcome) BeforeCreate(tx *gorm.DB) (err error) {
	if ro.ID == uuid.Nil {
		ro.ID = uuid.New()
	}
	return
}
