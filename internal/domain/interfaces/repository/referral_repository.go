package repository

import (
	"context"

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

type ReferralRepository interface {
	CreateReferralTransaction(ctx context.Context, referral *entity.Referral) error
	UpdateReferralTransaction(ctx context.Context, referral *entity.Referral) error
	DeleteReferral(ctx context.Context, id uuid.UUID) error
	GetReferralByID(ctx context.Context, id uuid.UUID) (*entity.Referral, error)
	ListReferrals(ctx context.Context, filter map[string]interface{}) ([]entity.Referral, error)
	CreateStatusHistory(ctx context.Context, history *entity.ReferralStatusHistory) error

	// Role-Based Queries
	ListForSystemAdmin(ctx context.Context, filter ReferralFilter) ([]entity.Referral, int64, error)
	GetHospitalLogsForAdmin(ctx context.Context, hospID uuid.UUID, limit, page int) ([]entity.ReferralStatusHistory, int64, error)
	ListForDoctor(ctx context.Context, doctorID uuid.UUID, filter ReferralFilter) ([]entity.Referral, int64, error)
	ListOutgoingForLiaison(ctx context.Context, hospID uuid.UUID, filter ReferralFilter) ([]entity.Referral, int64, error)
	ListIncomingForLiaison(ctx context.Context, hospID uuid.UUID, filter ReferralFilter) ([]entity.Referral, int64, error)
	ListForSpecialist(ctx context.Context, hospID uuid.UUID, filter ReferralFilter) ([]entity.Referral, int64, error)
	ListForReceptionist(ctx context.Context, hospID uuid.UUID, filter ReferralFilter) ([]entity.Referral, int64, error)
	GetDoctorStats(ctx context.Context, doctorID uuid.UUID) (total, pending, accepted, critical int64, err error)
	GetLatestPendingForDoctor(ctx context.Context, doctorID uuid.UUID, limit int) ([]entity.Referral, error)
}
