package dto

import (
	"time"

	"github.com/google/uuid"
)

type AddClinicalUpdateRequest struct {
	UpdateReason  string `json:"update_reason" binding:"required,oneof=MISSED_APPOINTMENT_RE_EVALUATION CONDITION_CHANGE SPECIALIST_NOTE"`
	ClinicalNotes string `json:"clinical_notes" binding:"required"`
}

type ClinicalUpdateResponse struct {
	ID             uuid.UUID `json:"id"`
	ReferralID     uuid.UUID `json:"referral_id"`
	UpdatedByID    uuid.UUID `json:"updated_by_id"`
	UpdateReason   string    `json:"update_reason"`
	ClinicalNotes  string    `json:"clinical_notes"`
	RequiresReview bool      `json:"requires_review"`
	CreatedAt      time.Time `json:"created_at"`
}

type RecordOutcomeRequest struct {
	Outcome                string `json:"outcome" binding:"required,oneof=improved deteriorated deceased transferred discharged"`
	LengthOfStayDays       *int   `json:"length_of_stay_days,omitempty"`
	WasReferralAppropriate *bool  `json:"was_referral_appropriate,omitempty"`
	OutcomeNotes           string `json:"outcome_notes,omitempty"`
}

type ReferralOutcomeResponse struct {
	ID                     uuid.UUID `json:"id"`
	ReferralID             uuid.UUID `json:"referral_id"`
	Outcome                string    `json:"outcome"`
	LengthOfStayDays       *int      `json:"length_of_stay_days,omitempty"`
	WasReferralAppropriate *bool     `json:"was_referral_appropriate,omitempty"`
	OutcomeNotes           string    `json:"outcome_notes,omitempty"`
	RecordedByID           uuid.UUID `json:"recorded_by_id"`
	RecordedAt             time.Time `json:"recorded_at"`
}
