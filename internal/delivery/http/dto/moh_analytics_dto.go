package dto

type MohDashboardSummaryResponse struct {
	TotalReferrals        int64   `json:"total_referrals"`
	TotalAccepted         int64   `json:"total_accepted"`
	TotalRejected         int64   `json:"total_rejected"`
	TotalAdmitted         int64   `json:"total_admitted"`
	AcceptanceRate        float64 `json:"acceptance_rate_percentage"`
	AverageMLSeverity     float64 `json:"average_ml_severity_score"`
	AverageTurnaroundHour float64 `json:"average_turnaround_hours"`
	BaseResponse
}

type MohReferralTrendPointResponse struct {
	Period             string `json:"period"`
	TotalReferrals     int64  `json:"total_referrals"`
	AcceptedReferrals  int64  `json:"accepted_referrals"`
	RejectedReferrals  int64  `json:"rejected_referrals"`
	EmergencyReferrals int64  `json:"emergency_referrals"`
}

type MohReferralTrendsResponse struct {
	Data []MohReferralTrendPointResponse `json:"data"`
	BaseResponse
}

type MohHospitalLoadMetricResponse struct {
	HospitalID             string  `json:"hospital_id"`
	HospitalName           string  `json:"hospital_name"`
	TierLevel              string  `json:"tier_level"`
	Region                 string  `json:"region"`
	TotalReferralsReceived int64   `json:"total_referrals_received"`
	TotalAccepted          int64   `json:"total_accepted"`
	TotalRejected          int64   `json:"total_rejected"`
	RejectionRate          float64 `json:"rejection_rate_percentage"`
	AverageSeverityScore   float64 `json:"average_severity_score"`
}

type MohHospitalLoadResponse struct {
	Data []MohHospitalLoadMetricResponse `json:"data"`
	BaseResponse
}

type MohDiseaseHotspotResponseItem struct {
	Region               string  `json:"region"`
	DepartmentName       string  `json:"department_name"`
	ReferralCount        int64   `json:"referral_count"`
	AverageSeverityScore float64 `json:"average_severity_score"`
}

type MohDiseaseHotspotsResponse struct {
	Data []MohDiseaseHotspotResponseItem `json:"data"`
	BaseResponse
}

type MohSeverityDistributionResponseItem struct {
	Region         string `json:"region"`
	CriticalCount  int64  `json:"critical_count"`
	UrgentCount    int64  `json:"urgent_count"`
	RoutineCount   int64  `json:"routine_count"`
	TotalReferrals int64  `json:"total_referrals"`
}

type MohSeverityDistributionResponse struct {
	Data []MohSeverityDistributionResponseItem `json:"data"`
	BaseResponse
}

type MohAnalyticsExportResponse struct {
	Summary      MohDashboardSummaryResponse     `json:"summary"`
	HospitalLoad []MohHospitalLoadMetricResponse `json:"hospital_load"`
	BaseResponse
}
