package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"Hospital-Referral-System/internal/domain/entity"
	irepository "Hospital-Referral-System/internal/domain/interfaces/repository"
	iusecase "Hospital-Referral-System/internal/domain/interfaces/usecase"
	"Hospital-Referral-System/internal/infrastructure/ml"
)

type mlUseCase struct {
	db           *gorm.DB
	referralRepo irepository.ReferralRepository
	mlRepo       irepository.MLPredictionRepository
	triageRepo   irepository.TriageQueueRepository
	configRepo   irepository.SystemConfigRepository
	auditRepo    irepository.AuditLogRepository
	client       ml.Client
	enabled      bool
	maxRetries   int
}

func NewMLUseCase(
	db *gorm.DB,
	referralRepo irepository.ReferralRepository,
	mlRepo irepository.MLPredictionRepository,
	triageRepo irepository.TriageQueueRepository,
	configRepo irepository.SystemConfigRepository,
	auditRepo irepository.AuditLogRepository,
	client ml.Client,
	enabled bool,
	maxRetries int,
) iusecase.MLUseCase {
	if maxRetries < 1 {
		maxRetries = 3
	}
	return &mlUseCase{
		db:           db,
		referralRepo: referralRepo,
		mlRepo:       mlRepo,
		triageRepo:   triageRepo,
		configRepo:   configRepo,
		auditRepo:    auditRepo,
		client:       client,
		enabled:      enabled,
		maxRetries:   maxRetries,
	}
}

func (u *mlUseCase) ScheduleScore(referralID uuid.UUID) {
	u.scheduleScore(referralID, false)
}

func (u *mlUseCase) ScheduleScoreForce(referralID uuid.UUID) {
	u.scheduleScore(referralID, true)
}

func (u *mlUseCase) scheduleScore(referralID uuid.UUID, force bool) {
	if !u.enabled {
		log.Printf("ml score skipped referral %s: ML integration disabled", referralID)
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
		defer cancel()
		if err := u.scoreReferral(ctx, referralID, force); err != nil {
			log.Printf("ml score referral %s: %v", referralID, err)
		}
	}()
}

func (u *mlUseCase) ScoreReferral(ctx context.Context, referralID uuid.UUID) error {
	return u.scoreReferral(ctx, referralID, false)
}

func (u *mlUseCase) ScoreReferralForce(ctx context.Context, referralID uuid.UUID) error {
	return u.scoreReferral(ctx, referralID, true)
}

func (u *mlUseCase) scoreReferral(ctx context.Context, referralID uuid.UUID, force bool) error {
	if !u.enabled {
		return nil
	}

	ref, err := u.referralRepo.GetReferralByID(ctx, referralID)
	if err != nil {
		return err
	}

	// 1. Quota Limit Check
	if ref.MLSuccessfulRerunCount >= 3 {
		return errors.New("ML scoring successful rerun limit reached (max 3)")
	}

	// 2. Stuck Pending Detection (older than 2.5 minutes)
	if ref.MLStatus == entity.MLStatusPending && ref.MLRunStartedAt != nil && time.Since(*ref.MLRunStartedAt) > 150*time.Second {
		log.Printf("ML pipeline timeout detected for referral %s (stuck PENDING for > 2.5 minutes). Auto-failing.", referralID)
		now := time.Now()
		ref.MLStatus = entity.MLStatusFailed
		ref.MLLastFailedAt = &now
		ref.MLRunStartedAt = nil
		ref.MLRetryCount = u.maxRetries // Force retry limit
		errMsg := "ML scoring timeout detected (> 2.5 minutes)"
		ref.MLLastError = &errMsg
		if err := u.referralRepo.Update(ctx, ref); err != nil {
			return err
		}
	}

	// 3. Rerun Cooldown Guard (if failed in the last 1 minute)
	if ref.MLStatus == entity.MLStatusFailed && ref.MLLastFailedAt != nil && time.Since(*ref.MLLastFailedAt) < 1*time.Minute {
		remaining := 1*time.Minute - time.Since(*ref.MLLastFailedAt)
		return fmt.Errorf("ML scoring is in cooldown, please wait %s before retrying", remaining.Round(time.Second))
	}

	if !u.shouldScoreReferral(ref, force) {
		log.Printf("ml score skipped referral %s: status=%s ml_status=%s force=%v",
			referralID, ref.Status, ref.MLStatus, force)
		return nil
	}

	_ = u.markMLStatus(ctx, referralID, entity.MLStatusPending, nil)

	scoreReq, inputJSON, buildErr := BuildMLScoreRequest(ref)
	if buildErr != nil {
		msg := buildErr.Error()
		_ = u.markMLStatus(ctx, referralID, entity.MLStatusSkipped, &msg)
		log.Printf("ml score skipped referral %s: %s", referralID, msg)
		return nil
	}

	trigger := "INITIAL"
	if _, err := u.mlRepo.GetLatestByReferralID(ctx, referralID); err == nil {
		trigger = "RESUBMIT"
	}
	if force {
		trigger = "RERUN"
	}

	log.Printf("ml calling POST /score for referral %s (status=%s)", referralID, ref.Status)

	// 4. Enforce strict 120-second timeout context for the API task execution
	apiCtx, apiCancel := context.WithTimeout(ctx, 120*time.Second)
	defer apiCancel()

	var scoreResp *ml.ScoreResponse
	var lastErr error
	for attempt := 0; attempt < u.maxRetries; attempt++ {
		scoreResp, lastErr = u.client.Score(apiCtx, scoreReq)
		if lastErr == nil {
			break
		}
		log.Printf("ml score attempt %d referral %s: %v", attempt+1, referralID, lastErr)
		if !isRetryableMLError(lastErr) {
			break
		}
		time.Sleep(time.Duration(attempt+1) * 500 * time.Millisecond)
	}
	if lastErr != nil {
		msg := lastErr.Error()
		_ = u.incrementMLFailure(ctx, referralID, msg)
		return lastErr
	}

	explanationJSON, _ := json.Marshal(scoreResp.Explanations)
	extID := scoreResp.PredictionID
	tier := scoreResp.SeverityTier
	procMs := scoreResp.ProcessingTimeMs

	// Process ML result via transaction and update referral record
	txErr := u.ProcessMLResult(ctx, referralID, scoreResp.SeverityScore, 0.0, tier, explanationJSON, scoreResp.ModelVersion, inputJSON, &extID, &procMs, trigger)
	if txErr != nil {
		_ = u.incrementMLFailure(ctx, referralID, txErr.Error())
		return txErr
	}

	log.Printf("ml score success referral %s: score=%.1f tier=%s prediction_id=%s",
		referralID, scoreResp.SeverityScore, tier, extID)
	return nil
}

// shouldScoreReferral decides whether scoring is allowed for the current referral state.
func (u *mlUseCase) shouldScoreReferral(ref *entity.Referral, force bool) bool {
	if ref == nil {
		return false
	}
	if force {
		switch ref.Status {
		case entity.StatusDraft, entity.StatusCancelled:
			return false
		default:
			return true
		}
	}
	if ref.Status == entity.StatusSubmitted {
		return true
	}
	// Recover referrals stuck FAILED when status is scorable.
	// If status is PENDING, we wait for timeout to transition it to FAILED before allowing rerun.
	if ref.MLStatus == entity.MLStatusFailed {
		return isMLScorableReferralStatus(ref.Status)
	}
	return false
}

func isMLScorableReferralStatus(status entity.ReferralStatus) bool {
	switch status {
	case entity.StatusSubmitted,
		entity.StatusUnderLiaisonReview,
		entity.StatusForwarded,
		entity.StatusUnderSpecialistReview,
		entity.StatusNeedRevision:
		return true
	default:
		return false
	}
}

func (u *mlUseCase) SendFeedbackAccept(ctx context.Context, referralID uuid.UUID) error {
	if !u.enabled {
		return nil
	}
	return u.sendFeedback(ctx, referralID, false, nil, nil)
}

func (u *mlUseCase) SendFeedbackOverride(ctx context.Context, referralID uuid.UUID, correctedScore float64, doctorExplanation string) error {
	if !u.enabled {
		return nil
	}
	score := int(correctedScore)
	return u.sendFeedback(ctx, referralID, true, &score, &doctorExplanation)
}

func (u *mlUseCase) sendFeedback(ctx context.Context, referralID uuid.UUID, overridden bool, correctedScore *int, explanation *string) error {
	pred, err := u.mlRepo.GetPendingFeedbackByReferralID(ctx, referralID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}
	if pred.ExternalPredictionID == nil || *pred.ExternalPredictionID == "" {
		return nil
	}

	req := ml.FeedbackRequest{
		PredictionID:  *pred.ExternalPredictionID,
		WasOverridden: overridden,
	}
	if overridden {
		req.CorrectedScore = correctedScore
		req.DoctorExplanation = explanation
	}

	log.Printf("ml calling POST /feedback for referral %s prediction_id=%s", referralID, req.PredictionID)
	if _, err := u.client.Feedback(ctx, req); err != nil {
		return err
	}

	now := time.Now()
	return u.db.WithContext(ctx).Model(&entity.MLPrediction{}).
		Where("id = ?", pred.ID).
		Update("feedback_sent_at", now).Error
}

func (u *mlUseCase) markMLStatus(ctx context.Context, referralID uuid.UUID, status string, errMsg *string) error {
	updates := map[string]interface{}{
		"ml_status": status,
	}
	if errMsg != nil {
		updates["ml_last_error"] = *errMsg
	}
	if status == entity.MLStatusPending {
		now := time.Now()
		updates["ml_run_started_at"] = &now
	} else {
		updates["ml_run_started_at"] = nil
	}
	if status == entity.MLStatusFailed {
		now := time.Now()
		updates["ml_last_failed_at"] = &now
	}
	return u.db.WithContext(ctx).Model(&entity.Referral{}).Where("id = ?", referralID).Updates(updates).Error
}

func (u *mlUseCase) incrementMLFailure(ctx context.Context, referralID uuid.UUID, msg string) error {
	now := time.Now()
	return u.db.WithContext(ctx).Model(&entity.Referral{}).Where("id = ?", referralID).
		Updates(map[string]interface{}{
			"ml_status":         entity.MLStatusFailed,
			"ml_last_error":     msg,
			"ml_retry_count":    gorm.Expr("ml_retry_count + 1"),
			"ml_last_failed_at": &now,
			"ml_run_started_at": nil,
		}).Error
}

func isRetryableMLError(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	return strings.Contains(s, "status 503") || strings.Contains(s, "status 500")
}

func (u *mlUseCase) ProcessMLResult(
	ctx context.Context,
	referralID uuid.UUID,
	score float64,
	confidence float64,
	severityTier string,
	explanation json.RawMessage,
	modelVersion string,
	inputFeatures json.RawMessage,
	externalPredictionID *string,
	processingTimeMs *float64,
	triggerReason string,
) error {
	// Fetch existing prediction or instantiate a new one
	var pred entity.MLPrediction
	err := u.db.WithContext(ctx).Where("referral_id = ?", referralID).First(&pred).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	isNew := errors.Is(err, gorm.ErrRecordNotFound)

	pred.ReferralID = referralID
	pred.OutputScore = score
	pred.TriggerReason = triggerReason
	pred.InputFeatures = inputFeatures
	pred.ExternalPredictionID = externalPredictionID
	pred.SeverityTier = &severityTier
	pred.Explanation = explanation
	pred.ModelVersion = modelVersion
	pred.ProcessingTimeMs = processingTimeMs
	if confidence > 0 {
		pred.ConfidenceLevel = &confidence
	} else {
		pred.ConfidenceLevel = nil
	}
	pred.PredictedAt = time.Now()

	txErr := u.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if isNew {
			if err := tx.Create(&pred).Error; err != nil {
				return err
			}
		} else {
			if err := tx.Save(&pred).Error; err != nil {
				return err
			}
		}

		if err := tx.Model(&entity.Referral{}).Where("id = ?", referralID).Updates(map[string]interface{}{
			"active_ml_prediction_id":   pred.ID,
			"ml_severity_score":         score,
			"ml_status":                 entity.MLStatusSuccess,
			"triage_status":             entity.TriageAutoScored,
			"ml_retry_count":            0,
			"ml_last_error":             nil,
			"ml_run_started_at":         nil,
			"ml_successful_rerun_count": gorm.Expr("ml_successful_rerun_count + 1"),
		}).Error; err != nil {
			return err
		}

		// Refresh the triage queue composite score with the new ML result.
		// When a referral is accepted before async scoring completes,
		// LandInQueue uses a 50.0 ML fallback. Once the real score arrives
		// we update the queue row so ordering reflects actual severity.
		queue, err := u.triageRepo.GetByReferralID(ctx, referralID)
		if err != nil || queue == nil {
			// Not yet in the queue — nothing to refresh.
			return nil
		}

		ref, err := u.referralRepo.GetReferralByID(ctx, referralID)
		if err != nil || ref == nil || ref.ReferralForm == nil {
			return nil
		}

		var ts float64
		switch strings.ToLower(strings.TrimSpace(ref.ReferralForm.ConditionAtReferral)) {
		case "critical":
			ts = 100
		case "urgent":
			ts = 70
		case "stable":
			ts = 30
		default:
			ts = 10
		}

		agingFactor := 1.0
		if cfg, err := u.configRepo.GetByKey(ctx, "aging_factor"); err == nil && cfg != nil {
			if af, err := strconv.ParseFloat(cfg.Value, 64); err == nil && af > 0 {
				agingFactor = af
			}
		}
		agingBonus := time.Since(ref.CreatedAt).Hours() / 24 * agingFactor
		queue.CompositeScore = (ts * 0.6) + (score * 0.3) + agingBonus

		return tx.Save(queue).Error
	})

	return txErr
}

func (u *mlUseCase) MLSeverityOverride(ctx context.Context, referralID, userID uuid.UUID, score float64, justification string) error {
	if score < 0 || score > 100 {
		return errors.New("severity score must be between 0 and 100")
	}

	if err := u.SendFeedbackOverride(ctx, referralID, score, justification); err != nil {
		log.Printf("ml feedback override referral %s: %v", referralID, err)
	}

	return u.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 1. Delete any existing ML predictions for this referral to completely remove manual inputs from ML prediction tables
		if err := tx.Where("referral_id = ?", referralID).Delete(&entity.MLPrediction{}).Error; err != nil {
			return err
		}

		// 2. Update Referral: set active_ml_prediction_id to nil, store the manual override score, and mark ml_status as MANUAL
		if err := tx.Model(&entity.Referral{}).
			Where("id = ?", referralID).
			Updates(map[string]interface{}{
				"active_ml_prediction_id": nil,
				"ml_severity_score":       score,
				"triage_status":           entity.TriageOverridden,
				"ml_status":               entity.MLStatusManual,
				"ml_run_started_at":       nil,
			}).Error; err != nil {
			return err
		}

		// 3. Recalculate TriageQueue composite score
		queue, err := u.triageRepo.GetByReferralID(ctx, referralID)
		if err == nil && queue != nil {
			ref, err := u.referralRepo.GetReferralByID(ctx, referralID)
			if err != nil {
				return err
			}
			if ref.ReferralForm == nil {
				return errors.New("referral form missing")
			}

			var triageScore float64
			switch strings.ToLower(strings.TrimSpace(ref.ReferralForm.ConditionAtReferral)) {
			case "critical":
				triageScore = 100
			case "urgent":
				triageScore = 70
			case "stable":
				triageScore = 30
			default:
				triageScore = 10
			}

			agingFactor := 1.0
			cfg, err := u.configRepo.GetByKey(ctx, "aging_factor")
			if err == nil && cfg != nil {
				if af, err := strconv.ParseFloat(cfg.Value, 64); err == nil && af > 0 {
					agingFactor = af
				}
			}
			daysWait := time.Since(ref.CreatedAt).Hours() / 24
			agingBonus := daysWait * agingFactor

			newComposite := (triageScore * 0.6) + (score * 0.3) + agingBonus
			queue.CompositeScore = newComposite

			if err := tx.Save(queue).Error; err != nil {
				return err
			}
		}

		// 4. Log audit event
		return u.auditRepo.LogWithContext(ctx, userID, entity.ActionOverrideMLScore, &referralID, nil, map[string]interface{}{
			"score":         score,
			"justification": justification,
		})
	})
}

