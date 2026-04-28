package entity

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type SchedulerCheckpoint struct {
	ID               uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	HospitalID       uuid.UUID  `gorm:"type:uuid;not null;uniqueIndex:idx_checkpoint_hosp_dept" json:"hospital_id"`
	DeptID           uuid.UUID  `gorm:"type:uuid;uniqueIndex:idx_checkpoint_hosp_dept" json:"dept_id"`
	LastProcessedAt  *time.Time `json:"last_processed_at,omitempty"`
	LeaseHolder      *string    `gorm:"type:varchar(255)" json:"lease_holder,omitempty"`
	LeaseExpiresAt   *time.Time `json:"lease_expires_at,omitempty"`
}

func (sc *SchedulerCheckpoint) BeforeCreate(tx *gorm.DB) (err error) {
	if sc.ID == uuid.Nil {
		sc.ID = uuid.New()
	}
	return
}
