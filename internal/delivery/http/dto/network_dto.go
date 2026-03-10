package dto

import "github.com/google/uuid"

type CreateNetworkRouteRequest struct {
	SenderHospitalID      uuid.UUID `json:"sender_hospital_id" binding:"required" example:"62af3d82-52ce-4e8f-af29-2c5e509e1e24"`
	ReceiverHospitalID    uuid.UUID `json:"receiver_hospital_id" binding:"required" example:"0f74f069-d52d-4482-9ba5-41b007fdc1e5"`
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
