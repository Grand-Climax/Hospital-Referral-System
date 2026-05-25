package interfaces

import (
	"context"

	"github.com/google/uuid"

	"Hospital-Referral-System/internal/domain/entity"
)

// ArrivalUseCase methods take a caller scope (callerHospID, callerDeptID)
// for the receptionist mutations. The scope is the hospital/department
// from the caller's JWT - we reject if a receptionist tries to act on a
// queue row outside their own hospital (and department, for AssignDoctor
// / RevokeDoctorAssignment).
//
// callerDeptID is uuid.Nil for callers without a department-bound
// session - in that case only the hospital check runs.
type ArrivalUseCase interface {
	GetTodayAndTomorrowSchedule(ctx context.Context, hospitalID, deptID uuid.UUID) ([]*entity.TriageQueue, error)
	ConfirmArrival(ctx context.Context, queueID uuid.UUID, userID uuid.UUID, callerHospID, callerDeptID uuid.UUID) error
	AssignDoctor(ctx context.Context, queueID uuid.UUID, doctorID uuid.UUID, userID uuid.UUID, reason string, callerHospID, callerDeptID uuid.UUID) error
	RevokeDoctorAssignment(ctx context.Context, queueID, userID uuid.UUID, reason string, callerHospID, callerDeptID uuid.UUID) error
	MarkMissed(ctx context.Context, queueID uuid.UUID, missReason entity.MissReason, userID uuid.UUID, callerHospID, callerDeptID uuid.UUID) error
	ListMissedByHospital(ctx context.Context, hospitalID uuid.UUID, limit, offset int) ([]*entity.TriageQueue, int64, error)
	GetTriageQueueByReferralID(ctx context.Context, referralID uuid.UUID) (*entity.TriageQueue, error)

	// Consult management
	GrantConsultAccess(ctx context.Context, referralID, granterID, doctorID uuid.UUID) error
	RevokeConsultAccess(ctx context.Context, referralID, granterID, doctorID uuid.UUID, reason string) error
	ReturnToTriage(ctx context.Context, queueID, userID uuid.UUID, callerHospID, callerDeptID uuid.UUID) error
}

