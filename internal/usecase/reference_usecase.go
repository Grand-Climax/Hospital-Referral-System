package usecase

import (
	"context"

	"github.com/google/uuid"

	"Hospital-Referral-System/internal/domain/entity"
	"Hospital-Referral-System/internal/repository"
)

type ReferenceUseCase interface {
	GetHospitals(ctx context.Context, tier string) ([]entity.Hospital, error)
	GetDepartments(ctx context.Context) ([]entity.Department, error)
	SearchICDCodes(ctx context.Context, query string) ([]entity.ICDCode, error)
	GetNetworkedHospitals(ctx context.Context, senderHospitalID uuid.UUID) ([]entity.Hospital, error)
	GetHospitalDepartments(ctx context.Context, hospitalID uuid.UUID) ([]entity.Department, error)
}

type referenceUseCase struct {
	referenceRepo repository.ReferenceRepository
}

func NewReferenceUseCase(repo repository.ReferenceRepository) ReferenceUseCase {
	return &referenceUseCase{referenceRepo: repo}
}

func (u *referenceUseCase) GetHospitals(ctx context.Context, tier string) ([]entity.Hospital, error) {
	return u.referenceRepo.GetHospitals(ctx, tier)
}

func (u *referenceUseCase) GetDepartments(ctx context.Context) ([]entity.Department, error) {
	return u.referenceRepo.GetDepartments(ctx)
}

func (u *referenceUseCase) SearchICDCodes(ctx context.Context, query string) ([]entity.ICDCode, error) {
	return u.referenceRepo.SearchICDCodes(ctx, query)
}

func (u *referenceUseCase) GetNetworkedHospitals(ctx context.Context, senderHospitalID uuid.UUID) ([]entity.Hospital, error) {
	hospitals, err := u.referenceRepo.GetNetworkedHospitals(ctx, senderHospitalID)
	if err != nil {
		return nil, err
	}

	return hospitals, nil
}

func (u *referenceUseCase) GetHospitalDepartments(ctx context.Context, hospitalID uuid.UUID) ([]entity.Department, error) {
	return u.referenceRepo.GetHospitalDepartments(ctx, hospitalID)
}
