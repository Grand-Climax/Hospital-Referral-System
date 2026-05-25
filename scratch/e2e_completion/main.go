// Package main is a standalone end-to-end exerciser for the
// post-arrival ("completion") arc of the referral lifecycle:
//
//   ARRIVED  -- assign-doctor -->  ARRIVED  (treating doctor linked)
//   ARRIVED  -- clinical update -->  ARRIVED  (note recorded)
//   ARRIVED  -- record-outcome -->  ADMITTED  (referral.status = COMPLETED)
//
// It deliberately does NOT submit a new referral. Instead it peeks at
// the DB and picks an existing ARRIVED + unassigned referral at TA
// Cardiology so it can be re-run any time without flooding the queue.
// SMS is assumed to be off (auto_notify=false) in this harness - that
// is by design, we are testing the state machine, not the notifier.
//
// While at it, the harness exercises every read endpoint a frontend
// receptionist page (incl. the offline IndexedDB prefetch) calls, so
// the FE team can copy-paste the request/response shapes from
// scratch/e2e_completion/output.json.
//
// Usage:
//   1. Make sure the backend is running on http://localhost:8081
//   2. Make sure the e2e_lifecycle harness (or any other flow) has
//      already produced at least one ARRIVED + unassigned referral at
//      TA Cardiology. If not, run scratch/e2e_lifecycle first.
//   3. go run ./scratch/e2e_completion
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"Hospital-Referral-System/internal/domain/entity"
)

const (
	baseURL = "http://localhost:8081/api/v1"

	hospTA   = "a1000000-0000-0000-0000-000000000001"
	deptCard = "b1000000-0000-0000-0000-000000000001"

	// Existing seeded users we log in as. The receptionist is in TA
	// Cardiology, which is exactly the scope we want to test.
	taReceptionEmail = "reception.ta@hospital.et"

	// TA Cardiology treating doctor. The seeder change in this PR adds
	// the user; the harness also upserts defensively so it can be run
	// against a Neon snapshot that hasn't been re-seeded yet.
	taCardioDoctorID    = "d1000000-0000-0000-0000-000000000020"
	taCardioDoctorEmail = "doctor.cardio1.ta@hospital.et"

	// Second TA Cardiology doctor - same hospital + department but
	// explicitly NOT the assigned treating doctor. We use this user
	// purely to confirm the strict outcome gate: even a colleague in
	// the same department gets 403 unless the receptionist has linked
	// them via /assign-doctor.
	taCardioOtherID    = "d1000000-0000-0000-0000-000000000021"
	taCardioOtherEmail = "doctor.cardio2.ta@hospital.et"

	defaultPwd = "password123"
)

// ───────────────────────── output ─────────────────────────

type Output struct {
	StartedAt time.Time              `json:"started_at"`
	BaseURL   string                 `json:"base_url"`
	Users     map[string]UserToken   `json:"users"`
	Picked    PickedReferral         `json:"picked_referral"`
	Steps     []StepResult           `json:"steps"`
	DBChecks  []DBCheck              `json:"db_checks"`
	Errors    []string               `json:"errors"`
	EndedAt   time.Time              `json:"ended_at"`
}

type UserToken struct {
	Email       string `json:"email"`
	Role        string `json:"role"`
	AccessToken string `json:"access_token"`
}

type PickedReferral struct {
	ReferralID    string `json:"referral_id"`
	PatientID     string `json:"patient_id"`
	ArrivalStatus string `json:"arrival_status_before"`
	ReferralState string `json:"referral_status_before"`
	HospitalID    string `json:"hospital_id"`
	DepartmentID  string `json:"department_id"`
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

type DBCheck struct {
	Name     string `json:"name"`
	Expected string `json:"expected"`
	Actual   string `json:"actual"`
	OK       bool   `json:"ok"`
}

// ───────────────────────── globals ─────────────────────────

var (
	db  *gorm.DB
	out = &Output{
		StartedAt: time.Now(),
		BaseURL:   baseURL,
		Users:     map[string]UserToken{},
	}
	outputPath = filepath.Join("scratch", "e2e_completion", "output.json")
)

// ───────────────────────── main ─────────────────────────

func main() {
	// Walk up to the repo root so we read .env.local and write
	// output.json in the same paths regardless of where the user
	// invoked us from (project root vs scratch subfolder).
	if _, err := os.Stat(".env.local"); err != nil {
		_ = os.Chdir("..")
		_ = os.Chdir("..")
	}
	if err := godotenv.Load(".env.local"); err != nil {
		log.Printf("[warn] .env.local not loaded: %v", err)
	}

	mustConnectDB()
	upsertCardioDoctor(taCardioDoctorID, "DOC-TA-CARDIO-001",
		taCardioDoctorEmail, "Bereket", "Cardio")
	upsertCardioDoctor(taCardioOtherID, "DOC-TA-CARDIO-002",
		taCardioOtherEmail, "Selamawit", "Cardio")

	loginAs("ta_receptionist", taReceptionEmail, "RECEPTIONIST")
	loginAs("ta_cardio_doctor", taCardioDoctorEmail, "REFERRING_DOCTOR")
	loginAs("ta_cardio_other", taCardioOtherEmail, "REFERRING_DOCTOR")

	picked := pickArrivedReferral()
	out.Picked = picked
	log.Printf("picked referral: %s (patient=%s, arrival=%s)",
		picked.ReferralID, picked.PatientID, picked.ArrivalStatus)

	// 1. Receptionist read endpoints - everything a FE page needs.
	receptionistReads(picked.ReferralID)

	// 2. Completion arc - doctor reads land BETWEEN assignment and
	//    outcome on purpose. RecordOutcome calls RevokeAllByReferral,
	//    so any /doctor/referrals/:id call AFTER outcome is supposed
	//    to return 403. We want to prove the treating-doctor fallback
	//    works while the grant is active, not after revocation.
	assignDoctor(picked.ReferralID, taCardioDoctorID)
	doctorReads(picked.ReferralID)

	// Negative gate: another doctor in the SAME department (cardio2)
	// tries to close the case. They are not assigned, so even though
	// the hospital + dept JWT scope matches and they could read the
	// referral via ReferralAccess (no, actually they can't - they
	// never received a grant), they MUST be rejected with 403. This
	// is the contract the FE relies on to decide whether to render
	// the "Record outcome" CTA.
	outcomeForbiddenByPeer(picked.ReferralID)

	clinicalUpdate(picked.ReferralID,
		"SPECIALIST_NOTE",
		"Patient stable post-consultation; cleared for discharge.")
	recordOutcome(picked.ReferralID, "discharged", 1, true,
		"Discharged from cardiology after observation; back to primary care.")
	// One more sanity hit AFTER outcome - we EXPECT a 403 here,
	// because the access grant has been revoked. This documents the
	// revocation behaviour for the FE team.
	doctorReadAfterOutcome(picked.ReferralID)

	// 4. DB assertions - belt-and-suspenders, in case the HTTP layer
	//    returned 200 but the persistence layer didn't follow through.
	verifyDB(picked.ReferralID)

	finalize()
}

// ───────────────────────── setup ─────────────────────────

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

// upsertCardioDoctor makes the harness re-runnable against
// pre-existing Neon dumps: the seeder adds the user on a fresh seed,
// but for already-seeded environments we don't want to re-seed
// everything just to add a doctor row. Idempotent via ON CONFLICT.
// We use it for both the assigned (cardio1) and the negative-test
// peer (cardio2) doctor.
func upsertCardioDoctor(id, nationalID, email, firstName, lastName string) {
	hash, _ := bcrypt.GenerateFromPassword([]byte(defaultPwd), 12)
	ctx := context.Background()
	if err := db.WithContext(ctx).Exec(`
		INSERT INTO users (id, national_id, email, first_name, middle_name, last_name,
		                   role, hospital_id, department_id, region, password_hash,
		                   is_active, is_deleted)
		VALUES (?, ?, ?, ?, '', ?, ?, ?, ?, ?, ?, true, false)
		ON CONFLICT (id) DO UPDATE SET
			email         = EXCLUDED.email,
			first_name    = EXCLUDED.first_name,
			last_name     = EXCLUDED.last_name,
			role          = EXCLUDED.role,
			hospital_id   = EXCLUDED.hospital_id,
			department_id = EXCLUDED.department_id,
			region        = EXCLUDED.region,
			password_hash = EXCLUDED.password_hash,
			is_active     = true,
			is_deleted    = false,
			deleted_at    = NULL,
			updated_at    = NOW()
	`, id, nationalID, email,
		firstName, lastName, entity.RoleReferringDoctor,
		hospTA, deptCard, entity.RegionAddisAbaba, string(hash)).Error; err != nil {
		log.Fatalf("upsert ta cardio doctor %s: %v", email, err)
	}
	log.Printf("ta cardio doctor upserted: %s", email)
}

// pickArrivedReferral peeks at the DB and returns an ARRIVED,
// unassigned triage row at TA Cardiology - the canonical candidate
// for "ready to be admitted". Fails hard if there are none, since
// without one the harness has nothing to test.
func pickArrivedReferral() PickedReferral {
	type row struct {
		ReferralID    string
		PatientID     string
		ArrivalStatus string
		ReferralState string
		HospitalID    string
		DepartmentID  string
	}
	var r row
	err := db.Raw(`
		SELECT tq.referral_id::text       AS referral_id,
		       r.patient_id::text         AS patient_id,
		       tq.arrival_status::text    AS arrival_status,
		       r.status::text             AS referral_state,
		       tq.hospital_id::text       AS hospital_id,
		       tq.department_id::text     AS department_id
		FROM   triage_queues tq
		JOIN   referrals     r ON r.id = tq.referral_id
		WHERE  tq.hospital_id   = ?::uuid
		  AND  tq.department_id = ?::uuid
		  AND  tq.arrival_status = 'ARRIVED'
		  AND  tq.assigned_doctor_id IS NULL
		  AND  r.status IN ('SCHEDULED', 'ACCEPTED')
		ORDER BY COALESCE(tq.arrived_at, tq.assigned_at) DESC
		LIMIT 1
	`, hospTA, deptCard).Scan(&r).Error
	if err != nil {
		fail("peek arrived referral: %v", err)
	}
	if r.ReferralID == "" {
		fail("no ARRIVED + unassigned referrals at TA Cardiology - run scratch/e2e_lifecycle first")
	}
	return PickedReferral{
		ReferralID:    r.ReferralID,
		PatientID:     r.PatientID,
		ArrivalStatus: r.ArrivalStatus,
		ReferralState: r.ReferralState,
		HospitalID:    r.HospitalID,
		DepartmentID:  r.DepartmentID,
	}
}

// ───────────────────────── login ─────────────────────────

func loginAs(key, email, role string) {
	body, _ := json.Marshal(map[string]string{"email": email, "password": defaultPwd})
	respBody, status, err := doRaw(http.MethodPost, "/auth/login", "", body, "application/json")
	if err != nil {
		fail("login %s: %v", email, err)
	}
	if status != 200 {
		fail("login %s status=%d body=%s", email, status, string(respBody))
	}
	var parsed struct {
		AccessToken string `json:"access_token"`
		MFAToken    string `json:"mfa_token"`
	}
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		fail("login %s parse: %v", email, err)
	}
	if parsed.AccessToken == "" {
		fail("login %s: no access_token (mfa=%q)", email, parsed.MFAToken)
	}
	out.Users[key] = UserToken{Email: email, Role: role, AccessToken: parsed.AccessToken}
	log.Printf("login ok: %s", email)
}

// ───────────────────────── receptionist read suite ─────────────────────────

// receptionistReads exercises every receptionist GET that a FE page
// (incl. the offline IndexedDB cache) calls. We don't assert the
// payload shape - the FE team reads the body fields from output.json
// - we just confirm 200 and capture the body for documentation.
func receptionistReads(refID string) {
	tk := "ta_receptionist"

	get(tk, "rec_list_referrals", "/receptionist/referrals?limit=20&page=1")
	get(tk, "rec_list_missed", "/receptionist/referrals/missed?limit=20&page=1")
	get(tk, "rec_list_upcoming", "/receptionist/referrals/upcoming")
	get(tk, "rec_offline_data", "/receptionist/referrals/offline-data")
	get(tk, "rec_list_triage_queue", "/receptionist/referrals/triage-queue?limit=20&page=1")
	get(tk, "rec_list_doctors", "/receptionist/doctors")

	get(tk, "rec_get_referral", "/receptionist/referrals/"+refID)
	get(tk, "rec_get_triage_detail", "/receptionist/referrals/"+refID+"/triage-detail")
}

// ───────────────────────── completion arc ─────────────────────────

func assignDoctor(refID, doctorID string) {
	postJSON("ta_receptionist", "receptionist_assign_doctor",
		"/receptionist/referrals/"+refID+"/assign-doctor",
		map[string]string{
			"doctor_id": doctorID,
			"reason":    "Routine post-arrival assignment by e2e_completion harness",
		})
}

func clinicalUpdate(refID, reason, notes string) {
	postJSON("ta_cardio_doctor", "doctor_clinical_update",
		"/referrals/"+refID+"/clinical/updates",
		map[string]string{
			"update_reason":  reason,
			"clinical_notes": notes,
		})
}

func recordOutcome(refID, outcome string, stayDays int, appropriate bool, notes string) {
	postJSON("ta_cardio_doctor", "doctor_record_outcome",
		"/referrals/"+refID+"/clinical/outcome",
		map[string]interface{}{
			"outcome":                  outcome,
			"length_of_stay_days":      stayDays,
			"was_referral_appropriate": appropriate,
			"outcome_notes":            notes,
		})
}

// outcomeForbiddenByPeer asserts the strict "assigned doctor only"
// gate. A second TA Cardiology doctor (same hospital, same dept,
// same role) attempts to close the case while cardio1 is the
// assigned treating doctor. The backend must respond 403 with the
// "only the assigned treating doctor" message.
func outcomeForbiddenByPeer(refID string) {
	body, _ := json.Marshal(map[string]interface{}{
		"outcome":                  "discharged",
		"length_of_stay_days":      1,
		"was_referral_appropriate": true,
		"outcome_notes":            "Negative test: peer doctor tries to close case.",
	})
	recordExpect("doctor_record_outcome_peer_expects_403",
		http.MethodPost,
		"/referrals/"+refID+"/clinical/outcome",
		"ta_cardio_other", body, "application/json", 403)
}

// ───────────────────────── doctor-side reads ─────────────────────────

// doctorReads verifies that the treating doctor (granted access via
// the assign-doctor step) can hit the same detail endpoint the sender
// already uses (the fallback we added in GetDetailsForDoctor) and
// that both "treating" and "TREATING_DOCTOR" work as the access_type
// filter on the assigned list.
func doctorReads(refID string) {
	tk := "ta_cardio_doctor"

	get(tk, "doc_assigned_treating_alias",
		"/doctor/referrals/assigned?access_type=treating&include_revoked=false")
	get(tk, "doc_assigned_treating_canonical",
		"/doctor/referrals/assigned?access_type=TREATING_DOCTOR&include_revoked=false")
	get(tk, "doc_get_single_as_grantee",
		"/doctor/referrals/"+refID)
}

// doctorReadAfterOutcome is intentionally allowed to return 403: the
// outcome auto-revokes all ReferralAccess grants, so the treating
// doctor loses read access once the case is closed. This is the
// audit-only state described in the FE doc.
func doctorReadAfterOutcome(refID string) {
	getExpect("ta_cardio_doctor", "doc_get_single_post_outcome_expects_403",
		"/doctor/referrals/"+refID, 403)
}

// ───────────────────────── verification ─────────────────────────

func verifyDB(refID string) {
	var refStatus string
	var arrivalStatus string
	if err := db.Raw(`SELECT status::text FROM referrals WHERE id = ?::uuid`, refID).
		Scan(&refStatus).Error; err != nil {
		fail("verify referral status: %v", err)
	}
	if err := db.Raw(`SELECT arrival_status::text FROM triage_queues WHERE referral_id = ?::uuid`, refID).
		Scan(&arrivalStatus).Error; err != nil {
		fail("verify arrival status: %v", err)
	}
	out.DBChecks = append(out.DBChecks,
		DBCheck{Name: "referral.status", Expected: "COMPLETED", Actual: refStatus, OK: refStatus == "COMPLETED"},
		DBCheck{Name: "triage_queue.arrival_status", Expected: "ADMITTED", Actual: arrivalStatus, OK: arrivalStatus == "ADMITTED"},
	)

	var outcomeCount int64
	if err := db.Raw(`SELECT COUNT(*) FROM referral_outcomes WHERE referral_id = ?::uuid`, refID).
		Scan(&outcomeCount).Error; err != nil {
		fail("verify outcome count: %v", err)
	}
	out.DBChecks = append(out.DBChecks, DBCheck{
		Name: "referral_outcomes row count", Expected: ">=1",
		Actual: fmt.Sprintf("%d", outcomeCount), OK: outcomeCount >= 1,
	})

	for _, c := range out.DBChecks {
		log.Printf("db_check %-40s expected=%s actual=%s ok=%v", c.Name, c.Expected, c.Actual, c.OK)
		if !c.OK {
			out.Errors = append(out.Errors,
				fmt.Sprintf("db_check failed: %s expected=%s actual=%s", c.Name, c.Expected, c.Actual))
		}
	}
}

// ───────────────────────── HTTP helpers ─────────────────────────

func get(tokenKey, name, path string) {
	record(name, http.MethodGet, path, tokenKey, nil, "")
}

// getExpect lets callers declare an expected non-2xx status (e.g. 403
// after the access grant has been revoked) without aborting the run.
func getExpect(tokenKey, name, path string, wantStatus int) {
	recordExpect(name, http.MethodGet, path, tokenKey, nil, "", wantStatus)
}

func postJSON(tokenKey, name, path string, payload interface{}) {
	body, _ := json.Marshal(payload)
	record(name, http.MethodPost, path, tokenKey, body, "application/json")
}

func record(name, method, path, tokenKey string, body []byte, ct string) []byte {
	return recordExpect(name, method, path, tokenKey, body, ct, 0)
}

// recordExpect runs an HTTP call and treats it as success if the
// status code is 2xx OR matches wantStatus (when wantStatus > 0).
// This lets us assert specific failure modes (e.g. 403 after access
// revocation) without aborting the harness.
func recordExpect(name, method, path, tokenKey string, body []byte, ct string, wantStatus int) []byte {
	start := time.Now()
	respBody, status, err := doRaw(method, path, out.Users[tokenKey].AccessToken, body, ct)
	sr := StepResult{
		N: len(out.Steps) + 1, Name: name, Method: method, Path: path,
		Status: status, Duration: time.Since(start).String(),
	}
	if err != nil {
		sr.Error = err.Error()
		out.Steps = append(out.Steps, sr)
		fail("%s: %v", name, err)
	}
	if wantStatus > 0 {
		sr.OK = status == wantStatus
	} else {
		sr.OK = status >= 200 && status < 300
	}
	if len(respBody) > 0 && len(respBody) < 8000 {
		sr.Body = json.RawMessage(respBody)
	} else if len(respBody) > 0 {
		sr.Body = json.RawMessage(respBody[:8000])
	}
	if !sr.OK {
		sr.Error = fmt.Sprintf("unexpected status %d (wanted %d)", status, wantStatus)
	}
	out.Steps = append(out.Steps, sr)
	log.Printf("step %-40s %s %-60s -> %d", name, method, path, status)
	if !sr.OK {
		fail("%s -> %d: %s", name, status, string(respBody))
	}
	return respBody
}

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

// ───────────────────────── finalize ─────────────────────────

func fail(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	out.Errors = append(out.Errors, msg)
	finalize()
	log.Fatal(msg)
}

func finalize() {
	out.EndedAt = time.Now()
	b, _ := json.MarshalIndent(out, "", "  ")
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		log.Printf("[warn] mkdir output: %v", err)
	}
	if err := os.WriteFile(outputPath, b, 0644); err != nil {
		log.Printf("[warn] write output: %v", err)
		return
	}
	log.Printf("wrote %s", outputPath)
}
