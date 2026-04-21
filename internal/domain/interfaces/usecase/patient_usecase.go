package interfaces

import (
	"context"

	"Hospital-Referral-System/internal/delivery/http/dto"
	"Hospital-Referral-System/internal/domain/entity"
)

type PatientUseCase interface {
	GetByNationalID(ctx context.Context, nationalID string) (*entity.Patient, error)
	SearchPatients(ctx context.Context, query string) ([]entity.Patient, error)
	LookupPatient(ctx context.Context, nationalID, phone, firstName string) (*entity.Patient, error)
	CreatePatient(ctx context.Context, req dto.CreatePatientRequest) (*entity.Patient, error)
}
