package entity

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Patient struct {
	ID             uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	NationalIDEnc  *string    `gorm:"type:text" json:"-"`
	NationalIDHash *string    `gorm:"type:varchar(64);uniqueIndex" json:"-"`
	PhoneNumber    *string    `gorm:"type:varchar(20)" json:"phone_number,omitempty"`
	FirstName      string     `gorm:"type:varchar(100);not null" json:"first_name"`
	MiddleName     string     `gorm:"type:varchar(100);not null" json:"middle_name"`
	LastName       string     `gorm:"type:varchar(100);not null" json:"last_name"`
	Sex            string     `gorm:"type:varchar(10);not null" json:"sex"`
	DateOfBirth    *time.Time `gorm:"type:date" json:"date_of_birth,omitempty"`
	HomeRegion     *string    `gorm:"type:varchar(100);index" json:"home_region,omitempty"`
	IsDeleted      bool       `gorm:"default:false;index" json:"-"`
	DeletedAt      gorm.DeletedAt `json:"-"`
}

func (p *Patient) BeforeCreate(tx *gorm.DB) (err error) {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return
}
