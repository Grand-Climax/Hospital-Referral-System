package dto

import "github.com/google/uuid"


type DoctorDashboardStats struct {
	TotalReferrals int64 `json:"total_referrals"`
	Pending        int64 `json:"pending"`
	Accepted       int64 `json:"accepted"`
	Critical       int64 `json:"critical"`
}

type DoctorDashboardStatsResponse struct {
	DoctorDashboardStats
	BaseResponse
}

type LatestPendingReferralsResponse struct {
	Data []ListReferralResponse `json:"data"`
	BaseResponse
}

type GrantConsultRequest struct {
	DoctorID uuid.UUID `json:"doctor_id" binding:"required"`
}

type RevokeConsultRequest struct {
	DoctorID uuid.UUID `json:"doctor_id" binding:"required"`
	Reason   string    `json:"reason"`
}

type AssignedReferralResponse struct {
	ListReferralResponse
	AccessType      string  `json:"access_type"`
	AccessGrantedAt string  `json:"access_granted_at"`
	AccessRevokedAt *string `json:"access_revoked_at,omitempty"`
}

type AssignedReferralListResponse struct {
	Data []AssignedReferralResponse `json:"data"`
	BaseResponse
}
