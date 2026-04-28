package entity

import (
	"time"

	"github.com/google/uuid"
)

type SystemConfig struct {
	Key       string     `gorm:"primaryKey;type:varchar(100)" json:"key"`
	Value     string     `gorm:"type:text;not null" json:"value"`
	UpdatedAt time.Time  `gorm:"default:now()" json:"updated_at"`
	UpdatedBy *uuid.UUID `gorm:"type:uuid" json:"updated_by,omitempty"`
}
