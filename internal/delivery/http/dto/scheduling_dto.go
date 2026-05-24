package dto

import (
	"time"
)

type SchedulingRequest struct {
	AppointmentDate time.Time `json:"appointment_date" binding:"required"`
	Notes           string    `json:"notes"`
	Override        bool      `json:"override"` // Request a capacity override
}

type CapacityStatusResponse struct {
	Date          time.Time `json:"date"`
	TotalCapacity int       `json:"total_capacity"`
	OverbookLimit int       `json:"overbook_limit"`
	BookedSlots   int       `json:"booked_slots"`
	IsFull        bool      `json:"is_full"`
}

type DailyScheduleListResponse struct {
	Schedules []CapacityStatusResponse `json:"schedules"`
}

type BatchScheduleResult struct {
	ScheduledCount int    `json:"scheduled_count"`
	WaitingCount   int    `json:"waiting_count"`
	Message        string `json:"message,omitempty"`
}

type UpdateMaxSlotsRequest struct {
	MaxSlots int `json:"max_slots" binding:"required,min=1"`
}

type SchedulingResponse struct {
	BaseResponse
	RescheduledFromMissed bool `json:"rescheduled_from_missed"`
}
