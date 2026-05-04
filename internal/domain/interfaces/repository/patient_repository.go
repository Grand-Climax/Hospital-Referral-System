package interfaces

import (
	"context"

	"Hospital-Referral-System/internal/domain/entity"
)

type PatientRepository interface {
	BaseRepository[entity.Patient]
	FindByNationalIDHash(ctx context.Context, hash string) (*entity.Patient, error)
	FindByPhoneHash(ctx context.Context, hash string) (*entity.Patient, error)
}
