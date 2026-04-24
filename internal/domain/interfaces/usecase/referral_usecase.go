package interfaces

import (
	"context"

	"github.com/google/uuid"

	"Hospital-Referral-System/internal/delivery/http/dto"
	"Hospital-Referral-System/internal/domain/entity"
	irepository "Hospital-Referral-System/internal/domain/interfaces/repository"
)

type ReferralUseCase interface {
	// --- Doctor Actions ---
	CreateDraftOrSubmit(ctx context.Context, doctorID uuid.UUID, senderHospitalID uuid.UUID, req dto.CreateReferralRequest) (*dto.ReferralCreationResponse, error)
	ListForDoctor(ctx context.Context, doctorID uuid.UUID, filter irepository.ReferralFilter) ([]entity.Referral, int64, error)
	GetDetailsForDoctor(ctx context.Context, id, doctorID uuid.UUID) (*entity.Referral, error)
	UpdateAndResubmit(ctx context.Context, id, doctorID uuid.UUID, req dto.UpdateReferralRequest, submit bool) (*dto.ReferralCreationResponse, error)
	CancelReferral(ctx context.Context, id, doctorID uuid.UUID, reason string) error
	GetDoctorDashboardStats(ctx context.Context, doctorID uuid.UUID) (*dto.DoctorDashboardStats, error)
	GetLatestPendingReferrals(ctx context.Context, doctorID uuid.UUID, limit int) ([]dto.ListReferralResponse, error)
	DeleteAttachmentsByReferralID(ctx context.Context, id, doctorID uuid.UUID) error

	// --- Liaison Actions ---
	ListOutgoingForLiaison(ctx context.Context, hospID uuid.UUID, filter irepository.ReferralFilter) ([]entity.Referral, int64, error)
	ListIncomingForLiaison(ctx context.Context, hospID uuid.UUID, filter irepository.ReferralFilter) ([]entity.Referral, int64, error)
	GetDetailsForLiaison(ctx context.Context, id, hospID uuid.UUID) (*entity.Referral, error)
	LiaisonRead(ctx context.Context, id, liaisonID, hospID uuid.UUID) error
	LiaisonForward(ctx context.Context, id, liaisonID, hospID uuid.UUID, comment string) error
	LiaisonReject(ctx context.Context, id, liaisonID, hospID uuid.UUID, reason string) error
	LiaisonRevise(ctx context.Context, id, liaisonID, hospID uuid.UUID, reason string) error
	LiaisonUnassignSpecialist(ctx context.Context, id, liaisonID, hospID uuid.UUID, reason string) error

	// --- Specialist Actions ---
	ListForSpecialist(ctx context.Context, hospID, specialistID uuid.UUID, filter irepository.ReferralFilter) ([]entity.Referral, int64, error)
	GetDetailsForSpecialist(ctx context.Context, id, hospID uuid.UUID) (*entity.Referral, error)
	SpecialistRead(ctx context.Context, id, specialistID, hospID uuid.UUID) error
	SpecialistAccept(ctx context.Context, id, specialistID, hospID uuid.UUID, severityScore *float64) error
	SpecialistReject(ctx context.Context, id, specialistID, hospID uuid.UUID, reason string) error
	SpecialistRelease(ctx context.Context, id, specialistID, hospID uuid.UUID, reason string) error
	SpecialistRerunML(ctx context.Context, id, specialistID, hospID uuid.UUID) error

	// --- Receptionist Actions ---
	ListForReceptionist(ctx context.Context, hospID uuid.UUID, filter irepository.ReferralFilter) ([]entity.Referral, int64, error)
	GetDetailsForReceptionist(ctx context.Context, id, hospID uuid.UUID) (*entity.Referral, error)
	ConfirmAttendance(ctx context.Context, id, receptionistID, hospID uuid.UUID, status string) error

	// --- Admin Actions ---
	ListForSystemAdmin(ctx context.Context, filter irepository.ReferralFilter) ([]entity.Referral, int64, error)
	GetHospitalLogsForAdmin(ctx context.Context, hospID uuid.UUID, limit, page int) ([]entity.ReferralStatusHistory, int64, error)
	GetReferralStatusHistoryForHospitalAdmin(ctx context.Context, hospID, referralID uuid.UUID, limit, page int) ([]entity.ReferralStatusHistory, int64, error)

	// --- Clinical & Outcome Actions ---
	AddClinicalUpdate(ctx context.Context, referralID, userID uuid.UUID, req dto.ClinicalUpdateRequest) (*dto.ClinicalUpdateResponse, error)
	RecordOutcome(ctx context.Context, referralID, userID uuid.UUID, req dto.ReferralOutcomeRequest) (*dto.ReferralOutcomeResponse, error)
	GetClinicalHistory(ctx context.Context, referralID uuid.UUID) ([]dto.ClinicalUpdateResponse, error)

	// --- Helpers ---
	IsValidStatus(status string) bool
}
