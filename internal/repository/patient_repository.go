package repository

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"Hospital-Referral-System/internal/domain/entity"
	irepository "Hospital-Referral-System/internal/domain/interfaces/repository"
)

type patientRepository struct {
	*BaseRepository[entity.Patient]
	db *gorm.DB
}

func NewPatientRepository(db *gorm.DB) irepository.PatientRepository {
	return &patientRepository{
		BaseRepository: NewBaseRepository[entity.Patient](db),
		db:             db,
	}
}

func (r *patientRepository) FindByNationalIDHash(ctx context.Context, hash string) (*entity.Patient, error) {
	var patient entity.Patient

	if err := r.db.WithContext(ctx).Where("national_id_hash = ?", hash).First(&patient).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &patient, nil
}

func (r *patientRepository) FindByPhoneHash(ctx context.Context, hash string) (*entity.Patient, error) {
	var patient entity.Patient
	err := r.db.WithContext(ctx).
		Where("phone_hash = ?", hash).
		First(&patient).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &patient, nil
}
