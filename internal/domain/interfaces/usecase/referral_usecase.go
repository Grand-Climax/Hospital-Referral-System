package interfaces

import (
	"context"

	"github.com/google/uuid"

	"Hospital-Referral-System/internal/delivery/http/dto"
	"Hospital-Referral-System/internal/domain/entity"
	irepository "Hospital-Referral-System/internal/domain/interfaces/repository"
)

type UploadedFileData struct {
	URL, PublicID, FileName, FileType, Category string
	FileSize int64
	Status   string
	Metadata map[string]interface{}
	Reason   string
}

type StatItem struct {
	Count  int64   `json:"count"`
	Change float64 `json:"change"` // percentage change, e.g., 12.5 means +12.5%
}

type LiaisonDashboardStats struct {
	TotalReferrals StatItem `json:"total_referrals"`
	PendingReview  StatItem `json:"pending_review"`
	ApprovedToday  StatItem `json:"approved_today"`
	Rejected       StatItem `json:"rejected"`
}

type ReferralUseCase interface {
	// --- Doctor Actions ---
	CreateDraftOrSubmit(ctx context.Context, doctorID uuid.UUID, senderHospitalID uuid.UUID, req dto.CreateReferralRequest) (*dto.ReferralCreationResponse, error)
	CreateReferralWithAttachments(ctx context.Context, doctorID, senderHospitalID uuid.UUID, req dto.CreateReferralRequest, refID uuid.UUID, uploads []UploadedFileData) (*dto.ReferralCreationResponse, error)
	ListForDoctor(ctx context.Context, doctorID uuid.UUID, filter irepository.ReferralFilter) ([]entity.Referral, int64, error)
	GetDetailsForDoctor(ctx context.Context, id, doctorID uuid.UUID) (*entity.Referral, error)
	UpdateAndResubmit(ctx context.Context, id, doctorID uuid.UUID, req dto.UpdateReferralRequest, submit bool) (*dto.ReferralCreationResponse, error)
	CancelReferral(ctx context.Context, id, doctorID uuid.UUID, reason string) error
	RejectAfterSend(ctx context.Context, referralID, userID, hospID uuid.UUID, role entity.UserRole, reason string) error
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
	GetLiaisonDashboardStats(ctx context.Context, hospID uuid.UUID) (*LiaisonDashboardStats, error)
	UpdateReviewChecklist(ctx context.Context, referralID, liaisonID, hospID uuid.UUID, req dto.ReviewChecklistRequest) error
	GetReviewChecklist(ctx context.Context, referralID, hospID uuid.UUID) (*dto.ReviewChecklistResponse, error)

	// --- Specialist Actions ---
	ListForSpecialist(ctx context.Context, hospID, specialistID uuid.UUID, filter irepository.ReferralFilter) ([]entity.Referral, int64, error)
	GetDetailsForSpecialist(ctx context.Context, id, hospID uuid.UUID) (*entity.Referral, error)
	SpecialistRead(ctx context.Context, id, specialistID, hospID uuid.UUID) error
	SpecialistAccept(ctx context.Context, id, specialistID, hospID uuid.UUID, severityScore *float64) error
	SpecialistReject(ctx context.Context, id, specialistID, hospID uuid.UUID, reason string) error
	SpecialistRelease(ctx context.Context, id, specialistID, hospID uuid.UUID, reason string) error
	SpecialistRerunML(ctx context.Context, id, specialistID, hospID uuid.UUID) error
	RedirectReferral(ctx context.Context, id, specialistID, hospID, targetHospitalID uuid.UUID, reason string, newDeptID *uuid.UUID) error
	ListRedirectionOptions(ctx context.Context, id, specialistID, hospID uuid.UUID, filterDeptID *uuid.UUID) ([]entity.Hospital, error)
	GetRedirectionHistory(ctx context.Context, referralID, userID uuid.UUID, role string, hospID uuid.UUID) ([]entity.ReferralRedirection, error)
	ChangeDepartment(ctx context.Context, referralID, specialistID, hospID, newDeptID uuid.UUID) error

	// --- Receptionist Actions ---
	ListForReceptionist(ctx context.Context, hospID uuid.UUID, filter irepository.ReferralFilter) ([]entity.Referral, int64, error)
	GetDetailsForReceptionist(ctx context.Context, id, hospID uuid.UUID) (*entity.Referral, error)

	// --- Admin Actions ---
	ListForSystemAdmin(ctx context.Context, filter irepository.ReferralFilter) ([]entity.Referral, int64, error)
	ListInboundForHospitalAdmin(ctx context.Context, hospID uuid.UUID, filter irepository.ReferralFilter) ([]entity.Referral, int64, error)
	ListOutboundForHospitalAdmin(ctx context.Context, hospID uuid.UUID, filter irepository.ReferralFilter) ([]entity.Referral, int64, error)
	ListPendingApprovalsForHospitalAdmin(ctx context.Context, hospID uuid.UUID, filter irepository.ReferralFilter) ([]entity.Referral, int64, error)
	ListRejectedRedirectedForHospitalAdmin(ctx context.Context, hospID uuid.UUID, filter irepository.ReferralFilter) ([]entity.Referral, int64, error)
	GetDetailsForHospitalAdmin(ctx context.Context, hospID, referralID uuid.UUID) (*entity.Referral, error)
	GetReferralStatusCountsForHospitalAdmin(ctx context.Context, hospID uuid.UUID) ([]irepository.ReferralStatusCount, error)
	GetMonthlyReferralTotalsForHospitalAdmin(ctx context.Context, hospID uuid.UUID, months int) ([]irepository.MonthlyReferralTotal, error)
	GetAcceptanceRejectionRateForHospitalAdmin(ctx context.Context, hospID uuid.UUID) (float64, float64, error)
	GetMissedAppointmentRateForHospitalAdmin(ctx context.Context, hospID uuid.UUID) (float64, error)
	GetBusiestDepartmentsForHospitalAdmin(ctx context.Context, hospID uuid.UUID, limit int) ([]irepository.DepartmentReferralLoad, error)
	GetAverageWaitTimeForHospitalAdmin(ctx context.Context, hospID uuid.UUID) (float64, error)
	GetTopReferringHospitalsForHospitalAdmin(ctx context.Context, hospID uuid.UUID, limit int) ([]irepository.ReferringHospitalCount, error)
	GetHospitalLogsForAdmin(ctx context.Context, hospID uuid.UUID, limit, page int) ([]entity.ReferralStatusHistory, int64, error)
	GetReferralStatusHistoryForHospitalAdmin(ctx context.Context, hospID, referralID uuid.UUID, limit, page int) ([]entity.ReferralStatusHistory, int64, error)


	// --- Shared Actions ---
	MarkDeceased(ctx context.Context, referralID, userID uuid.UUID, role entity.UserRole, hospID uuid.UUID, reason string) error

	// --- Helpers ---
	IsValidStatus(status string) bool
}
