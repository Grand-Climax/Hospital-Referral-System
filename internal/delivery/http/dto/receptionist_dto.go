package dto

import "github.com/google/uuid"

type AssignDoctorRequest struct {
	DoctorID uuid.UUID `json:"doctor_id" binding:"required"`
}

type MarkMissedRequest struct {
	MissReason string `json:"miss_reason" binding:"required,oneof=PATIENT_NO_SHOW PATIENT_CONTACTED_RESCHEDULE HOSPITAL_CANCELLED HOSPITAL_CAPACITY_ISSUE"`
}

type WalkInRequest struct {
	ReferralID uuid.UUID `json:"referral_id" binding:"required"`
}
