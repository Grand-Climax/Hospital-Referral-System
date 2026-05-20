package ml

// ScoreRequest matches the ML service POST /score body.
type ScoreRequest struct {
	Vitals        VitalsPayload        `json:"vitals"`
	ClinicalText  ClinicalTextPayload  `json:"clinical_text"`
}

type VitalsPayload struct {
	BPSystolic   int     `json:"bp_systolic"`
	BPDiastolic  int     `json:"bp_diastolic"`
	HeartRate    int     `json:"heart_rate"`
	SpO2         float64 `json:"spo2"`
	Temperature  float64 `json:"temperature"`
}

type ClinicalTextPayload struct {
	CombinedNote              string `json:"combined_note,omitempty"`
	ReasonForReferral         string `json:"reason_for_referral,omitempty"`
	ChiefComplaint            string `json:"chief_complaint,omitempty"`
	HistoryOfPresentIllness   string `json:"history_of_present_illness,omitempty"`
	PhysicalExamination       string `json:"physical_examination,omitempty"`
	Diagnosis                 string `json:"diagnosis,omitempty"`
	TreatmentGiven            string `json:"treatment_given,omitempty"`
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
	PredictionID       string  `json:"prediction_id"`
	WasOverridden      bool    `json:"was_overridden"`
	CorrectedScore     *int    `json:"corrected_score,omitempty"`
	DoctorExplanation  *string `json:"doctor_explanation,omitempty"`
}

type FeedbackResponse struct {
	Status        string `json:"status"`
	PredictionID  string `json:"prediction_id"`
	WasOverridden bool   `json:"was_overridden"`
	TrainingLabel int    `json:"training_label"`
	Message       string `json:"message"`
}
