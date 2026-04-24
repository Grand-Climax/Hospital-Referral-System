package usecase

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"Hospital-Referral-System/internal/delivery/http/dto"
	"Hospital-Referral-System/internal/domain/entity"
	irepository "Hospital-Referral-System/internal/domain/interfaces/repository"
	iusecase "Hospital-Referral-System/internal/domain/interfaces/usecase"
)

type triageUseCase struct {
	db           *gorm.DB
	referralRepo irepository.ReferralRepository
	triageRepo   irepository.TriageQueueRepository
	mlRepo       irepository.MLPredictionRepository
	configRepo   irepository.SystemConfigRepository
	auditRepo    irepository.AuditLogRepository
}

func NewTriageUseCase(
	db *gorm.DB,
	rRepo irepository.ReferralRepository,
	tRepo irepository.TriageQueueRepository,
	mlRepo irepository.MLPredictionRepository,
	cfgRepo irepository.SystemConfigRepository,
	auditRepo irepository.AuditLogRepository,
) iusecase.TriageUseCase {
	return &triageUseCase{
		db:           db,
		referralRepo: rRepo,
		triageRepo:   tRepo,
		mlRepo:       mlRepo,
		configRepo:   cfgRepo,
		auditRepo:    auditRepo,
	}
}

func (u *triageUseCase) LandInQueue(ctx context.Context, referralID uuid.UUID) error {
	ref, err := u.referralRepo.GetReferralByID(ctx, referralID)
	if err != nil {
		return err
	}

	queue := &entity.TriageQueue{
		ReferralID:  referralID,
		DeptID:      ref.TargetDeptID,
		QueueStatus: entity.QueueWaiting,
	}

	score, _ := u.CalculateCompositeScore(ctx, referralID)
	queue.CompositeScore = score

	return u.triageRepo.Create(ctx, queue)
}

func (u *triageUseCase) CalculateCompositeScore(ctx context.Context, referralID uuid.UUID) (float64, error) {
	ref, err := u.referralRepo.GetReferralByID(ctx, referralID)
	if err != nil {
		return 0, err
	}

	var triageScore float64
	switch ref.ReferralForm.ConditionAtReferral {
	case "critical":
		triageScore = 100
	case "urgent":
		triageScore = 70
	case "stable":
		triageScore = 30
	default:
		triageScore = 10
	}

	mlSeverity := 0.0
	pred, err := u.mlRepo.GetLatestByReferralID(ctx, referralID)
	if err == nil && pred != nil {
		mlSeverity = pred.OutputScore
	}

	agingFactor := 1.0
	cfg, err := u.configRepo.GetByKey(ctx, "aging_factor")
	if err == nil {
		af, _ := strconv.ParseFloat(cfg.Value, 64)
		if af > 0 {
			agingFactor = af
		}
	}
	daysWait := time.Since(ref.CreatedAt).Hours() / 24
	agingBonus := daysWait * agingFactor

	composite := (triageScore * 0.6) + (mlSeverity * 0.3) + agingBonus
	return composite, nil
}

func (u *triageUseCase) ListForTriage(ctx context.Context, hospitalID uuid.UUID, limit, offset int) ([]dto.TriageListResponse, int64, error) {
	queues, count, err := u.triageRepo.ListForTriage(ctx, hospitalID, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	var resp []dto.TriageListResponse
	for _, q := range queues {
		ref, _ := u.referralRepo.GetReferralByID(ctx, q.ReferralID)
		name := "Unknown"
		if ref != nil && ref.Patient != nil {
			name = fmt.Sprintf("%s %s", ref.Patient.FirstName, ref.Patient.LastName)
		}
		resp = append(resp, dto.TriageListResponse{
			ReferralID:      q.ReferralID,
			PatientName:     name,
			TargetDept:      ref.TargetDeptID.String(),
			CompositeScore:  q.CompositeScore,
			QueueStatus:     string(q.QueueStatus),
			ArrivalBoost:    float64(q.ArrivalBoost),
			AppointmentDate: q.AppointmentDate,
		})
	}
	return resp, count, nil
}

func (u *triageUseCase) ReviewTriage(ctx context.Context, referralID, userID uuid.UUID, req dto.TriageReviewRequest) error {
	queue, err := u.triageRepo.GetByReferralID(ctx, referralID)
	if err != nil {
		return err
	}

	switch req.Action {
	case "APPROVE":
		// logic
	case "OVERRIDE":
		if req.CompositeScore != nil {
			queue.CompositeScore = *req.CompositeScore
		}
	case "REJECT":
		return errors.New("rejection logic not yet implemented")
	}

	if err := u.triageRepo.Update(ctx, queue); err != nil {
		return err
	}

	return u.auditRepo.LogWithContext(ctx, userID, entity.ActionOverrideQueue, &referralID, nil, req)
}

func (u *triageUseCase) ListScheduledInRange(ctx context.Context, hospitalID, deptID uuid.UUID, start, end time.Time) ([]entity.TriageQueue, error) {
	return u.triageRepo.ListScheduledInRange(ctx, hospitalID, deptID, start, end)
}
