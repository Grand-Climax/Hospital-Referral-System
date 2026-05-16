package interfaces

import (
	"context"
	"time"

	"github.com/google/uuid"

	"Hospital-Referral-System/internal/delivery/http/dto"
)

type SchedulingUseCase interface {
	GetCapacityStatus(ctx context.Context, hospitalID, deptID uuid.UUID, dateRangeDays int) ([]dto.CapacityStatusResponse, error)
	ScheduleAppointment(ctx context.Context, referralID, userID uuid.UUID, req dto.SchedulingRequest) error
	// Specialized Scheduling
	ManualEmergencySchedule(ctx context.Context, referralID uuid.UUID, appointmentDate time.Time, justification string, userID uuid.UUID) error
	BatchSchedule(ctx context.Context, hospitalID, deptID, userID uuid.UUID, sendNotifications bool) (*dto.BatchScheduleResult, error)
	ProcessMissedAppointments(ctx context.Context) error
}
