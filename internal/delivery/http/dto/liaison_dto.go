package dto

import "github.com/google/uuid"

// LiaisonApproveRequest is the body for POST /liaison/referrals/:id/approve.
type LiaisonApproveRequest struct {
	Comment string `json:"comment,omitempty"`
}

// LiaisonRejectRequest is the body for POST /liaison/referrals/:id/reject.
type LiaisonRejectRequest struct {
	Reason string `json:"reason" binding:"required" example:"Incomplete clinical summary"`
}

// LiaisonForwardRequest is the body for POST /liaison/referrals/:id/forward.
type LiaisonForwardRequest struct {
	TargetHospitalID uuid.UUID `json:"target_hospital_id" binding:"required" example:"0f74f069-d52d-4482-9ba5-41b007fdc1e5"`
	Comment          string    `json:"comment,omitempty"`
}

// LiaisonActionResponse is the uniform response for all liaison actions.
type LiaisonActionResponse struct {
	Message    string `json:"message"`
	ReferralID string `json:"referral_id"`
	NewStatus  string `json:"new_status"`
}
