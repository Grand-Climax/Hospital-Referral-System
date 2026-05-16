package entity

import (
	"time"
)

type JobCheckpoint struct {
	JobName   string     `gorm:"type:varchar(100);primaryKey" json:"job_name"`
	LastRunAt *time.Time `json:"last_run_at,omitempty"`
}
