// Package main is a one-shot end-to-end exerciser for the referral
// lifecycle. Run AFTER the seeder has populated the canonical
// hospitals/departments/users (we don't truncate; we only insert the
// two new patients and one new Y12 referring doctor, deleting any prior
// run's leftovers by national_id first).
//
// Flow:
//   Y12 (Secondary, Internal Med) --> Tikur Anbessa (Tertiary, Cardiology)
//   Muhammed  (Amhara,      +251947753588) -> happy path, Amharic SMS
//   Jaefer    (Addis Ababa, +251714855110) -> missed -> reschedule, English SMS
//
// All step results, tokens, referral ids and SMS rows land in
// scratch/e2e_lifecycle/output.json so the user (and any subsequent run)
// can inspect what happened without scraping logs.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"Hospital-Referral-System/internal/domain/entity"
	"Hospital-Referral-System/internal/infrastructure/crypto"
)

var _ = uuid.Nil // keep uuid import for buildPatient

const (
	baseURL = "http://localhost:8081/api/v1"

	// Canonical IDs from the seeder
	hospY12   = "a9000000-0000-0000-0000-000000000009"
	hospTA    = "a1000000-0000-0000-0000-000000000001"
	deptCard  = "b1000000-0000-0000-0000-000000000001"
	deptInt   = "b4000000-0000-0000-0000-000000000004"

	// Pre-existing seeded users we will log in as
	y12Liaison   = "liaison.y12@hospital.et"
	taLiaison    = "liaison.ta@hospital.et"
	taSpecialist = "specialist.ta@hospital.et"
	taReception  = "reception.ta@hospital.et"
	taDeptHead   = "depthead.ta@hospital.et"

	// New user we insert ourselves (seeder has no doctor at Y12)
	y12DoctorID    = "d9000000-0000-0000-0000-000000000004"
	y12DoctorEmail = "doctor.y12@hospital.et"
	y12DoctorPwd   = "password123"

	// Patients
	jaeferID       = "e1000000-0000-0000-0000-00000000000a"
	muhammedID     = "e1000000-0000-0000-0000-00000000000b"
	jaeferNatID    = "NAT-E2E-JAEFER-001"
	muhammedNatID  = "NAT-E2E-MUHAMMED-001"

	icdCode = "I10" // Essential (primary) hypertension - confirmed in icd_codes table

	defaultPwd = "password123"
)

// ───────────────────────── output structure ─────────────────────────

type Output struct {
	StartedAt time.Time              `json:"started_at"`
	BaseURL   string                 `json:"base_url"`
	Users     map[string]UserToken   `json:"users"`
	Patients  map[string]PatientInfo `json:"patients"`
	Flows     map[string]*FlowResult `json:"flows"`
	Errors    []string               `json:"errors"`
	EndedAt   time.Time              `json:"ended_at"`
}

type UserToken struct {
	UserID       string `json:"user_id"`
	Email        string `json:"email"`
	Role         string `json:"role"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token,omitempty"`
}

type PatientInfo struct {
	ID         string `json:"id"`
	NationalID string `json:"national_id"`
	Phone      string `json:"phone"`
	Region     string `json:"region"`
	Language   string `json:"language"`
}

type FlowResult struct {
	Label      string              `json:"label"`
	ReferralID string              `json:"referral_id"`
	Steps      []StepResult        `json:"steps"`
	SMSCheck   []NotificationCheck `json:"sms_check"`
}

type StepResult struct {
	N        int             `json:"step"`
	Name     string          `json:"name"`
	Method   string          `json:"method"`
	Path     string          `json:"path"`
	Status   int             `json:"status"`
	OK       bool            `json:"ok"`
	Body     json.RawMessage `json:"body,omitempty"`
	Error    string          `json:"error,omitempty"`
	Duration string          `json:"duration"`
}

type NotificationCheck struct {
	NotificationType string `json:"notification_type"`
	DeliveryStatus   string `json:"delivery_status"`
	Phone            string `json:"phone"`
	ContentPreview   string `json:"content_preview"`
	Language         string `json:"language_detected"`
	Lifecycle        string `json:"lifecycle_step"`
	CreatedAt        string `json:"created_at"`
}

// ───────────────────────── globals ─────────────────────────

var (
	db        *gorm.DB
	cryptoSvc *crypto.PatientCryptoService
	out       = &Output{
		StartedAt: time.Now(),
		BaseURL:   baseURL,
		Users:     map[string]UserToken{},
		Patients:  map[string]PatientInfo{},
		Flows:     map[string]*FlowResult{},
	}
	outputPath = filepath.Join("scratch", "e2e_lifecycle", "output.json")
)

func main() {
	// Find repo root so we can load .env.local + write output.json in
	// the same paths whether the user runs us via `go run scratch/...`
	// or from inside the e2e_lifecycle folder.
	if _, err := os.Stat(".env.local"); err != nil {
		_ = os.Chdir("..")
		_ = os.Chdir("..")
	}

	if err := godotenv.Load(".env.local"); err != nil {
		log.Printf("[warn] .env.local not loaded: %v", err)
	}

	mustConnectDB()
	mustInitCrypto()

	step("setup", func() error { return setupCleanupAndSeed() })
	step("config", func() error { return ensureAutoNotify() })

	loginAll()

	runHappyPath()
	runMissedPath()

	finalize()
}

// ───────────────────────── setup / DB helpers ─────────────────────────

func mustConnectDB() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL not set in .env.local")
	}
	conn, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}
	db = conn
}

func mustInitCrypto() {
	aesKey := os.Getenv("PATIENT_AES_KEY")
	hmacKey := os.Getenv("PATIENT_HMAC_KEY")
	if aesKey == "" || hmacKey == "" {
		log.Fatal("PATIENT_AES_KEY / PATIENT_HMAC_KEY not set; cannot encrypt patients")
	}
	svc, err := crypto.NewPatientCryptoService(aesKey, hmacKey)
	if err != nil {
		log.Fatalf("crypto init: %v", err)
	}
	cryptoSvc = svc
}

func setupCleanupAndSeed() error {
	ctx := context.Background()

	// 1. Wipe any prior run's data for these two patients + the Y12 doctor.
	natHashes := []string{
		cryptoSvc.GenerateHMAC(jaeferNatID),
		cryptoSvc.GenerateHMAC(muhammedNatID),
	}
	// Resolve patient + referral IDs first so the cascade deletes can
	// use a simple `IN (?)` with a uuid slice (gorm cleanly expands a
	// slice with the `IN ?` form, which is portable across drivers).
	var patientIDs []string
	if err := db.WithContext(ctx).Raw(
		`SELECT id::text FROM patients WHERE national_id_hash IN ?`, natHashes,
	).Scan(&patientIDs).Error; err != nil {
		return fmt.Errorf("lookup patients: %w", err)
	}
	if len(patientIDs) > 0 {
		var referralIDs []string
		if err := db.WithContext(ctx).Raw(
			`SELECT id::text FROM referrals WHERE patient_id IN ?`, patientIDs,
		).Scan(&referralIDs).Error; err != nil {
			return fmt.Errorf("lookup referrals: %w", err)
		}
		if len(referralIDs) > 0 {
			refScoped := []string{
				"notifications", "in_app_notifications", "audit_logs",
				"triage_queues", "ml_predictions", "clinical_updates",
				"referral_outcomes", "referral_accesses",
				"referral_redirections", "referral_status_histories",
				"referral_diagnoses", "vitals",
				"referral_emergency_details", "referral_forms",
				"attachments",
			}
			for _, t := range refScoped {
				if err := db.WithContext(ctx).Exec(
					fmt.Sprintf(`DELETE FROM %s WHERE referral_id IN ?`, t),
					referralIDs,
				).Error; err != nil {
					return fmt.Errorf("wipe %s: %w", t, err)
				}
			}
			if err := db.WithContext(ctx).Exec(
				`DELETE FROM referrals WHERE id IN ?`, referralIDs,
			).Error; err != nil {
				return fmt.Errorf("wipe referrals: %w", err)
			}
		}
		if err := db.WithContext(ctx).Exec(
			`DELETE FROM patients WHERE id IN ?`, patientIDs,
		).Error; err != nil {
			return fmt.Errorf("wipe patients: %w", err)
		}
	}
	if err := db.WithContext(ctx).Exec(`DELETE FROM sessions WHERE user_id = ?`, y12DoctorID).Error; err != nil {
		return fmt.Errorf("wipe y12 doctor sessions: %w", err)
	}

	// 2. Upsert Y12 referring doctor (seeder has none, and we can't
	// hard-delete because audit_logs reference the user). Re-hash the
	// password every run so the harness is hermetic.
	hash, _ := bcrypt.GenerateFromPassword([]byte(y12DoctorPwd), 12)
	if err := db.WithContext(ctx).Exec(`
		INSERT INTO users (id, national_id, email, first_name, middle_name, last_name,
		                   role, hospital_id, region, password_hash, is_active, is_deleted)
		VALUES (?, ?, ?, ?, '', ?, ?, ?, ?, ?, true, false)
		ON CONFLICT (id) DO UPDATE SET
			email         = EXCLUDED.email,
			first_name    = EXCLUDED.first_name,
			last_name     = EXCLUDED.last_name,
			role          = EXCLUDED.role,
			hospital_id   = EXCLUDED.hospital_id,
			region        = EXCLUDED.region,
			password_hash = EXCLUDED.password_hash,
			is_active     = true,
			is_deleted    = false,
			deleted_at    = NULL,
			updated_at    = NOW()
	`, y12DoctorID, "DOC-Y12-001", y12DoctorEmail, "Selam", "Doctor",
		entity.RoleReferringDoctor, hospY12, entity.RegionAddisAbaba, string(hash)).Error; err != nil {
		return fmt.Errorf("upsert y12 doctor: %w", err)
	}

	// 3. Insert two patients (encrypted fields).
	dob, _ := time.Parse("2006-01-02", "1990-06-15")
	jaefer := buildPatient(jaeferID, "Jaefer", "Ali", "Mohammed",
		"+251714855110", jaeferNatID, entity.RegionAddisAbaba, dob, "male")
	muhammed := buildPatient(muhammedID, "Muhammed", "Samson", "Jemal",
		"+251947753588", muhammedNatID, entity.RegionAmhara, dob, "male")
	if err := db.WithContext(ctx).Create(&jaefer).Error; err != nil {
		return fmt.Errorf("insert jaefer: %w", err)
	}
	if err := db.WithContext(ctx).Create(&muhammed).Error; err != nil {
		return fmt.Errorf("insert muhammed: %w", err)
	}

	out.Patients["jaefer"] = PatientInfo{
		ID: jaeferID, NationalID: jaeferNatID, Phone: "+251714855110",
		Region: string(entity.RegionAddisAbaba), Language: "English (en)"}
	out.Patients["muhammed"] = PatientInfo{
		ID: muhammedID, NationalID: muhammedNatID, Phone: "+251947753588",
		Region: string(entity.RegionAmhara), Language: "Amharic (am)"}
	return nil
}

func buildPatient(id, first, mid, last, phone, natID string, region entity.EthiopianRegion, dob time.Time, sex string) entity.Patient {
	enc := func(s string) string { v, _ := cryptoSvc.Encrypt([]byte(s)); return v }
	encPtr := func(s string) *string { v := enc(s); return &v }
	hashPtr := func(s string) *string { h := cryptoSvc.GenerateHMAC(s); return &h }
	normPhone, _ := crypto.NormalizePhone(phone)
	return entity.Patient{
		ID:             uuid.MustParse(id),
		FirstNameEnc:   enc(first),
		MiddleNameEnc:  enc(mid),
		LastNameEnc:    enc(last),
		Sex:            sex,
		DateOfBirth:    &dob,
		HomeRegion:     &region,
		AllowSMS:       true,
		PhoneNumberEnc: encPtr(normPhone),
		PhoneHash:      hashPtr(normPhone),
		NationalIDEnc:  encPtr(natID),
		NationalIDHash: hashPtr(natID),
	}
}

func ensureAutoNotify() error {
	ctx := context.Background()
	// Upsert auto_notify = true so every QueueNotification triggers
	// a real SMS send via AfroMessage. Also flip enable_cron_jobs in
	// case the manual flush ever runs.
	for k, v := range map[string]string{
		"auto_notify":      "true",
		"enable_cron_jobs": "true",
	} {
		if err := db.WithContext(ctx).Exec(`
			INSERT INTO system_configs (key, value) VALUES (?, ?)
			ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value
		`, k, v).Error; err != nil {
			return fmt.Errorf("system_configs %s: %w", k, err)
		}
	}
	return nil
}

// ───────────────────────── HTTP helpers ─────────────────────────

func loginAll() {
	required := map[string]bool{
		"y12_doctor":      true,
		"y12_liaison":     true,
		"ta_liaison":      true,
		"ta_specialist":   true,
		"ta_receptionist": true,
		"ta_depthead":     false, // captured for convenience only
	}
	logins := []struct {
		key, email, role string
	}{
		{"y12_doctor", y12DoctorEmail, "REFERRING_DOCTOR"},
		{"y12_liaison", y12Liaison, "LIAISON_OFFICER"},
		{"ta_liaison", taLiaison, "LIAISON_OFFICER"},
		{"ta_specialist", taSpecialist, "RECEIVING_SPECIALIST"},
		{"ta_receptionist", taReception, "RECEPTIONIST"},
		{"ta_depthead", taDeptHead, "DEPT_HEAD"},
	}
	for _, l := range logins {
		l := l
		stepCtx := "login_" + l.key
		mandatory := required[l.key]
		start := time.Now()
		err := loginOne(l.key, l.email, l.role)
		if err != nil {
			msg := fmt.Sprintf("[%s] FAIL after %s: %v", stepCtx, time.Since(start), err)
			if mandatory {
				out.Errors = append(out.Errors, msg)
				finalize()
				log.Fatal(msg)
			}
			log.Printf("[%s] skipped (non-fatal): %v", stepCtx, err)
			out.Errors = append(out.Errors, "[non-fatal] "+msg)
			continue
		}
		log.Printf("[%s] ok (%s)", stepCtx, time.Since(start))
	}
}

func loginOne(key, email, role string) error {
	body, _ := json.Marshal(map[string]string{"email": email, "password": defaultPwd})
	resp, status, err := doRaw(http.MethodPost, "/auth/login", "", body, "application/json")
	if err != nil {
		return err
	}
	if status != 200 {
		return fmt.Errorf("status=%d body=%s", status, string(resp))
	}
	var parsed struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		MFAToken     string `json:"mfa_token"`
	}
	if err := json.Unmarshal(resp, &parsed); err != nil {
		return fmt.Errorf("parse: %w", err)
	}
	if parsed.AccessToken == "" {
		return fmt.Errorf("no access_token (mfa_token=%q)", parsed.MFAToken)
	}
	out.Users[key] = UserToken{
		Email: email, Role: role,
		AccessToken: parsed.AccessToken, RefreshToken: parsed.RefreshToken,
	}
	return nil
}

func tokenOf(key string) string { return out.Users[key].AccessToken }

func doRaw(method, path, token string, body []byte, contentType string) ([]byte, int, error) {
	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}
	req, err := http.NewRequest(method, baseURL+path, bodyReader)
	if err != nil {
		return nil, 0, err
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	return b, resp.StatusCode, err
}

func doJSON(flow *FlowResult, name, method, path, tokenKey string, payload interface{}, wantStatuses ...int) []byte {
	var body []byte
	if payload != nil {
		body, _ = json.Marshal(payload)
	}
	return record(flow, name, method, path, tokenKey, body, "application/json", wantStatuses...)
}

func record(flow *FlowResult, name, method, path, tokenKey string, body []byte, ct string, wantStatuses ...int) []byte {
	start := time.Now()
	respBody, status, err := doRaw(method, path, tokenOf(tokenKey), body, ct)
	sr := StepResult{
		N: len(flow.Steps) + 1, Name: name, Method: method, Path: path,
		Status: status, Duration: time.Since(start).String(),
	}
	if err != nil {
		sr.Error = err.Error()
		flow.Steps = append(flow.Steps, sr)
		die(flow, fmt.Errorf("%s: %w", name, err))
	}
	sr.OK = false
	if len(wantStatuses) == 0 {
		sr.OK = status >= 200 && status < 300
	} else {
		for _, w := range wantStatuses {
			if status == w {
				sr.OK = true
				break
			}
		}
	}
	// Embed the response body (truncated if huge) for debugging.
	if len(respBody) > 0 && len(respBody) < 8000 {
		sr.Body = json.RawMessage(respBody)
	} else if len(respBody) > 0 {
		sr.Body = json.RawMessage(respBody[:8000])
	}
	if !sr.OK {
		sr.Error = fmt.Sprintf("unexpected status %d", status)
	}
	flow.Steps = append(flow.Steps, sr)
	if !sr.OK {
		die(flow, fmt.Errorf("%s -> %d: %s", name, status, string(respBody)))
	}
	return respBody
}

// ───────────────────────── lifecycle paths ─────────────────────────

func runHappyPath() {
	flow := &FlowResult{Label: "Muhammed (Amhara, Amharic) - happy path"}
	out.Flows["muhammed"] = flow

	refID := submitReferral(flow, "y12_doctor", muhammedID, "UNSTABLE",
		"Recurrent chest pain, palpitations", "Hypertension, diabetes 8y")
	flow.ReferralID = refID

	liaisonForward(flow, refID, "y12_liaison")
	// Schedule for TODAY so arrive succeeds (arrival only allowed on
	// the scheduled date - by design, see arrival_usecase.go:72).
	specialistAcceptAndSchedule(flow, refID, todayISO())
	receptionistConfirmArrival(flow, refID)

	collectSMS(flow, refID)
}

func runMissedPath() {
	flow := &FlowResult{Label: "Jaefer (Addis Ababa, English) - missed -> rescheduled -> arrived"}
	out.Flows["jaefer"] = flow

	refID := submitReferral(flow, "y12_doctor", jaeferID, "UNSTABLE",
		"Chest pain on exertion; suspected angina", "Hypertension 5y, smoker")
	flow.ReferralID = refID

	liaisonForward(flow, refID, "y12_liaison")
	specialistAcceptAndSchedule(flow, refID, todayISO())
	receptionistMarkMissed(flow, refID)
	// Emergency reschedule (override) to TODAY so the eventual arrive
	// step lands on the right day.
	specialistEmergencySchedule(flow, refID, todayISO())
	receptionistConfirmArrival(flow, refID)

	collectSMS(flow, refID)
}

func todayISO() string {
	return time.Now().Format("2006-01-02T15:04:05Z")
}

func submitReferral(flow *FlowResult, doctorKey, patientID, condition, summary, history string) string {
	// build multipart body
	payload := map[string]interface{}{
		"patient_id":          patientID,
		"target_hospital_id":  hospTA,
		"target_dept_id":      deptCard,
		"liaison_officer_id":  "d9000000-0000-0000-0000-000000000002",
		"clinical_summary":    summary,
		"patient_history":     history,
		"reason_of_referral":  "Cardiology evaluation required",
		"condition_at_referral": condition,
		"status":              "SUBMITTED",
		"diagnoses": []map[string]interface{}{
			{"icd_code": icdCode, "is_primary": true, "diagnosis_certainty": "SUSPECTED"},
		},
		"vitals": map[string]interface{}{
			"systolic_bp":      150,
			"diastolic_bp":     95,
			"heart_rate":       102,
			"sp_o2":            96.5,
			"temperature":      37.1,
			"respiratory_rate": 22,
			"gcs_score":        15,
		},
	}
	jsonBytes, _ := json.Marshal(payload)

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	_ = mw.WriteField("referral", string(jsonBytes))
	_ = mw.WriteField("attachment_category", "GENERAL_CLINICAL")
	mw.Close()

	body := record(flow, "doctor_create_submit", http.MethodPost,
		"/doctor/referrals", doctorKey, buf.Bytes(), mw.FormDataContentType(), 201)

	var parsed struct {
		Referral struct{ ID string } `json:"referral"`
		// Some response shapes nest differently; try a few.
		Data struct{ ID string } `json:"data"`
		ID   string              `json:"id"`
	}
	_ = json.Unmarshal(body, &parsed)
	if parsed.Referral.ID != "" {
		return parsed.Referral.ID
	}
	if parsed.Data.ID != "" {
		return parsed.Data.ID
	}
	if parsed.ID != "" {
		return parsed.ID
	}
	die(flow, fmt.Errorf("could not extract referral ID from response: %s", string(body)))
	return ""
}

func liaisonForward(flow *FlowResult, refID, liaisonKey string) {
	doJSON(flow, "liaison_read", http.MethodPost,
		"/liaison/referrals/"+refID+"/read", liaisonKey, nil)

	// Fill the review checklist (all four flags) so Forward passes
	checklist := map[string]bool{
		"patient_identity_verified": true,
		"clinical_history_attached": true,
		"vitals_included":           true,
		"attachments_included":      true,
	}
	doJSON(flow, "liaison_checklist", http.MethodPut,
		"/liaison/referrals/"+refID+"/review-checklist", liaisonKey, checklist)

	doJSON(flow, "liaison_forward", http.MethodPost,
		"/liaison/referrals/"+refID+"/forward", liaisonKey, nil)
}

func specialistAcceptAndSchedule(flow *FlowResult, refID, apptDate string) {
	doJSON(flow, "specialist_read", http.MethodPost,
		"/specialist/referrals/"+refID+"/read", "ta_specialist", nil)

	forceMLOverride(flow, refID)

	doJSON(flow, "specialist_accept", http.MethodPost,
		"/specialist/referrals/"+refID+"/accept", "ta_specialist", map[string]interface{}{})
	doJSON(flow, "specialist_schedule", http.MethodPost,
		"/specialist/referrals/"+refID+"/schedule", "ta_specialist", map[string]interface{}{
			"appointment_date": apptDate,
			"notes":            "First available cardiology slot",
		})
}

// forceMLOverride skips the ML wait entirely and just stamps a manual
// severity score so Accept is never blocked by a cold-starting Cloud
// Run ML instance. The harness is a lifecycle exerciser, not an ML
// smoke test - if you want to test the ML pipeline, use rerun-ml or
// the ML service directly.
func forceMLOverride(flow *FlowResult, refID string) {
	doJSON(flow, "ml_override", http.MethodPost,
		"/specialist/referrals/"+refID+"/ml-severity-override", "ta_specialist",
		map[string]interface{}{
			"score":         55.0,
			"justification": "E2E lifecycle harness: deterministic override (score=55, MEDIUM tier) so Accept never blocks on async ML.",
		})
}

func specialistEmergencySchedule(flow *FlowResult, refID, apptDate string) {
	// Emergency-schedule accepts YYYY-MM-DD only
	dateOnly := apptDate[:10]
	doJSON(flow, "specialist_emergency_schedule", http.MethodPost,
		"/specialist/referrals/"+refID+"/emergency-schedule", "ta_specialist", map[string]string{
			"appointment_date": dateOnly,
			"justification":    "Patient previously missed; clinical instability flagged",
		})
}

func receptionistConfirmArrival(flow *FlowResult, refID string) {
	doJSON(flow, "receptionist_confirm_arrival", http.MethodPost,
		"/receptionist/referrals/"+refID+"/arrive", "ta_receptionist", nil)
}

func receptionistMarkMissed(flow *FlowResult, refID string) {
	doJSON(flow, "receptionist_mark_missed", http.MethodPost,
		"/receptionist/referrals/"+refID+"/miss", "ta_receptionist", map[string]string{
			"miss_reason": "PATIENT_NO_SHOW",
		})
}


// ───────────────────────── SMS verification ─────────────────────────

func collectSMS(flow *FlowResult, refID string) {
	type row struct {
		NotificationType string    `gorm:"column:notification_type"`
		DeliveryStatus   string    `gorm:"column:delivery_status"`
		PhoneNumber      string    `gorm:"column:phone_number"`
		Content          string    `gorm:"column:content"`
		CreatedAt        time.Time `gorm:"column:created_at"`
	}
	var rows []row
	db.Raw(`SELECT notification_type, delivery_status, phone_number, content, created_at
	        FROM notifications WHERE referral_id = ? ORDER BY created_at ASC`, refID).Scan(&rows)
	for _, r := range rows {
		flow.SMSCheck = append(flow.SMSCheck, NotificationCheck{
			NotificationType: r.NotificationType,
			DeliveryStatus:   r.DeliveryStatus,
			Phone:            r.PhoneNumber,
			ContentPreview:   truncate(r.Content, 200),
			Language:         detectLang(r.Content),
			Lifecycle:        flow.Label,
			CreatedAt:        r.CreatedAt.Format(time.RFC3339),
		})
	}
}

func detectLang(s string) string {
	for _, r := range s {
		switch {
		case r >= 0x1200 && r <= 0x137F:
			return "am (Amharic)"
		}
	}
	if strings.Contains(s, "Rifeeraaliin") || strings.Contains(s, "Beellam") {
		return "om (Afaan Oromoo)"
	}
	return "en (English)"
}

// ───────────────────────── misc helpers ─────────────────────────

func step(name string, fn func() error) {
	start := time.Now()
	err := fn()
	if err != nil {
		msg := fmt.Sprintf("[%s] FAIL after %s: %v", name, time.Since(start), err)
		out.Errors = append(out.Errors, msg)
		finalize()
		log.Fatal(msg)
	}
	log.Printf("[%s] ok (%s)", name, time.Since(start))
}

func die(flow *FlowResult, err error) {
	out.Errors = append(out.Errors,
		fmt.Sprintf("[flow=%s ref=%s] %v", flow.Label, flow.ReferralID, err))
	finalize()
	log.Fatal(err)
}

func finalize() {
	out.EndedAt = time.Now()
	b, _ := json.MarshalIndent(out, "", "  ")
	if err := os.WriteFile(outputPath, b, 0644); err != nil {
		log.Printf("[warn] writing output: %v", err)
		return
	}
	log.Printf("wrote %s", outputPath)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

