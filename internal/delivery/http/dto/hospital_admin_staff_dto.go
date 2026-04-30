package dto

import "Hospital-Referral-System/internal/domain/entity"

type HospitalAdminCreateStaffRequest struct {
	Email        string          `json:"email" binding:"required,email" example:"staff@hospital.et"`
	Password     string          `json:"password" binding:"required,min=8" example:"password123"`
	FirstName    string          `json:"first_name" binding:"required" example:"Abebe"`
	MiddleName   string          `json:"middle_name" binding:"required" example:"Kebede"`
	LastName     string          `json:"last_name" binding:"required" example:"Molla"`
	NationalID   string          `json:"national_id" example:"ETH-0001"`
	Role         entity.UserRole `json:"role" binding:"required" example:"LIAISON_OFFICER"`
	DepartmentID *string         `json:"department_id" example:"dfc2b777-a5d5-424b-911a-976b2e8d8614"`
}

type HospitalAdminChangeRoleRequest struct {
	Role entity.UserRole `json:"role" binding:"required" example:"RECEIVING_SPECIALIST"`
}

type HospitalAdminReplaceStaffRequest struct {
	FirstName  string `json:"first_name" binding:"required" example:"Abebe"`
	MiddleName string `json:"middle_name" binding:"required" example:"Kebede"`
	LastName   string `json:"last_name" binding:"required" example:"Molla"`
	Email      string `json:"email" binding:"required,email" example:"replacement@hospital.et"`
	Password   string `json:"password" binding:"required,min=8" example:"newStrongPass123"`
	Reason     string `json:"reason" binding:"required,min=5" example:"Staff transition and account continuity"`
}

type HospitalAdminSetStaffActiveRequest struct {
	IsActive *bool `json:"is_active" binding:"required" example:"false"`
}

type HospitalAdminReassignDepartmentRequest struct {
	DepartmentID *string `json:"department_id" example:"dfc2b777-a5d5-424b-911a-976b2e8d8614"`
}

type HospitalAdminSessionResponse struct {
	ID        string  `json:"id"`
	UserID    string  `json:"user_id"`
	IPAddress *string `json:"ip_address,omitempty"`
	UserAgent *string `json:"user_agent,omitempty"`
	ExpiresAt string  `json:"expires_at"`
	CreatedAt string  `json:"created_at"`
}

type HospitalAdminSessionListResponse struct {
	Data  []HospitalAdminSessionResponse `json:"data"`
	Total int64                          `json:"total"`
	Page  int                            `json:"page"`
	BaseResponse
}

type HospitalAdminForceLogoutResponse struct {
	RevokedSessions int64 `json:"revoked_sessions"`
	BaseResponse
}
