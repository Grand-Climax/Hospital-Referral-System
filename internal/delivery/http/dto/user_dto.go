package dto

import "Hospital-Referral-System/internal/domain/entity"

type UserResponse struct {
	ID             string          `json:"id"`
	Email          string          `json:"email"`
	FirstName      string          `json:"first_name"`
	LastName       string          `json:"last_name"`
	NationalID     string          `json:"national_id"`
	Role           entity.UserRole `json:"role"`
	HospitalID     *string         `json:"hospital_id,omitempty"`
	DepartmentID   *string         `json:"department_id,omitempty"`
	HospitalName   string          `json:"hospital_name,omitempty"`
	DepartmentName string          `json:"department_name,omitempty"`
	IsActive       bool            `json:"is_active"`
	CreatedAt      string          `json:"created_at"`
	UpdatedAt      string          `json:"updated_at"`
	BaseResponse
}

type UserListResponse struct {
	Data  []UserResponse `json:"data"`
	Total int64          `json:"total"`
	Page  int            `json:"page"`
	BaseResponse
}
