package repository

import (
	"context"

	"fmt"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"Hospital-Referral-System/internal/domain/entity"
	irepository "Hospital-Referral-System/internal/domain/interfaces/repository"
)

type referralRepository struct {
	*BaseRepository[entity.Referral]
	db *gorm.DB
}

func NewReferralRepository(db *gorm.DB) irepository.ReferralRepository {
	return &referralRepository{
		BaseRepository: NewBaseRepository[entity.Referral](db),
		db:             db,
	}
}

func (r *referralRepository) CreateReferralTransaction(ctx context.Context, referral *entity.Referral) error {
	// GORM automatically manages the transaction for associated creations
	// Patient, ReferralForm, and Diagnoses will be inserted atomically.
	return r.db.WithContext(ctx).Create(referral).Error
}

func (r *referralRepository) UpdateReferralTransaction(ctx context.Context, referral *entity.Referral) error {
	// Use a transaction to safely clear old relations (like diagnoses/vitals) and insert new ones
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Manually delete the existing dependent records from the DB instead of using Association.Replace()
		// which tries to SET NULL on columns with NOT NULL constraints before deleting them.
		if err := tx.Unscoped().Where("referral_id = ?", referral.ID).Delete(&entity.ReferralDiagnosis{}).Error; err != nil {
			return err
		}
		if err := tx.Unscoped().Where("referral_id = ?", referral.ID).Delete(&entity.Vital{}).Error; err != nil {
			return err
		}

		// Save the parent and its associations (ReferralForm, EmergencyDetail, and the new Diagnoses/Vitals).
		// FullSaveAssociations: true ensures that GORM saves everything in the entity graph.
		return tx.Session(&gorm.Session{FullSaveAssociations: true}).Save(referral).Error
	})
}

func (r *referralRepository) DeleteReferral(ctx context.Context, id uuid.UUID) error {
	// Only delete the root Referral envelope, Cascade should handle Form, Vitals etc. if set.
	// We also ensure we only delete DRAFTs from the usecase layer, but doing a targeted delete here.
	return r.db.WithContext(ctx).Delete(&entity.Referral{}, id).Error
}

func (r *referralRepository) GetReferralByID(ctx context.Context, id uuid.UUID) (*entity.Referral, error) {
	var referral entity.Referral
	err := r.db.WithContext(ctx).
		Preload("Patient").
		Preload("ReferralForm").
		Preload("Diagnoses").
		Preload("Diagnoses.CodeInfo").
		Preload("Vitals").
		Preload("EmergencyDetail").
		Preload("Attachments").
		Preload("ReceiverHospital").
		Preload("TargetDepartment").
		Where("id = ?", id).
		First(&referral).Error
	return &referral, err
}

func (r *referralRepository) CreateStatusHistory(ctx context.Context, history *entity.ReferralStatusHistory) error {
	return r.db.WithContext(ctx).Create(history).Error
}

func (r *referralRepository) ListReferrals(ctx context.Context, filter map[string]interface{}) ([]entity.Referral, error) {
	var referrals []entity.Referral
	query := r.db.WithContext(ctx).
		Preload("Patient").
		Preload("ReferralForm").
		Preload("Diagnoses").
		Preload("Diagnoses.CodeInfo").
		Preload("Vitals").
		Preload("EmergencyDetail")

	if status, ok := filter["status"]; ok {
		query = query.Where("status = ?", status)
	}

	if senderID, ok := filter["sender_hospital_id"]; ok {
		query = query.Where("sender_hospital_id = ?", senderID)
	}

	if targetID, ok := filter["target_hospital_id"]; ok {
		query = query.Where("target_hospital_id = ?", targetID)
	}

	if doctorID, ok := filter["referring_doctor_id"]; ok {
		query = query.Where("referring_doctor_id = ?", doctorID)
	}

	if deptID, ok := filter["target_dept_id"]; ok {
		query = query.Where("target_dept_id = ?", deptID)
	}

	if startDate, ok := filter["start_date"]; ok {
		query = query.Where("created_at >= ?", startDate)
	}

	if endDate, ok := filter["end_date"]; ok {
		query = query.Where("created_at <= ?", endDate)
	}

	err := query.Order("created_at desc").Find(&referrals).Error
	return referrals, err
}

func (r *referralRepository) applyFilter(query *gorm.DB, filter irepository.ReferralFilter) *gorm.DB {
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}

	if filter.Region != "" {
		// Case-insensitive region search using Joins to Patient
		query = query.Joins("JOIN patients ON patients.id = referrals.patient_id").
			Where("LOWER(patients.home_region) LIKE LOWER(?)", "%"+filter.Region+"%")
	}

	if filter.PatientName != "" {
		// Split name into tokens to support any order (e.g., "Doe John" matches "John ... Doe")
		tokens := strings.Fields(strings.ToLower(filter.PatientName))
		// If region JOIN wasn't already added
		if filter.Region == "" {
			query = query.Joins("JOIN patients ON patients.id = referrals.patient_id")
		}
		for _, token := range tokens {
			pattern := "%" + token + "%"
			query = query.Where("(LOWER(patients.first_name) LIKE ? OR LOWER(patients.middle_name) LIKE ? OR LOWER(patients.last_name) LIKE ?)", pattern, pattern, pattern)
		}
	}

	// Sorting
	sortOrder := "desc"
	if strings.ToLower(filter.Sort) == "asc" {
		sortOrder = "asc"
	}
	query = query.Order(fmt.Sprintf("referrals.created_at %s", sortOrder))

	return query
}

func (r *referralRepository) ListForSystemAdmin(ctx context.Context, filter irepository.ReferralFilter) ([]entity.Referral, int64, error) {
	var referrals []entity.Referral
	var count int64
	offset := (filter.Page - 1) * filter.Limit

	query := r.db.WithContext(ctx).Model(&entity.Referral{})
	query = r.applyFilter(query, filter)

	err := query.Count(&count).Limit(filter.Limit).Offset(offset).
		Preload("Patient").Preload("ReferralForm").Preload("Diagnoses").Preload("Diagnoses.CodeInfo").Find(&referrals).Error
	return referrals, count, err
}

func (r *referralRepository) GetHospitalLogsForAdmin(ctx context.Context, hospID uuid.UUID, limit, page int) ([]entity.ReferralStatusHistory, int64, error) {
	var logs []entity.ReferralStatusHistory
	var count int64
	offset := (page - 1) * limit
	query := r.db.WithContext(ctx).Model(&entity.ReferralStatusHistory{}).
		Joins("JOIN referrals ON referral_status_histories.referral_id = referrals.id").
		Where("referrals.sender_hospital_id = ? OR referrals.target_hospital_id = ?", hospID, hospID)
	err := query.Count(&count).Limit(limit).Offset(offset).Order("referral_status_histories.changed_at desc").Find(&logs).Error
	return logs, count, err
}

func (r *referralRepository) GetReferralStatusHistoryForHospital(ctx context.Context, hospID, referralID uuid.UUID, limit, page int) ([]entity.ReferralStatusHistory, int64, error) {
	var logs []entity.ReferralStatusHistory
	var count int64
	offset := (page - 1) * limit

	query := r.db.WithContext(ctx).Model(&entity.ReferralStatusHistory{}).
		Joins("JOIN referrals ON referral_status_histories.referral_id = referrals.id").
		Where("referral_status_histories.referral_id = ?", referralID).
		Where("referrals.sender_hospital_id = ? OR referrals.target_hospital_id = ?", hospID, hospID)

	err := query.Count(&count).Limit(limit).Offset(offset).Order("referral_status_histories.changed_at desc").Find(&logs).Error
	return logs, count, err
}

func (r *referralRepository) ListForDoctor(ctx context.Context, doctorID uuid.UUID, filter irepository.ReferralFilter) ([]entity.Referral, int64, error) {
	var referrals []entity.Referral
	var count int64
	offset := (filter.Page - 1) * filter.Limit

	query := r.db.WithContext(ctx).Model(&entity.Referral{}).Where("referring_doctor_id = ?", doctorID)
	query = r.applyFilter(query, filter)

	err := query.Count(&count).Limit(filter.Limit).Offset(offset).
		Preload("Patient").Preload("ReferralForm").Preload("Diagnoses").Preload("Diagnoses.CodeInfo").Find(&referrals).Error
	return referrals, count, err
}

func (r *referralRepository) ListOutgoingForLiaison(ctx context.Context, hospID uuid.UUID, filter irepository.ReferralFilter) ([]entity.Referral, int64, error) {
	var referrals []entity.Referral
	var count int64
	offset := (filter.Page - 1) * filter.Limit

	query := r.db.WithContext(ctx).Model(&entity.Referral{}).Where("sender_hospital_id = ? AND status != ?", hospID, entity.StatusDraft)
	query = r.applyFilter(query, filter)

	err := query.Count(&count).Limit(filter.Limit).Offset(offset).
		Preload("Patient").Preload("ReferralForm").Preload("Diagnoses").Preload("Diagnoses.CodeInfo").Find(&referrals).Error
	return referrals, count, err
}

func (r *referralRepository) ListIncomingForLiaison(ctx context.Context, hospID uuid.UUID, filter irepository.ReferralFilter) ([]entity.Referral, int64, error) {
	var referrals []entity.Referral
	var count int64
	offset := (filter.Page - 1) * filter.Limit

	allowedStatuses := []entity.ReferralStatus{
		entity.StatusForwarded,
		entity.StatusUnderSpecialistReview,
		entity.StatusAccepted,
		entity.StatusScheduled,
		entity.StatusAssigned,
		entity.StatusCompleted,
		entity.StatusRejectedBySpecialist,
		entity.StatusMissed,
		entity.StatusRescheduled,
	}

	query := r.db.WithContext(ctx).Model(&entity.Referral{}).
		Where("target_hospital_id = ? AND status IN ?", hospID, allowedStatuses)

	query = r.applyFilter(query, filter)

	err := query.Count(&count).Limit(filter.Limit).Offset(offset).
		Preload("Patient").Preload("ReferralForm").Preload("Diagnoses").Preload("Diagnoses.CodeInfo").Find(&referrals).Error
	return referrals, count, err
}

func (r *referralRepository) ListForSpecialist(ctx context.Context, hospID uuid.UUID, filter irepository.ReferralFilter) ([]entity.Referral, int64, error) {
	var referrals []entity.Referral
	var count int64
	offset := (filter.Page - 1) * filter.Limit
	allowedStatuses := []entity.ReferralStatus{
		entity.StatusForwarded, entity.StatusUnderSpecialistReview, entity.StatusAccepted,
		entity.StatusScheduled, entity.StatusAssigned, entity.StatusCompleted,
		entity.StatusRejectedBySpecialist, entity.StatusMissed, entity.StatusRescheduled,
	}

	query := r.db.WithContext(ctx).Model(&entity.Referral{}).
		Where("target_hospital_id = ? AND status IN ?", hospID, allowedStatuses)

	query = r.applyFilter(query, filter)

	err := query.Count(&count).Limit(filter.Limit).Offset(offset).
		Preload("Patient").Preload("ReferralForm").Preload("Diagnoses").Preload("Diagnoses.CodeInfo").Find(&referrals).Error
	return referrals, count, err
}

func (r *referralRepository) ListForReceptionist(ctx context.Context, hospID uuid.UUID, filter irepository.ReferralFilter) ([]entity.Referral, int64, error) {
	var referrals []entity.Referral
	var count int64
	offset := (filter.Page - 1) * filter.Limit
	allowedStatuses := []entity.ReferralStatus{
		entity.StatusAccepted, entity.StatusScheduled, entity.StatusAssigned,
		entity.StatusMissed, entity.StatusRescheduled,
	}
	query := r.db.WithContext(ctx).Model(&entity.Referral{}).
		Where("target_hospital_id = ? AND status IN ?", hospID, allowedStatuses)

	query = r.applyFilter(query, filter)

	err := query.Count(&count).Limit(filter.Limit).Offset(offset).
		Preload("Patient").Preload("ReferralForm").Preload("Diagnoses").Preload("Diagnoses.CodeInfo").Find(&referrals).Error
	return referrals, count, err
}
func (r *referralRepository) GetDoctorStats(ctx context.Context, doctorID uuid.UUID) (total, pending, accepted, critical int64, err error) {
	// Total
	if err := r.db.WithContext(ctx).Model(&entity.Referral{}).Where("referring_doctor_id = ?", doctorID).Count(&total).Error; err != nil {
		return 0, 0, 0, 0, err
	}

	// Pending
	pendingStatuses := []entity.ReferralStatus{
		entity.StatusSubmitted, entity.StatusUnderLiaisonReview, entity.StatusForwarded,
		entity.StatusUnderSpecialistReview, entity.StatusNeedRevision,
	}
	if err := r.db.WithContext(ctx).Model(&entity.Referral{}).Where("referring_doctor_id = ? AND status IN ?", doctorID, pendingStatuses).Count(&pending).Error; err != nil {
		return 0, 0, 0, 0, err
	}

	// Accepted
	acceptedStatuses := []entity.ReferralStatus{
		entity.StatusAccepted, entity.StatusScheduled, entity.StatusAssigned, entity.StatusCompleted,
	}
	if err := r.db.WithContext(ctx).Model(&entity.Referral{}).Where("referring_doctor_id = ? AND status IN ?", doctorID, acceptedStatuses).Count(&accepted).Error; err != nil {
		return 0, 0, 0, 0, err
	}

	// Critical
	excludedStatuses := []entity.ReferralStatus{
		entity.StatusRejectedByLiaison, entity.StatusRejectedBySpecialist, entity.StatusCancelled, entity.StatusCompleted,
	}
	if err := r.db.WithContext(ctx).Model(&entity.Referral{}).
		Joins("JOIN referral_forms ON referral_forms.referral_id = referrals.id").
		Where("referrals.referring_doctor_id = ? AND referrals.status NOT IN ? AND LOWER(referral_forms.condition_at_referral) = ?", doctorID, excludedStatuses, "critical").
		Count(&critical).Error; err != nil {
		return 0, 0, 0, 0, err
	}

	return total, pending, accepted, critical, nil
}

func (r *referralRepository) GetLatestPendingForDoctor(ctx context.Context, doctorID uuid.UUID, limit int) ([]entity.Referral, error) {
	var referrals []entity.Referral
	pendingStatuses := []entity.ReferralStatus{
		entity.StatusSubmitted, entity.StatusUnderLiaisonReview, entity.StatusForwarded,
		entity.StatusUnderSpecialistReview, entity.StatusNeedRevision,
	}
	err := r.db.WithContext(ctx).Model(&entity.Referral{}).
		Where("referring_doctor_id = ? AND status IN ?", doctorID, pendingStatuses).
		Order("created_at desc").
		Limit(limit).
		Preload("Patient").
		Preload("Diagnoses").
		Preload("Diagnoses.CodeInfo").
		Preload("ReferralForm").
		Find(&referrals).Error
	return referrals, err
}

// ─────────────────────────────────────────────────────────────────────────────
// ReferralOutcomeRepository
// Persists the final clinical outcome when a referral episode is complete.
// ─────────────────────────────────────────────────────────────────────────────

type referralOutcomeRepository struct {
	*BaseRepository[entity.ReferralOutcome]
	db *gorm.DB
}

func NewReferralOutcomeRepository(db *gorm.DB) irepository.ReferralOutcomeRepository {
	return &referralOutcomeRepository{
		BaseRepository: NewBaseRepository[entity.ReferralOutcome](db),
		db:             db,
	}
}

func (r *referralOutcomeRepository) GetByReferralID(ctx context.Context, referralID uuid.UUID) (*entity.ReferralOutcome, error) {
	var outcome entity.ReferralOutcome
	err := r.db.WithContext(ctx).Where("referral_id = ?", referralID).First(&outcome).Error
	return &outcome, err
}

// ─────────────────────────────────────────────────────────────────────────────
// ReferralRedirectionRepository
// Records when a referral was redirected to a different hospital or department.
// ─────────────────────────────────────────────────────────────────────────────

type referralRedirectionRepository struct {
	*BaseRepository[entity.ReferralRedirection]
	db *gorm.DB
}

func NewReferralRedirectionRepository(db *gorm.DB) irepository.ReferralRedirectionRepository {
	return &referralRedirectionRepository{
		BaseRepository: NewBaseRepository[entity.ReferralRedirection](db),
		db:             db,
	}
}

func (r *referralRedirectionRepository) GetByReferralID(ctx context.Context, referralID uuid.UUID) (*entity.ReferralRedirection, error) {
	var redirection entity.ReferralRedirection
	err := r.db.WithContext(ctx).Where("referral_id = ?", referralID).First(&redirection).Error
	return &redirection, err
}

// ─────────────────────────────────────────────────────────────────────────────
// ReferralAccessRepository
// Tracks which doctors have been granted read/update access to a referral
// (treating doctor vs. consulted doctor).
// ─────────────────────────────────────────────────────────────────────────────

type referralAccessRepository struct {
	*BaseRepository[entity.ReferralAccess]
	db *gorm.DB
}

func NewReferralAccessRepository(db *gorm.DB) irepository.ReferralAccessRepository {
	return &referralAccessRepository{
		BaseRepository: NewBaseRepository[entity.ReferralAccess](db),
		db:             db,
	}
}

func (r *referralAccessRepository) GetAccess(ctx context.Context, referralID, userID uuid.UUID) (*entity.ReferralAccess, error) {
	var access entity.ReferralAccess
	err := r.db.WithContext(ctx).Where("referral_id = ? AND user_id = ? AND revoked_at IS NULL", referralID, userID).First(&access).Error
	return &access, err
}

func (r *referralAccessRepository) CheckAccess(ctx context.Context, referralID, userID uuid.UUID) (bool, error) {
	// 1. Check if user is the referring doctor or assigned specialist
	var ref entity.Referral
	if err := r.db.WithContext(ctx).Select("id, referring_doctor_id, specialist_id").Where("id = ?", referralID).First(&ref).Error; err != nil {
		return false, err
	}

	if ref.ReferringDoctorID == userID || (ref.SpecialistID != nil && *ref.SpecialistID == userID) {
		return true, nil
	}

	// 2. Check for explicit active access grant
	var count int64
	if err := r.db.WithContext(ctx).Model(&entity.ReferralAccess{}).
		Where("referral_id = ? AND user_id = ? AND revoked_at IS NULL", referralID, userID).
		Count(&count).Error; err != nil {
		return false, err
	}

	return count > 0, nil
}
