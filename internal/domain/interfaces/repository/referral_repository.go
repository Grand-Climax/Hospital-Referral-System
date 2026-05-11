package interfaces

import (
	"context"
	"time"

	"github.com/google/uuid"

	"Hospital-Referral-System/internal/domain/entity"
)

type ReferralFilter struct {
	Status      string
	PatientName string
	Region      string
	Sort        string // "asc" or "desc"
	Limit       int
	Page        int
}

type ReferralStatusCount struct {
	Status entity.ReferralStatus `json:"status"`
	Count  int64                 `json:"count"`
}

type MonthlyReferralTotal struct {
	Month string `json:"month"`
	Count int64  `json:"count"`
}

type DepartmentReferralLoad struct {
	DepartmentID uuid.UUID `json:"department_id"`
	Count        int64     `json:"count"`
}

type ReferringHospitalCount struct {
	HospitalID   uuid.UUID `json:"hospital_id"`
	HospitalName string    `json:"hospital_name"`
	Count        int64     `json:"count"`
}

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
	ListInboundForHospitalAdmin(ctx context.Context, hospID uuid.UUID, filter ReferralFilter) ([]entity.Referral, int64, error)
	ListOutboundForHospitalAdmin(ctx context.Context, hospID uuid.UUID, filter ReferralFilter) ([]entity.Referral, int64, error)
	ListByStatusesForHospitalAdmin(ctx context.Context, hospID uuid.UUID, filter ReferralFilter, statuses []entity.ReferralStatus) ([]entity.Referral, int64, error)
	GetDetailsForHospitalAdmin(ctx context.Context, hospID, referralID uuid.UUID) (*entity.Referral, error)
	GetReferralStatusCounts(ctx context.Context, hospitalID uuid.UUID) ([]ReferralStatusCount, error)
	UpdateTargetDeptAndStatus(ctx context.Context, referralID, targetHospID, targetDeptID uuid.UUID, status entity.ReferralStatus) error
	GetMonthlyReferralTotalsForHospitalAdmin(ctx context.Context, hospID uuid.UUID, months int) ([]MonthlyReferralTotal, error)
	GetAcceptanceRejectionRateForHospitalAdmin(ctx context.Context, hospID uuid.UUID) (acceptedRate float64, rejectedRate float64, err error)
	GetMissedAppointmentRateForHospitalAdmin(ctx context.Context, hospID uuid.UUID) (float64, error)
	GetBusiestDepartmentsForHospitalAdmin(ctx context.Context, hospID uuid.UUID, limit int) ([]DepartmentReferralLoad, error)
	GetAverageWaitTimeForHospitalAdmin(ctx context.Context, hospID uuid.UUID) (float64, error)
	GetTopReferringHospitalsForHospitalAdmin(ctx context.Context, hospID uuid.UUID, limit int) ([]ReferringHospitalCount, error)
	GetHospitalLogsForAdmin(ctx context.Context, hospID uuid.UUID, limit, page int) ([]entity.ReferralStatusHistory, int64, error)
	GetReferralStatusHistoryForHospital(ctx context.Context, hospID, referralID uuid.UUID, limit, page int) ([]entity.ReferralStatusHistory, int64, error)
	ListForDoctor(ctx context.Context, doctorID uuid.UUID, filter ReferralFilter) ([]entity.Referral, int64, error)
	ListOutgoingForLiaison(ctx context.Context, hospID uuid.UUID, filter ReferralFilter) ([]entity.Referral, int64, error)
	ListIncomingForLiaison(ctx context.Context, hospID uuid.UUID, filter ReferralFilter) ([]entity.Referral, int64, error)
	ListForSpecialist(ctx context.Context, hospID uuid.UUID, filter ReferralFilter) ([]entity.Referral, int64, error)
	ListForReceptionist(ctx context.Context, hospID uuid.UUID, filter ReferralFilter) ([]entity.Referral, int64, error)
	GetDoctorStats(ctx context.Context, doctorID uuid.UUID) (total, pending, accepted, critical int64, err error)
	GetLatestPendingForDoctor(ctx context.Context, doctorID uuid.UUID, limit int) ([]entity.Referral, error)

	// Dashboard Stats
	CountBySenderHospitalAndStatuses(ctx context.Context, hospID uuid.UUID, statuses []entity.ReferralStatus, excludeDraft bool, startDate, endDate *time.Time) (int64, error)
	CountAcceptedOrCompletedToday(ctx context.Context, hospID uuid.UUID) (int64, error)
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
	ListByReferralID(ctx context.Context, referralID uuid.UUID) ([]entity.ReferralRedirection, error)
}

// ReferralAccessRepository tracks which doctors have been granted access to a referral (treating vs. consulted).
type ReferralAccessRepository interface {
	BaseRepository[entity.ReferralAccess]
	Create(ctx context.Context, access *entity.ReferralAccess) error
	GetAccess(ctx context.Context, referralID, userID uuid.UUID) (*entity.ReferralAccess, error)
	CheckAccess(ctx context.Context, referralID, userID uuid.UUID) (bool, error)
}
