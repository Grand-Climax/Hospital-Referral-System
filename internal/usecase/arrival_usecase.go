package usecase

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"Hospital-Referral-System/internal/domain/entity"
	irepository "Hospital-Referral-System/internal/domain/interfaces/repository"
	iusecase "Hospital-Referral-System/internal/domain/interfaces/usecase"
)

type arrivalUseCase struct {
	db         *gorm.DB
	triageRepo irepository.TriageQueueRepository
	auditRepo  irepository.AuditLogRepository
}

func NewArrivalUseCase(
	db *gorm.DB,
	tRepo irepository.TriageQueueRepository,
	auditRepo irepository.AuditLogRepository,
) iusecase.ArrivalUseCase {
	return &arrivalUseCase{
		db:         db,
		triageRepo: tRepo,
		auditRepo:  auditRepo,
	}
}

func (u *arrivalUseCase) MarkExpected(ctx context.Context, referralID uuid.UUID) error {
	queue, err := u.triageRepo.GetByReferralID(ctx, referralID)
	if err != nil {
		return err
	}
	queue.ArrivalStatus = entity.ArrivalExpected
	return u.triageRepo.Update(ctx, queue)
}

func (u *arrivalUseCase) ConfirmArrival(ctx context.Context, referralID, receptionistID uuid.UUID) error {
	queue, err := u.triageRepo.GetByReferralID(ctx, referralID)
	if err != nil {
		return err
	}

	queue.ArrivalStatus = entity.ArrivalArrived
	now := time.Now()
	queue.ArrivedAt = &now

	if err := u.triageRepo.Update(ctx, queue); err != nil {
		return err
	}

	return u.auditRepo.LogWithContext(ctx, receptionistID, entity.ActionUpdatePatientStatus, &referralID, nil, "ARRIVED")
}

func (u *arrivalUseCase) UpdateArrivalStatus(ctx context.Context, referralID uuid.UUID, status entity.ArrivalStatus) error {
	queue, err := u.triageRepo.GetByReferralID(ctx, referralID)
	if err != nil {
		return err
	}
	queue.ArrivalStatus = status
	return u.triageRepo.Update(ctx, queue)
}
