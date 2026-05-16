package dto

import (
	"Hospital-Referral-System/internal/domain/entity"
	"github.com/google/uuid"
)

type AssignDoctorRequest struct {
	DoctorID uuid.UUID `json:"doctor_id" binding:"required"`
	Reason   string    `json:"reason"` // optional, for reassignment
}

type MarkMissedRequest struct {
	MissReason string `json:"miss_reason" binding:"required,oneof=PATIENT_NO_SHOW PATIENT_CONTACTED_RESCHEDULE HOSPITAL_CANCELLED HOSPITAL_CAPACITY_ISSUE"`
}

type RevokeDoctorRequest struct {
	Reason string `json:"reason" binding:"required"`
}

type DoctorInfo struct {
	ID        uuid.UUID `json:"id"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
}

type ReceptionistOfflineDataResponse struct {
	Schedule []*entity.TriageQueue `json:"schedule"`
	Doctors  []DoctorInfo          `json:"doctors"`
}

type DoctorListResponse struct {
	BaseResponse
	Data []DoctorInfo `json:"data"`
}
