package repository

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"Hospital-Referral-System/internal/domain/entity"
	irepository "Hospital-Referral-System/internal/domain/interfaces/repository"
)

type referralRepository struct {
	db *gorm.DB
}

func NewReferralRepository(db *gorm.DB) irepository.ReferralRepository {
	return &referralRepository{db: db}
}

func (r *referralRepository) CreateReferralTransaction(ctx context.Context, referral *entity.Referral) error {
	// GORM automatically manages the transaction for associated creations
	// Patient, ReferralForm, and Diagnoses will be inserted atomically.
	return r.db.WithContext(ctx).Create(referral).Error
}

func (r *referralRepository) UpdateReferralTransaction(ctx context.Context, referral *entity.Referral) error {
	// Use a transaction to safely clear old relations (like diagnoses/vitals) and insert new ones
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Replace associated slice collections entirely to avoid orphans
		if err := tx.Model(referral).Association("Diagnoses").Replace(referral.Diagnoses); err != nil {
			return err
		}
		if err := tx.Model(referral).Association("Vitals").Replace(referral.Vitals); err != nil {
			return err
		}

		// Save the parent and one-to-one models
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

func (r *referralRepository) ListForSystemAdmin(ctx context.Context, limit, offset int, statusFilter string) ([]entity.Referral, int64, error) {
	var referrals []entity.Referral
	var count int64
	query := r.db.WithContext(ctx).Model(&entity.Referral{})
	if statusFilter != "" {
		query = query.Where("status = ?", statusFilter)
	}
	err := query.Count(&count).Limit(limit).Offset(offset).Order("created_at desc").
		Preload("Patient").Preload("Diagnoses").Preload("Diagnoses.CodeInfo").Find(&referrals).Error
	return referrals, count, err
}

func (r *referralRepository) GetHospitalLogsForAdmin(ctx context.Context, hospID uuid.UUID, limit, offset int) ([]entity.ReferralStatusHistory, int64, error) {
	var logs []entity.ReferralStatusHistory
	var count int64
	query := r.db.WithContext(ctx).Model(&entity.ReferralStatusHistory{}).
		Joins("JOIN referrals ON referral_status_histories.referral_id = referrals.id").
		Where("referrals.sender_hospital_id = ? OR referrals.target_hospital_id = ?", hospID, hospID)
	err := query.Count(&count).Limit(limit).Offset(offset).Order("created_at desc").Find(&logs).Error
	return logs, count, err
}

func (r *referralRepository) ListForDoctor(ctx context.Context, doctorID uuid.UUID, limit, offset int, statusFilter string) ([]entity.Referral, int64, error) {
	var referrals []entity.Referral
	var count int64
	query := r.db.WithContext(ctx).Model(&entity.Referral{}).Where("referring_doctor_id = ?", doctorID)
	if statusFilter != "" {
		query = query.Where("status = ?", statusFilter)
	}
	err := query.Count(&count).Limit(limit).Offset(offset).Order("created_at desc").
		Preload("Patient").Preload("Diagnoses").Preload("Diagnoses.CodeInfo").Find(&referrals).Error
	return referrals, count, err
}

func (r *referralRepository) ListForLiaison(ctx context.Context, hospID uuid.UUID, limit, offset int, statusFilter string) ([]entity.Referral, int64, error) {
	var referrals []entity.Referral
	var count int64
	query := r.db.WithContext(ctx).Model(&entity.Referral{}).Where("sender_hospital_id = ? AND status != ?", hospID, entity.StatusDraft)
	if statusFilter != "" {
		query = query.Where("status = ?", statusFilter)
	}
	err := query.Count(&count).Limit(limit).Offset(offset).Order("created_at desc").
		Preload("Patient").Preload("Diagnoses").Preload("Diagnoses.CodeInfo").Find(&referrals).Error
	return referrals, count, err
}

func (r *referralRepository) ListForSpecialist(ctx context.Context, hospID uuid.UUID, specialistID uuid.UUID, limit, offset int, statusFilter string) ([]entity.Referral, int64, error) {
	var referrals []entity.Referral
	var count int64
	allowedStatuses := []entity.ReferralStatus{
		entity.StatusForwarded, entity.StatusUnderSpecialistReview, entity.StatusAccepted, 
		entity.StatusScheduled, entity.StatusAssigned, entity.StatusCompleted, 
		entity.StatusRejectedBySpecialist, entity.StatusMissed, entity.StatusRescheduled,
	}
	
	// Either they are in the allowed statuses, OR they were specifically rejected by this specialist
	query := r.db.WithContext(ctx).Model(&entity.Referral{}).
		Where("target_hospital_id = ? AND (status IN ? OR (status = ? AND specialist_id = ?))", hospID, allowedStatuses, entity.StatusRejectedBySpecialist, specialistID)
	if statusFilter != "" {
		query = query.Where("status = ?", statusFilter)
	}
	err := query.Count(&count).Limit(limit).Offset(offset).Order("created_at desc").
		Preload("Patient").Preload("Diagnoses").Preload("Diagnoses.CodeInfo").Find(&referrals).Error
	return referrals, count, err
}

func (r *referralRepository) ListForReceptionist(ctx context.Context, hospID uuid.UUID, limit, offset int, statusFilter string) ([]entity.Referral, int64, error) {
	var referrals []entity.Referral
	var count int64
	allowedStatuses := []entity.ReferralStatus{
		entity.StatusAccepted, entity.StatusScheduled, entity.StatusAssigned, 
		entity.StatusMissed, entity.StatusRescheduled,
	}
	query := r.db.WithContext(ctx).Model(&entity.Referral{}).
		Where("target_hospital_id = ? AND status IN ?", hospID, allowedStatuses)
	if statusFilter != "" {
		query = query.Where("status = ?", statusFilter)
	}
	err := query.Count(&count).Limit(limit).Offset(offset).Order("created_at desc").
		Preload("Patient").Preload("Diagnoses").Preload("Diagnoses.CodeInfo").Find(&referrals).Error
	return referrals, count, err
}
