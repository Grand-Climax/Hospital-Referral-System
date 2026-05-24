package interfaces

import (
	"context"
	"time"

	"github.com/google/uuid"

	"Hospital-Referral-System/internal/delivery/http/dto"
)

type SchedulingUseCase interface {
	GetCapacityStatus(ctx context.Context, hospitalID, deptID uuid.UUID, dateRangeDays int) ([]dto.CapacityStatusResponse, error)
	ScheduleAppointment(ctx context.Context, referralID, userID uuid.UUID, req dto.SchedulingRequest) (bool, error)
	// Specialized Scheduling
	ManualEmergencySchedule(ctx context.Context, referralID uuid.UUID, appointmentDate time.Time, justification string, userID uuid.UUID) (bool, error)
	BatchSchedule(ctx context.Context, hospitalID, deptID, userID uuid.UUID, sendNotifications bool) (*dto.BatchScheduleResult, error)
	ProcessMissedAppointments(ctx context.Context) error

	// EffectiveCapacity is the single source of truth for "is there room?".
	// It returns the effective maxSlots and overbookLimit for a given
	// (hospital, department, date) plus the live booked count.
	//
	//   - booked        : live count from TriageQueue (Expected/Arrived/Admitted)
	//   - maxSlots      : HospitalDepartment.StandardDailyLimit, overridden by
	//     CapacityOverride.NewLimit when an active override exists
	//   - overbookLimit : HospitalDepartment.OverbookLimit (department-level)
	//
	// Exposed so the capacity-management/calendar/specialist views can
	// reuse the same logic without duplicating SQL.
	EffectiveCapacity(ctx context.Context, hospitalID, deptID uuid.UUID, date time.Time) (maxSlots, overbookLimit int, booked int64, err error)

	// ListScheduleOptions returns the next N viable dates for a referral,
	// starting at today + buffer_days, with the live available capacity
	// for each date. Used by the specialist UI to surface routine
	// scheduling choices.
	ListScheduleOptions(ctx context.Context, referralID uuid.UUID, days int) ([]dto.ScheduleOption, error)
}
