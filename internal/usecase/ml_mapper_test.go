package usecase

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"Hospital-Referral-System/internal/domain/entity"
)

func TestBuildMLScoreRequest_MapsClinicalAndVitals(t *testing.T) {
	sys := int16(85)
	dia := int16(55)
	hr := int16(130)
	spo2 := float32(84)
	temp := float32(39.5)
	pe := "febrile"
	treat := "none"

	ref := &entity.Referral{
		ReferralForm: &entity.ReferralForm{
			ReasonOfReferral:             "Cerebral malaria",
			ClinicalSummary:              "Unconscious, high fever",
			PatientHistory:               "7-year-old male",
			PhysicalExaminationFindings:  &pe,
			TreatmentGivenBeforeReferral:   &treat,
		},
		Vitals: []entity.Vital{{
			SystolicBP:  &sys,
			DiastolicBP: &dia,
			HeartRate:   &hr,
			SpO2:        &spo2,
			Temperature: &temp,
		}},
		Diagnoses: []entity.ReferralDiagnosis{{
			IsPrimary: true,
			ICDCode:   "B50.0",
			CodeInfo:  &entity.ICDCode{Code: "B50.0", Description: "Plasmodium falciparum malaria"},
		}},
	}

	req, _, err := BuildMLScoreRequest(ref)
	require.NoError(t, err)
	assert.Equal(t, 85, req.Vitals.BPSystolic)
	assert.Equal(t, "Cerebral malaria", req.ClinicalText.ReasonForReferral)
	assert.Equal(t, "Plasmodium falciparum malaria", req.ClinicalText.Diagnosis)
}

func TestBuildMLScoreRequest_SkipsWhenVitalsMissing(t *testing.T) {
	ref := &entity.Referral{
		ReferralForm: &entity.ReferralForm{
			ReasonOfReferral: "test",
			ClinicalSummary:  "test",
			PatientHistory:   "test",
		},
	}
	_, _, err := BuildMLScoreRequest(ref)
	assert.ErrorIs(t, err, errMLVitalsMissing)
}
