package usecase

import (
	"context"

	"Hospital-Referral-System/internal/domain/entity"
	"Hospital-Referral-System/internal/repository"
)

type PatientUseCase interface {
	GetByNationalID(ctx context.Context, nationalID string) (*entity.Patient, error)
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
