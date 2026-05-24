package dto

import "time"

// DepartmentHeadDashboardStats is the landing-page summary for the
// /department-head/dashboard/stats endpoint. Every field is computed
// live from the existing capacity / queue / referral repositories - the
// dept head never sees a stale snapshot.
//
// Field semantics:
//   - TodayCapacity   : the same payload returned by GET /capacity/detail
//     for today's date (max, booked, overbook, staff hint, etc.)
//   - WaitingQueueSize: number of unscheduled EXPECTED triage rows for the
//     caller's dept.
//   - OldestWaitingDays: integer days since the oldest unscheduled
//     EXPECTED triage row was created. Zero when the queue is empty.
//   - ScheduledToday   : Expected/Arrived/Admitted triage rows whose
//     appointment_date = today.
//   - ScheduledNext7Days: same, summed across [today+1, today+7]
//   - MissedLast7Days  : MISSED triage rows whose appointment_date is in
//     [today-7, today-1].
//   - PendingReferrals  : referrals in ACCEPTED status targeted at the
//     caller's department that have not yet been scheduled.
//   - CompletedLast30Days: referrals in COMPLETED status created within
//     the last 30 days for the dept (sender = inbound).
//   - ActiveStaff       : count of active REFERRING_DOCTOR + RECEPTIONIST
//     users in the caller's department.
//   - ActiveOverrides   : number of CapacityOverride rows with is_active
//     = true for the dept (informs "any overrides in effect?").
//   - StatusCounts       : detailed per-status counts for the dept's
//     inbound referrals (used to render the status bar chart).
type DepartmentHeadDashboardStats struct {
	TodayCapacity       *CapacityDetailResponse  `json:"today_capacity"`
	WaitingQueueSize    int                      `json:"waiting_queue_size"`
	OldestWaitingDays   int                      `json:"oldest_waiting_days"`
	ScheduledToday      int64                    `json:"scheduled_today"`
	ScheduledNext7Days  int64                    `json:"scheduled_next_7_days"`
	MissedLast7Days     int64                    `json:"missed_last_7_days"`
	PendingReferrals    int64                    `json:"pending_referrals"`
	CompletedLast30Days int64                    `json:"completed_last_30_days"`
	ActiveStaff         int                      `json:"active_staff"`
	ActiveOverrides     int                      `json:"active_overrides"`
	StatusCounts        []DeptReferralStatusItem `json:"status_counts"`
}

// DeptReferralStatusItem is one row in the per-status breakdown returned
// by the dashboard. The frontend renders these as a horizontal bar chart.
type DeptReferralStatusItem struct {
	Status string `json:"status"`
	Count  int64  `json:"count"`
}

// DepartmentHeadTrendPoint is one entry in the trends time-series used by
// the dept-head dashboard chart (last N days). booked_slots is the live
// triage-queue count for the day (Expected/Arrived/Admitted) and
// max_slots is the effective max (override-aware), so the front-end can
// derive utilization = booked / max when max > 0.
type DepartmentHeadTrendPoint struct {
	Date           string  `json:"date"`
	MaxSlots       int     `json:"max_slots"`
	BookedSlots    int64   `json:"booked_slots"`
	OverbookLimit  int     `json:"overbook_limit"`
	AvailableSlots int     `json:"available_slots"`
	Utilization    float64 `json:"utilization"` // 0..1
	HasOverride    bool    `json:"has_override"`
}

// PriorityBucketResponse breaks the dept's waiting queue down by clinical
// priority (CRITICAL / URGENT / STABLE from the referral form) and by
// ML severity tier (HIGH / MEDIUM / LOW), so the dept head can spot
// surges at a glance.
type PriorityBucketResponse struct {
	TotalWaiting int64                       `json:"total_waiting"`
	ByCondition  []PriorityBucket            `json:"by_condition"`
	BySeverity   []PriorityBucket            `json:"by_severity"`
	TopWaiting   []PriorityBucketWaitingItem `json:"top_waiting"` // top 5 highest composite_score
}

type PriorityBucket struct {
	Label string `json:"label"`
	Count int64  `json:"count"`
}

type PriorityBucketWaitingItem struct {
	ReferralID     string    `json:"referral_id"`
	PatientName    string    `json:"patient_name"`
	CompositeScore float64   `json:"composite_score"`
	WaitingDays    int       `json:"waiting_days"`
	CreatedAt      time.Time `json:"created_at"`
}

// StaffSummaryResponse is the per-role active/inactive count for the
// caller's department plus a "doctors with bookings today" hint that
// reuses the existing CountAssignedDoctorsByDeptAndDate helper.
type StaffSummaryResponse struct {
	Department       string             `json:"department"`
	Doctors          StaffRoleCount     `json:"doctors"`
	Receptionists    StaffRoleCount     `json:"receptionists"`
	DoctorsAssignedToday int64           `json:"doctors_assigned_today"`
	StaffCapacityHint int               `json:"staff_capacity_hint"` // MaxCapacityOfStaff
	Members          []StaffMember      `json:"members"`             // doctor + receptionist roster
}

type StaffRoleCount struct {
	Active   int `json:"active"`
	Inactive int `json:"inactive"`
	Total    int `json:"total"`
}

// StaffMember is a lightweight roster entry (the frontend can deep-link
// to the full user profile by ID when needed).
type StaffMember struct {
	ID           string `json:"id"`
	FirstName    string `json:"first_name"`
	LastName     string `json:"last_name"`
	Email        string `json:"email"`
	Role         string `json:"role"`
	IsActive     bool   `json:"is_active"`
	ProfileImage string `json:"profile_image,omitempty"`
}

// DepartmentHeadActivityItem is a normalized recent-activity row sourced
// from the audit log so the dept head can see, in one place: batch
// scheduling runs, override creates/deletes, emergency schedules, staff
// changes, etc.
type DepartmentHeadActivityItem struct {
	ID         string                 `json:"id"`
	Timestamp  time.Time              `json:"timestamp"`
	ActionType string                 `json:"action_type"`
	ActorID    string                 `json:"actor_id"`
	ActorName  string                 `json:"actor_name,omitempty"`
	ActorRole  string                 `json:"actor_role,omitempty"`
	ReferralID *string                `json:"referral_id,omitempty"`
	Summary    string                 `json:"summary"`
	NewValue   map[string]interface{} `json:"new_value,omitempty"`
}
