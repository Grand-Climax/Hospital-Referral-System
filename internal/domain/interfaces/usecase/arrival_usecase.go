package interfaces

import (
	"context"

	"github.com/google/uuid"

	"Hospital-Referral-System/internal/domain/entity"
)

type ArrivalUseCase interface {
	GetTodayAndTomorrowSchedule(ctx context.Context, hospitalID, deptID uuid.UUID) ([]*entity.TriageQueue, error)
	ConfirmArrival(ctx context.Context, queueID uuid.UUID, userID uuid.UUID) error
	AssignDoctor(ctx context.Context, queueID uuid.UUID, doctorID uuid.UUID, userID uuid.UUID, reason string) error
	RevokeDoctorAssignment(ctx context.Context, queueID, userID uuid.UUID, reason string) error
	MarkMissed(ctx context.Context, queueID uuid.UUID, missReason entity.MissReason, userID uuid.UUID) error
	ListMissedByHospital(ctx context.Context, hospitalID uuid.UUID, limit, offset int) ([]*entity.TriageQueue, int64, error)
	GetTriageQueueByReferralID(ctx context.Context, referralID uuid.UUID) (*entity.TriageQueue, error)


	// Consult management
	GrantConsultAccess(ctx context.Context, referralID, granterID, doctorID uuid.UUID) error
	RevokeConsultAccess(ctx context.Context, referralID, granterID, doctorID uuid.UUID, reason string) error
	ReturnToTriage(ctx context.Context, queueID, userID uuid.UUID) error
}

