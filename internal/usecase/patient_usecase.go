package usecase

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"

	"Hospital-Referral-System/internal/delivery/http/dto"
	"Hospital-Referral-System/internal/domain/entity"
	irepository "Hospital-Referral-System/internal/domain/interfaces/repository"
	iusecase "Hospital-Referral-System/internal/domain/interfaces/usecase"
)

type patientUseCase struct {
	patientRepo irepository.PatientRepository
}

func NewPatientUseCase(patientRepo irepository.PatientRepository) iusecase.PatientUseCase {
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

func (u *patientUseCase) SearchPatients(ctx context.Context, query string) ([]entity.Patient, error) {
	// Reverted to support explicit matching
	return nil, errors.New("method deprecated: use explicit lookup methods")
}

func (u *patientUseCase) LookupPatient(ctx context.Context, nationalID, phone, firstName string) (*entity.Patient, error) {
	if nationalID != "" {
		return u.patientRepo.FindByNationalID(ctx, nationalID)
	}

	if phone != "" && firstName != "" {
		// Validate Ethiopian phone number
		if err := u.validateEthiopianPhone(phone); err != nil {
			return nil, err
		}

		patient, err := u.patientRepo.FindByPhoneAndName(ctx, phone, firstName)
		if err != nil {
			return nil, err
		}
		if patient == nil {
			return nil, nil
		}
		return patient, nil
	}

	return nil, errors.New("insufficient lookup parameters: provide national_id OR (phone_number AND first_name)")
}

func (u *patientUseCase) CreatePatient(ctx context.Context, req dto.CreatePatientRequest) (*entity.Patient, error) {
	// 0. Validate Ethiopian phone number
	if err := u.validateEthiopianPhone(req.PhoneNumber); err != nil {
		return nil, err
	}

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

	// 2. Uniqueness check by Phone
	existingPhones, err := u.patientRepo.SearchPatients(ctx, req.PhoneNumber)
	if err != nil {
		return nil, err
	}
	for _, p := range existingPhones {
		if p.PhoneNumber != nil && *p.PhoneNumber == req.PhoneNumber && strings.EqualFold(p.FirstName, req.FirstName) {
			return nil, errors.New("a patient with this Phone Number and First Name already exists")
		}
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

func (u *patientUseCase) validateEthiopianPhone(phone string) error {
	if !strings.HasPrefix(phone, "09") && !strings.HasPrefix(phone, "07") && !strings.HasPrefix(phone, "+2519") && !strings.HasPrefix(phone, "+2517") {
		return errors.New("invalid phone format: must be an Ethiopian number (09/07 or +251)")
	}
	if len(phone) < 10 || len(phone) > 13 {
		return errors.New("invalid phone format length")
	}
	return nil
}
