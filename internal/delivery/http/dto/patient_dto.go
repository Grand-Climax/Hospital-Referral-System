package dto

import (
	"time"
)

// CreatePatientRequest is used for POST /api/v1/patients
type CreatePatientRequest struct {
	// Optional Primary ID
	NationalID string `json:"national_id" example:"NAT-SEED-001"`

	// Required fields
	PhoneNumber string     `json:"phone_number" binding:"required"          example:"+251911000001"`
	FirstName   string     `json:"first_name"   binding:"required,min=2,max=100" example:"Abebe"`
	LastName    string     `json:"last_name"    binding:"required,min=2,max=100" example:"Balcha"`
	Sex         string     `json:"sex"          binding:"required,oneof=male female unknown" example:"male"`

	// Optional Fields
	MiddleName  string     `json:"middle_name"                                            example:"Kebede"`
	DateOfBirth *time.Time `json:"date_of_birth"`
	HomeRegion  string     `json:"home_region"                                            example:"Addis Ababa"`
}

type PatientResponse struct {
	ID          string `json:"id"`
	FirstName   string `json:"first_name"`
	MiddleName  string    `json:"middle_name"`
	LastName    string    `json:"last_name"`
	NationalID  string    `json:"national_id,omitempty"`
	Sex         string    `json:"sex"`
	DateOfBirth string    `json:"date_of_birth"`
	PhoneNumber string    `json:"phone_number"`
	HomeRegion  string    `json:"home_region"`
	BaseResponse
}
