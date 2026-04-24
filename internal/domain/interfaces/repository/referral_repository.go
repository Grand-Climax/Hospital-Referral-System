package interfaces

import (
	"context"

	"github.com/google/uuid"

	"Hospital-Referral-System/internal/domain/entity"
)

// ReferralFilter is used to filter referral list queries.
type ReferralFilter struct {
	Status      string
	PatientName string
	Region      string
	Sort        string // "asc" or "desc"
	Limit       int
	Page        int
}

// ReferralRepository handles the core referral lifecycle persistence.
type ReferralRepository interface {
	BaseRepository[entity.Referral]
	CreateReferralTransaction(ctx context.Context, referral *entity.Referral) error
	UpdateReferralTransaction(ctx context.Context, referral *entity.Referral) error
	DeleteReferral(ctx context.Context, id uuid.UUID) error
	GetReferralByID(ctx context.Context, id uuid.UUID) (*entity.Referral, error)
	ListReferrals(ctx context.Context, filter map[string]interface{}) ([]entity.Referral, error)
	CreateStatusHistory(ctx context.Context, history *entity.ReferralStatusHistory) error

	// Role-Based Queries
	ListForSystemAdmin(ctx context.Context, filter ReferralFilter) ([]entity.Referral, int64, error)
	GetHospitalLogsForAdmin(ctx context.Context, hospID uuid.UUID, limit, page int) ([]entity.ReferralStatusHistory, int64, error)
	GetReferralStatusHistoryForHospital(ctx context.Context, hospID, referralID uuid.UUID, limit, page int) ([]entity.ReferralStatusHistory, int64, error)
	ListForDoctor(ctx context.Context, doctorID uuid.UUID, filter ReferralFilter) ([]entity.Referral, int64, error)
	ListOutgoingForLiaison(ctx context.Context, hospID uuid.UUID, filter ReferralFilter) ([]entity.Referral, int64, error)
	ListIncomingForLiaison(ctx context.Context, hospID uuid.UUID, filter ReferralFilter) ([]entity.Referral, int64, error)
	ListForSpecialist(ctx context.Context, hospID uuid.UUID, filter ReferralFilter) ([]entity.Referral, int64, error)
	ListForReceptionist(ctx context.Context, hospID uuid.UUID, filter ReferralFilter) ([]entity.Referral, int64, error)
	GetDoctorStats(ctx context.Context, doctorID uuid.UUID) (total, pending, accepted, critical int64, err error)
	GetLatestPendingForDoctor(ctx context.Context, doctorID uuid.UUID, limit int) ([]entity.Referral, error)
}

// ReferralOutcomeRepository persists the final clinical outcome of a referral episode.
type ReferralOutcomeRepository interface {
	BaseRepository[entity.ReferralOutcome]
	Create(ctx context.Context, outcome *entity.ReferralOutcome) error
	GetByReferralID(ctx context.Context, referralID uuid.UUID) (*entity.ReferralOutcome, error)
}

// ReferralRedirectionRepository stores records of referrals that were redirected to another hospital/department.
type ReferralRedirectionRepository interface {
	BaseRepository[entity.ReferralRedirection]
	Create(ctx context.Context, redirection *entity.ReferralRedirection) error
	GetByReferralID(ctx context.Context, referralID uuid.UUID) (*entity.ReferralRedirection, error)
}

// ReferralAccessRepository tracks which doctors have been granted access to a referral (treating vs. consulted).
type ReferralAccessRepository interface {
	BaseRepository[entity.ReferralAccess]
	Create(ctx context.Context, access *entity.ReferralAccess) error
	GetAccess(ctx context.Context, referralID, doctorID uuid.UUID) (*entity.ReferralAccess, error)
}
