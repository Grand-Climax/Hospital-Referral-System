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

	if startDate, ok := filter["start_date"]; ok {
		query = query.Where("created_at >= ?", startDate)
	}

	if endDate, ok := filter["end_date"]; ok {
		query = query.Where("created_at <= ?", endDate)
	}

	err := query.Order("created_at desc").Find(&referrals).Error
	return referrals, err
}
