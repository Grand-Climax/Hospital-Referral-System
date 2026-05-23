package entity

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type MLPrediction struct {
	ID                    uuid.UUID       `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	ReferralID            uuid.UUID       `gorm:"type:uuid;not null;uniqueIndex" json:"referral_id"`
	ExternalPredictionID  *string         `gorm:"type:varchar(128);uniqueIndex" json:"external_prediction_id,omitempty"`
	TriggerReason         string          `gorm:"type:varchar(30);not null;default:'INITIAL'" json:"trigger_reason"`
	InputFeatures         json.RawMessage `gorm:"type:jsonb;not null" json:"input_features" swaggertype:"object"`
	OutputScore           float64         `gorm:"type:numeric(5,2);not null" json:"output_score"`
	SeverityTier          *string         `gorm:"type:varchar(30)" json:"severity_tier,omitempty"`
	ConfidenceLevel       *float64        `gorm:"type:numeric(5,2)" json:"confidence_level,omitempty"`
	Explanation           json.RawMessage `gorm:"type:jsonb" json:"explanation,omitempty" swaggertype:"object"`
	ModelVersion          string          `gorm:"type:varchar(50);not null;index:idx_ml_prediction_model_override" json:"model_version"`
	ProcessingTimeMs      *float64        `gorm:"type:numeric(10,2)" json:"processing_time_ms,omitempty"`
	PredictedAt           time.Time       `gorm:"default:now()" json:"predicted_at"`
	FeedbackSentAt        *time.Time      `json:"feedback_sent_at,omitempty"`
	IsOverridden          bool            `gorm:"default:false;index:idx_ml_prediction_model_override" json:"is_overridden"`
	OverriddenScore       *float64        `gorm:"type:numeric(5,2)" json:"overridden_score,omitempty"`
	OverriddenBy          *uuid.UUID      `gorm:"type:uuid" json:"overridden_by,omitempty"`
	OverrideJustification *string         `gorm:"type:text" json:"override_justification,omitempty"`
	OverriddenAt          *time.Time      `json:"overridden_at,omitempty"`

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
