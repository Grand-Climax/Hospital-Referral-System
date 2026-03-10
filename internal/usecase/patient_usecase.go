package usecase

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"

	"Hospital-Referral-System/internal/delivery/http/dto"
	"Hospital-Referral-System/internal/domain/entity"
	"Hospital-Referral-System/internal/repository"
)

type PatientUseCase interface {
	GetByNationalID(ctx context.Context, nationalID string) (*entity.Patient, error)
	LookupOrCreate(ctx context.Context, req dto.LookupPatientRequest) (*entity.Patient, bool, error)
}

type patientUseCase struct {
	patientRepo repository.PatientRepository
}

func NewPatientUseCase(patientRepo repository.PatientRepository) PatientUseCase {
	return &patientUseCase{
		patientRepo: patientRepo,
	}
}

func (u *patientUseCase) GetByNationalID(ctx context.Context, nationalID string) (*entity.Patient, error) {
	patient, err := u.patientRepo.FindByNationalID(ctx, nationalID)
	if err != nil {
		return nil, err
	}
	// Return nil if not found, let handler decide 404
	return patient, nil
}

func (u *patientUseCase) LookupOrCreate(ctx context.Context, req dto.LookupPatientRequest) (*entity.Patient, bool, error) {
	// 1. Try NationalID
	if req.NationalID != "" {
		patient, err := u.patientRepo.FindByNationalID(ctx, req.NationalID)
		if err != nil {
			return nil, false, err
		}
		if patient != nil {
			return patient, false, nil // Found by National ID
		}
	}

	// 2. Fallback to Phone & FirstName
	if req.PhoneNumber != "" && req.FirstName != "" {
		patient, err := u.patientRepo.FindByPhoneAndName(ctx, req.PhoneNumber, req.FirstName)
		if err != nil {
			return nil, false, err
		}
		if patient != nil {
			return patient, false, nil // Found by Phone + Name
		}
	}

	// 3. Validation for Auto-Create
	if req.LastName == "" || req.Sex == "" {
		return nil, false, errors.New("last_name and sex are required for new patient creation")
	}

	if req.NationalID == "" && (req.PhoneNumber == "" || req.FirstName == "") {
		return nil, false, errors.New("must provide either national_id OR (phone_number AND first_name)")
	}

	// 4. Auto-Create New Patient
	newPatient := &entity.Patient{
		PhoneNumber: &req.PhoneNumber,
		FirstName:   req.FirstName,
		MiddleName:  req.MiddleName,
		LastName:    req.LastName,
		Sex:         strings.ToLower(req.Sex),
		DateOfBirth: req.DateOfBirth,
	}

	if req.HomeRegion != "" {
		newPatient.HomeRegion = &req.HomeRegion
	}

	// Hash and store NationalID if provided
	if req.NationalID != "" {
		newPatient.NationalIDEnc = &req.NationalID
		
		hash := sha256.Sum256([]byte(req.NationalID))
		hashStr := hex.EncodeToString(hash[:])
		newPatient.NationalIDHash = &hashStr
	}

	if err := u.patientRepo.Create(ctx, newPatient); err != nil {
		return nil, true, err
	}

	return newPatient, true, nil
}
