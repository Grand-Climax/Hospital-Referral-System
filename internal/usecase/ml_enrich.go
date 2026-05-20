package usecase

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"

	"Hospital-Referral-System/internal/domain/entity"
)

func mlExplanationsFromPrediction(pred *entity.MLPrediction) []string {
	if pred == nil || len(pred.Explanation) == 0 {
		return nil
	}
	var out []string
	if err := json.Unmarshal(pred.Explanation, &out); err != nil {
		return nil
	}
	return out
}

func applyMLPredictionFields(ref *entity.Referral, pred *entity.MLPrediction) {
	if ref == nil || pred == nil {
		return
	}
	ref.MLSeverityTier = pred.SeverityTier
	ref.MLModelVersion = &pred.ModelVersion
	ref.MLExplanations = mlExplanationsFromPrediction(pred)
}

func (u *referralUseCase) enrichReferralML(ctx context.Context, ref *entity.Referral) {
	if ref == nil || u.mlRepo == nil {
		return
	}
	var pred *entity.MLPrediction
	var err error
	if ref.ActiveMLPredictionID != nil {
		pred, err = u.mlRepo.FindByID(ctx, *ref.ActiveMLPredictionID)
	} else {
		pred, err = u.mlRepo.GetActiveByReferralID(ctx, ref.ID)
	}
	if err != nil || pred == nil {
		return
	}
	applyMLPredictionFields(ref, pred)
}

func (u *referralUseCase) enrichReferralsMLBatch(ctx context.Context, referrals []entity.Referral) {
	if u.mlRepo == nil || len(referrals) == 0 {
		return
	}
	ids := make([]uuid.UUID, len(referrals))
	for i := range referrals {
		ids[i] = referrals[i].ID
	}
	predMap, err := u.mlRepo.MapActiveByReferralIDs(ctx, ids)
	if err != nil {
		return
	}
	for i := range referrals {
		if pred, ok := predMap[referrals[i].ID]; ok {
			applyMLPredictionFields(&referrals[i], pred)
		}
	}
}
