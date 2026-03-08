package entity

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type DiagnosisCertainty string

const (
	CertaintyConfirmed DiagnosisCertainty = "CONFIRMED"
	CertaintySuspected DiagnosisCertainty = "SUSPECTED"
	CertaintyRuledOut  DiagnosisCertainty = "RULED_OUT"
)

type ICDCode struct {
	Code        string `gorm:"type:varchar(10);primary_key"`
	Description string `gorm:"type:text;not null"`

	// Timestamps
	CreatedAt time.Time `gorm:"default:now()"`
	UpdatedAt time.Time `gorm:"default:now()"`
}

type ReferralDiagnosis struct {
	ID                 uuid.UUID          `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	ReferralID         uuid.UUID          `gorm:"type:uuid;not null;index;uniqueIndex:idx_referral_icd"`
	ICDCodeValue       string             `gorm:"column:icd_code;type:varchar(10);not null;index;uniqueIndex:idx_referral_icd"`
	IsPrimary          bool               `gorm:"default:true"`
	DiagnosisCertainty DiagnosisCertainty `gorm:"type:varchar(50);not null;default:'CONFIRMED'"`

	// Timestamps
	CreatedAt time.Time `gorm:"default:now()"`
	UpdatedAt time.Time `gorm:"default:now()"`

	// Relationships
	Referral Referral `gorm:"foreignKey:ReferralID"`
	ICDCode  ICDCode  `gorm:"foreignKey:ICDCodeValue;references:Code"`
}

func (rd *ReferralDiagnosis) BeforeCreate(tx *gorm.DB) (err error) {
	if rd.ID == uuid.Nil {
		rd.ID = uuid.New()
	}
	return
}
