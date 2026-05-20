package usecase

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"Hospital-Referral-System/internal/domain/entity"
	"Hospital-Referral-System/internal/infrastructure/ml"
)

var errMLVitalsMissing = errors.New("vitals incomplete for ML scoring")
var errMLClinicalEmpty = errors.New("no clinical text fields for ML scoring")

// BuildMLScoreRequest maps a referral into the ML POST /score payload.
// Returns an error when vitals or clinical text cannot satisfy ML validation.
func BuildMLScoreRequest(ref *entity.Referral) (ml.ScoreRequest, json.RawMessage, error) {
	if ref == nil || ref.ReferralForm == nil {
		return ml.ScoreRequest{}, nil, errors.New("referral form missing")
	}

	vitals, err := mapVitalsForML(ref.Vitals)
	if err != nil {
		return ml.ScoreRequest{}, nil, err
	}

	clinical := mapClinicalTextForML(ref)
	if !hasClinicalText(clinical) {
		return ml.ScoreRequest{}, nil, errMLClinicalEmpty
	}

	req := ml.ScoreRequest{
		Vitals:       vitals,
		ClinicalText: clinical,
	}

	raw, err := json.Marshal(req)
	if err != nil {
		return ml.ScoreRequest{}, nil, err
	}
	return req, raw, nil
}

func mapVitalsForML(vitals []entity.Vital) (ml.VitalsPayload, error) {
	if len(vitals) == 0 {
		return ml.VitalsPayload{}, errMLVitalsMissing
	}
	v := vitals[0]
	if v.SystolicBP == nil || v.DiastolicBP == nil || v.HeartRate == nil || v.SpO2 == nil || v.Temperature == nil {
		return ml.VitalsPayload{}, errMLVitalsMissing
	}

	sys := int(*v.SystolicBP)
	dia := int(*v.DiastolicBP)
	hr := int(*v.HeartRate)
	spo2 := float64(*v.SpO2)
	temp := float64(*v.Temperature)

	if sys < 50 || sys > 250 {
		return ml.VitalsPayload{}, fmt.Errorf("bp_systolic %d outside ML range 50-250", sys)
	}
	if dia < 20 || dia > 200 {
		return ml.VitalsPayload{}, fmt.Errorf("bp_diastolic %d outside ML range 20-200", dia)
	}
	if hr < 30 || hr > 250 {
		return ml.VitalsPayload{}, fmt.Errorf("heart_rate %d outside ML range 30-250", hr)
	}
	if spo2 < 50 || spo2 > 100 {
		return ml.VitalsPayload{}, fmt.Errorf("spo2 %.1f outside ML range 50-100", spo2)
	}
	if temp < 30.0 || temp > 43.0 {
		return ml.VitalsPayload{}, fmt.Errorf("temperature %.1f outside ML range 30.0-43.0", temp)
	}

	return ml.VitalsPayload{
		BPSystolic:  sys,
		BPDiastolic: dia,
		HeartRate:   hr,
		SpO2:        spo2,
		Temperature: temp,
	}, nil
}

func mapClinicalTextForML(ref *entity.Referral) ml.ClinicalTextPayload {
	f := ref.ReferralForm
	out := ml.ClinicalTextPayload{
		ReasonForReferral:       strings.TrimSpace(f.ReasonOfReferral),
		ChiefComplaint:          strings.TrimSpace(f.ClinicalSummary),
		HistoryOfPresentIllness: strings.TrimSpace(f.PatientHistory),
	}
	if f.PhysicalExaminationFindings != nil {
		out.PhysicalExamination = strings.TrimSpace(*f.PhysicalExaminationFindings)
	}
	if f.TreatmentGivenBeforeReferral != nil {
		out.TreatmentGiven = strings.TrimSpace(*f.TreatmentGivenBeforeReferral)
	}
	out.Diagnosis = primaryDiagnosisLabel(ref.Diagnoses)
	return out
}

func primaryDiagnosisLabel(diagnoses []entity.ReferralDiagnosis) string {
	for _, d := range diagnoses {
		if d.IsPrimary && d.CodeInfo != nil && d.CodeInfo.Description != "" {
			return strings.TrimSpace(d.CodeInfo.Description)
		}
	}
	for _, d := range diagnoses {
		if d.CodeInfo != nil && d.CodeInfo.Description != "" {
			return strings.TrimSpace(d.CodeInfo.Description)
		}
	}
	if len(diagnoses) > 0 {
		return strings.TrimSpace(diagnoses[0].ICDCode)
	}
	return ""
}

func hasClinicalText(c ml.ClinicalTextPayload) bool {
	fields := []string{
		c.CombinedNote,
		c.ReasonForReferral,
		c.ChiefComplaint,
		c.HistoryOfPresentIllness,
		c.PhysicalExamination,
		c.Diagnosis,
		c.TreatmentGiven,
	}
	for _, f := range fields {
		if f != "" {
			return true
		}
	}
	return false
}
