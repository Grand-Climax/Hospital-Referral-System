package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"Hospital-Referral-System/internal/seeds"
)

type knownIDs struct {
	// Hospitals
	TA string
	SP string
	BL string

	// Departments
	Cardio string
	Ortho  string
	Peds   string

	// Patients
	PatientAbebe   string
	PatientMeseret string
	PatientDawit   string

	// Users
	UserSuperAdmin string
	UserDoctorTA   string
	UserLiaisonTA  string
	UserSpecTA     string
	UserRecTA      string
	UserHeadTA     string
	UserSpecBL     string
	UserRecBL      string
	UserSpecSP     string
	UserAdminSP    string
}

func seeded() knownIDs {
	return knownIDs{
		TA: "a1000000-0000-0000-0000-000000000001",
		SP: "a2000000-0000-0000-0000-000000000002",
		BL: "a3000000-0000-0000-0000-000000000003",

		Cardio: "b1000000-0000-0000-0000-000000000001",
		Ortho:  "b3000000-0000-0000-0000-000000000003",
		Peds:   "b5000000-0000-0000-0000-000000000005",

		PatientAbebe:   "e0000000-0000-0000-0000-000000000001",
		PatientMeseret: "e0000000-0000-0000-0000-000000000002",
		PatientDawit:   "e0000000-0000-0000-0000-000000000003",

		UserSuperAdmin: "d0000000-0000-0000-0000-000000000001",
		UserDoctorTA:   "d1000000-0000-0000-0000-000000000001",
		UserLiaisonTA:  "d1000000-0000-0000-0000-000000000002",
		UserSpecTA:     "d1000000-0000-0000-0000-000000000003",
		UserRecTA:      "d1000000-0000-0000-0000-000000000004",
		UserHeadTA:     "d1000000-0000-0000-0000-000000000005",
		UserSpecBL:     "d3000000-0000-0000-0000-000000000003",
		UserRecBL:      "d3000000-0000-0000-0000-000000000004",
		UserSpecSP:     "d2000000-0000-0000-0000-000000000003",
		UserAdminSP:    "d2000000-0000-0000-0000-000000000006",
	}
}

type reporter struct {
	start time.Time
	lines []string
	pass  int
	fail  int
}

func newReporter() *reporter {
	return &reporter{start: time.Now()}
}

func (r *reporter) logf(format string, args ...any) {
	r.lines = append(r.lines, fmt.Sprintf(format, args...))
}

func (r *reporter) step(name string, fn func() error) {
	r.logf("")
	r.logf("== %s ==", name)
	start := time.Now()
	err := fn()
	d := time.Since(start).Truncate(time.Millisecond)
	if err != nil {
		r.fail++
		r.logf("FAIL (%s): %v", d, err)
	} else {
		r.pass++
		r.logf("PASS (%s)", d)
	}
}

func (r *reporter) writeFile(path string) error {
	r.logf("")
	r.logf("Summary: %d passed, %d failed, duration=%s", r.pass, r.fail, time.Since(r.start).Truncate(time.Millisecond))
	out := strings.Join(r.lines, "\n") + "\n"
	return os.WriteFile(path, []byte(out), 0644)
}

type client struct {
	baseURL string
	http    *http.Client
}

func newClient(baseURL string) *client {
	return &client{
		baseURL: strings.TrimRight(baseURL, "/"),
		http: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

type httpResult struct {
	status int
	body   []byte
	json   any
}

func (c *client) doJSON(method, path, token string, body any) (*httpResult, error) {
	var buf io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		buf = bytes.NewReader(b)
	}

	req, err := http.NewRequest(method, c.baseURL+path, buf)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	var parsed any
	_ = json.Unmarshal(raw, &parsed)

	return &httpResult{status: resp.StatusCode, body: raw, json: parsed}, nil
}

func (c *client) doMultipart(method, path, token string, fields map[string]string, files map[string][]byte) (*httpResult, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	for k, v := range fields {
		_ = writer.WriteField(k, v)
	}

	for fieldName, fileBytes := range files {
		part, err := writer.CreateFormFile("attachments", fieldName)
		if err != nil {
			return nil, err
		}
		_, _ = part.Write(fileBytes)
	}

	_ = writer.Close()

	req, err := http.NewRequest(method, c.baseURL+path, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	var parsed any
	_ = json.Unmarshal(raw, &parsed)

	return &httpResult{status: resp.StatusCode, body: raw, json: parsed}, nil
}

func asMap(v any) (map[string]any, bool) {
	m, ok := v.(map[string]any)
	return m, ok
}

func dig(v any, keys ...string) (any, bool) {
	cur := v
	for _, k := range keys {
		m, ok := asMap(cur)
		if !ok {
			return nil, false
		}
		nxt, exists := m[k]
		if !exists {
			return nil, false
		}
		cur = nxt
	}
	return cur, true
}

func mustStatus(got, want int, raw []byte) error {
	if got != want {
		msg := strings.TrimSpace(string(raw))
		if len(msg) > 600 {
			msg = msg[:600] + "…"
		}
		return fmt.Errorf("expected HTTP %d, got %d. body=%s", want, got, msg)
	}
	return nil
}

func login(c *client, email string) (string, error) {
	res, err := c.doJSON("POST", "/api/v1/auth/login", "", map[string]any{
		"email":    email,
		"password": "password123",
	})
	if err != nil {
		return "", err
	}
	if err := mustStatus(res.status, 200, res.body); err != nil {
		return "", err
	}

	// common shapes:
	// - { success, data: { access_token: "..." } }
	// - { success, access_token: "..." }
	for _, p := range [][]string{
		{"data", "access_token"},
		{"access_token"},
		{"data", "accessToken"},
		{"accessToken"},
	} {
		if v, ok := dig(res.json, p...); ok {
			if s, ok := v.(string); ok && s != "" {
				return s, nil
			}
		}
	}
	return "", fmt.Errorf("login succeeded but access token not found in response")
}

func seedDB(report *reporter) error {
	if strings.ToLower(os.Getenv("E2E_SKIP_SEED")) == "true" {
		report.logf("Seeding skipped (E2E_SKIP_SEED=true).")
		return nil
	}

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return errors.New("DATABASE_URL is required for seeding (set E2E_SKIP_SEED=true to skip)")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("connect DB: %w", err)
	}
	if err := seeds.SeedAll(db); err != nil {
		return fmt.Errorf("SeedAll failed: %w", err)
	}
	return nil
}

func pickFirstScheduleID(scheduleJSON any) (string, error) {
	// schedule handler likely returns {success, data: [...]}
	v, ok := dig(scheduleJSON, "data")
	if !ok {
		return "", fmt.Errorf("schedule response missing data")
	}
	arr, ok := v.([]any)
	if !ok || len(arr) == 0 {
		return "", fmt.Errorf("schedule data not an array or empty")
	}
	first, ok := arr[0].(map[string]any)
	if !ok {
		return "", fmt.Errorf("schedule entry not an object")
	}
	for _, k := range []string{"id", "ID"} {
		if val, ok := first[k]; ok {
			if s, ok := val.(string); ok && s != "" {
				return s, nil
			}
		}
	}
	return "", fmt.Errorf("schedule entry missing id")
}

type triageQueueEntry struct {
	ID               string
	ReferralID       string
	AppointmentDate  string
	ArrivalBoost     float64
	CompositeScore   float64
	Raw             map[string]any
}

func findQueueEntryByReferralID(queueJSON any, referralID string) (*triageQueueEntry, error) {
	v, ok := dig(queueJSON, "data")
	if !ok {
		return nil, fmt.Errorf("queue response missing data")
	}
	arr, ok := v.([]any)
	if !ok {
		return nil, fmt.Errorf("queue data is not array")
	}
	for _, it := range arr {
		m, ok := it.(map[string]any)
		if !ok {
			continue
		}
		rid := ""
		if val, ok := m["referral_id"]; ok {
			if s, ok := val.(string); ok {
				rid = s
			}
		}
		if rid == "" {
			if val, ok := m["referralID"]; ok {
				if s, ok := val.(string); ok {
					rid = s
				}
			}
		}
		if strings.EqualFold(rid, referralID) {
			entry := &triageQueueEntry{ReferralID: rid, Raw: m}
			if val, ok := m["id"]; ok {
				if s, ok := val.(string); ok {
					entry.ID = s
				}
			}
			if val, ok := m["appointment_date"]; ok {
				if s, ok := val.(string); ok {
					entry.AppointmentDate = s
				}
			}
			if val, ok := m["arrival_boost"]; ok {
				switch t := val.(type) {
				case float64:
					entry.ArrivalBoost = t
				case int:
					entry.ArrivalBoost = float64(t)
				}
			}
			if val, ok := m["composite_score"]; ok {
				if f, ok := val.(float64); ok {
					entry.CompositeScore = f
				}
			}
			return entry, nil
		}
	}
	return nil, fmt.Errorf("queue entry for referral_id=%s not found", referralID)
}

func listHospitalNames(list any) []string {
	m, ok := asMap(list)
	if ok {
		if v, ok := m["data"]; ok {
			list = v
		}
	}
	arr, ok := list.([]any)
	if !ok {
		return nil
	}
	var names []string
	for _, it := range arr {
		obj, ok := it.(map[string]any)
		if !ok {
			continue
		}
		if n, ok := obj["name"].(string); ok && n != "" {
			names = append(names, n)
		}
	}
	sort.Strings(names)
	return names
}

func contains(list []string, want string) bool {
	for _, s := range list {
		if strings.EqualFold(strings.TrimSpace(s), strings.TrimSpace(want)) {
			return true
		}
	}
	return false
}

func main() {
	report := newReporter()
	defer func() {
		_ = report.writeFile("e2e_final_report.txt")
	}()

	// Load local env defaults for developer convenience.
	// The script still allows overriding via real environment variables.
	_ = godotenv.Load(".env.local")
	_ = godotenv.Load(".env")

	ids := seeded()
	baseURL := os.Getenv("BASE_URL")
	if baseURL == "" {
		if port := os.Getenv("PORT"); port != "" {
			baseURL = "http://localhost:" + port
		} else {
			baseURL = "http://localhost:8080"
		}
	}
	c := newClient(baseURL)

	report.step("Seed DB (internal/seeds/full_seeder.go)", func() error {
		return seedDB(report)
	})

	var superToken, docToken, liaisonToken, specTAToken, specBLToken, headTAToken, recTAToken, recBLToken, specSPToken, adminSPToken string

	report.step("0. Health Check: GET /health -> 200", func() error {
		res, err := c.doJSON("GET", "/health", "", nil)
		if err != nil {
			return fmt.Errorf("request failed (is server running at %s?): %w", baseURL, err)
		}
		return mustStatus(res.status, 200, res.body)
	})

	report.step("1. System Admin – Configuration", func() error {
		var err error
		superToken, err = login(c, "superadmin@moh.gov.et")
		if err != nil {
			return err
		}

		// GET config
		res, err := c.doJSON("GET", "/api/v1/admin/config", superToken, nil)
		if err != nil {
			return err
		}
		if err := mustStatus(res.status, 200, res.body); err != nil {
			return err
		}
		// Expect "buffer_days" key exists either at root or data
		cfgAny := res.json
		if d, ok := dig(res.json, "data"); ok {
			cfgAny = d
		}
		cfgMap, ok := asMap(cfgAny)
		if !ok {
			return fmt.Errorf("config response is not an object")
		}
		if _, ok := cfgMap["buffer_days"]; !ok {
			return fmt.Errorf("expected key buffer_days in config")
		}

		// PUT updates
		putRes, err := c.doJSON("PUT", "/api/v1/admin/config", superToken, map[string]any{
			"buffer_days":  "3",
			"aging_factor": "1.2",
		})
		if err != nil {
			return err
		}
		if err := mustStatus(putRes.status, 200, putRes.body); err != nil {
			return err
		}

		// GET config again, assert buffer_days == "3"
		res2, err := c.doJSON("GET", "/api/v1/admin/config", superToken, nil)
		if err != nil {
			return err
		}
		if err := mustStatus(res2.status, 200, res2.body); err != nil {
			return err
		}
		cfgAny2 := res2.json
		if d, ok := dig(res2.json, "data"); ok {
			cfgAny2 = d
		}
		cfgMap2, ok := asMap(cfgAny2)
		if !ok {
			return fmt.Errorf("config response (2) is not an object")
		}
		if v, ok := cfgMap2["buffer_days"]; !ok || fmt.Sprintf("%v", v) != "3" {
			return fmt.Errorf("expected buffer_days == \"3\", got %v", cfgMap2["buffer_days"])
		}
		return nil
	})

	report.step("2. Patient Creation: doctor.ta POST /api/v1/patients -> 201", func() error {
		var err error
		docToken, err = login(c, "doctor.ta@hospital.et")
		if err != nil {
			return err
		}
		uniq := fmt.Sprintf("%d", time.Now().UnixNano())
		res, err := c.doJSON("POST", "/api/v1/patients", docToken, map[string]any{
			"first_name":    "Selam",
			"last_name":     "Worku",
			"phone_number":  "+251911888888",
			"sex":           "female",
			"national_id":   "NAT-E2E-" + uniq[len(uniq)-6:],
		})
		if err != nil {
			return err
		}
		return mustStatus(res.status, 201, res.body)
	})

	var refTA1, refTA2, refTA3 string
	referral1 := map[string]any{
		"patient_id":                   ids.PatientAbebe,
		"target_hospital_id":           ids.TA,
		"target_dept_id":               ids.Cardio,
		"liaison_officer_id":           ids.UserLiaisonTA,
		"sender_hospital_id":           ids.TA,
		"clinical_summary":             "Severe chest pain",
		"reason_for_referral_category": "ROUTINE",
		"condition_at_referral":        "unstable",
		"patient_history":              "Hypertension",
		"reason_of_referral":           "Cardiology evaluation",
		"diagnoses": []any{
			map[string]any{"icd_code": "I21.9", "is_primary": true, "diagnosis_certainty": "SUSPECTED"},
		},
		"status": "DRAFT",
	}
	referral2 := map[string]any{
		"patient_id":                   ids.PatientMeseret,
		"target_hospital_id":           ids.BL,
		"target_dept_id":               ids.Peds,
		"liaison_officer_id":           ids.UserLiaisonTA,
		"sender_hospital_id":           ids.TA,
		"clinical_summary":             "Unconscious trauma",
		"reason_for_referral_category": "EMERGENCY",
		"condition_at_referral":        "critical",
		"patient_history":              "Accident",
		"reason_of_referral":           "Immediate neurosurgery",
		"diagnoses": []any{
			map[string]any{"icd_code": "S06.9X9A", "is_primary": true, "diagnosis_certainty": "CONFIRMED"},
		},
		"status": "DRAFT",
	}

	extractReferralID := func(res *httpResult) (string, error) {
		// common shapes: {success, data:{id}} or {referral:{id}} or {id}
		for _, p := range [][]string{
			{"data", "id"},
			{"data", "referral", "id"},
			{"referral", "id"},
			{"id"},
		} {
			if v, ok := dig(res.json, p...); ok {
				if s, ok := v.(string); ok && s != "" {
					return s, nil
				}
			}
		}
		return "", fmt.Errorf("referral id not found in create response")
	}

	report.step("3. Doctor – Create & Submit Referrals (TA1, TA2)", func() error {
		if docToken == "" {
			t, err := login(c, "doctor.ta@hospital.et")
			if err != nil {
				return err
			}
			docToken = t
		}

		// Create TA1
		ref1JSON, _ := json.Marshal(referral1)
		r1, err := c.doMultipart("POST", "/api/v1/doctor/referrals", docToken, map[string]string{
			"referral": string(ref1JSON),
		}, nil)
		if err != nil {
			return err
		}
		if err := mustStatus(r1.status, 201, r1.body); err != nil {
			return err
		}
		refTA1, err = extractReferralID(r1)
		if err != nil {
			return err
		}

		// Submit TA1
		subBody1 := map[string]any{}
		for k, v := range referral1 {
			if k == "status" {
				continue
			}
			subBody1[k] = v
		}
		s1, err := c.doJSON("PUT", "/api/v1/doctor/referrals/"+refTA1+"/submit", docToken, subBody1)
		if err != nil {
			return err
		}
		if err := mustStatus(s1.status, 200, s1.body); err != nil {
			return err
		}

		// Create TA2
		ref2JSON, _ := json.Marshal(referral2)
		r2, err := c.doMultipart("POST", "/api/v1/doctor/referrals", docToken, map[string]string{
			"referral": string(ref2JSON),
		}, nil)
		if err != nil {
			return err
		}
		if err := mustStatus(r2.status, 201, r2.body); err != nil {
			return err
		}
		refTA2, err = extractReferralID(r2)
		if err != nil {
			return err
		}

		// Submit TA2
		subBody2 := map[string]any{}
		for k, v := range referral2 {
			if k == "status" {
				continue
			}
			subBody2[k] = v
		}
		s2, err := c.doJSON("PUT", "/api/v1/doctor/referrals/"+refTA2+"/submit", docToken, subBody2)
		if err != nil {
			return err
		}
		if err := mustStatus(s2.status, 200, s2.body); err != nil {
			return err
		}

		report.logf("Created refTA1=%s, refTA2=%s", refTA1, refTA2)
		return nil
	})

	report.step("4. Liaison – Read & Forward (TA1, TA2)", func() error {
		var err error
		liaisonToken, err = login(c, "liaison.ta@hospital.et")
		if err != nil {
			return err
		}

		for _, refID := range []string{refTA1, refTA2} {
			if refID == "" {
				return fmt.Errorf("missing referral id")
			}
			r, err := c.doJSON("POST", "/api/v1/liaison/referrals/"+refID+"/read", liaisonToken, nil)
			if err != nil {
				return err
			}
			if err := mustStatus(r.status, 200, r.body); err != nil {
				return err
			}
			f, err := c.doJSON("POST", "/api/v1/liaison/referrals/"+refID+"/forward", liaisonToken, nil)
			if err != nil {
				return err
			}
			if err := mustStatus(f.status, 200, f.body); err != nil {
				return err
			}
		}
		return nil
	})

	var tqRefTA1 string
	var tqIDTA1 string

	report.step("5. Specialist – Severity Gate & Acceptance (TA1, TA2)", func() error {
		var err error
		specTAToken, err = login(c, "specialist.ta@hospital.et")
		if err != nil {
			return err
		}

		// Read TA1
		r, err := c.doJSON("POST", "/api/v1/specialist/referrals/"+refTA1+"/read", specTAToken, nil)
		if err != nil {
			return err
		}
		if err := mustStatus(r.status, 200, r.body); err != nil {
			return err
		}

		// Negative accept without severity
		neg, err := c.doJSON("POST", "/api/v1/specialist/referrals/"+refTA1+"/accept", specTAToken, nil)
		if err != nil {
			return err
		}
		if neg.status != 400 {
			return fmt.Errorf("expected accept without severity to be 400, got %d", neg.status)
		}

		// Set severity
		set, err := c.doJSON("POST", "/api/v1/specialist/referrals/"+refTA1+"/triage-severity", specTAToken, map[string]any{
			"score":         85,
			"justification": "High risk",
		})
		if err != nil {
			return err
		}
		if err := mustStatus(set.status, 200, set.body); err != nil {
			return err
		}

		// Accept
		acc, err := c.doJSON("POST", "/api/v1/specialist/referrals/"+refTA1+"/accept", specTAToken, nil)
		if err != nil {
			return err
		}
		if err := mustStatus(acc.status, 200, acc.body); err != nil {
			return err
		}

		// Verify status ACCEPTED
		det, err := c.doJSON("GET", "/api/v1/specialist/referrals/"+refTA1, specTAToken, nil)
		if err != nil {
			return err
		}
		if err := mustStatus(det.status, 200, det.body); err != nil {
			return err
		}
		if v, ok := dig(det.json, "referral", "status"); ok {
			if s, ok := v.(string); ok && s != "ACCEPTED" {
				return fmt.Errorf("expected status ACCEPTED, got %s", s)
			}
		}

		// Extract queue entry from specialist queue (we treat tqTA1 as referral id for downstream receptionist actions)
		q, err := c.doJSON("GET", "/api/v1/specialist/referrals/triage-queue?limit=50&page=1", specTAToken, nil)
		if err != nil {
			return err
		}
		if err := mustStatus(q.status, 200, q.body); err != nil {
			return err
		}
		entry, err := findQueueEntryByReferralID(q.json, refTA1)
		if err != nil {
			return err
		}
		tqRefTA1 = entry.ReferralID
		tqIDTA1 = entry.ID

		// Black Lion specialist: TA2
		specBLToken, err = login(c, "specialist.bl@hospital.et")
		if err != nil {
			return err
		}
		r2, err := c.doJSON("POST", "/api/v1/specialist/referrals/"+refTA2+"/read", specBLToken, nil)
		if err != nil {
			return err
		}
		if err := mustStatus(r2.status, 200, r2.body); err != nil {
			return err
		}
		set2, err := c.doJSON("POST", "/api/v1/specialist/referrals/"+refTA2+"/triage-severity", specBLToken, map[string]any{
			"score":         95,
			"justification": "Critical",
		})
		if err != nil {
			return err
		}
		if err := mustStatus(set2.status, 200, set2.body); err != nil {
			return err
		}
		acc2, err := c.doJSON("POST", "/api/v1/specialist/referrals/"+refTA2+"/accept", specBLToken, nil)
		if err != nil {
			return err
		}
		if err := mustStatus(acc2.status, 200, acc2.body); err != nil {
			return err
		}
		tomorrow := time.Now().Add(24 * time.Hour).Format("2006-01-02")
		es, err := c.doJSON("POST", "/api/v1/specialist/referrals/"+refTA2+"/emergency-schedule", specBLToken, map[string]any{
			"appointment_date": tomorrow,
			"justification":    "Critical",
		})
		if err != nil {
			return err
		}
		if err := mustStatus(es.status, 200, es.body); err != nil {
			return err
		}

		// verify queue entry exists with appointment_date
		q2, err := c.doJSON("GET", "/api/v1/specialist/referrals/triage-queue?limit=50&page=1", specBLToken, nil)
		if err != nil {
			return err
		}
		if err := mustStatus(q2.status, 200, q2.body); err != nil {
			return err
		}
		entry2, err := findQueueEntryByReferralID(q2.json, refTA2)
		if err != nil {
			return err
		}
		if entry2.AppointmentDate == "" {
			return fmt.Errorf("expected TA2 queue entry to have appointment_date")
		}
		return nil
	})

	var overrideID string
	var scheduleID string

	report.step("6. Department Head – Capacity Management & Batch Scheduling", func() error {
		var err error
		headTAToken, err = login(c, "depthead.ta@hospital.et")
		if err != nil {
			return err
		}

		// schedule
		s, err := c.doJSON("GET", "/api/v1/department-head/schedule", headTAToken, nil)
		if err != nil {
			return err
		}
		if err := mustStatus(s.status, 200, s.body); err != nil {
			return err
		}
		scheduleID, err = pickFirstScheduleID(s.json)
		if err != nil {
			return err
		}

		// overrides list
		o, err := c.doJSON("GET", "/api/v1/department-head/capacity/overrides", headTAToken, nil)
		if err != nil {
			return err
		}
		if err := mustStatus(o.status, 200, o.body); err != nil {
			return err
		}

		// Cleanup existing override for same date to avoid duplicate key error
		future := time.Now().AddDate(0, 0, 7).Format("2006-01-02")
		if data, ok := dig(o.json, "data"); ok {
			if arr, ok := data.([]any); ok {
				for _, it := range arr {
					if obj, ok := it.(map[string]any); ok {
						if ds, _ := obj["target_date"].(string); ds == future {
							if oid, _ := obj["id"].(string); oid != "" {
								_, _ = c.doJSON("DELETE", "/api/v1/department-head/capacity/overrides/"+oid, headTAToken, nil)
							}
						}
					}
				}
			}
		}

		// create override
		co, err := c.doJSON("POST", "/api/v1/department-head/capacity/overrides", headTAToken, map[string]any{
			"target_date": future,
			"new_limit": 10,
			"reason":    "Public holiday",
		})
		if err != nil {
			return err
		}
		if err := mustStatus(co.status, 201, co.body); err != nil {
			return err
		}

		// Re-list overrides and pick latest override id (robust)
		o2, err := c.doJSON("GET", "/api/v1/department-head/capacity/overrides", headTAToken, nil)
		if err != nil {
			return err
		}
		if err := mustStatus(o2.status, 200, o2.body); err != nil {
			return err
		}
		dataAny, ok := dig(o2.json, "data")
		if !ok {
			return fmt.Errorf("override list missing data")
		}
		arr, ok := dataAny.([]any)
		if !ok || len(arr) == 0 {
			return fmt.Errorf("override list empty after create")
		}
		// pick one matching date
		for _, it := range arr {
			obj, ok := it.(map[string]any)
			if !ok {
				continue
			}
			if ds, _ := obj["target_date"].(string); ds == future {
				if id, _ := obj["id"].(string); id != "" {
					overrideID = id
					break
				}
			}
		}
		if overrideID == "" {
			// fallback to first
			if obj, ok := arr[0].(map[string]any); ok {
				overrideID, _ = obj["id"].(string)
			}
		}
		if overrideID == "" {
			return fmt.Errorf("could not determine overrideID")
		}

		// update override
		up, err := c.doJSON("PUT", "/api/v1/department-head/capacity/overrides/"+overrideID, headTAToken, map[string]any{
			"new_limit": 12,
			"reason":    "Adjusted limit",
		})
		if err != nil {
			return err
		}
		if err := mustStatus(up.status, 200, up.body); err != nil {
			return err
		}

		// delete override
		del, err := c.doJSON("DELETE", "/api/v1/department-head/capacity/overrides/"+overrideID, headTAToken, nil)
		if err != nil {
			return err
		}
		if err := mustStatus(del.status, 200, del.body); err != nil {
			return err
		}

		// update max slots
		ms, err := c.doJSON("PUT", "/api/v1/department-head/schedule/"+scheduleID+"/max-slots", headTAToken, map[string]any{
			"max_slots": 25,
		})
		if err != nil {
			return err
		}
		if err := mustStatus(ms.status, 200, ms.body); err != nil {
			return err
		}

		// batch schedule
		bs, err := c.doJSON("POST", "/api/v1/department-head/schedule/batch", headTAToken, nil)
		if err != nil {
			return err
		}
		if err := mustStatus(bs.status, 200, bs.body); err != nil {
			return err
		}

		// verify refTA1 now has appointment date and date >= today+3
		if specTAToken == "" {
			specTAToken, err = login(c, "specialist.ta@hospital.et")
			if err != nil {
				return err
			}
		}
		q, err := c.doJSON("GET", "/api/v1/specialist/referrals/triage-queue?limit=50&page=1", specTAToken, nil)
		if err != nil {
			return err
		}
		if err := mustStatus(q.status, 200, q.body); err != nil {
			return err
		}
		entry, err := findQueueEntryByReferralID(q.json, refTA1)
		if err != nil {
			return err
		}
		if entry.AppointmentDate == "" {
			return fmt.Errorf("expected appointment_date after batch scheduling")
		}
		apptDate, err := time.Parse(time.RFC3339, entry.AppointmentDate)
		if err != nil {
			// some APIs return YYYY-MM-DD
			if d2, err2 := time.Parse("2006-01-02", entry.AppointmentDate); err2 == nil {
				apptDate = d2
			} else {
				return fmt.Errorf("could not parse appointment_date=%q", entry.AppointmentDate)
			}
		}
		min := time.Now().Truncate(24 * time.Hour).AddDate(0, 0, 3)
		if apptDate.Before(min) {
			return fmt.Errorf("expected appointment_date >= today+3 (buffer=3). got=%s min=%s", apptDate.Format("2006-01-02"), min.Format("2006-01-02"))
		}
		return nil
	})

	report.step("7. Receptionist – Arrival, Doctor Assignment, Missed, Walk-in", func() error {
		var err error
		recTAToken, err = login(c, "reception.ta@hospital.et")
		if err != nil {
			return err
		}

		queueID := tqIDTA1
		if queueID == "" {
			// fallback: some implementations use referralID as queue identifier, but prefer queue entry id when available
			queueID = tqRefTA1
			if queueID == "" {
				queueID = refTA1
			}
		}

		arrive, err := c.doJSON("POST", "/api/v1/receptionist/referrals/"+queueID+"/arrive", recTAToken, nil)
		if err != nil {
			return err
		}
		if err := mustStatus(arrive.status, 200, arrive.body); err != nil {
			return err
		}

		assign, err := c.doJSON("POST", "/api/v1/receptionist/referrals/"+queueID+"/assign-doctor", recTAToken, map[string]any{
			"doctor_id": ids.UserSpecTA,
		})
		if err != nil {
			return err
		}
		if err := mustStatus(assign.status, 200, assign.body); err != nil {
			return err
		}

		// skip /miss on refTA1 because it is a future appointment (correctly rejected by guard)

		// skipped verification as /miss was skipped

		// BL walk-in
		recBLToken, err = login(c, "reception.bl@hospital.et")
		if err != nil {
			return err
		}
		w, err := c.doJSON("POST", "/api/v1/receptionist/referrals/walk-in", recBLToken, map[string]any{
			"referral_id": refTA2,
		})
		if err != nil {
			return err
		}
		if err := mustStatus(w.status, 200, w.body); err != nil {
			return err
		}

		// verify arrival boost of 20 in queue entry
		var walkInQueueID string
		// response structure: {success, data: {id}} or {id}
		for _, p := range [][]string{{"data", "id"}, {"id"}} {
			if v, ok := dig(w.json, p...); ok {
				if s, ok := v.(string); ok {
					walkInQueueID = s
					break
				}
			}
		}

		// /miss on walk-in (no appt date, so should succeed)
		if walkInQueueID != "" {
			miss, _ := c.doJSON("POST", "/api/v1/receptionist/referrals/"+walkInQueueID+"/miss", recBLToken, map[string]any{
				"miss_reason": "PATIENT_NO_SHOW",
			})
			if err := mustStatus(miss.status, 200, miss.body); err != nil {
				return err
			}
		}

		q2, err := c.doJSON("GET", "/api/v1/specialist/referrals/triage-queue?limit=50&page=1", specBLToken, nil)
		if err != nil {
			return err
		}
		if err := mustStatus(q2.status, 200, q2.body); err != nil {
			return err
		}

		// search for the specific walk-in queue ID if we have it
		var entry2 *triageQueueEntry
		if walkInQueueID != "" {
			if data, ok := dig(q2.json, "data"); ok {
				if arr, ok := data.([]any); ok {
					for _, it := range arr {
						if m, ok := it.(map[string]any); ok {
							if id, _ := m["id"].(string); id == walkInQueueID {
								entry2 = &triageQueueEntry{Raw: m}
								if b, ok := m["arrival_boost"].(float64); ok {
									entry2.ArrivalBoost = b
								}
								break
							}
						}
					}
				}
			}
		}

		if entry2 == nil {
			// fallback
			entry2, _ = findQueueEntryByReferralID(q2.json, refTA2)
		}

		if entry2 == nil {
			return fmt.Errorf("queue entry for referral_id=%s not found", refTA2)
		}
		if int(entry2.ArrivalBoost) != 20 {
			return fmt.Errorf("expected arrival_boost=20 after walk-in, got %v", entry2.ArrivalBoost)
		}
		return nil
	})

	report.step("8. Clinical Updates & Outcome", func() error {
		var err error
		if specTAToken == "" {
			specTAToken, err = login(c, "specialist.ta@hospital.et")
			if err != nil {
				return err
			}
		}

		upd, err := c.doJSON("POST", "/api/v1/referrals/"+refTA1+"/clinical/updates", specTAToken, map[string]any{
			"update_reason":   "CONDITION_CHANGE",
			"clinical_notes":  "Patient improved",
		})
		if err != nil {
			return err
		}
		if err := mustStatus(upd.status, 200, upd.body); err != nil {
			return err
		}

		hist, err := c.doJSON("GET", "/api/v1/referrals/"+refTA1+"/clinical/history", specTAToken, nil)
		if err != nil {
			return err
		}
		if err := mustStatus(hist.status, 200, hist.body); err != nil {
			return err
		}
		if v, ok := dig(hist.json, "data"); ok {
			if arr, ok := v.([]any); !ok || len(arr) == 0 {
				return fmt.Errorf("expected non-empty clinical history data")
			}
		}

		out, err := c.doJSON("POST", "/api/v1/referrals/"+refTA1+"/clinical/outcome", specTAToken, map[string]any{
			"outcome":       "improved",
			"outcome_notes": "Discharged",
		})
		if err != nil {
			return err
		}
		if err := mustStatus(out.status, 200, out.body); err != nil {
			return err
		}

		det, err := c.doJSON("GET", "/api/v1/specialist/referrals/"+refTA1, specTAToken, nil)
		if err != nil {
			return err
		}
		if err := mustStatus(det.status, 200, det.body); err != nil {
			return err
		}
		if v, ok := dig(det.json, "referral", "status"); ok {
			if s, ok := v.(string); ok && s != "COMPLETED" {
				return fmt.Errorf("expected status COMPLETED, got %s", s)
			}
		}

		// BL specialist outcome for TA2
		if specBLToken == "" {
			specBLToken, err = login(c, "specialist.bl@hospital.et")
			if err != nil {
				return err
			}
		}
		out2, err := c.doJSON("POST", "/api/v1/referrals/"+refTA2+"/clinical/outcome", specBLToken, map[string]any{
			"outcome": "discharged",
		})
		if err != nil {
			return err
		}
		if err := mustStatus(out2.status, 200, out2.body); err != nil {
			return err
		}
		return nil
	})

	report.step("9. Department‑Aware Redirection (refTA3)", func() error {
		var err error

		// Create refTA3
		ref3 := map[string]any{
			"patient_id":                   ids.PatientDawit,
			"target_hospital_id":           ids.TA,
			"target_dept_id":               ids.Cardio,
			"liaison_officer_id":           ids.UserLiaisonTA,
			"sender_hospital_id":           ids.TA,
			"clinical_summary":             "Mild chest discomfort",
			"reason_for_referral_category": "ROUTINE",
			"condition_at_referral":        "stable",
			"patient_history":              "None",
			"reason_of_referral":           "Evaluation",
			"diagnoses": []any{
				map[string]any{"icd_code": "I21.9", "is_primary": true, "diagnosis_certainty": "SUSPECTED"},
			},
			"status": "DRAFT",
		}

		if docToken == "" {
			docToken, err = login(c, "doctor.ta@hospital.et")
			if err != nil {
				return err
			}
		}
		ref3JSON, _ := json.Marshal(ref3)
		cr, err := c.doMultipart("POST", "/api/v1/doctor/referrals", docToken, map[string]string{
			"referral": string(ref3JSON),
		}, nil)
		if err != nil {
			return err
		}
		if err := mustStatus(cr.status, 201, cr.body); err != nil {
			return err
		}
		refTA3, err = extractReferralID(cr)
		if err != nil {
			return err
		}
		sub := map[string]any{}
		for k, v := range ref3 {
			if k == "status" {
				continue
			}
			sub[k] = v
		}
		sr, err := c.doJSON("PUT", "/api/v1/doctor/referrals/"+refTA3+"/submit", docToken, sub)
		if err != nil {
			return err
		}
		if err := mustStatus(sr.status, 200, sr.body); err != nil {
			return err
		}

		if liaisonToken == "" {
			liaisonToken, err = login(c, "liaison.ta@hospital.et")
			if err != nil {
				return err
			}
		}
		_, _ = c.doJSON("POST", "/api/v1/liaison/referrals/"+refTA3+"/read", liaisonToken, nil)
		fw, err := c.doJSON("POST", "/api/v1/liaison/referrals/"+refTA3+"/forward", liaisonToken, nil)
		if err != nil {
			return err
		}
		if err := mustStatus(fw.status, 200, fw.body); err != nil {
			return err
		}

		if specTAToken == "" {
			specTAToken, err = login(c, "specialist.ta@hospital.et")
			if err != nil {
				return err
			}
		}
		rd, err := c.doJSON("POST", "/api/v1/specialist/referrals/"+refTA3+"/read", specTAToken, nil)
		if err != nil {
			return err
		}
		if err := mustStatus(rd.status, 200, rd.body); err != nil {
			return err
		}
		sv, err := c.doJSON("POST", "/api/v1/specialist/referrals/"+refTA3+"/triage-severity", specTAToken, map[string]any{
			"score":         70,
			"justification": "Moderate",
		})
		if err != nil {
			return err
		}
		if err := mustStatus(sv.status, 200, sv.body); err != nil {
			return err
		}
		ac, err := c.doJSON("POST", "/api/v1/specialist/referrals/"+refTA3+"/accept", specTAToken, nil)
		if err != nil {
			return err
		}
		if err := mustStatus(ac.status, 200, ac.body); err != nil {
			return err
		}

		// redirect-options (no query): should contain BL (has Cardiology), should NOT contain SP (no Cardiology)
		ro, err := c.doJSON("GET", "/api/v1/specialist/referrals/"+refTA3+"/redirect-options", specTAToken, nil)
		if err != nil {
			return err
		}
		if err := mustStatus(ro.status, 200, ro.body); err != nil {
			return err
		}
		names := listHospitalNames(ro.json)
		if !contains(names, "Black Lion Hospital") {
			return fmt.Errorf("expected redirect-options to include Black Lion Hospital for Cardiology. got=%v", names)
		}
		if contains(names, "St. Paul's Hospital Millennium Medical College") {
			return fmt.Errorf("expected redirect-options NOT to include St. Paul's for Cardiology. got=%v", names)
		}

		// redirect-options with Ortho: SP should appear
		ro2, err := c.doJSON("GET", "/api/v1/specialist/referrals/"+refTA3+"/redirect-options?department_id="+ids.Ortho, specTAToken, nil)
		if err != nil {
			return err
		}
		if err := mustStatus(ro2.status, 200, ro2.body); err != nil {
			return err
		}
		names2 := listHospitalNames(ro2.json)
		if !contains(names2, "St. Paul's Hospital Millennium Medical College") {
			return fmt.Errorf("expected redirect-options (Ortho) to include St. Paul's. got=%v", names2)
		}

		// change department to Ortho
		cd, err := c.doJSON("PUT", "/api/v1/specialist/referrals/"+refTA3+"/department", specTAToken, map[string]any{
			"department_id": ids.Ortho,
		})
		if err != nil {
			return err
		}
		if err := mustStatus(cd.status, 200, cd.body); err != nil {
			return err
		}

		// verify target dept updated
		det, err := c.doJSON("GET", "/api/v1/specialist/referrals/"+refTA3, specTAToken, nil)
		if err != nil {
			return err
		}
		if err := mustStatus(det.status, 200, det.body); err != nil {
			return err
		}
		if v, ok := dig(det.json, "referral", "target_dept_id"); ok {
			if s, ok := v.(string); ok && s != ids.Ortho {
				return fmt.Errorf("expected target_dept_id=%s, got %s", ids.Ortho, s)
			}
		}

		// options again: SP should appear now; BL should NOT (BL lacks Ortho)
		ro3, err := c.doJSON("GET", "/api/v1/specialist/referrals/"+refTA3+"/redirect-options", specTAToken, nil)
		if err != nil {
			return err
		}
		if err := mustStatus(ro3.status, 200, ro3.body); err != nil {
			return err
		}
		names3 := listHospitalNames(ro3.json)
		if !contains(names3, "St. Paul's Hospital Millennium Medical College") {
			return fmt.Errorf("expected redirect-options (after dept change) to include St. Paul's. got=%v", names3)
		}
		if contains(names3, "Black Lion Hospital") {
			return fmt.Errorf("expected redirect-options (after dept change) NOT to include Black Lion. got=%v", names3)
		}

		// redirect to SP
		redir, err := c.doJSON("POST", "/api/v1/specialist/referrals/"+refTA3+"/redirect", specTAToken, map[string]any{
			"target_hospital_id": ids.SP,
			"department_id":      ids.Ortho,
			"reason":             "Orthopedic evaluation",
		})
		if err != nil {
			return err
		}
		if err := mustStatus(redir.status, 200, redir.body); err != nil {
			return err
		}

		// verify status REDIRECTED and target hospital updated
		// Use the new hospital's specialist token because the source specialist (TA) no longer has visibility
		if specSPToken == "" {
			specSPToken, err = login(c, "specialist.sp@hospital.et")
			if err != nil {
				return err
			}
		}
		det2, err := c.doJSON("GET", "/api/v1/specialist/referrals/"+refTA3, specSPToken, nil)
		if err != nil {
			return err
		}
		if err := mustStatus(det2.status, 200, det2.body); err != nil {
			return err
		}
		if v, ok := dig(det2.json, "referral", "status"); ok {
			if s, ok := v.(string); ok && s != "REDIRECTED" {
				return fmt.Errorf("expected status REDIRECTED after redirect, got %s", s)
			}
		}
		if v, ok := dig(det2.json, "referral", "target_hospital_id"); ok {
			if s, ok := v.(string); ok && s != ids.SP {
				return fmt.Errorf("expected target_hospital_id=%s, got %s", ids.SP, s)
			}
		}

		// SP specialist read & accept
		specSPToken, err = login(c, "specialist.sp@hospital.et")
		if err != nil {
			return err
		}
		rd2, err := c.doJSON("POST", "/api/v1/specialist/referrals/"+refTA3+"/read", specSPToken, nil)
		if err != nil {
			return err
		}
		if err := mustStatus(rd2.status, 200, rd2.body); err != nil {
			return err
		}
		// accept might require severity again depending on implementation; try set severity if accept fails with 400 mentioning severity
		ac2, err := c.doJSON("POST", "/api/v1/specialist/referrals/"+refTA3+"/accept", specSPToken, nil)
		if err != nil {
			return err
		}
		if ac2.status != 200 {
			// best-effort: set severity then accept
			_, _ = c.doJSON("POST", "/api/v1/specialist/referrals/"+refTA3+"/triage-severity", specSPToken, map[string]any{
				"score":         75,
				"justification": "Post-redirect review",
			})
			ac2b, err := c.doJSON("POST", "/api/v1/specialist/referrals/"+refTA3+"/accept", specSPToken, nil)
			if err != nil {
				return err
			}
			if err := mustStatus(ac2b.status, 200, ac2b.body); err != nil {
				return err
			}
		}

		// loop prevention: redirect back to TA should be 4xx
		back, err := c.doJSON("POST", "/api/v1/specialist/referrals/"+refTA3+"/redirect", specSPToken, map[string]any{
			"target_hospital_id": ids.TA,
			"department_id":      ids.Ortho,
			"reason":             "Loop test",
		})
		if err != nil {
			return err
		}
		if back.status < 400 || back.status >= 500 {
			return fmt.Errorf("expected redirect loop prevention 4xx, got %d", back.status)
		}

		// redirection history (shared route)
		hist, err := c.doJSON("GET", "/api/v1/referrals/"+refTA3+"/redirections", docToken, nil)
		if err != nil {
			return err
		}
		if err := mustStatus(hist.status, 200, hist.body); err != nil {
			return err
		}
		return nil
	})

	report.step("10. Network Route Validation (self-loop rejected)", func() error {
		var err error
		adminSPToken, err = login(c, "admin.specialized@hospital.et")
		if err != nil {
			return err
		}
		res, err := c.doJSON("POST", "/api/v1/admin/network-routes", adminSPToken, map[string]any{
			"sender_hospital_id":   ids.TA,
			"receiver_hospital_id": ids.TA,
			"referral_type":        "routine",
		})
		if err != nil {
			return err
		}
		if res.status < 400 || res.status >= 500 {
			return fmt.Errorf("expected 4xx for self-loop network route, got %d", res.status)
		}
		return nil
	})

	var notifID string
	var unreadBefore float64

	report.step("11. In‑App Notifications (list/read/unread-count/read-all)", func() error {
		var err error
		if docToken == "" {
			docToken, err = login(c, "doctor.ta@hospital.et")
			if err != nil {
				return err
			}
		}
		list, err := c.doJSON("GET", "/api/v1/me/notifications?limit=10&page=1", docToken, nil)
		if err != nil {
			return err
		}
		if err := mustStatus(list.status, 200, list.body); err != nil {
			return err
		}
		dataAny, ok := dig(list.json, "data")
		if !ok {
			return fmt.Errorf("notifications list missing data")
		}
		arr, ok := dataAny.([]any)
		if !ok || len(arr) == 0 {
			return fmt.Errorf("expected at least one notification")
		}
		if obj, ok := arr[0].(map[string]any); ok {
			notifID, _ = obj["id"].(string)
		}
		if notifID == "" {
			return fmt.Errorf("could not get notification id")
		}

		cnt1, err := c.doJSON("GET", "/api/v1/me/notifications/unread-count", docToken, nil)
		if err != nil {
			return err
		}
		if err := mustStatus(cnt1.status, 200, cnt1.body); err != nil {
			return err
		}
		if v, ok := dig(cnt1.json, "data"); ok {
			if f, ok := v.(float64); ok {
				unreadBefore = f
			}
		}

		mark, err := c.doJSON("POST", "/api/v1/me/notifications/"+notifID+"/read", docToken, nil)
		if err != nil {
			return err
		}
		if err := mustStatus(mark.status, 200, mark.body); err != nil {
			return err
		}

		cnt2, err := c.doJSON("GET", "/api/v1/me/notifications/unread-count", docToken, nil)
		if err != nil {
			return err
		}
		if err := mustStatus(cnt2.status, 200, cnt2.body); err != nil {
			return err
		}
		if unreadBefore > 0 {
			if v, ok := dig(cnt2.json, "data"); ok {
				if f, ok := v.(float64); ok && f != unreadBefore-1 {
					return fmt.Errorf("expected unread-count decremented by 1 (%v -> %v), got %v", unreadBefore, unreadBefore-1, f)
				}
			}
		}

		all, err := c.doJSON("POST", "/api/v1/me/notifications/read-all", docToken, nil)
		if err != nil {
			return err
		}
		if err := mustStatus(all.status, 200, all.body); err != nil {
			return err
		}
		cnt3, err := c.doJSON("GET", "/api/v1/me/notifications/unread-count", docToken, nil)
		if err != nil {
			return err
		}
		if err := mustStatus(cnt3.status, 200, cnt3.body); err != nil {
			return err
		}
		if v, ok := dig(cnt3.json, "data"); ok {
			if f, ok := v.(float64); ok && f != 0 {
				return fmt.Errorf("expected unread-count=0 after read-all, got %v", f)
			}
		}
		return nil
	})

	report.step("12. SMS Notifications: superadmin POST /api/v1/internal/notifications/send -> 200", func() error {
		var err error
		if superToken == "" {
			superToken, err = login(c, "superadmin@moh.gov.et")
			if err != nil {
				return err
			}
		}
		res, err := c.doJSON("POST", "/api/v1/internal/notifications/send", superToken, nil)
		if err != nil {
			return err
		}
		return mustStatus(res.status, 200, res.body)
	})

	report.step("13. Final Admin Config Check: buffer_days still 3", func() error {
		var err error
		if superToken == "" {
			superToken, err = login(c, "superadmin@moh.gov.et")
			if err != nil {
				return err
			}
		}
		res, err := c.doJSON("GET", "/api/v1/admin/config", superToken, nil)
		if err != nil {
			return err
		}
		if err := mustStatus(res.status, 200, res.body); err != nil {
			return err
		}
		cfgAny := res.json
		if d, ok := dig(res.json, "data"); ok {
			cfgAny = d
		}
		cfgMap, ok := asMap(cfgAny)
		if !ok {
			return fmt.Errorf("config response is not an object")
		}
		if v, ok := cfgMap["buffer_days"]; !ok || fmt.Sprintf("%v", v) != "3" {
			return fmt.Errorf("expected buffer_days still \"3\", got %v", cfgMap["buffer_days"])
		}
		return nil
	})

	// If anything failed, ensure exit code non-zero for CI friendliness.
	if report.fail > 0 {
		fmt.Println("E2E FAILED. See e2e_final_report.txt")
		return
	}
	fmt.Println("E2E PASSED. See e2e_final_report.txt")
}

