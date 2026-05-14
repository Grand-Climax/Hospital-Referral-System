package interfaces

import (
	"context"
	"time"

	"github.com/google/uuid"

	"Hospital-Referral-System/internal/domain/entity"
)

type ReferralFilter struct {
	Status      string
	Statuses    []entity.ReferralStatus
	PatientID   *uuid.UUID
	PatientName string
	Region      string
	SortBy      string // "created_at" or "updated_at"
	SortOrder   string // "asc" or "desc"
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

type ReferralUpdateFields struct {
	PatientIdentityVerified *bool
	ClinicalHistoryAttached *bool
	VitalsIncluded          *bool
	AttachmentsIncluded     *bool
}

type MohAnalyticsFilter struct {
	From       *time.Time
	To         *time.Time
	Region     *string
	HospitalID *uuid.UUID
	TierLevel  *entity.HospitalTier
}

type MohDashboardSummary struct {
	TotalReferrals      int64   `json:"total_referrals"`
	TotalAccepted       int64   `json:"total_accepted"`
	TotalRejected       int64   `json:"total_rejected"`
	TotalAdmitted       int64   `json:"total_admitted"`
	AcceptanceRate      float64 `json:"acceptance_rate"`
	AverageMLSeverity   float64 `json:"average_ml_severity"`
	AverageTurnaroundHr float64 `json:"average_turnaround_hours"`
}

type MohReferralTrendPoint struct {
	Period             string `json:"period"`
	TotalReferrals     int64  `json:"total_referrals"`
	AcceptedReferrals  int64  `json:"accepted_referrals"`
	RejectedReferrals  int64  `json:"rejected_referrals"`
	EmergencyReferrals int64  `json:"emergency_referrals"`
}

type MohHospitalLoadMetric struct {
	HospitalID      uuid.UUID           `json:"hospital_id"`
	HospitalName    string              `json:"hospital_name"`
	TierLevel       entity.HospitalTier `json:"tier_level"`
	Region          string              `json:"region"`
	TotalReceived   int64               `json:"total_referrals_received"`
	TotalAccepted   int64               `json:"total_accepted"`
	TotalRejected   int64               `json:"total_rejected"`
	RejectionRate   float64             `json:"rejection_rate"`
	AverageSeverity float64             `json:"average_severity"`
}

type MohDiseaseHotspot struct {
	Region          string  `json:"region"`
	DepartmentName  string  `json:"department_name"`
	ReferralCount   int64   `json:"referral_count"`
	AverageSeverity float64 `json:"average_severity"`
}

type MohSeverityDistribution struct {
	Region         string `json:"region"`
	CriticalCount  int64  `json:"critical_count"`
	UrgentCount    int64  `json:"urgent_count"`
	RoutineCount   int64  `json:"routine_count"`
	TotalReferrals int64  `json:"total_referrals"`
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
	UpdateFields(ctx context.Context, referralID uuid.UUID, updates ReferralUpdateFields) error
	GetMohDashboardSummary(ctx context.Context, filter MohAnalyticsFilter) (*MohDashboardSummary, error)
	GetMohReferralTrends(ctx context.Context, filter MohAnalyticsFilter, granularity string) ([]MohReferralTrendPoint, error)
	GetMohHospitalLoad(ctx context.Context, filter MohAnalyticsFilter) ([]MohHospitalLoadMetric, error)
	GetMohDiseaseHotspots(ctx context.Context, filter MohAnalyticsFilter) ([]MohDiseaseHotspot, error)
	GetMohSeverityDistribution(ctx context.Context, filter MohAnalyticsFilter) ([]MohSeverityDistribution, error)
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
