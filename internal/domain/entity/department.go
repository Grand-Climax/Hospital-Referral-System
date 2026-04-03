package entity

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Department struct {
	ID          uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	Name        string    `gorm:"type:varchar(255);not null;unique" json:"name"`
	Description *string   `gorm:"type:text" json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (d *Department) BeforeCreate(tx *gorm.DB) (err error) {
	if d.ID == uuid.Nil {
		d.ID = uuid.New()
	}
	return
}

type HospitalDepartment struct {
	ID                 uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	HospitalID         uuid.UUID `gorm:"type:uuid;not null;index:idx_hosp_dept,unique" json:"hospital_id"`
	DepartmentID       uuid.UUID `gorm:"type:uuid;not null;index:idx_hosp_dept,unique" json:"department_id"`
	StandardDailyLimit int       `gorm:"not null;default:20" json:"standard_daily_limit"`
	IsActive           bool      `gorm:"default:true" json:"is_active"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`

	Hospital   Hospital   `gorm:"foreignKey:HospitalID" json:"hospital,omitempty"`
	Department Department `gorm:"foreignKey:DepartmentID" json:"department,omitempty"`
}

func (hd *HospitalDepartment) BeforeCreate(tx *gorm.DB) (err error) {
	if hd.ID == uuid.Nil {
		hd.ID = uuid.New()
	}
	return
}
