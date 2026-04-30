package interfaces

import (
	"context"
	"time"

	"github.com/google/uuid"

	"Hospital-Referral-System/internal/delivery/http/dto"
	"Hospital-Referral-System/internal/domain/entity"
)

type CapacityManagementUseCase interface {
	GetSchedule(ctx context.Context, hospitalID, deptID uuid.UUID, startDate, endDate time.Time) ([]entity.DailySchedule, error)
	GetOverrides(ctx context.Context, hospitalID, deptID uuid.UUID) ([]entity.CapacityOverride, error)
	CreateOverride(ctx context.Context, hospitalID, deptID uuid.UUID, date time.Time, newLimit int, reason string, userID uuid.UUID) error
	UpdateOverride(ctx context.Context, overrideID uuid.UUID, newLimit int, reason string, userID uuid.UUID) error
	DeleteOverride(ctx context.Context, overrideID, userID uuid.UUID) error
	UpdateMaxSlots(ctx context.Context, scheduleID uuid.UUID, maxSlots int, userID uuid.UUID) error
	ExtendSchedules(ctx context.Context) error
	BatchSchedule(ctx context.Context, hospitalID, deptID, userID uuid.UUID) (*dto.BatchScheduleResult, error)
}
