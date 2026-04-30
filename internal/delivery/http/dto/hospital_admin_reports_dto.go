package dto

import "Hospital-Referral-System/internal/domain/entity"

type AuditLogResponse struct {
	ID         string            `json:"id"`
	UserID     string            `json:"user_id"`
	ReferralID *string           `json:"referral_id,omitempty"`
	ActionType entity.ActionType `json:"action_type"`
	Resource   *string           `json:"resource,omitempty"`
	ResourceID *string           `json:"resource_id,omitempty"`
	IPAddress  *string           `json:"ip_address,omitempty"`
	UserAgent  *string           `json:"user_agent,omitempty"`
	Timestamp  string            `json:"timestamp"`
}

type AuditLogListResponse struct {
	Data  []AuditLogResponse `json:"data"`
	Total int64              `json:"total"`
	Page  int                `json:"page"`
	BaseResponse
}

type MonthlyReferralTotalResponse struct {
	Month string `json:"month"`
	Count int64  `json:"count"`
}

type MonthlyReferralTotalsResponse struct {
	Data []MonthlyReferralTotalResponse `json:"data"`
	BaseResponse
}

type AcceptanceRejectionRateResponse struct {
	AcceptanceRate float64 `json:"acceptance_rate"`
	RejectionRate  float64 `json:"rejection_rate"`
	BaseResponse
}

type MissedAppointmentRateResponse struct {
	MissedAppointmentRate float64 `json:"missed_appointment_rate"`
	BaseResponse
}

type DepartmentLoadResponse struct {
	DepartmentID string `json:"department_id"`
	Count        int64  `json:"count"`
}

type DepartmentLoadListResponse struct {
	Data []DepartmentLoadResponse `json:"data"`
	BaseResponse
}

type AverageWaitTimeResponse struct {
	AverageWaitTime float64 `json:"average_wait_time"`
	BaseResponse
}

type TopReferringHospitalResponse struct {
	HospitalID   string `json:"hospital_id"`
	HospitalName string `json:"hospital_name"`
	Count        int64  `json:"count"`
}

type TopReferringHospitalListResponse struct {
	Data []TopReferringHospitalResponse `json:"data"`
	BaseResponse
}
