package dto

import "Hospital-Referral-System/internal/domain/entity"

type UserResponse struct {
	ID             string          `json:"id"`
	Email          string          `json:"email"`
	FirstName      string          `json:"first_name"`
	MiddleName     string          `json:"middle_name"`
	LastName       string          `json:"last_name"`
	NationalID     string          `json:"national_id"`
	Role           entity.UserRole `json:"role"`
	HospitalID     *string         `json:"hospital_id,omitempty"`
	DepartmentID   *string         `json:"department_id,omitempty"`
	Region         *string         `json:"region,omitempty"`
	PhoneNumber    *string         `json:"phone_number,omitempty"`
	Hospital       *HospitalResponse `json:"hospital,omitempty"`
	Department     *DepartmentResponse `json:"department,omitempty"`
	ProfileImageURL string         `json:"profile_image_url,omitempty"`
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

type UpdateProfileImageRequest struct {
	PublicID string `json:"public_id" binding:"required" example:"profiles/hosp_1/user_1/abc"`
	ImageURL string `json:"image_url" binding:"required" example:"https://res.cloudinary.com/..."`
}
