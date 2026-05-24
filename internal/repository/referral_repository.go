package repository

import (
	"context"
	"errors"

	"math"
	"time"

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

func preloadReferralListRelations(query *gorm.DB) *gorm.DB {
	return query.
		Preload("Patient").
		Preload("ReferralForm").
		Preload("Diagnoses").
		Preload("Diagnoses.CodeInfo").
		Preload("TargetDepartment")
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
		Preload("Redirections").
		Preload("Redirections.RedirectedFromHospital").
		Preload("Redirections.RedirectedToHospital").
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
		Preload("EmergencyDetail").
		Where("is_archived = false")

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
	query = query.Where("is_archived = false")

	if len(filter.Statuses) > 0 {
		query = query.Where("status IN ?", filter.Statuses)
	} else if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}

	if filter.PatientID != nil {
		query = query.Where("patient_id = ?", *filter.PatientID)
	}

	if filter.Region != "" {
		// Joins to Patient to filter by region
		query = query.Joins("Patient").Where("\"Patient\".home_region = ?", filter.Region)
	}

	if len(filter.ReferralIDs) > 0 {
		query = query.Where("id IN ?", filter.ReferralIDs)
	}

	// Dynamic Sorting
	sortBy := "created_at"
	if filter.SortBy == "updated_at" {
		sortBy = "updated_at"
	}
	sortOrder := "desc"
	if filter.SortOrder == "asc" {
		sortOrder = "asc"
	}
	query = query.Order(sortBy + " " + sortOrder)

	return query
}

func (r *referralRepository) applyMohReferralFilter(query *gorm.DB, filter irepository.MohAnalyticsFilter, joinPatients bool) *gorm.DB {
	query = query.Where("referrals.is_archived = false")

	if filter.From != nil {
		query = query.Where("referrals.created_at >= ?", *filter.From)
	}
	if filter.To != nil {
		query = query.Where("referrals.created_at <= ?", *filter.To)
	}
	if filter.HospitalID != nil {
		query = query.Where("(referrals.sender_hospital_id = ? OR referrals.target_hospital_id = ?)", *filter.HospitalID, *filter.HospitalID)
	}
	if joinPatients && filter.Region != nil && *filter.Region != "" {
		query = query.Joins("JOIN patients ON patients.id = referrals.patient_id").Where("patients.home_region = ?", *filter.Region)
	}

	return query
}

func (r *referralRepository) ListForSystemAdmin(ctx context.Context, filter irepository.ReferralFilter) ([]entity.Referral, int64, error) {
	var referrals []entity.Referral
	var count int64
	offset := (filter.Page - 1) * filter.Limit

	query := r.db.WithContext(ctx).Model(&entity.Referral{})
	query = r.applyFilter(query, filter)

	err := preloadReferralListRelations(query.Count(&count).Limit(filter.Limit).Offset(offset)).Find(&referrals).Error
	return referrals, count, err
}

func (r *referralRepository) listHospitalAdminReferrals(ctx context.Context, hospID uuid.UUID, filter irepository.ReferralFilter, scope string, statuses []entity.ReferralStatus) ([]entity.Referral, int64, error) {
	var referrals []entity.Referral
	var count int64
	offset := (filter.Page - 1) * filter.Limit

	query := r.db.WithContext(ctx).Model(&entity.Referral{})
	switch scope {
	case "inbound":
		query = query.Where("target_hospital_id = ?", hospID)
	case "outbound":
		query = query.Where("sender_hospital_id = ? AND status != ?", hospID, entity.StatusDraft)
	default:
		query = query.Where("(sender_hospital_id = ? OR target_hospital_id = ?)", hospID, hospID)
	}
	if len(statuses) > 0 {
		query = query.Where("status IN ?", statuses)
	}
	query = r.applyFilter(query, filter)

	err := preloadReferralListRelations(query.Count(&count).Limit(filter.Limit).Offset(offset)).Find(&referrals).Error
	return referrals, count, err
}

func (r *referralRepository) ListInboundForHospitalAdmin(ctx context.Context, hospID uuid.UUID, filter irepository.ReferralFilter) ([]entity.Referral, int64, error) {
	return r.listHospitalAdminReferrals(ctx, hospID, filter, "inbound", nil)
}

func (r *referralRepository) ListOutboundForHospitalAdmin(ctx context.Context, hospID uuid.UUID, filter irepository.ReferralFilter) ([]entity.Referral, int64, error) {
	return r.listHospitalAdminReferrals(ctx, hospID, filter, "outbound", nil)
}

func (r *referralRepository) ListByStatusesForHospitalAdmin(ctx context.Context, hospID uuid.UUID, filter irepository.ReferralFilter, statuses []entity.ReferralStatus) ([]entity.Referral, int64, error) {
	return r.listHospitalAdminReferrals(ctx, hospID, filter, "both", statuses)
}

func (r *referralRepository) GetDetailsForHospitalAdmin(ctx context.Context, hospID, referralID uuid.UUID) (*entity.Referral, error) {
	var referral entity.Referral
	err := r.db.WithContext(ctx).
		Preload("Patient").
		Preload("ReferralForm").
		Preload("Diagnoses").
		Preload("Diagnoses.CodeInfo").
		Preload("Vitals").
		Preload("EmergencyDetail").
		Preload("Attachments").
		Preload("Redirections").
		Preload("Redirections.RedirectedFromHospital").
		Preload("Redirections.RedirectedToHospital").
		Where("id = ?", referralID).
		Where("(sender_hospital_id = ? OR target_hospital_id = ?)", hospID, hospID).
		First(&referral).Error
	if err != nil {
		return nil, err
	}
	return &referral, nil
}

func (r *referralRepository) GetReferralStatusCounts(ctx context.Context, hospID uuid.UUID) ([]irepository.ReferralStatusCount, error) {
	var rows []irepository.ReferralStatusCount
	err := r.db.WithContext(ctx).
		Model(&entity.Referral{}).
		Select("status, COUNT(*) as count").
		Where("(sender_hospital_id = ? OR target_hospital_id = ?) AND is_archived = false", hospID, hospID).
		Group("status").
		Scan(&rows).Error
	return rows, err
}

func (r *referralRepository) GetMonthlyReferralTotalsForHospitalAdmin(ctx context.Context, hospID uuid.UUID, months int) ([]irepository.MonthlyReferralTotal, error) {
	if months <= 0 {
		months = 6
	}
	cutoff := time.Now().AddDate(0, -months+1, 0)
	type row struct {
		Month time.Time
		Count int64
	}
	var rows []row
	err := r.db.WithContext(ctx).
		Model(&entity.Referral{}).
		Select("DATE_TRUNC('month', created_at) as month, COUNT(*) as count").
		Where("(sender_hospital_id = ? OR target_hospital_id = ?) AND created_at >= ?", hospID, hospID, cutoff).
		Group("month").
		Order("month ASC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	resp := make([]irepository.MonthlyReferralTotal, 0, len(rows))
	for _, rw := range rows {
		resp = append(resp, irepository.MonthlyReferralTotal{
			Month: rw.Month.Format("2006-01"),
			Count: rw.Count,
		})
	}
	return resp, nil
}

func (r *referralRepository) GetAcceptanceRejectionRateForHospitalAdmin(ctx context.Context, hospID uuid.UUID) (float64, float64, error) {
	var accepted int64
	var rejected int64

	err := r.db.WithContext(ctx).Model(&entity.Referral{}).
		Where("(sender_hospital_id = ? OR target_hospital_id = ?) AND status = ?", hospID, hospID, entity.StatusAccepted).
		Count(&accepted).Error
	if err != nil {
		return 0, 0, err
	}
	err = r.db.WithContext(ctx).Model(&entity.Referral{}).
		Where("(sender_hospital_id = ? OR target_hospital_id = ?) AND status IN ?", hospID, hospID, []entity.ReferralStatus{
			entity.StatusRejectedByLiaison,
			entity.StatusRejectedBySpecialist,
		}).
		Count(&rejected).Error
	if err != nil {
		return 0, 0, err
	}

	total := accepted + rejected
	if total == 0 {
		return 0, 0, nil
	}
	return (float64(accepted) / float64(total)) * 100, (float64(rejected) / float64(total)) * 100, nil
}

func (r *referralRepository) GetMissedAppointmentRateForHospitalAdmin(ctx context.Context, hospID uuid.UUID) (float64, error) {
	var missed int64
	var tracked int64

	err := r.db.WithContext(ctx).Model(&entity.Referral{}).
		Where("target_hospital_id = ? AND status = 'COMPLETED' AND is_archived = true", hospID).
		Count(&missed).Error
	if err != nil {
		return 0, err
	}

	err = r.db.WithContext(ctx).Model(&entity.Referral{}).
		Where("target_hospital_id = ? AND status IN ?", hospID, []entity.ReferralStatus{
			entity.StatusScheduled,
			entity.StatusCompleted,
		}).
		Count(&tracked).Error
	if err != nil {
		return 0, err
	}

	if tracked == 0 {
		return 0, nil
	}
	return (float64(missed) / float64(tracked)) * 100, nil
}

func (r *referralRepository) GetBusiestDepartmentsForHospitalAdmin(ctx context.Context, hospID uuid.UUID, limit int) ([]irepository.DepartmentReferralLoad, error) {
	if limit <= 0 {
		limit = 5
	}
	var rows []irepository.DepartmentReferralLoad
	err := r.db.WithContext(ctx).Model(&entity.Referral{}).
		Select("target_dept_id as department_id, COUNT(*) as count").
		Where("target_hospital_id = ?", hospID).
		Group("target_dept_id").
		Order("count DESC").
		Limit(limit).
		Scan(&rows).Error
	return rows, err
}

func (r *referralRepository) GetAverageWaitTimeForHospitalAdmin(ctx context.Context, hospID uuid.UUID) (float64, error) {
	type row struct {
		Avg float64
	}
	var rw row
	err := r.db.WithContext(ctx).Model(&entity.Referral{}).
		Select("COALESCE(AVG(waiting_hours_weight), 0) as avg").
		Where("sender_hospital_id = ? OR target_hospital_id = ?", hospID, hospID).
		Scan(&rw).Error
	if err != nil {
		return 0, err
	}
	// keep to 2 dp for API readability
	return math.Round(rw.Avg*100) / 100, nil
}

func (r *referralRepository) GetTopReferringHospitalsForHospitalAdmin(ctx context.Context, hospID uuid.UUID, limit int) ([]irepository.ReferringHospitalCount, error) {
	if limit <= 0 {
		limit = 5
	}
	var rows []irepository.ReferringHospitalCount
	err := r.db.WithContext(ctx).Model(&entity.Referral{}).
		Select("sender_hospital_id as hospital_id, hospitals.name as hospital_name, COUNT(*) as count").
		Joins("JOIN hospitals ON hospitals.id = referrals.sender_hospital_id").
		Where("referrals.target_hospital_id = ?", hospID).
		Group("sender_hospital_id, hospitals.name").
		Order("count DESC").
		Limit(limit).
		Scan(&rows).Error
	return rows, err
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

	query := r.db.WithContext(ctx).Model(&entity.Referral{})
	if len(filter.ReferralIDs) == 0 {
		query = query.Where("referring_doctor_id = ?", doctorID)
	}
	query = r.applyFilter(query, filter)

	err := preloadReferralListRelations(query.Count(&count).Limit(filter.Limit).Offset(offset)).Find(&referrals).Error
	return referrals, count, err
}

func (r *referralRepository) ListOutgoingForLiaison(ctx context.Context, hospID uuid.UUID, filter irepository.ReferralFilter) ([]entity.Referral, int64, error) {
	var referrals []entity.Referral
	var count int64
	offset := (filter.Page - 1) * filter.Limit

	query := r.db.WithContext(ctx).Model(&entity.Referral{}).Where("sender_hospital_id = ? AND status != ?", hospID, entity.StatusDraft)
	query = r.applyFilter(query, filter)

	err := preloadReferralListRelations(query.Count(&count).Limit(filter.Limit).Offset(offset)).Find(&referrals).Error
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
		entity.StatusCompleted,
		entity.StatusRejectedBySpecialist,
		entity.StatusRedirected,
		entity.StatusRejectedAfterSend,
	}

	query := r.db.WithContext(ctx).Model(&entity.Referral{}).
		Where("target_hospital_id = ? AND status IN ?", hospID, allowedStatuses)

	query = r.applyFilter(query, filter)

	err := preloadReferralListRelations(query.Count(&count).Limit(filter.Limit).Offset(offset)).Find(&referrals).Error
	return referrals, count, err
}

func (r *referralRepository) ListForSpecialist(ctx context.Context, hospID uuid.UUID, filter irepository.ReferralFilter) ([]entity.Referral, int64, error) {
	var referrals []entity.Referral
	var count int64
	offset := (filter.Page - 1) * filter.Limit
	allowedStatuses := []entity.ReferralStatus{
		entity.StatusForwarded, entity.StatusUnderSpecialistReview, entity.StatusAccepted,
		entity.StatusRejectedBySpecialist,
		entity.StatusRedirected, entity.StatusRejectedAfterSend,
	}

	query := r.db.WithContext(ctx).Model(&entity.Referral{}).
		Where("target_hospital_id = ? AND status IN ?", hospID, allowedStatuses)

	query = r.applyFilter(query, filter)

	err := preloadReferralListRelations(query.Count(&count).Limit(filter.Limit).Offset(offset)).Find(&referrals).Error
	return referrals, count, err
}

func (r *referralRepository) ListForReceptionist(ctx context.Context, hospID uuid.UUID, filter irepository.ReferralFilter) ([]entity.Referral, int64, error) {
	var referrals []entity.Referral
	var count int64
	offset := (filter.Page - 1) * filter.Limit
	allowedStatuses := []entity.ReferralStatus{
		entity.StatusAccepted, entity.StatusScheduled,
	}
	query := r.db.WithContext(ctx).Model(&entity.Referral{}).
		Where("target_hospital_id = ? AND status IN ?", hospID, allowedStatuses)

	query = r.applyFilter(query, filter)

	err := preloadReferralListRelations(query.Count(&count).Limit(filter.Limit).Offset(offset)).Find(&referrals).Error
	return referrals, count, err
}

func (r *referralRepository) UpdateTargetDeptAndStatus(ctx context.Context, referralID, targetHospID, targetDeptID uuid.UUID, status entity.ReferralStatus) error {
	return r.db.WithContext(ctx).Model(&entity.Referral{}).
		Where("id = ?", referralID).
		Updates(map[string]interface{}{
			"target_hospital_id": targetHospID,
			"target_dept_id":     targetDeptID,
			"status":             status,
			"specialist_id":      nil,
		}).Error
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
		entity.StatusAccepted, entity.StatusScheduled, entity.StatusCompleted,
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
	err := preloadReferralListRelations(r.db.WithContext(ctx).Model(&entity.Referral{}).
		Where("referring_doctor_id = ? AND status IN ?", doctorID, pendingStatuses).
		Order("created_at desc").
		Limit(limit)).Find(&referrals).Error
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

func (r *referralRedirectionRepository) ListByReferralID(ctx context.Context, referralID uuid.UUID) ([]entity.ReferralRedirection, error) {
	var redirections []entity.ReferralRedirection
	err := r.db.WithContext(ctx).
		Where("referral_id = ?", referralID).
		Preload("RedirectedFromHospital").
		Preload("RedirectedToHospital").
		Order("created_at ASC").
		Find(&redirections).Error
	return redirections, err
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

func (r *referralAccessRepository) ListActiveByReferral(ctx context.Context, referralID uuid.UUID) ([]entity.ReferralAccess, error) {
	var accesses []entity.ReferralAccess
	err := r.db.WithContext(ctx).Where("referral_id = ? AND revoked_at IS NULL", referralID).Find(&accesses).Error
	return accesses, err
}

func (r *referralAccessRepository) RevokeAllByReferral(ctx context.Context, referralID uuid.UUID, reason string) error {
	return r.db.WithContext(ctx).Model(&entity.ReferralAccess{}).
		Where("referral_id = ? AND revoked_at IS NULL", referralID).
		Updates(map[string]interface{}{
			"revoked_at":    time.Now(),
			"revoke_reason": reason,
		}).Error
}

func (r *referralAccessRepository) ListByDoctor(ctx context.Context, doctorID uuid.UUID) ([]entity.ReferralAccess, error) {
	var accesses []entity.ReferralAccess
	err := r.db.WithContext(ctx).Where("user_id = ?", doctorID).Order("granted_at DESC").Find(&accesses).Error
	return accesses, err
}

func (r *referralAccessRepository) GetActiveAccess(ctx context.Context, referralID, userID uuid.UUID) (*entity.ReferralAccess, error) {
	var access entity.ReferralAccess
	err := r.db.WithContext(ctx).Where("referral_id = ? AND user_id = ? AND revoked_at IS NULL", referralID, userID).First(&access).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &access, nil
}



func (r *referralRepository) CountBySenderHospitalAndStatuses(
	ctx context.Context,
	hospID uuid.UUID,
	statuses []entity.ReferralStatus,
	excludeDraft bool,
	startDate, endDate *time.Time,
) (int64, error) {
	query := r.db.WithContext(ctx).Model(&entity.Referral{}).
		Where("sender_hospital_id = ? AND is_archived = false", hospID)

	if len(statuses) > 0 {
		query = query.Where("status IN ?", statuses)
	}

	if excludeDraft {
		query = query.Where("status != ?", entity.StatusDraft)
	}

	if startDate != nil {
		query = query.Where("created_at >= ?", *startDate)
	}
	if endDate != nil {
		query = query.Where("created_at <= ?", *endDate)
	}

	var count int64
	if err := query.Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

func (r *referralRepository) CountAcceptedOrCompletedToday(ctx context.Context, hospID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Table("referrals").
		Where("sender_hospital_id = ? AND is_archived = false", hospID).
		Where("id IN (?)",
			r.db.WithContext(ctx).
				Table("referral_status_histories").
				Select("referral_id").
				Where("to_status IN ?", []entity.ReferralStatus{
					entity.StatusAccepted,
					entity.StatusCompleted,
				}).
				Where("CAST(changed_at AS DATE) = CURRENT_DATE"),
		).
		Count(&count).Error
	return count, err
}

// CountByTargetDeptAndStatuses returns one ReferralStatusCount per requested
// status for inbound referrals at (target hospital, target department).
// Archived rows are excluded. If startDate / endDate are non-nil they bound
// referrals.created_at. Statuses that yield zero rows still appear in the
// result with Count = 0 so the front-end can render every bucket
// deterministically.
func (r *referralRepository) CountByTargetDeptAndStatuses(
	ctx context.Context,
	hospID, deptID uuid.UUID,
	statuses []entity.ReferralStatus,
	startDate, endDate *time.Time,
) ([]irepository.ReferralStatusCount, error) {
	if len(statuses) == 0 {
		return []irepository.ReferralStatusCount{}, nil
	}

	query := r.db.WithContext(ctx).Model(&entity.Referral{}).
		Where("target_hospital_id = ? AND target_dept_id = ? AND is_archived = false", hospID, deptID).
		Where("status IN ?", statuses)

	if startDate != nil {
		query = query.Where("created_at >= ?", *startDate)
	}
	if endDate != nil {
		query = query.Where("created_at <= ?", *endDate)
	}

	type row struct {
		Status string
		Count  int64
	}
	var rows []row
	if err := query.Select("status, COUNT(*) AS count").Group("status").Scan(&rows).Error; err != nil {
		return nil, err
	}

	got := make(map[entity.ReferralStatus]int64, len(rows))
	for _, r := range rows {
		got[entity.ReferralStatus(r.Status)] = r.Count
	}

	out := make([]irepository.ReferralStatusCount, 0, len(statuses))
	for _, s := range statuses {
		out = append(out, irepository.ReferralStatusCount{Status: s, Count: got[s]})
	}
	return out, nil
}

func (r *referralRepository) UpdateFields(ctx context.Context, referralID uuid.UUID, updates irepository.ReferralUpdateFields) error {
	return r.db.WithContext(ctx).Model(&entity.Referral{}).Where("id = ?", referralID).Updates(updates).Error
}

func (r *referralRepository) GetMohDashboardSummary(ctx context.Context, filter irepository.MohAnalyticsFilter) (*irepository.MohDashboardSummary, error) {
	type row struct {
		TotalReferrals      int64
		TotalAccepted       int64
		TotalRejected       int64
		TotalAdmitted       int64
		AverageMLSeverity   float64
		AverageTurnaroundHr float64
	}

	var rw row
	query := r.db.WithContext(ctx).Model(&entity.Referral{}).
		Select(`
			COUNT(*) AS total_referrals,
			COUNT(*) FILTER (WHERE referrals.status = ?) AS total_accepted,
			COUNT(*) FILTER (WHERE referrals.status IN ?) AS total_rejected,
			COUNT(*) FILTER (WHERE admitted.admitted_at IS NOT NULL) AS total_admitted,
			COALESCE(AVG(referrals.ml_severity_score), 0) AS average_ml_severity,
			COALESCE(AVG(EXTRACT(EPOCH FROM (admitted.admitted_at - referrals.created_at))/3600), 0) AS average_turnaround_hr
		`,
			entity.StatusAccepted,
			[]entity.ReferralStatus{
				entity.StatusRejectedByLiaison,
				entity.StatusRejectedBySpecialist,
				entity.StatusRejectedAfterSend,
			},
		).
		Joins(`
			LEFT JOIN (
				SELECT referral_id, MIN(changed_at) AS admitted_at
				FROM referral_status_histories
				WHERE to_status = ?
				GROUP BY referral_id
			) admitted ON admitted.referral_id = referrals.id
		`, entity.StatusCompleted) // Replaced StatusAdmitted with StatusCompleted

	query = r.applyMohReferralFilter(query, filter, true)
	if err := query.Scan(&rw).Error; err != nil {
		return nil, err
	}

	acceptanceRate := 0.0
	if rw.TotalReferrals > 0 {
		acceptanceRate = (float64(rw.TotalAccepted) / float64(rw.TotalReferrals)) * 100
	}

	return &irepository.MohDashboardSummary{
		TotalReferrals:      rw.TotalReferrals,
		TotalAccepted:       rw.TotalAccepted,
		TotalRejected:       rw.TotalRejected,
		TotalAdmitted:       rw.TotalAdmitted,
		AcceptanceRate:      math.Round(acceptanceRate*100) / 100,
		AverageMLSeverity:   math.Round(rw.AverageMLSeverity*100) / 100,
		AverageTurnaroundHr: math.Round(rw.AverageTurnaroundHr*100) / 100,
	}, nil
}

func (r *referralRepository) GetMohReferralTrends(ctx context.Context, filter irepository.MohAnalyticsFilter, granularity string) ([]irepository.MohReferralTrendPoint, error) {
	type row struct {
		Period             time.Time
		TotalReferrals     int64
		AcceptedReferrals  int64
		RejectedReferrals  int64
		EmergencyReferrals int64
	}

	var rows []row
	query := r.db.WithContext(ctx).Model(&entity.Referral{}).
		Select(`
			DATE_TRUNC(?, referrals.created_at) AS period,
			COUNT(*) AS total_referrals,
			COUNT(*) FILTER (WHERE referrals.status = ?) AS accepted_referrals,
			COUNT(*) FILTER (WHERE referrals.status IN ?) AS rejected_referrals,
			COUNT(*) FILTER (
				WHERE LOWER(COALESCE(referral_forms.reason_for_referral_category, '')) = 'emergency'
				   OR LOWER(COALESCE(referral_forms.condition_at_referral, '')) = 'critical'
			) AS emergency_referrals
		`,
			granularity,
			entity.StatusAccepted,
			[]entity.ReferralStatus{
				entity.StatusRejectedByLiaison,
				entity.StatusRejectedBySpecialist,
				entity.StatusRejectedAfterSend,
			},
		).
		Joins("LEFT JOIN referral_forms ON referral_forms.referral_id = referrals.id")

	query = r.applyMohReferralFilter(query, filter, true)
	if err := query.Group("period").Order("period ASC").Scan(&rows).Error; err != nil {
		return nil, err
	}

	resp := make([]irepository.MohReferralTrendPoint, 0, len(rows))
	for _, rw := range rows {
		period := rw.Period.Format("2006-01")
		switch granularity {
		case "day":
			period = rw.Period.Format("2006-01-02")
		case "week":
			period = rw.Period.Format("2006-01-02")
		}
		resp = append(resp, irepository.MohReferralTrendPoint{
			Period:             period,
			TotalReferrals:     rw.TotalReferrals,
			AcceptedReferrals:  rw.AcceptedReferrals,
			RejectedReferrals:  rw.RejectedReferrals,
			EmergencyReferrals: rw.EmergencyReferrals,
		})
	}

	return resp, nil
}

func (r *referralRepository) GetMohHospitalLoad(ctx context.Context, filter irepository.MohAnalyticsFilter) ([]irepository.MohHospitalLoadMetric, error) {
	type row struct {
		HospitalID      uuid.UUID
		HospitalName    string
		TierLevel       entity.HospitalTier
		Region          string
		TotalReceived   int64
		TotalAccepted   int64
		TotalRejected   int64
		RejectionRate   float64
		AverageSeverity float64
	}

	var rows []row
	query := r.db.WithContext(ctx).Model(&entity.Referral{}).
		Select(`
			hospitals.id AS hospital_id,
			hospitals.name AS hospital_name,
			hospitals.tier_level AS tier_level,
			hospitals.region AS region,
			COUNT(*) AS total_received,
			COUNT(*) FILTER (WHERE referrals.status = ?) AS total_accepted,
			COUNT(*) FILTER (WHERE referrals.status IN ?) AS total_rejected,
			CASE
				WHEN COUNT(*) = 0 THEN 0
				ELSE (COUNT(*) FILTER (WHERE referrals.status IN ?) * 100.0) / COUNT(*)
			END AS rejection_rate,
			COALESCE(AVG(referrals.ml_severity_score), 0) AS average_severity
		`,
			entity.StatusAccepted,
			[]entity.ReferralStatus{
				entity.StatusRejectedByLiaison,
				entity.StatusRejectedBySpecialist,
				entity.StatusRejectedAfterSend,
			},
			[]entity.ReferralStatus{
				entity.StatusRejectedByLiaison,
				entity.StatusRejectedBySpecialist,
				entity.StatusRejectedAfterSend,
			},
		).
		Joins("JOIN hospitals ON hospitals.id = referrals.target_hospital_id").
		Where("referrals.is_archived = false")

	if filter.From != nil {
		query = query.Where("referrals.created_at >= ?", *filter.From)
	}
	if filter.To != nil {
		query = query.Where("referrals.created_at <= ?", *filter.To)
	}
	if filter.Region != nil && *filter.Region != "" {
		query = query.Where("hospitals.region = ?", *filter.Region)
	}
	if filter.TierLevel != nil {
		query = query.Where("hospitals.tier_level = ?", *filter.TierLevel)
	}
	if filter.HospitalID != nil {
		query = query.Where("hospitals.id = ?", *filter.HospitalID)
	}

	if err := query.Group("hospitals.id, hospitals.name, hospitals.tier_level, hospitals.region").
		Order("total_received DESC").
		Scan(&rows).Error; err != nil {
		return nil, err
	}

	resp := make([]irepository.MohHospitalLoadMetric, 0, len(rows))
	for _, rw := range rows {
		resp = append(resp, irepository.MohHospitalLoadMetric{
			HospitalID:      rw.HospitalID,
			HospitalName:    rw.HospitalName,
			TierLevel:       rw.TierLevel,
			Region:          rw.Region,
			TotalReceived:   rw.TotalReceived,
			TotalAccepted:   rw.TotalAccepted,
			TotalRejected:   rw.TotalRejected,
			RejectionRate:   math.Round(rw.RejectionRate*100) / 100,
			AverageSeverity: math.Round(rw.AverageSeverity*100) / 100,
		})
	}

	return resp, nil
}

func (r *referralRepository) GetMohDiseaseHotspots(ctx context.Context, filter irepository.MohAnalyticsFilter) ([]irepository.MohDiseaseHotspot, error) {
	type row struct {
		Region          string
		DepartmentName  string
		ReferralCount   int64
		AverageSeverity float64
	}

	var rows []row
	query := r.db.WithContext(ctx).Model(&entity.Referral{}).
		Select(`
			COALESCE(patients.home_region, 'Unknown') AS region,
			departments.name AS department_name,
			COUNT(*) AS referral_count,
			COALESCE(AVG(referrals.ml_severity_score), 0) AS average_severity
		`).
		Joins("JOIN patients ON patients.id = referrals.patient_id").
		Joins("JOIN departments ON departments.id = referrals.target_dept_id")

	query = r.applyMohReferralFilter(query, filter, false)
	if filter.Region != nil && *filter.Region != "" {
		query = query.Where("patients.home_region = ?", *filter.Region)
	}

	if err := query.Group("region, departments.name").Order("referral_count DESC").Scan(&rows).Error; err != nil {
		return nil, err
	}

	resp := make([]irepository.MohDiseaseHotspot, 0, len(rows))
	for _, rw := range rows {
		resp = append(resp, irepository.MohDiseaseHotspot{
			Region:          rw.Region,
			DepartmentName:  rw.DepartmentName,
			ReferralCount:   rw.ReferralCount,
			AverageSeverity: math.Round(rw.AverageSeverity*100) / 100,
		})
	}

	return resp, nil
}

func (r *referralRepository) GetMohSeverityDistribution(ctx context.Context, filter irepository.MohAnalyticsFilter) ([]irepository.MohSeverityDistribution, error) {
	type row struct {
		Region         string
		CriticalCount  int64
		UrgentCount    int64
		RoutineCount   int64
		TotalReferrals int64
	}

	var rows []row
	query := r.db.WithContext(ctx).Model(&entity.Referral{}).
		Select(`
			COALESCE(patients.home_region, 'Unknown') AS region,
			COUNT(*) FILTER (WHERE COALESCE(referrals.ml_severity_score, 0) >= 67) AS critical_count,
			COUNT(*) FILTER (WHERE COALESCE(referrals.ml_severity_score, 0) >= 34 AND COALESCE(referrals.ml_severity_score, 0) < 67) AS urgent_count,
			COUNT(*) FILTER (WHERE COALESCE(referrals.ml_severity_score, 0) < 34) AS routine_count,
			COUNT(*) AS total_referrals
		`).
		Joins("JOIN patients ON patients.id = referrals.patient_id")

	query = r.applyMohReferralFilter(query, filter, false)
	if filter.Region != nil && *filter.Region != "" {
		query = query.Where("patients.home_region = ?", *filter.Region)
	}

	if err := query.Group("region").Order("critical_count DESC").Scan(&rows).Error; err != nil {
		return nil, err
	}

	resp := make([]irepository.MohSeverityDistribution, 0, len(rows))
	for _, rw := range rows {
		resp = append(resp, irepository.MohSeverityDistribution{
			Region:         rw.Region,
			CriticalCount:  rw.CriticalCount,
			UrgentCount:    rw.UrgentCount,
			RoutineCount:   rw.RoutineCount,
			TotalReferrals: rw.TotalReferrals,
		})
	}

	return resp, nil
}
