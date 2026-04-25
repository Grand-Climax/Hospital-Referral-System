package interfaces

import (
	"context"

	"github.com/google/uuid"

	"Hospital-Referral-System/internal/domain/entity"
)

type ArrivalUseCase interface {
	GetTodayAndTomorrowSchedule(ctx context.Context, hospitalID, deptID uuid.UUID) ([]*entity.TriageQueue, error)
	ConfirmArrival(ctx context.Context, queueID uuid.UUID, userID uuid.UUID) error
	AssignDoctor(ctx context.Context, queueID uuid.UUID, doctorID uuid.UUID, userID uuid.UUID) error
	RegisterWalkIn(ctx context.Context, referralID uuid.UUID, hospitalID uuid.UUID, deptID uuid.UUID, userID uuid.UUID) (*entity.TriageQueue, error)
	MarkMissed(ctx context.Context, queueID uuid.UUID, missReason entity.MissReason, userID uuid.UUID) error
}
