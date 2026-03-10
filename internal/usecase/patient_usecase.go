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
	GetByPhoneAndName(ctx context.Context, phone, firstName string) (*entity.Patient, error)
	CreatePatient(ctx context.Context, req dto.CreatePatientRequest) (*entity.Patient, error)
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

func (u *patientUseCase) GetByPhoneAndName(ctx context.Context, phone, firstName string) (*entity.Patient, error) {
	patient, err := u.patientRepo.FindByPhoneAndName(ctx, phone, firstName)
	if err != nil {
		return nil, err
	}
	return patient, nil
}

func (u *patientUseCase) CreatePatient(ctx context.Context, req dto.CreatePatientRequest) (*entity.Patient, error) {
	// 1. Uniqueness check by National ID (if provided)
	if req.NationalID != "" {
		existing, err := u.patientRepo.FindByNationalID(ctx, req.NationalID)
		if err != nil {
			return nil, err
		}
		if existing != nil {
			return nil, errors.New("a patient with this National ID already exists")
		}
	}

	// 2. Uniqueness check by Phone + First Name
	existing, err := u.patientRepo.FindByPhoneAndName(ctx, req.PhoneNumber, req.FirstName)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, errors.New("a patient with this Phone Number and First Name already exists")
	}

	// 3. Create New Patient
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
		return nil, err
	}

	return newPatient, nil
}
