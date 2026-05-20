package usecase

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"

	"Hospital-Referral-System/internal/domain/entity"
)

func TestApplyMLPredictionFields(t *testing.T) {
	tier := "Critical"
	ver := "1.0.0"
	raw, _ := json.Marshal([]string{"Low SpO2", "High fever"})
	ref := &entity.Referral{}
	pred := &entity.MLPrediction{
		SeverityTier:  &tier,
		ModelVersion:  ver,
		Explanation:   raw,
	}
	applyMLPredictionFields(ref, pred)
	assert.Equal(t, "Critical", *ref.MLSeverityTier)
	assert.Equal(t, []string{"Low SpO2", "High fever"}, ref.MLExplanations)
	assert.Equal(t, "1.0.0", *ref.MLModelVersion)
}
