package interfaces

import (
	"context"

	"time"

	"github.com/google/uuid"

	"Hospital-Referral-System/internal/delivery/http/dto"
	"Hospital-Referral-System/internal/domain/entity"
)

type TriageUseCase interface {
	LandInQueue(ctx context.Context, referralID uuid.UUID) error
	CalculateCompositeScore(ctx context.Context, referralID uuid.UUID) (float64, error)
	ListForTriage(ctx context.Context, hospitalID uuid.UUID, limit, offset int) ([]dto.TriageListResponse, int64, error)
	ListForTriageByDepartment(ctx context.Context, hospitalID, deptID uuid.UUID, limit, offset int) ([]dto.TriageListResponse, int64, error)
	ReviewTriage(ctx context.Context, referralID, userID uuid.UUID, req dto.TriageReviewRequest) error
	ListScheduledInRange(ctx context.Context, hospitalID, deptID uuid.UUID, start, end time.Time) ([]entity.TriageQueue, error)

	// ListTriageFiltered is the role-aware filterable list used by every
	// triage queue endpoint (specialist, receptionist, dept-head). The
	// handler is responsible for forcing role-specific scopes (e.g. the
	// dept-head handler must inject DepartmentID from the JWT).
	ListTriageFiltered(ctx context.Context, filter dto.TriageListFilter) ([]dto.TriageListItem, int64, error)

	// GetTriageDetailForSpecialist returns the full clinical view for a
	// receiving specialist. {id} is a referral UUID.
	GetTriageDetailForSpecialist(ctx context.Context, referralID, userID uuid.UUID) (*dto.TriageDetailSpecialistResponse, error)

	// GetTriageDetailForReceptionist returns the redacted operational
	// view for a receptionist. {id} is a referral UUID.
	GetTriageDetailForReceptionist(ctx context.Context, referralID, userID uuid.UUID) (*dto.TriageDetailReceptionistResponse, error)

	// GetTriageDetailForDeptHead returns the capacity-oriented view for
	// a department head. {id} is a Referral UUID, matching the
	// specialist + receptionist semantics so the FE has ONE id model
	// across roles. The list endpoint returns both queue_id and
	// referral_id; pass referral_id to this endpoint.
	GetTriageDetailForDeptHead(ctx context.Context, referralID, userID uuid.UUID) (*dto.TriageDetailDeptHeadResponse, error)
}
