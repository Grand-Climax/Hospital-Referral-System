package dto

import (
	"time"

	"github.com/google/uuid"

	"Hospital-Referral-System/internal/domain/entity"
)

// TriageLandingResponse is returned by POST /specialist/referrals/:id/accept.
// It surfaces the initial composite score so the FE can confirm the row
// landed in the queue with the expected priority.
type TriageLandingResponse struct {
	ReferralID     uuid.UUID           `json:"referral_id"`
	TriageStatus   entity.TriageStatus `json:"triage_status"`
	CompositeScore float64             `json:"composite_score"`
	MLSeverity     *float64            `json:"ml_severity"`
}

// TriageListItem is the rich row shape returned by the role-aware triage
// queue list endpoints (specialist, dept-head, receptionist). Fields are
// stable across roles; what changes between roles is which rows the
// caller can see (scope) and what the detail endpoint returns.
type TriageListItem struct {
	QueueID             uuid.UUID  `json:"queue_id"`
	ReferralID          uuid.UUID  `json:"referral_id"`
	PatientID           uuid.UUID  `json:"patient_id"`
	PatientName         string     `json:"patient_name"`
	CompositeScore      float64    `json:"composite_score"`
	AppointmentDate     *time.Time `json:"appointment_date"`
	ArrivalStatus       string     `json:"arrival_status"`
	ReferralStatus      string     `json:"referral_status"`
	ConditionAtReferral string     `json:"condition_at_referral"`
	DepartmentID        uuid.UUID  `json:"department_id"`
	DepartmentName      string     `json:"department_name"`
	HasDoctorAssigned   bool       `json:"has_doctor_assigned"`
	AssignedDoctorID    *uuid.UUID `json:"assigned_doctor_id,omitempty"`
	AssignedDoctorName  string     `json:"assigned_doctor_name,omitempty"`
	CreatedAt           time.Time  `json:"created_at"`
}

// TriageListResponse is kept as a backwards-compatible alias for the
// richer TriageListItem so existing imports (e.g. DeptHeadTriageQueueResponse
// in department_head_response_dto.go) keep compiling. The wire format
// is now the richer item; old field names like target_dept are gone.
type TriageListResponse = TriageListItem

// TriageListEnvelope is the standard paginated wrapper returned by the
// role-aware list endpoints. has_more lets the FE drive infinite scroll
// without recomputing (total / limit) on the client.
type TriageListEnvelope struct {
	Success bool             `json:"success" example:"true"`
	Data    []TriageListItem `json:"data"`
	Total   int64            `json:"total" example:"42"`
	Page    int              `json:"page" example:"1"`
	Limit   int              `json:"limit" example:"20"`
	HasMore bool             `json:"has_more" example:"true"`
}

// TriageListFilter is the parameter bag use case methods accept from
// handlers. Mirrors the repository TriageQueueFilter but adds the raw
// national_id string; the use case hashes it before forwarding so PII
// never leaves the handler -> use case boundary.
type TriageListFilter struct {
	HospitalID        uuid.UUID
	DepartmentID      *uuid.UUID
	ArrivalStatuses   []entity.ArrivalStatus
	ReferralStatuses  []entity.ReferralStatus
	PatientID         *uuid.UUID
	NationalID        string
	HasDoctorAssigned *bool
	IncludeTerminal   bool
	SortBy            string
	SortOrder         string
	Limit             int
	Offset            int
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
