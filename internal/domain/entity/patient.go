package entity

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Patient struct {
	ID             uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	NationalIDEnc  *string    `gorm:"type:text"`
	NationalIDHash *string    `gorm:"type:varchar(64);uniqueIndex"`
	PhoneNumber    *string    `gorm:"type:varchar(20)"`
	FirstName      string     `gorm:"type:varchar(100);not null"`
	MiddleName     string     `gorm:"type:varchar(100);not null"`
	LastName       string     `gorm:"type:varchar(100);not null"`
	Sex            string     `gorm:"type:varchar(10);not null"`
	DateOfBirth    *time.Time `gorm:"type:date"`
	HomeRegion     *string    `gorm:"type:varchar(100);index"`
	IsDeleted      bool       `gorm:"default:false;index"`
	DeletedAt      gorm.DeletedAt
}

func (p *Patient) BeforeCreate(tx *gorm.DB) (err error) {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return
}
