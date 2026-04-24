package interfaces

import (
	"context"

	"github.com/google/uuid"

	"Hospital-Referral-System/internal/domain/entity"
	irepository "Hospital-Referral-System/internal/domain/interfaces/repository"
)

type HospitalUseCase interface {
	CreateHospital(ctx context.Context, hospital *entity.Hospital) error
	GetHospitalByID(ctx context.Context, id uuid.UUID) (*entity.Hospital, error)
	UpdateHospital(ctx context.Context, hospital *entity.Hospital) error
	DeleteHospital(ctx context.Context, id uuid.UUID) error
	ListHospitals(ctx context.Context, filter irepository.HospitalListFilter) ([]entity.Hospital, int64, error)

	// Admin Logic absorbed from legacy AdminUseCase
	UpdateSystemConfig(ctx context.Context, userID uuid.UUID, req map[string]string) error
	GetSystemConfigs(ctx context.Context) (map[string]string, error)
}
