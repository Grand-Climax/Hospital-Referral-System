package dto

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
