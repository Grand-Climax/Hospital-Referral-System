package dto

import (
	"time"

	"github.com/google/uuid"
)

// This file defines the strongly-typed envelope responses for every
// Department Head endpoint. Each Response type mirrors the exact JSON
// shape the handler emits via gin.H{"success": true, ...} so the
// Swagger documentation describes the wire format precisely rather
// than degrading to map[string]interface{}.
//
// Conventions:
//   - Every success envelope embeds {success: true} as the first field.
//   - List envelopes always include a `total` count.
//   - When the handler also echoes the request filters (date, year,
//     month, days, limit), the envelope echoes them too so the
//     frontend cache key can be derived from the response alone.

// ── /department-head/triage-queue ─────────────────────────────────────────────

// DeptHeadTriageQueueResponse is the paginated list of triage rows
// returned by GET /department-head/triage-queue. The page/limit are
// echoed so the frontend can keep pagination state in the response.
type DeptHeadTriageQueueResponse struct {
	Success bool                 `json:"success" example:"true"`
	Data    []TriageListResponse `json:"data"`
	Total   int64                `json:"total" example:"42"`
	Page    int                  `json:"page" example:"1"`
	Limit   int                  `json:"limit" example:"50"`
}

// ── /department-head/schedule/batch ───────────────────────────────────────────

// DeptHeadBatchScheduleResponse wraps BatchScheduleResult in the
// standard success envelope. The Message field on the inner Result is
// populated when the soft lease blocks a concurrent run or when no
// patients could be placed despite a non-empty queue.
type DeptHeadBatchScheduleResponse struct {
	Success bool                `json:"success" example:"true"`
	Data    BatchScheduleResult `json:"data"`
}

// ── /department-head/capacity/overrides ───────────────────────────────────────

// DeptHeadOverrideListResponse is the read-only listing returned by
// GET /capacity/overrides. Active and inactive rows are both returned
// so the dept head can see history before deciding to delete +
// recreate (overrides are immutable once created).
type DeptHeadOverrideListResponse struct {
	Success bool                       `json:"success" example:"true"`
	Data    []CapacityOverrideListItem `json:"data"`
}

// CapacityOverrideListItem is a single override row. The Reason field
// is a pointer in the entity but flattened to an empty string here so
// the JSON shape is stable.
type CapacityOverrideListItem struct {
	ID           uuid.UUID `json:"id" example:"6c3c2b54-6a4e-4f0c-b8d1-7c1e3a2b1c4f"`
	HospitalID   uuid.UUID `json:"hospital_id" example:"3a5d8c14-2e1d-4c2f-9a3b-1c2d3e4f5a6b"`
	DepartmentID uuid.UUID `json:"department_id" example:"9f8e7d6c-5b4a-3c2d-1e0f-9a8b7c6d5e4f"`
	TargetDate   time.Time `json:"target_date" example:"2026-06-15T00:00:00Z"`
	NewLimit     int       `json:"new_limit" example:"35"`
	Reason       string    `json:"reason,omitempty" example:"Public holiday — reduced staffing"`
	SetByID      uuid.UUID `json:"set_by_id" example:"11111111-2222-3333-4444-555555555555"`
	IsActive     bool      `json:"is_active" example:"true"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// DeptHeadOverrideByMonthResponse echoes the filters used so the
// caller can verify the response matches its query.
type DeptHeadOverrideByMonthResponse struct {
	Success bool                       `json:"success" example:"true"`
	Year    int                        `json:"year" example:"2026"`
	Month   int                        `json:"month" example:"6"`
	Data    []CapacityOverrideListItem `json:"data"`
}

// DeptHeadOverrideDetailResponse is the single-row response of
// GET /capacity/overrides/{id}.
type DeptHeadOverrideDetailResponse struct {
	Success bool                     `json:"success" example:"true"`
	Data    CapacityOverrideListItem `json:"data"`
}

// ── /department-head/schedule ─────────────────────────────────────────────────

// DeptHeadScheduleResponse is the response of GET /department-head/schedule.
// The endpoint emits one of three shapes depending on the query:
//   - single-day with a log row: {data: DailyScheduleSnapshot, has_schedule: true}
//   - single-day with no log:     {data: null, has_schedule: false, message: "..."}
//   - date-range with rows:       {data: [DailyScheduleSnapshot...], has_schedule: true}
//
// To represent this honestly in Swagger we declare Data as interface{}
// rather than committing to one of the three shapes. The frontend should
// branch on `has_schedule` first.
type DeptHeadScheduleResponse struct {
	Success     bool        `json:"success" example:"true"`
	HasSchedule bool        `json:"has_schedule" example:"true"`
	Message     string      `json:"message,omitempty" example:"No schedule log for this date yet"`
	Data        interface{} `json:"data" swaggertype:"object"`
}

// DailyScheduleSnapshot mirrors entity.DailySchedule without the
// nested Hospital / Department pointers so the wire shape is stable
// and lightweight.
type DailyScheduleSnapshot struct {
	ID            uuid.UUID `json:"id"`
	HospitalID    uuid.UUID `json:"hospital_id"`
	DepartmentID  uuid.UUID `json:"department_id"`
	ScheduleDate  time.Time `json:"schedule_date"`
	BookedSlots   int       `json:"booked_slots" example:"24"`
	MaxSlots      int       `json:"max_slots" example:"50"`
	OverbookLimit int       `json:"overbook_limit" example:"5"`
}

// ── /department-head/schedule/patients ────────────────────────────────────────

// DeptHeadScheduledPatientsResponse is the rich per-date roster used
// by the "Today" / "Tomorrow" tabs. The total is precomputed so the
// frontend does not need to len() the array.
type DeptHeadScheduledPatientsResponse struct {
	Success bool                          `json:"success" example:"true"`
	Date    string                        `json:"date" example:"2026-06-15"`
	Total   int                           `json:"total" example:"12"`
	Data    []ScheduledPatientListItem    `json:"data"`
}

// ScheduledPatientListItem flattens the most useful TriageQueue +
// Referral + Patient fields into a single row so the UI does not need
// to descend into nested objects. The full nested object is still
// available under `referral` when the FE wants to deep-link.
type ScheduledPatientListItem struct {
	QueueID          uuid.UUID  `json:"id"`
	ReferralID       uuid.UUID  `json:"referral_id"`
	PatientID        uuid.UUID  `json:"patient_id"`
	PatientName      string     `json:"patient_name" example:"Abebe Kebede"`
	PatientSex       string     `json:"patient_sex,omitempty" example:"male"`
	PatientAgeYears  *int       `json:"patient_age_years,omitempty" example:"35"`
	AppointmentDate  *time.Time `json:"appointment_date,omitempty"`
	ArrivalStatus    string     `json:"arrival_status" example:"EXPECTED"`
	CompositeScore   float64    `json:"composite_score" example:"82.5"`
	AssignedDoctorID *uuid.UUID `json:"assigned_doctor_id,omitempty"`
}

// ── /department-head/capacity/detail ──────────────────────────────────────────

// DeptHeadCapacityDetailResponse wraps the existing CapacityDetailResponse
// in the standard success envelope.
type DeptHeadCapacityDetailResponse struct {
	Success bool                   `json:"success" example:"true"`
	Data    CapacityDetailResponse `json:"data"`
}

// ── /department-head/capacity/calendar ────────────────────────────────────────

// DeptHeadCapacityCalendarResponse is the month rollup used by the
// calendar view. The year/month are echoed for caching.
type DeptHeadCapacityCalendarResponse struct {
	Success bool                  `json:"success" example:"true"`
	Year    int                   `json:"year" example:"2026"`
	Month   int                   `json:"month" example:"6"`
	Data    []CapacityCalendarDay `json:"data"`
}

// ── /department-head/staff-capacity ───────────────────────────────────────────

// DeptHeadStaffCapacityUpdateResponse is the BaseResponse-shaped
// confirmation returned after PUT /staff-capacity. Documented
// explicitly so the FE knows no body payload is returned.
type DeptHeadStaffCapacityUpdateResponse struct {
	Success bool   `json:"success" example:"true"`
	Message string `json:"message" example:"staff capacity (soft hint) updated"`
}

// ── /department-head/dashboard/stats ──────────────────────────────────────────

// DeptHeadDashboardStatsResponse is the success envelope for the
// landing-page KPI block.
type DeptHeadDashboardStatsResponse struct {
	Success bool                         `json:"success" example:"true"`
	Data    DepartmentHeadDashboardStats `json:"data"`
}

// ── /department-head/dashboard/trends ─────────────────────────────────────────

// DeptHeadTrendsResponse is the multi-day utilisation series. days is
// echoed so the frontend can cache by window.
type DeptHeadTrendsResponse struct {
	Success bool                       `json:"success" example:"true"`
	Days    int                        `json:"days" example:"14"`
	Data    []DepartmentHeadTrendPoint `json:"data"`
}

// ── /department-head/triage-queue/buckets ─────────────────────────────────────

// DeptHeadPriorityBucketsResponse wraps the priority/severity bucket
// breakdown used by the "Surge view" panel.
type DeptHeadPriorityBucketsResponse struct {
	Success bool                   `json:"success" example:"true"`
	Data    PriorityBucketResponse `json:"data"`
}

// ── /department-head/staff/summary ────────────────────────────────────────────

// DeptHeadStaffSummaryResponse wraps the per-role staff summary used
// by the "Team" panel.
type DeptHeadStaffSummaryResponse struct {
	Success bool                 `json:"success" example:"true"`
	Data    StaffSummaryResponse `json:"data"`
}

// ── /department-head/activity ────────────────────────────────────────────────

// DeptHeadActivityResponse is the recent-activity audit feed.
type DeptHeadActivityResponse struct {
	Success bool                         `json:"success" example:"true"`
	Total   int                          `json:"total" example:"20"`
	Data    []DepartmentHeadActivityItem `json:"data"`
}

// ── Shared error envelope for 400/401/404/409/500 ─────────────────────────────

// DeptHeadErrorResponse documents the failure body returned by every
// dept-head endpoint. It is identical in shape to BaseResponse +
// success=false, kept as a separate type so Swagger groups it under
// the Department Head tag.
type DeptHeadErrorResponse struct {
	Success bool   `json:"success" example:"false"`
	Message string `json:"message" example:"invalid date format; expected YYYY-MM-DD"`
}
