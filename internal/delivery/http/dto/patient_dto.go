package dto

import "time"

// LookupPatientRequest is used for the POST /patients/lookup endpoint.
// The system tries NationalID first, then falls back to phone+name, and
// auto-creates a new patient record if neither lookup succeeds.
type LookupPatientRequest struct {
	// Primary lookup key — plain text, will be SHA-256 hashed server-side
	NationalID string `json:"national_id" example:"NAT-SEED-001"`

	// Fallback lookup keys
	PhoneNumber string `json:"phone_number" example:"+251911000001"`
	FirstName   string `json:"first_name"   example:"Abebe"`

	// Required fields for auto-create (also used to enrich existing records)
	LastName   string     `json:"last_name"    binding:"required"                        example:"Kebede"`
	MiddleName string     `json:"middle_name"                                            example:"Tilahun"`
	Sex        string     `json:"sex"          binding:"required,oneof=male female unknown" example:"male"`
	DateOfBirth *time.Time `json:"date_of_birth"`
	HomeRegion string     `json:"home_region"                                            example:"Addis Ababa"`
}
