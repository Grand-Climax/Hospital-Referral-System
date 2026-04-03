package usecase

import (
	"context"

	"github.com/google/uuid"

	"Hospital-Referral-System/internal/domain/entity"
	irepository "Hospital-Referral-System/internal/domain/interfaces/repository"
	iusecase "Hospital-Referral-System/internal/domain/interfaces/usecase"
)

type referenceUseCase struct {
	referenceRepo irepository.ReferenceRepository
}

func NewReferenceUseCase(repo irepository.ReferenceRepository) iusecase.ReferenceUseCase {
	return &referenceUseCase{referenceRepo: repo}
}

func (u *referenceUseCase) GetHospitals(ctx context.Context, tier string) ([]entity.Hospital, error) {
	return u.referenceRepo.GetHospitals(ctx, tier)
}

func (u *referenceUseCase) GetDepartments(ctx context.Context) ([]entity.Department, error) {
	return u.referenceRepo.GetDepartments(ctx)
}

func (u *referenceUseCase) ListICDCodes(ctx context.Context) ([]entity.ICDCode, error) {
	return u.referenceRepo.ListICDCodes(ctx)
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

func (u *referenceUseCase) GetLiaisonsByHospital(ctx context.Context, hospitalID uuid.UUID) ([]entity.User, error) {
	return u.referenceRepo.GetLiaisonsByHospital(ctx, hospitalID)
}
