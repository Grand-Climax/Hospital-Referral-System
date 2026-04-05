package dto

type SpecialistRejectRequest struct {
	Reason string `json:"reason" binding:"required" example:"Patient condition not suitable for this department"`
}

type SpecialistActionResponse struct {
	BaseResponse
	Message    string `json:"message" example:"Referral rejected and returned to liaison review"`
	ReferralID string `json:"referral_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	NewStatus  string `json:"new_status" example:"UNDER_LIAISON_REVIEW"`
}
