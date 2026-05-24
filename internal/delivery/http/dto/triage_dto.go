package dto

import (
	"time"

	"github.com/google/uuid"

	"Hospital-Referral-System/internal/domain/entity"
)

type TriageLandingResponse struct {
	ReferralID     uuid.UUID          `json:"referral_id"`
	TriageStatus   entity.TriageStatus `json:"triage_status"`
	CompositeScore float64            `json:"composite_score"`
	MLSeverity     *float64           `json:"ml_severity"`
}

type TriageListResponse struct {
	QueueID         uuid.UUID  `json:"id"`
	ReferralID      uuid.UUID `json:"referral_id"`
	PatientName     string    `json:"patient_name"`
	TargetDept      string    `json:"target_dept"`
	CompositeScore  float64   `json:"composite_score"`
	AppointmentDate *time.Time `json:"appointment_date"`
}

type TriageReviewRequest struct {
	Action         string   `json:"action" binding:"required"` // "APPROVE", "OVERRIDE", "REJECT"
	CompositeScore *float64 `json:"composite_score"`
	Reason         string   `json:"reason"`
}


type ManualEmergencyScheduleRequest struct {
	AppointmentDate string `json:"appointment_date" binding:"required"`
	Justification   string `json:"justification" binding:"required"`
}
