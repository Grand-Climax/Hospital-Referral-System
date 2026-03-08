package entity

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Sex string

const (
	SexMale    Sex = "MALE"
	SexFemale  Sex = "FEMALE"
	SexUnknown Sex = "UNKNOWN"
)

type PreferredLanguage string

const (
	LangEnglish    PreferredLanguage = "ENGLISH"
	LangAmharic    PreferredLanguage = "AMHARIC"
	LangAfaanOromo PreferredLanguage = "AFAAN_OROMO"
)

type Patient struct {
	ID              uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	NationalIDEnc   *string   `gorm:"type:text"`
	NationalIDHash  *string   `gorm:"type:varchar(64);uniqueIndex"`
	PhoneNumber     *string   `gorm:"type:varchar(20)"`
	FirstName       string    `gorm:"type:varchar(100);not null"`
	MiddleName      string    `gorm:"type:varchar(100);not null"`
	LastName        string    `gorm:"type:varchar(100);not null"`
	Sex             Sex       `gorm:"type:varchar(10);not null"`
	DateOfBirth     *time.Time `gorm:"type:date"`
	HomeRegion      *string           `gorm:"type:varchar(100);index"`
	PreferredLang   PreferredLanguage `gorm:"type:varchar(20);not null;default:'AMHARIC'"`
	IsDeleted       bool      `gorm:"default:false;index"`
	DeletedAt       gorm.DeletedAt
	CreatedAt       time.Time
	UpdatedAt       time.Time

	// Relationships (one-to-many)
	Referrals []Referral `gorm:"foreignKey:PatientID"`
}

func (p *Patient) BeforeCreate(tx *gorm.DB) (err error) {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return
}
