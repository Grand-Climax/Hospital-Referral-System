package usecase

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"Hospital-Referral-System/internal/domain/entity"
	irepository "Hospital-Referral-System/internal/domain/interfaces/repository"
	iusecase "Hospital-Referral-System/internal/domain/interfaces/usecase"
)

var (
	ErrHospitalNotFound = errors.New("hospital not found")
)

type hospitalUseCase struct {
	repo       irepository.HospitalRepository
	configRepo irepository.SystemConfigRepository
	auditRepo  irepository.AuditLogRepository
}

func NewHospitalUseCase(
	repo irepository.HospitalRepository,
	configRepo irepository.SystemConfigRepository,
	auditRepo irepository.AuditLogRepository,
) iusecase.HospitalUseCase {
	return &hospitalUseCase{
		repo:       repo,
		configRepo: configRepo,
		auditRepo:  auditRepo,
	}
}

func (u *hospitalUseCase) CreateHospital(ctx context.Context, hospital *entity.Hospital) error {
	hospital.IsActive = true
	hospital.IsDeleted = false
	return u.repo.Create(ctx, hospital)
}

func (u *hospitalUseCase) GetHospitalByID(ctx context.Context, id uuid.UUID) (*entity.Hospital, error) {
	hospital, err := u.repo.FindByID(ctx, id)
	if err != nil {
		return nil, ErrHospitalNotFound
	}
	if hospital.IsDeleted {
		return nil, ErrHospitalNotFound
	}
	return hospital, nil
}

func (u *hospitalUseCase) UpdateHospital(ctx context.Context, hospital *entity.Hospital) error {
	existing, err := u.repo.FindByID(ctx, hospital.ID)
	if err != nil || existing.IsDeleted {
		return ErrHospitalNotFound
	}

	hospital.CreatedAt = existing.CreatedAt
	hospital.IsDeleted = existing.IsDeleted

	return u.repo.Update(ctx, hospital)
}

func (u *hospitalUseCase) DeleteHospital(ctx context.Context, id uuid.UUID) error {
	hospital, err := u.repo.FindByID(ctx, id)
	if err != nil {
		return ErrHospitalNotFound
	}
	hospital.IsDeleted = true
	hospital.IsActive = false
	return u.repo.Update(ctx, hospital)
}

func (u *hospitalUseCase) ListHospitals(ctx context.Context, filter irepository.HospitalListFilter) ([]entity.Hospital, int64, error) {
	return u.repo.ListHospitals(ctx, filter)
}

func (u *hospitalUseCase) UpdateSystemConfig(ctx context.Context, userID uuid.UUID, req map[string]string) error {
	for k, v := range req {
		cfg := &entity.SystemConfig{
			Key:   k,
			Value: v,
		}
		if err := u.configRepo.Update(ctx, cfg); err != nil {
			return err
		}
	}
	return u.auditRepo.LogWithContext(ctx, userID, entity.ActionUpdateSystemConfig, nil, nil, req)
}

func (u *hospitalUseCase) GetSystemConfigs(ctx context.Context) (map[string]string, error) {
	return u.configRepo.GetAll(ctx)
}
