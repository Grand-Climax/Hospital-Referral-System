package interfaces

import (
	"context"

	"github.com/google/uuid"

	"Hospital-Referral-System/internal/delivery/http/dto"
	"Hospital-Referral-System/internal/domain/entity"
)

type PatientUseCase interface {
	GetByNationalID(ctx context.Context, nationalID string) (*entity.Patient, error)
	LookupByNationalID(ctx context.Context, nationalID string) (*uuid.UUID, error)
	LookupPatient(ctx context.Context, nationalID, phone string) (*entity.Patient, error)
	CreatePatient(ctx context.Context, req dto.CreatePatientRequest) (*entity.Patient, error)
}
