package usecase

import (
	"context"

	"github.com/google/uuid"

	"Hospital-Referral-System/internal/delivery/http/dto"
	"Hospital-Referral-System/internal/domain/entity"
)

type ReferralUseCase interface {
	CreateReferral(ctx context.Context, doctorID uuid.UUID, senderHospitalID uuid.UUID, req dto.CreateReferralRequest) (*entity.Referral, error)
	GetReferral(ctx context.Context, id, userID, hospID, deptID uuid.UUID, userRole entity.UserRole) (*entity.Referral, error)
	ListReferrals(ctx context.Context, userID, hospID, deptID uuid.UUID, userRole entity.UserRole, statusFilter, dateFrom, dateTo string) ([]entity.Referral, error)
	UpdateDraft(ctx context.Context, id, userID uuid.UUID, req dto.CreateReferralRequest) (*entity.Referral, error)
	DeleteDraft(ctx context.Context, id, userID uuid.UUID) error
	UpdateReferralStatus(ctx context.Context, id uuid.UUID, newStatus entity.ReferralStatus, userID uuid.UUID, reason string) error
}
