package entity

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Department struct {
	ID          uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	Name        string    `gorm:"type:varchar(255);not null;unique"`
	Description *string   `gorm:"type:text"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (d *Department) BeforeCreate(tx *gorm.DB) (err error) {
	if d.ID == uuid.Nil {
		d.ID = uuid.New()
	}
	return
}

type HospitalDepartment struct {
	ID                 uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	HospitalID         uuid.UUID `gorm:"type:uuid;not null;index:idx_hosp_dept,unique"`
	DepartmentID       uuid.UUID `gorm:"type:uuid;not null;index:idx_hosp_dept,unique"`
	StandardDailyLimit int       `gorm:"not null;default:20"`
	IsActive           bool      `gorm:"default:true"`
	CreatedAt          time.Time
	UpdatedAt          time.Time

	Hospital   Hospital   `gorm:"foreignKey:HospitalID"`
	Department Department `gorm:"foreignKey:DepartmentID"`
}

func (hd *HospitalDepartment) BeforeCreate(tx *gorm.DB) (err error) {
	if hd.ID == uuid.Nil {
		hd.ID = uuid.New()
	}
	return
}
