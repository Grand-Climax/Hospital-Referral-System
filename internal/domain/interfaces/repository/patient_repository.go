package repository

import (
	"context"

	"Hospital-Referral-System/internal/domain/entity"
)

type PatientRepository interface {
	BaseRepository[entity.Patient]
	FindByNationalID(ctx context.Context, nationalID string) (*entity.Patient, error)
	FindByPhoneAndName(ctx context.Context, phone, firstName string) (*entity.Patient, error)
}
