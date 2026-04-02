package dto

import "github.com/google/uuid"

type CreateNetworkRouteRequest struct {
	SenderHospitalID      uuid.UUID `json:"sender_hospital_id" binding:"required"`
	ReceiverHospitalID    uuid.UUID `json:"receiver_hospital_id" binding:"required"`
	ReferralType          string    `json:"referral_type" binding:"omitempty" example:"INPATIENT"`
	RequiresAdminApproval bool      `json:"requires_admin_approval" example:"true"`
}

type NetworkRouteResponse struct {
	ID                    uuid.UUID `json:"id"`
	SenderHospitalID      uuid.UUID `json:"sender_hospital_id"`
	ReceiverHospitalID    uuid.UUID `json:"receiver_hospital_id"`
	ReferralType          string    `json:"referral_type"`
	RequiresAdminApproval bool      `json:"requires_admin_approval"`
}
