package entity

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// DailySchedule is an immutable history log created on the first booking for a
// given (hospital, department, date) tuple under the "Schedule-on-Demand" model.
// After creation, MaxSlots and OverbookLimit are frozen; only BookedSlots is
// updated as a snapshot after each booking. Live capacity decisions are made
// from TriageQueue + HospitalDepartment + CapacityOverride, NOT from this row.
type DailySchedule struct {
	ID             uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	HospitalID     uuid.UUID `gorm:"type:uuid;not null;index:idx_schedule_hosp_dept_date,priority:1" json:"hospital_id"`
	DepartmentID   uuid.UUID `gorm:"type:uuid;not null;index:idx_schedule_hosp_dept_date,priority:2" json:"department_id"`
	ScheduleDate   time.Time `gorm:"type:date;not null;index:idx_schedule_hosp_dept_date,priority:3" json:"schedule_date"`
	BookedSlots    int       `gorm:"default:0" json:"booked_slots"`
	MaxSlots       int       `gorm:"not null" json:"max_slots"`
	OverbookLimit  int       `gorm:"default:0" json:"overbook_limit"`

	Hospital   *Hospital   `gorm:"foreignKey:HospitalID" json:"hospital,omitempty"`
	Department *Department `gorm:"foreignKey:DepartmentID" json:"department,omitempty"`
}

func (ds *DailySchedule) BeforeCreate(tx *gorm.DB) (err error) {
	if ds.ID == uuid.Nil {
		ds.ID = uuid.New()
	}
	return
}
