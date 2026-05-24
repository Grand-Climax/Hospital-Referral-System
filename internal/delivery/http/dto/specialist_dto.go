package dto

import (
	"Hospital-Referral-System/internal/domain/entity"
)

type SpecialistRejectRequest struct {
	Reason string `json:"reason" binding:"required" example:"Patient condition not suitable for this department"`
}

type SpecialistActionResponse struct {
	BaseResponse
	Message    string `json:"message" example:"Referral rejected and returned to liaison review"`
	ReferralID string `json:"referral_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	NewStatus  string `json:"new_status" example:"UNDER_LIAISON_REVIEW"`
}

type MLPredictionResponse struct {
	Success bool                 `json:"success" example:"true"`
	Message string               `json:"message" example:"ML prediction retrieved successfully"`
	Data    *entity.MLPrediction `json:"data"`
}

type MLSeverityOverrideRequest struct {
	Score         float64 `json:"score" binding:"required,min=0,max=100" example:"85.0"`
	Justification string  `json:"justification" binding:"required" example:"Patient risk factors not fully captured by automatic vitals scoring"`
}
