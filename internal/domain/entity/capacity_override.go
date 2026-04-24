package entity

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type CapacityOverride struct {
	ID         uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	HospitalID uuid.UUID `gorm:"type:uuid;not null;index:idx_override_hosp_dept_date" json:"hospital_id"`
	DeptID     uuid.UUID `gorm:"type:uuid;not null;index:idx_override_hosp_dept_date" json:"dept_id"`
	TargetDate time.Time `gorm:"type:date;not null;index:idx_override_hosp_dept_date" json:"target_date"`
	NewLimit   int       `gorm:"not null" json:"new_limit"`
	Reason     *string   `gorm:"type:text" json:"reason,omitempty"`
	SetByID    uuid.UUID `gorm:"type:uuid;not null" json:"set_by_id"`
	IsActive   bool      `gorm:"default:true" json:"is_active"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func (co *CapacityOverride) BeforeCreate(tx *gorm.DB) (err error) {
	if co.ID == uuid.Nil {
		co.ID = uuid.New()
	}
	return
}
