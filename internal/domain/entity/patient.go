package entity

import (
	"time"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"Hospital-Referral-System/internal/infrastructure/crypto"
)

type Patient struct {
	ID             uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	NationalIDEnc  *string    `gorm:"type:text" json:"-"`
	NationalIDHash *string    `gorm:"type:varchar(64);uniqueIndex" json:"-"`
	PhoneNumberEnc *string    `gorm:"type:text" json:"-"`
	PhoneHash      *string    `gorm:"type:varchar(64);index" json:"-"`
	FirstNameEnc   string     `gorm:"type:text;not null;default:''" json:"-"`
	MiddleNameEnc  string     `gorm:"type:text;not null;default:''" json:"-"`
	LastNameEnc    string     `gorm:"type:text;not null;default:''" json:"-"`
	Sex            string     `gorm:"type:varchar(10);not null" json:"sex"`
	DateOfBirth    *time.Time `gorm:"type:date" json:"date_of_birth,omitempty"`
	HomeRegion     *EthiopianRegion `gorm:"type:varchar(100);index" json:"home_region,omitempty"`
	AllowSMS       bool       `gorm:"default:true" json:"allow_sms"`
	IsDeleted      bool       `gorm:"default:false;index" json:"-"`
	DeletedAt      gorm.DeletedAt `json:"-"`

	// Transient fields for plaintext data
	FirstNamePlain  string `gorm:"-" json:"first_name"`
	MiddleNamePlain string `gorm:"-" json:"middle_name"`
	LastNamePlain   string `gorm:"-" json:"last_name"`
	NationalIDPlain string `gorm:"-" json:"national_id,omitempty"`
	PhonePlain      string `gorm:"-" json:"phone_number,omitempty"`
}

func (p *Patient) BeforeCreate(tx *gorm.DB) (err error) {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return
}

// DecryptFields decrypts the encrypted fields and populates the transient plain fields.
func (p *Patient) DecryptFields(svc *crypto.PatientCryptoService) error {
	if svc == nil {
		return fmt.Errorf("crypto service is nil")
	}

	if p.FirstNameEnc != "" {
		dec, err := svc.Decrypt(p.FirstNameEnc)
		if err != nil {
			return err
		}
		p.FirstNamePlain = string(dec)
	}

	if p.MiddleNameEnc != "" {
		dec, err := svc.Decrypt(p.MiddleNameEnc)
		if err != nil {
			return err
		}
		p.MiddleNamePlain = string(dec)
	}

	if p.LastNameEnc != "" {
		dec, err := svc.Decrypt(p.LastNameEnc)
		if err != nil {
			return err
		}
		p.LastNamePlain = string(dec)
	}

	if p.PhoneNumberEnc != nil && *p.PhoneNumberEnc != "" {
		dec, err := svc.Decrypt(*p.PhoneNumberEnc)
		if err != nil {
			return err
		}
		p.PhonePlain = string(dec)
	}

	if p.NationalIDEnc != nil && *p.NationalIDEnc != "" {
		dec, err := svc.Decrypt(*p.NationalIDEnc)
		if err != nil {
			return err
		}
		p.NationalIDPlain = string(dec)
	}

	return nil
}
