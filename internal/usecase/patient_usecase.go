package usecase

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"

	"Hospital-Referral-System/internal/delivery/http/dto"
	"Hospital-Referral-System/internal/domain/entity"
	irepository "Hospital-Referral-System/internal/domain/interfaces/repository"
	iusecase "Hospital-Referral-System/internal/domain/interfaces/usecase"
	"Hospital-Referral-System/internal/infrastructure/crypto"
)

type patientUseCase struct {
	patientRepo   irepository.PatientRepository
	cryptoSvc     *crypto.PatientCryptoService
	auditRepo     irepository.AuditLogRepository
}

func NewPatientUseCase(patientRepo irepository.PatientRepository, cryptoSvc *crypto.PatientCryptoService, auditRepo irepository.AuditLogRepository) iusecase.PatientUseCase {
	return &patientUseCase{
		patientRepo: patientRepo,
		cryptoSvc:   cryptoSvc,
		auditRepo:   auditRepo,
	}
}

func (u *patientUseCase) decryptPatient(ctx context.Context, p *entity.Patient) error {
	if p == nil {
		return nil
	}
	err := p.DecryptFields(u.cryptoSvc)
	if err != nil {
		return err
	}

	// Optional: Log audit for decryption, if user info is available in context.
	// For simplicity, we just decrypt. The handler can log if needed, or we assume
	// generic read access here. If strict audit is needed:
	// u.auditRepo.Create(ctx, &entity.AuditLog{...})
	
	return nil
}

func (u *patientUseCase) GetByNationalID(ctx context.Context, nationalID string) (*entity.Patient, error) {
	if nationalID == "" {
		return nil, errors.New("national ID is required")
	}

	hash := u.cryptoSvc.GenerateHMAC(nationalID)
	patient, err := u.patientRepo.FindByNationalIDHash(ctx, hash)
	if err != nil {
		return nil, err
	}

	if err := u.decryptPatient(ctx, patient); err != nil {
		return nil, err
	}

	return patient, nil
}

func (u *patientUseCase) LookupByNationalID(ctx context.Context, nationalID string) (*uuid.UUID, error) {
	if nationalID == "" {
		return nil, nil
	}
	hash := u.cryptoSvc.GenerateHMAC(nationalID)
	patient, err := u.patientRepo.FindByNationalIDHash(ctx, hash)
	if err != nil {
		return nil, err
	}
	if patient == nil {
		return nil, nil
	}
	return &patient.ID, nil
}

func (u *patientUseCase) LookupPatient(ctx context.Context, nationalID, phone string) (*entity.Patient, error) {
	if nationalID != "" {
		hash := u.cryptoSvc.GenerateHMAC(nationalID)
		patient, err := u.patientRepo.FindByNationalIDHash(ctx, hash)
		if err != nil {
			return nil, err
		}
		if err := u.decryptPatient(ctx, patient); err != nil {
			return nil, err
		}
		return patient, nil
	}

	if phone != "" {
		normalizedPhone, err := crypto.NormalizePhone(phone)
		if err != nil {
			return nil, err
		}

		hash := u.cryptoSvc.GenerateHMAC(normalizedPhone)
		patient, err := u.patientRepo.FindByPhoneHash(ctx, hash)
		if err != nil {
			return nil, err
		}
		if patient == nil {
			return nil, nil
		}
		
		if err := u.decryptPatient(ctx, patient); err != nil {
			return nil, err
		}
		return patient, nil
	}

	return nil, errors.New("insufficient lookup parameters: provide national_id OR phone_number")
}

func (u *patientUseCase) CreatePatient(ctx context.Context, req dto.CreatePatientRequest) (*entity.Patient, error) {
	normalizedPhone, err := crypto.NormalizePhone(req.PhoneNumber)
	if err != nil {
		return nil, err
	}

	var natIDHash string
	if req.NationalID != "" {
		natIDHash = u.cryptoSvc.GenerateHMAC(req.NationalID)
		existing, err := u.patientRepo.FindByNationalIDHash(ctx, natIDHash)
		if err != nil {
			return nil, err
		}
		if existing != nil {
			return nil, errors.New("a patient with this National ID already exists")
		}
	}

	phoneHash := u.cryptoSvc.GenerateHMAC(normalizedPhone)
	existingPhone, err := u.patientRepo.FindByPhoneHash(ctx, phoneHash)
	if err != nil {
		return nil, err
	}
	if existingPhone != nil {
		return nil, errors.New("a patient with this Phone Number already exists")
	}

	// Encrypt fields
	encPhone, err := u.cryptoSvc.Encrypt([]byte(normalizedPhone))
	if err != nil { return nil, err }
	
	encFirst, err := u.cryptoSvc.Encrypt([]byte(req.FirstName))
	if err != nil { return nil, err }

	encMiddle, err := u.cryptoSvc.Encrypt([]byte(req.MiddleName))
	if err != nil { return nil, err }

	encLast, err := u.cryptoSvc.Encrypt([]byte(req.LastName))
	if err != nil { return nil, err }

	var encNatID *string
	if req.NationalID != "" {
		enc, err := u.cryptoSvc.Encrypt([]byte(req.NationalID))
		if err != nil { return nil, err }
		encNatID = &enc
	}

	newPatient := &entity.Patient{
		PhoneNumberEnc: &encPhone,
		PhoneHash:      &phoneHash,
		FirstNameEnc:   encFirst,
		MiddleNameEnc:  encMiddle,
		LastNameEnc:    encLast,
		Sex:            strings.ToLower(req.Sex),
		DateOfBirth:    req.DateOfBirth,
	}

	if req.HomeRegion != "" {
		newPatient.HomeRegion = &req.HomeRegion
	}

	if req.NationalID != "" {
		newPatient.NationalIDEnc = encNatID
		newPatient.NationalIDHash = &natIDHash
	}

	if err := u.patientRepo.Create(ctx, newPatient); err != nil {
		return nil, err
	}

	// Decrypt for return
	if err := u.decryptPatient(ctx, newPatient); err != nil {
		return nil, err
	}

	return newPatient, nil
}
