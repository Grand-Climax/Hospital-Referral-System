package usecase

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"Hospital-Referral-System/internal/domain/entity"
	"Hospital-Referral-System/internal/infrastructure/ml"
)

// ── helpers ──────────────────────────────────────────────────────────────────

func baseVitals() entity.Vital {
	sys := int16(85)
	dia := int16(55)
	hr := int16(130)
	spo2 := float32(84)
	temp := float32(39.5)
	return entity.Vital{
		SystolicBP:  &sys,
		DiastolicBP: &dia,
		HeartRate:   &hr,
		SpO2:        &spo2,
		Temperature: &temp,
	}
}

func baseForm() *entity.ReferralForm {
	pe := "febrile"
	treat := "none"
	return &entity.ReferralForm{
		ReasonOfReferral:            "Cerebral malaria",
		ClinicalSummary:             "Unconscious, high fever",
		PatientHistory:              "7-year-old male",
		PhysicalExaminationFindings: &pe,
		TreatmentGivenBeforeReferral: &treat,
		ConditionAtReferral:         "critical",
	}
}

// ── Original core mapping ────────────────────────────────────────────────────

func TestBuildMLScoreRequest_MapsClinicalAndVitals(t *testing.T) {
	ref := &entity.Referral{
		ReferralForm: baseForm(),
		Vitals:       []entity.Vital{baseVitals()},
		Diagnoses: []entity.ReferralDiagnosis{{
			IsPrimary: true,
			ICDCode:   "B50.0",
			CodeInfo:  &entity.ICDCode{Code: "B50.0", Description: "Plasmodium falciparum malaria"},
		}},
	}

	req, raw, err := BuildMLScoreRequest(ref)
	require.NoError(t, err)
	assert.Equal(t, 85, req.Vitals.BPSystolic)
	assert.Equal(t, 55, req.Vitals.BPDiastolic)
	assert.Equal(t, 130, req.Vitals.HeartRate)
	assert.InDelta(t, 84.0, req.Vitals.SpO2, 0.1)
	assert.InDelta(t, 39.5, req.Vitals.Temperature, 0.1)
	assert.Equal(t, "Cerebral malaria", req.ClinicalText.ReasonForReferral)
	assert.Equal(t, "Plasmodium falciparum malaria", req.ClinicalText.Diagnosis)
	assert.NotEmpty(t, raw)
}

func TestBuildMLScoreRequest_SkipsWhenVitalsMissing(t *testing.T) {
	ref := &entity.Referral{
		ReferralForm: &entity.ReferralForm{
			ReasonOfReferral:    "test",
			ClinicalSummary:     "test",
			PatientHistory:      "test",
			ConditionAtReferral: "stable",
		},
	}
	_, _, err := BuildMLScoreRequest(ref)
	assert.ErrorIs(t, err, errMLVitalsMissing)
}

// ── condition_at_referral normalization ──────────────────────────────────────

func TestBuildMLScoreRequest_ConditionAtReferral_Lowercase(t *testing.T) {
	cases := []struct {
		input    string
		expected string
	}{
		{"critical", "critical"},
		{"CRITICAL", "critical"},
		{"Critical", "critical"},
		{"URGENT", "urgent"},
		{"Urgent", "urgent"},
		{"STABLE", "stable"},
		{"  critical  ", "critical"}, // whitespace trimmed
	}

	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			f := baseForm()
			f.ConditionAtReferral = tc.input
			ref := &entity.Referral{
				ReferralForm: f,
				Vitals:       []entity.Vital{baseVitals()},
			}
			req, _, err := BuildMLScoreRequest(ref)
			require.NoError(t, err)
			assert.Equal(t, tc.expected, req.ClinicalText.ConditionAtReferral)
		})
	}
}

func TestBuildMLScoreRequest_ConditionAtReferral_EmptyIsOmitted(t *testing.T) {
	f := baseForm()
	f.ConditionAtReferral = ""
	ref := &entity.Referral{
		ReferralForm: f,
		Vitals:       []entity.Vital{baseVitals()},
	}
	req, raw, err := BuildMLScoreRequest(ref)
	require.NoError(t, err)
	assert.Empty(t, req.ClinicalText.ConditionAtReferral)

	// omitempty means the field should not appear in the JSON at all
	var m map[string]interface{}
	require.NoError(t, json.Unmarshal(raw, &m))
	ct := m["clinical_text"].(map[string]interface{})
	_, exists := ct["condition_at_referral"]
	assert.False(t, exists, "empty condition should be omitted from JSON")
}

func TestBuildMLScoreRequest_ConditionAtReferral_WhitespaceOnlyIsOmitted(t *testing.T) {
	f := baseForm()
	f.ConditionAtReferral = "   "
	ref := &entity.Referral{
		ReferralForm: f,
		Vitals:       []entity.Vital{baseVitals()},
	}
	req, _, err := BuildMLScoreRequest(ref)
	require.NoError(t, err)
	assert.Empty(t, req.ClinicalText.ConditionAtReferral)
}

// ── investigation_results ────────────────────────────────────────────────────

func TestBuildMLScoreRequest_InvestigationResults_Populated(t *testing.T) {
	inv := "Hb: 6.2, Parasitemia ++"
	f := baseForm()
	f.InvestigationResults = &inv
	ref := &entity.Referral{
		ReferralForm: f,
		Vitals:       []entity.Vital{baseVitals()},
	}
	req, raw, err := BuildMLScoreRequest(ref)
	require.NoError(t, err)
	assert.Equal(t, "Hb: 6.2, Parasitemia ++", req.ClinicalText.InvestigationResults)

	var m map[string]interface{}
	require.NoError(t, json.Unmarshal(raw, &m))
	ct := m["clinical_text"].(map[string]interface{})
	assert.Equal(t, "Hb: 6.2, Parasitemia ++", ct["investigation_results"])
}

func TestBuildMLScoreRequest_InvestigationResults_NilOmitted(t *testing.T) {
	f := baseForm()
	f.InvestigationResults = nil
	ref := &entity.Referral{
		ReferralForm: f,
		Vitals:       []entity.Vital{baseVitals()},
	}
	req, raw, err := BuildMLScoreRequest(ref)
	require.NoError(t, err)
	assert.Empty(t, req.ClinicalText.InvestigationResults)

	var m map[string]interface{}
	require.NoError(t, json.Unmarshal(raw, &m))
	ct := m["clinical_text"].(map[string]interface{})
	_, exists := ct["investigation_results"]
	assert.False(t, exists, "nil investigation_results should be omitted from JSON")
}

func TestBuildMLScoreRequest_InvestigationResults_WhitespaceOnlyOmitted(t *testing.T) {
	inv := "   "
	f := baseForm()
	f.InvestigationResults = &inv
	ref := &entity.Referral{
		ReferralForm: f,
		Vitals:       []entity.Vital{baseVitals()},
	}
	req, _, err := BuildMLScoreRequest(ref)
	require.NoError(t, err)
	assert.Empty(t, req.ClinicalText.InvestigationResults)
}

// Only investigation_results (no other text fields) should NOT allow scoring
// — `hasClinicalText` includes it but condition_at_referral is not text evidence.
func TestBuildMLScoreRequest_InvestigationResultsAloneAllowsScoring(t *testing.T) {
	inv := "Hb: 4.1 g/dL, WBC elevated"
	ref := &entity.Referral{
		ReferralForm: &entity.ReferralForm{
			ReasonOfReferral:    "",
			ClinicalSummary:     "",
			PatientHistory:      "",
			ConditionAtReferral: "critical",
			InvestigationResults: &inv,
		},
		Vitals: []entity.Vital{baseVitals()},
	}
	_, _, err := BuildMLScoreRequest(ref)
	// investigation_results is included in hasClinicalText check → should pass
	require.NoError(t, err)
}

// ── extended vitals: respiratory_rate and gcs_score ─────────────────────────

func TestBuildMLScoreRequest_ExtendedVitals_PopulatedWhenPresent(t *testing.T) {
	v := baseVitals()
	rr := int16(28)
	gcs := int16(9)
	v.RespiratoryRate = &rr
	v.GCSScore = &gcs

	ref := &entity.Referral{
		ReferralForm: baseForm(),
		Vitals:       []entity.Vital{v},
	}
	req, raw, err := BuildMLScoreRequest(ref)
	require.NoError(t, err)
	require.NotNil(t, req.Vitals.RespiratoryRate)
	assert.Equal(t, 28, *req.Vitals.RespiratoryRate)
	require.NotNil(t, req.Vitals.GCSScore)
	assert.Equal(t, 9, *req.Vitals.GCSScore)

	var m map[string]interface{}
	require.NoError(t, json.Unmarshal(raw, &m))
	vitals := m["vitals"].(map[string]interface{})
	assert.Equal(t, float64(28), vitals["respiratory_rate"])
	assert.Equal(t, float64(9), vitals["gcs_score"])
}

func TestBuildMLScoreRequest_ExtendedVitals_OmittedWhenNil(t *testing.T) {
	ref := &entity.Referral{
		ReferralForm: baseForm(),
		Vitals:       []entity.Vital{baseVitals()}, // no RR or GCS
	}
	req, raw, err := BuildMLScoreRequest(ref)
	require.NoError(t, err)
	assert.Nil(t, req.Vitals.RespiratoryRate)
	assert.Nil(t, req.Vitals.GCSScore)

	var m map[string]interface{}
	require.NoError(t, json.Unmarshal(raw, &m))
	vitals := m["vitals"].(map[string]interface{})
	_, hasRR := vitals["respiratory_rate"]
	_, hasGCS := vitals["gcs_score"]
	assert.False(t, hasRR, "absent RR should be omitted from JSON")
	assert.False(t, hasGCS, "absent GCS should be omitted from JSON")
}

func TestBuildMLScoreRequest_ExtendedVitals_OutOfRangeOmitted(t *testing.T) {
	// RR=2 is below the valid range (5-60), GCS=16 is above (3-15)
	v := baseVitals()
	rr := int16(2)
	gcs := int16(16)
	v.RespiratoryRate = &rr
	v.GCSScore = &gcs

	ref := &entity.Referral{
		ReferralForm: baseForm(),
		Vitals:       []entity.Vital{v},
	}
	req, _, err := BuildMLScoreRequest(ref)
	require.NoError(t, err)
	assert.Nil(t, req.Vitals.RespiratoryRate, "out-of-range RR should be omitted")
	assert.Nil(t, req.Vitals.GCSScore, "out-of-range GCS should be omitted")
}

func TestBuildMLScoreRequest_ExtendedVitals_BoundaryValues(t *testing.T) {
	// RR exactly at boundaries (5 and 60 should be included)
	v := baseVitals()
	rr := int16(5)
	gcs := int16(3)
	v.RespiratoryRate = &rr
	v.GCSScore = &gcs

	ref := &entity.Referral{ReferralForm: baseForm(), Vitals: []entity.Vital{v}}
	req, _, err := BuildMLScoreRequest(ref)
	require.NoError(t, err)
	require.NotNil(t, req.Vitals.RespiratoryRate)
	assert.Equal(t, 5, *req.Vitals.RespiratoryRate)
	require.NotNil(t, req.Vitals.GCSScore)
	assert.Equal(t, 3, *req.Vitals.GCSScore)

	v2 := baseVitals()
	rr2 := int16(60)
	gcs2 := int16(15)
	v2.RespiratoryRate = &rr2
	v2.GCSScore = &gcs2
	ref2 := &entity.Referral{ReferralForm: baseForm(), Vitals: []entity.Vital{v2}}
	req2, _, err2 := BuildMLScoreRequest(ref2)
	require.NoError(t, err2)
	assert.Equal(t, 60, *req2.Vitals.RespiratoryRate)
	assert.Equal(t, 15, *req2.Vitals.GCSScore)
}

// ── patient context ──────────────────────────────────────────────────────────

func TestBuildMLScoreRequest_PatientContext_FullyPopulated(t *testing.T) {
	dob := time.Now().AddDate(-35, 0, 0)
	region := entity.EthiopianRegion("Oromia")
	ref := &entity.Referral{
		ReferralForm: baseForm(),
		Vitals:       []entity.Vital{baseVitals()},
		Patient: &entity.Patient{
			DateOfBirth: &dob,
			Sex:         "male",
			HomeRegion:  &region,
		},
	}
	req, raw, err := BuildMLScoreRequest(ref)
	require.NoError(t, err)
	require.NotNil(t, req.PatientContext)
	require.NotNil(t, req.PatientContext.AgeYears)
	assert.Equal(t, 35, *req.PatientContext.AgeYears)
	assert.Equal(t, "male", req.PatientContext.Sex)
	assert.Equal(t, "Oromia", req.PatientContext.HomeRegion)

	var m map[string]interface{}
	require.NoError(t, json.Unmarshal(raw, &m))
	_, hasCtx := m["patient_context"]
	assert.True(t, hasCtx)
}

func TestBuildMLScoreRequest_PatientContext_NilWhenNoPatient(t *testing.T) {
	ref := &entity.Referral{
		ReferralForm: baseForm(),
		Vitals:       []entity.Vital{baseVitals()},
		Patient:      nil,
	}
	req, raw, err := BuildMLScoreRequest(ref)
	require.NoError(t, err)
	assert.Nil(t, req.PatientContext)

	var m map[string]interface{}
	require.NoError(t, json.Unmarshal(raw, &m))
	_, hasCtx := m["patient_context"]
	assert.False(t, hasCtx, "nil patient_context must be omitted from JSON")
}

func TestBuildMLScoreRequest_PatientContext_NilWhenAllFieldsEmpty(t *testing.T) {
	// Patient exists but has no DOB, no sex, no region
	ref := &entity.Referral{
		ReferralForm: baseForm(),
		Vitals:       []entity.Vital{baseVitals()},
		Patient:      &entity.Patient{Sex: ""},
	}
	req, _, err := BuildMLScoreRequest(ref)
	require.NoError(t, err)
	assert.Nil(t, req.PatientContext, "empty patient context must be nil")
}

func TestBuildMLScoreRequest_PatientContext_SexNormalizedLowercase(t *testing.T) {
	ref := &entity.Referral{
		ReferralForm: baseForm(),
		Vitals:       []entity.Vital{baseVitals()},
		Patient:      &entity.Patient{Sex: "FEMALE"},
	}
	req, _, err := BuildMLScoreRequest(ref)
	require.NoError(t, err)
	require.NotNil(t, req.PatientContext)
	assert.Equal(t, "female", req.PatientContext.Sex)
}

// ── core vitals validation ───────────────────────────────────────────────────

func TestMapVitalsForML_RejectsOutOfRangeCore(t *testing.T) {
	cases := []struct {
		name string
		mutate func(*entity.Vital)
	}{
		{"systolic too low", func(v *entity.Vital) { s := int16(49); v.SystolicBP = &s }},
		{"systolic too high", func(v *entity.Vital) { s := int16(251); v.SystolicBP = &s }},
		{"diastolic too low", func(v *entity.Vital) { d := int16(19); v.DiastolicBP = &d }},
		{"diastolic too high", func(v *entity.Vital) { d := int16(201); v.DiastolicBP = &d }},
		{"hr too low", func(v *entity.Vital) { h := int16(29); v.HeartRate = &h }},
		{"hr too high", func(v *entity.Vital) { h := int16(251); v.HeartRate = &h }},
		{"spo2 too low", func(v *entity.Vital) { s := float32(49); v.SpO2 = &s }},
		{"temp too low", func(v *entity.Vital) { t := float32(29.9); v.Temperature = &t }},
		{"temp too high", func(v *entity.Vital) { t := float32(43.1); v.Temperature = &t }},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			v := baseVitals()
			tc.mutate(&v)
			ref := &entity.Referral{
				ReferralForm: baseForm(),
				Vitals:       []entity.Vital{v},
			}
			_, _, err := BuildMLScoreRequest(ref)
			assert.Error(t, err, "expected validation error for: %s", tc.name)
		})
	}
}

func TestMapVitalsForML_RejectsPartialVitals(t *testing.T) {
	// Only systolic present — diastolic nil → should error
	sys := int16(120)
	ref := &entity.Referral{
		ReferralForm: baseForm(),
		Vitals:       []entity.Vital{{SystolicBP: &sys}},
	}
	_, _, err := BuildMLScoreRequest(ref)
	assert.ErrorIs(t, err, errMLVitalsMissing)
}

// ── clinical text guard ──────────────────────────────────────────────────────

func TestBuildMLScoreRequest_SkipsWhenAllClinicalTextEmpty(t *testing.T) {
	ref := &entity.Referral{
		ReferralForm: &entity.ReferralForm{
			ReasonOfReferral:    "",
			ClinicalSummary:     "",
			PatientHistory:      "",
			ConditionAtReferral: "critical", // condition alone is not "clinical text"
		},
		Vitals: []entity.Vital{baseVitals()},
	}
	_, _, err := BuildMLScoreRequest(ref)
	assert.ErrorIs(t, err, errMLClinicalEmpty)
}

func TestBuildMLScoreRequest_NilReferralForm(t *testing.T) {
	_, _, err := BuildMLScoreRequest(&entity.Referral{})
	assert.Error(t, err)
}

func TestBuildMLScoreRequest_NilReferral(t *testing.T) {
	_, _, err := BuildMLScoreRequest(nil)
	assert.Error(t, err)
}

// ── JSON serialization round-trip ────────────────────────────────────────────

func TestBuildMLScoreRequest_JSONRoundTrip_AllFields(t *testing.T) {
	inv := "WBC 14k, CRP elevated"
	rr := int16(22)
	gcs := int16(13)
	v := baseVitals()
	v.RespiratoryRate = &rr
	v.GCSScore = &gcs

	dob := time.Now().AddDate(-42, 0, 0)
	region := entity.EthiopianRegion("Amhara")

	ref := &entity.Referral{
		ReferralForm: &entity.ReferralForm{
			ReasonOfReferral:             "Sepsis",
			ClinicalSummary:              "High fever, confusion",
			PatientHistory:               "Diabetic",
			ConditionAtReferral:          "URGENT", // uppercase — must be normalised
			InvestigationResults:         &inv,
		},
		Vitals: []entity.Vital{v},
		Diagnoses: []entity.ReferralDiagnosis{{
			IsPrimary: true,
			ICDCode:   "A41.9",
			CodeInfo:  &entity.ICDCode{Code: "A41.9", Description: "Sepsis, unspecified organism"},
		}},
		Patient: &entity.Patient{
			DateOfBirth: &dob,
			Sex:         "Female",
			HomeRegion:  &region,
		},
	}

	req, raw, err := BuildMLScoreRequest(ref)
	require.NoError(t, err)

	// Struct assertions
	assert.Equal(t, "urgent", req.ClinicalText.ConditionAtReferral)
	assert.Equal(t, "WBC 14k, CRP elevated", req.ClinicalText.InvestigationResults)
	assert.Equal(t, 22, *req.Vitals.RespiratoryRate)
	assert.Equal(t, 13, *req.Vitals.GCSScore)
	assert.Equal(t, "female", req.PatientContext.Sex)
	assert.Equal(t, "Amhara", req.PatientContext.HomeRegion)

	// JSON key assertions
	var m map[string]interface{}
	require.NoError(t, json.Unmarshal(raw, &m))
	ct := m["clinical_text"].(map[string]interface{})
	assert.Equal(t, "urgent", ct["condition_at_referral"])
	assert.Equal(t, "WBC 14k, CRP elevated", ct["investigation_results"])
	assert.Equal(t, "Sepsis", ct["reason_for_referral"])
	vitals := m["vitals"].(map[string]interface{})
	assert.Equal(t, float64(22), vitals["respiratory_rate"])

	pc := m["patient_context"].(map[string]interface{})
	assert.Equal(t, "female", pc["sex"])
	assert.Equal(t, "Amhara", pc["home_region"])
}

// ── diagnosis label extraction ───────────────────────────────────────────────

func TestPrimaryDiagnosisLabel_PrefersIsPrimary(t *testing.T) {
	diagnoses := []entity.ReferralDiagnosis{
		{IsPrimary: false, ICDCode: "Z00.0", CodeInfo: &entity.ICDCode{Description: "Secondary"}},
		{IsPrimary: true, ICDCode: "A41.9", CodeInfo: &entity.ICDCode{Description: "Primary sepsis"}},
	}
	assert.Equal(t, "Primary sepsis", primaryDiagnosisLabel(diagnoses))
}

func TestPrimaryDiagnosisLabel_FallsBackToFirstWithDescription(t *testing.T) {
	diagnoses := []entity.ReferralDiagnosis{
		{IsPrimary: false, ICDCode: "A41.9", CodeInfo: &entity.ICDCode{Description: "Sepsis"}},
	}
	assert.Equal(t, "Sepsis", primaryDiagnosisLabel(diagnoses))
}

func TestPrimaryDiagnosisLabel_FallsBackToICDCode(t *testing.T) {
	diagnoses := []entity.ReferralDiagnosis{
		{IsPrimary: false, ICDCode: "A41.9", CodeInfo: &entity.ICDCode{Description: ""}},
	}
	assert.Equal(t, "A41.9", primaryDiagnosisLabel(diagnoses))
}

func TestPrimaryDiagnosisLabel_EmptySlice(t *testing.T) {
	assert.Equal(t, "", primaryDiagnosisLabel(nil))
	assert.Equal(t, "", primaryDiagnosisLabel([]entity.ReferralDiagnosis{}))
}

// ── ml.ScoreRequest struct matches expected JSON keys ────────────────────────

func TestScoreRequestJSONKeys(t *testing.T) {
	rr := 18
	gcs := 14
	age := 30
	req := ml.ScoreRequest{
		Vitals: ml.VitalsPayload{
			BPSystolic:      120,
			BPDiastolic:     80,
			HeartRate:       72,
			SpO2:            98,
			Temperature:     37,
			RespiratoryRate: &rr,
			GCSScore:        &gcs,
		},
		ClinicalText: ml.ClinicalTextPayload{
			ReasonForReferral:    "test",
			ConditionAtReferral:  "stable",
			InvestigationResults: "normal",
		},
		PatientContext: &ml.PatientContextPayload{
			AgeYears:   &age,
			Sex:        "male",
			HomeRegion: "Tigray",
		},
	}

	raw, err := json.Marshal(req)
	require.NoError(t, err)

	var m map[string]interface{}
	require.NoError(t, json.Unmarshal(raw, &m))

	vitals := m["vitals"].(map[string]interface{})
	assert.Contains(t, vitals, "bp_systolic")
	assert.Contains(t, vitals, "respiratory_rate")
	assert.Contains(t, vitals, "gcs_score")

	ct := m["clinical_text"].(map[string]interface{})
	assert.Contains(t, ct, "condition_at_referral")
	assert.Contains(t, ct, "investigation_results")

	pc := m["patient_context"].(map[string]interface{})
	assert.Contains(t, pc, "age_years")
	assert.Contains(t, pc, "sex")
	assert.Contains(t, pc, "home_region")
}
