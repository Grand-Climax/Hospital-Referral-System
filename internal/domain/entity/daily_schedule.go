package entity

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type DailySchedule struct {
	ID             uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	HospitalID     uuid.UUID `gorm:"type:uuid;not null;index:idx_schedule_hosp_dept_date" json:"hospital_id"`
	DeptID         uuid.UUID `gorm:"type:uuid;not null;index:idx_schedule_hosp_dept_date" json:"dept_id"`
	ScheduleDate   time.Time `gorm:"type:date;not null;index:idx_schedule_hosp_dept_date" json:"schedule_date"`
	BookedSlots    int       `gorm:"default:0" json:"booked_slots"`
	MaxSlots       int       `gorm:"not null" json:"max_slots"`
	OverbookLimit  int       `gorm:"default:0" json:"overbook_limit"`
	Version        int       `gorm:"default:1" json:"version"`
}

func (ds *DailySchedule) BeforeCreate(tx *gorm.DB) (err error) {
	if ds.ID == uuid.Nil {
		ds.ID = uuid.New()
	}
	return
}
