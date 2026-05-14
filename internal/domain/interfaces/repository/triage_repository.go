package interfaces

import (
	"context"
	"time"

	"github.com/google/uuid"

	"Hospital-Referral-System/internal/domain/entity"
)

// TriageQueueRepository manages the triage priority queue for incoming referrals.
type TriageQueueRepository interface {
	BaseRepository[entity.TriageQueue]
	GetByReferralID(ctx context.Context, referralID uuid.UUID) (*entity.TriageQueue, error)
	DeleteByReferralID(ctx context.Context, referralID uuid.UUID) error
	ListForTriage(ctx context.Context, hospitalID uuid.UUID, limit, offset int) ([]entity.TriageQueue, int64, error)
	ListScheduledInRange(ctx context.Context, hospitalID, deptID uuid.UUID, start, end time.Time) ([]entity.TriageQueue, error)
	GetWaitingByDept(ctx context.Context, hospitalID, deptID uuid.UUID) ([]entity.TriageQueue, error)
	FindWaitingByHospitalAndDept(ctx context.Context, hospitalID, departmentID uuid.UUID) ([]entity.TriageQueue, error)
	FindScheduledByHospitalAndDept(ctx context.Context, hospitalID, deptID uuid.UUID, startDate, endDate time.Time) ([]*entity.TriageQueue, error)
	FindByHospitalAndDept(ctx context.Context, hospitalID, deptID uuid.UUID, limit, offset int) ([]*entity.TriageQueue, int64, error)
	FindAppointmentsForReminders(ctx context.Context, date time.Time) ([]*entity.TriageQueue, error)
	IncrementWaitingWeights(ctx context.Context) (int64, error)
}
