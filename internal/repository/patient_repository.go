package repository

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
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

func hashNationalID(nationalID string) string {
	hash := sha256.Sum256([]byte(nationalID))
	return hex.EncodeToString(hash[:])
}

func (r *patientRepository) FindByNationalID(ctx context.Context, nationalID string) (*entity.Patient, error) {
	var patient entity.Patient
	hashedID := hashNationalID(nationalID)

	if err := r.db.WithContext(ctx).Where("national_id_hash = ?", hashedID).First(&patient).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // Return nil, nil for not found instead of error for cleaner logic
		}
		return nil, err
	}

	return &patient, nil
}

func (r *patientRepository) SearchPatients(ctx context.Context, query string) ([]entity.Patient, error) {
	var patients []entity.Patient
	
	wildcard := "%" + query + "%"
	hashedID := hashNationalID(query)

	err := r.db.WithContext(ctx).
		Where("phone_number ILIKE ? OR national_id_hash = ? OR first_name ILIKE ? OR last_name ILIKE ?", 
		wildcard, hashedID, wildcard, wildcard).
		Limit(20).
		Find(&patients).Error

	return patients, err
}

func (r *patientRepository) FindByPhoneAndName(ctx context.Context, phone, firstName string) (*entity.Patient, error) {
	var patient entity.Patient
	err := r.db.WithContext(ctx).
		Where("phone_number = ? AND LOWER(first_name) = LOWER(?)", phone, firstName).
		First(&patient).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &patient, nil
}
