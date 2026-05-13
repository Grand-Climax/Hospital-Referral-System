package entity

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type MLPrediction struct {
	ID                    uuid.UUID       `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	ReferralID            uuid.UUID       `gorm:"type:uuid;not null;index:idx_ml_prediction_referral_active" json:"referral_id"`
	TriggerReason         string          `gorm:"type:varchar(30);not null;default:'INITIAL'" json:"trigger_reason"`
	InputFeatures         json.RawMessage `gorm:"type:jsonb;not null" json:"input_features"`
	OutputScore           float64         `gorm:"type:numeric(5,2);not null" json:"output_score"`
	ConfidenceLevel       *float64        `gorm:"type:numeric(5,2)" json:"confidence_level,omitempty"`
	Explanation           json.RawMessage `gorm:"type:jsonb" json:"explanation,omitempty"`
	ModelVersion          string          `gorm:"type:varchar(50);not null;index:idx_ml_prediction_model_override" json:"model_version"`
	PredictedAt           time.Time       `gorm:"default:now()" json:"predicted_at"`
	IsOverridden          bool            `gorm:"default:false;index:idx_ml_prediction_model_override" json:"is_overridden"`
	OverriddenScore       *float64        `gorm:"type:numeric(5,2)" json:"overridden_score,omitempty"`
	OverriddenBy          *uuid.UUID      `gorm:"type:uuid" json:"overridden_by,omitempty"`
	OverrideJustification *string         `gorm:"type:text" json:"override_justification,omitempty"`
	OverriddenAt          *time.Time      `json:"overridden_at,omitempty"`
	IsActive              bool            `gorm:"default:true;index:idx_ml_prediction_referral_active" json:"is_active"`

	// Relationships
	Referral         *Referral `gorm:"foreignKey:ReferralID" json:"referral,omitempty"`
	OverriddenByUser *User     `gorm:"foreignKey:OverriddenBy" json:"overridden_by_user,omitempty"`
}

func (m *MLPrediction) BeforeCreate(tx *gorm.DB) (err error) {
	if m.ID == uuid.Nil {
		m.ID = uuid.New()
	}
	return
}
