package dto

import (
	"time"

	"github.com/google/uuid"
)

type ClinicalUpdateRequest struct {
	Summary        string `json:"summary" binding:"required"`
	RequiresReview bool   `json:"requires_review"`
}

type ClinicalUpdateResponse struct {
	ID             uuid.UUID `json:"id"`
	ReferralID      uuid.UUID `json:"referral_id"`
	UpdatedByID     uuid.UUID `json:"updated_by_id"`
	Summary        string    `json:"summary"`
	RequiresReview bool      `json:"requires_review"`
	CreatedAt      time.Time `json:"created_at"`
}

type ReferralOutcomeRequest struct {
	OutcomeType    string `json:"outcome_type" binding:"required"` // "DISCHARGED", "ADMITTED", "EXPIRED", "TRANSFERRED"
	FinalSummary   string `json:"final_summary"`
	FollowUpNeeded bool   `json:"follow_up_needed"`
}

type ReferralOutcomeResponse struct {
	ID             uuid.UUID `json:"id"`
	ReferralID      uuid.UUID `json:"referral_id"`
	OutcomeType    string    `json:"outcome_type"`
	FinalSummary   string    `json:"final_summary"`
	FollowUpNeeded bool      `json:"follow_up_needed"`
	CreatedAt      time.Time `json:"created_at"`
}
