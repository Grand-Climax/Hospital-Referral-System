package usecase

import (
	"context"

	"github.com/google/uuid"

	"Hospital-Referral-System/internal/delivery/http/dto"
	"Hospital-Referral-System/internal/domain/entity"
)

type ReferralUseCase interface {
	// --- Doctor Actions ---
	CreateDraftOrSubmit(ctx context.Context, doctorID uuid.UUID, senderHospitalID uuid.UUID, req dto.CreateReferralRequest) (*entity.Referral, error)
	ListForDoctor(ctx context.Context, doctorID uuid.UUID, limit, page int, statusFilter string) ([]entity.Referral, int64, error)
	GetDetailsForDoctor(ctx context.Context, id, doctorID uuid.UUID) (*entity.Referral, error)
	UpdateAndResubmit(ctx context.Context, id, doctorID uuid.UUID, req dto.CreateReferralRequest) (*entity.Referral, error)
	CancelReferral(ctx context.Context, id, doctorID uuid.UUID, reason string) error

	// --- Liaison Actions ---
	ListForLiaison(ctx context.Context, hospID uuid.UUID, limit, page int, statusFilter string) ([]entity.Referral, int64, error)
	GetDetailsForLiaison(ctx context.Context, id, hospID uuid.UUID) (*entity.Referral, error)
	LiaisonRead(ctx context.Context, id, liaisonID, hospID uuid.UUID) error
	LiaisonForward(ctx context.Context, id, liaisonID, hospID uuid.UUID, comment string) error
	LiaisonReject(ctx context.Context, id, liaisonID, hospID uuid.UUID, reason string) error
	LiaisonRevise(ctx context.Context, id, liaisonID, hospID uuid.UUID, reason string) error

	// --- Specialist Actions ---
	ListForSpecialist(ctx context.Context, hospID, specialistID uuid.UUID, limit, page int, statusFilter string) ([]entity.Referral, int64, error)
	GetDetailsForSpecialist(ctx context.Context, id, hospID uuid.UUID) (*entity.Referral, error)
	SpecialistRead(ctx context.Context, id, specialistID, hospID uuid.UUID) error
	SpecialistAccept(ctx context.Context, id, specialistID, hospID uuid.UUID, severityScore *float64) error
	SpecialistReject(ctx context.Context, id, specialistID, hospID uuid.UUID, reason string) error
	SpecialistRerunML(ctx context.Context, id, specialistID, hospID uuid.UUID) error

	// --- Receptionist Actions ---
	ListForReceptionist(ctx context.Context, hospID uuid.UUID, limit, page int, statusFilter string) ([]entity.Referral, int64, error)
	GetDetailsForReceptionist(ctx context.Context, id, hospID uuid.UUID) (*entity.Referral, error)
	ConfirmAttendance(ctx context.Context, id, receptionistID, hospID uuid.UUID, status string) error

	// --- Admin Actions ---
	ListForSystemAdmin(ctx context.Context, limit, page int, statusFilter string) ([]entity.Referral, int64, error)
	GetHospitalLogsForAdmin(ctx context.Context, hospID uuid.UUID, limit, page int) ([]entity.ReferralStatusHistory, int64, error)
}
