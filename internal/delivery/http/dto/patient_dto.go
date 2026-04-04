package dto

import (
	"time"

	"github.com/google/uuid"
)

// CreatePatientRequest is used for POST /api/v1/patients
type CreatePatientRequest struct {
	// Optional Primary ID
	NationalID string `json:"national_id" example:"NAT-SEED-001"`

	// Required fields
	PhoneNumber string     `json:"phone_number" binding:"required,e164"          example:"+251911000001"`
	FirstName   string     `json:"first_name"   binding:"required,min=2,max=100" example:"Abebe"`
	LastName    string     `json:"last_name"    binding:"required,min=2,max=100" example:"Kebede"`
	Sex         string     `json:"sex"          binding:"required,oneof=male female unknown" example:"male"`

	// Optional Fields
	MiddleName  string     `json:"middle_name"                                            example:"Tilahun"`
	DateOfBirth *time.Time `json:"date_of_birth"`
	HomeRegion  string     `json:"home_region"                                            example:"Addis Ababa"`
}

type PatientResponse struct {
	BaseResponse
	ID             uuid.UUID `json:"id"`
	FirstName      string    `json:"first_name"`
	MiddleName     string    `json:"middle_name"`
	LastName       string    `json:"last_name"`
	Sex            string    `json:"sex"`
	DateOfBirth    string    `json:"date_of_birth"`
	PhoneNumber    string    `json:"phone_number"`
	HomeRegion     string    `json:"home_region"`
}
