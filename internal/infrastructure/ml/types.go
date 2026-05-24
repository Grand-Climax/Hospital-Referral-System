package ml

// ScoreRequest matches the ML service POST /score body.
// All fields in VitalsPayload and ClinicalTextPayload mirror the column
// names the ML model was trained on so the keys must not be renamed
// without coordinating with the ML team. Optional fields are
// omitempty so they are omitted when not collected rather than sent
// as zero values, which the model treats as "unknown".
type ScoreRequest struct {
	Vitals         VitalsPayload          `json:"vitals"`
	ClinicalText   ClinicalTextPayload    `json:"clinical_text"`
	PatientContext *PatientContextPayload `json:"patient_context,omitempty"`
}

type VitalsPayload struct {
	// Required core vitals — ML scoring is skipped if any are missing.
	BPSystolic  int     `json:"bp_systolic"`
	BPDiastolic int     `json:"bp_diastolic"`
	HeartRate   int     `json:"heart_rate"`
	SpO2        float64 `json:"spo2"`
	Temperature float64 `json:"temperature"`

	// Optional extended vitals — sent when recorded by the clinician.
	// The model uses them when present; a missing field is treated as
	// "not measured" rather than normal.
	RespiratoryRate *int `json:"respiratory_rate,omitempty"`
	GCSScore        *int `json:"gcs_score,omitempty"`
}

type ClinicalTextPayload struct {
	// combined_note is a catch-all field that can hold a single
	// free-text block instead of individual fields.
	CombinedNote string `json:"combined_note,omitempty"`

	// Individual clinical text fields — at least one must be non-empty
	// or scoring is skipped.
	ReasonForReferral       string `json:"reason_for_referral,omitempty"`
	ChiefComplaint          string `json:"chief_complaint,omitempty"`
	HistoryOfPresentIllness string `json:"history_of_present_illness,omitempty"`
	PhysicalExamination     string `json:"physical_examination,omitempty"`
	Diagnosis               string `json:"diagnosis,omitempty"`
	TreatmentGiven          string `json:"treatment_given,omitempty"`

	// Additional fields populated from the referral form that give the
	// model extra context. These were not in the original payload and
	// are the primary reason critical referrals were under-scored.
	InvestigationResults string `json:"investigation_results,omitempty"`

	// condition_at_referral is the clinician-assigned severity label
	// (critical / urgent / stable / unstable). Sending it explicitly
	// alongside the free text ensures the model sees the exact label
	// rather than having to infer it from prose.
	ConditionAtReferral string `json:"condition_at_referral,omitempty"`
}

// PatientContextPayload carries demographic metadata that correlates
// with severity and outcome in the training data. All fields are
// optional; nil/empty values are omitted from the JSON.
type PatientContextPayload struct {
	AgeYears   *int   `json:"age_years,omitempty"`
	Sex        string `json:"sex,omitempty"`
	HomeRegion string `json:"home_region,omitempty"`
}

type ScoreResponse struct {
	PredictionID     string   `json:"prediction_id"`
	SeverityScore    float64  `json:"severity_score"`
	SeverityTier     string   `json:"severity_tier"`
	Explanations     []string `json:"explanations"`
	ModelVersion     string   `json:"model_version"`
	ProcessingTimeMs float64  `json:"processing_time_ms"`
}

type HealthResponse struct {
	Status       string `json:"status"`
	ModelsLoaded bool   `json:"models_loaded"`
	ModelVersion string `json:"model_version"`
}

type FeedbackRequest struct {
	PredictionID      string  `json:"prediction_id"`
	WasOverridden     bool    `json:"was_overridden"`
	CorrectedScore    *int    `json:"corrected_score,omitempty"`
	DoctorExplanation *string `json:"doctor_explanation,omitempty"`
}

type FeedbackResponse struct {
	Status        string `json:"status"`
	PredictionID  string `json:"prediction_id"`
	WasOverridden bool   `json:"was_overridden"`
	TrainingLabel int    `json:"training_label"`
	Message       string `json:"message"`
}
