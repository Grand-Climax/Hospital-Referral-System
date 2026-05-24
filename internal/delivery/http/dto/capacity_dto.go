package dto

type CreateOverrideRequest struct {
	Date     string `json:"target_date" binding:"required"`
	NewLimit int    `json:"new_limit" binding:"required,min=0"`
	Reason   string `json:"reason" binding:"required"`
}

type CapacityOverrideResponse struct {
	ID         string `json:"id"`
	TargetDate string `json:"target_date"`
	NewLimit   int    `json:"new_limit"`
	Reason     string `json:"reason"`
	IsActive   bool   `json:"is_active"`
}

// CapacityDetailResponse is the rich daily capacity view used by
// dept-head dashboards. StaffCapacity / StaffAssigned are advisory:
// the booking engine does not gate on them.
type CapacityDetailResponse struct {
	Date           string `json:"date"`
	MaxSlots       int    `json:"max_slots"`
	OverbookLimit  int    `json:"overbook_limit"`
	BookedSlots    int64  `json:"booked_slots"`
	StaffCapacity  int    `json:"staff_capacity"`   // HospitalDepartment.MaxCapacityOfStaff - soft hint only
	StaffAssigned  int64  `json:"staff_assigned"`   // distinct assigned_doctor_ids on TriageQueue for that date
	AvailableSlots int    `json:"available_slots"`  // max - booked, clamped to 0
	IsFull         bool   `json:"is_full"`          // booked >= max + overbook
	HasOverride    bool   `json:"has_override"`
}

// CapacityCalendarDay is one row in the per-day rollup returned by
// GET /capacity/calendar.
type CapacityCalendarDay struct {
	Date           string `json:"date"`
	MaxSlots       int    `json:"max_slots"`
	OverbookLimit  int    `json:"overbook_limit"`
	BookedSlots    int64  `json:"booked_slots"`
	AvailableSlots int    `json:"available_slots"`
	HasOverride    bool   `json:"has_override"`
	HasLog         bool   `json:"has_log"`
}

// UpdateStaffCapacityRequest changes HospitalDepartment.MaxCapacityOfStaff.
// This value is a soft hint surfaced through GetCapacityDetail only; it
// does not block bookings.
type UpdateStaffCapacityRequest struct {
	MaxCapacityOfStaff int `json:"max_capacity_of_staff" binding:"required,min=0"`
}

// UpdateDailyCapacityRequest changes the baseline daily capacity for
// the caller's HospitalDepartment. Both fields are required so the
// engine never sees a partial update where one column is at the new
// value and the other is stale.
//
// NOTE: `binding:"min=0"` is used WITHOUT `required` because the int
// zero-value (0) is a legitimate input (a department can be paused by
// setting standard_daily_limit to 0). With `required` Gin would reject
// a literal 0 as "missing".
type UpdateDailyCapacityRequest struct {
	StandardDailyLimit int `json:"standard_daily_limit" binding:"min=0" example:"30"`
	OverbookLimit      int `json:"overbook_limit"        binding:"min=0" example:"5"`
}

// ScheduleOption is one entry returned by /referrals/{id}/schedule-options.
type ScheduleOption struct {
	Date           string `json:"date"`
	MaxSlots       int    `json:"max_slots"`
	BookedSlots    int64  `json:"booked_slots"`
	AvailableSlots int    `json:"available_slots"`
	OverbookLimit  int    `json:"overbook_limit"`
	HasOverride    bool   `json:"has_override"`
}
