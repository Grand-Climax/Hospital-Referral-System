package entity

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ReasonForReferralCategory string

const (
	ReasonBetterInvestigation ReasonForReferralCategory = "BETTER_INVESTIGATION"
	ReasonHigherTreatment     ReasonForReferralCategory = "HIGHER_TREATMENT"
	ReasonSpecialistOpinion   ReasonForReferralCategory = "SPECIALIST_OPINION"
	ReasonBedUnavailability   ReasonForReferralCategory = "BED_UNAVAILABILITY"
)

type ConditionAtReferral string

const (
	ConditionStable   ConditionAtReferral = "STABLE"
	ConditionUnstable ConditionAtReferral = "UNSTABLE"
	ConditionCritical ConditionAtReferral = "CRITICAL"
)

type ModeOfTransport string

const (
	TransportAmbulance ModeOfTransport = "AMBULANCE"
	TransportPrivate   ModeOfTransport = "PRIVATE"
	TransportPublic    ModeOfTransport = "PUBLIC"
)

type ReferralForm struct {
	ID         uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	ReferralID uuid.UUID `gorm:"type:uuid;not null;uniqueIndex"`

	// What the referring doctor observed and documented
	ClinicalSummary            string  `gorm:"type:text;not null"`
	PatientHistory             string  `gorm:"type:text;not null"`
	PhysicalExaminationFindings *string `gorm:"type:text"`

	// What was already done at sender hospital
	InvestigationResults          *string `gorm:"type:text"`
	TreatmentGivenBeforeReferral  *string `gorm:"type:text"`
	MedicationOnTransfer          *string `gorm:"type:text"`

	// Why referring
	ReasonOfReferral         string                    `gorm:"type:text;not null"`
	ReasonForReferralCategory ReasonForReferralCategory `gorm:"type:varchar(50);not null"`

	// Patient condition and transport
	ConditionAtReferral ConditionAtReferral `gorm:"type:varchar(50);not null"`
	ModeOfTransport     *ModeOfTransport    `gorm:"type:varchar(50)"`

	// Accompanying person
	AccompanyingPersonName  *string `gorm:"type:varchar(100)"`
	AccompanyingPersonPhone *string `gorm:"type:varchar(20)"`

	// Timestamps
	CreatedAt time.Time `gorm:"default:now()"`
	UpdatedAt time.Time `gorm:"default:now()"`

	// Relationship
	Referral Referral `gorm:"foreignKey:ReferralID"`
}

func (rf *ReferralForm) BeforeCreate(tx *gorm.DB) (err error) {
	if rf.ID == uuid.Nil {
		rf.ID = uuid.New()
	}
	return
}
