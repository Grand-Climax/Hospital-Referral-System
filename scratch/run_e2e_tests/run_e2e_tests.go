package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

var (
	baseURL = "http://localhost:8081"
	report  = []string{}
)

func logStep(step, method, path string, expected, actual int, errStr string) {
	pass := expected == actual
	passStr := "FAIL"
	if pass {
		passStr = "PASS"
	}

	result := fmt.Sprintf("| %-30s | %-6s | %-60s | %d | %d | %s |", step, method, path, expected, actual, passStr)
	fmt.Println(result)
	report = append(report, result)
	if !pass && errStr != "" {
		fmt.Printf("   Error details: %s\n", errStr)
		report = append(report, fmt.Sprintf("   Error details: %s", errStr))
	}
}

func doReq(method, path, token string, body interface{}) (int, []byte) {
	var bodyReader io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		bodyReader = bytes.NewReader(b)
	}

	req, _ := http.NewRequest(method, baseURL+path, bodyReader)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return 0, []byte(err.Error())
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, respBody
}

func login(email string) string {
	code, body := doReq("POST", "/api/v1/auth/login", "", map[string]string{
		"email":    email,
		"password": "password123",
	})
	if code != 200 {
		log.Fatalf("Login failed for %s: %s", email, string(body))
	}
	var res struct {
		AccessToken string `json:"access_token"`
	}
	json.Unmarshal(body, &res)
	if res.AccessToken == "" {
		log.Fatalf("Token empty for %s: %s", email, string(body))
	}
	return res.AccessToken
}

func extractRefID(body []byte) string {
	var res struct {
		Referral struct {
			ID string `json:"id"`
		} `json:"referral"`
	}
	json.Unmarshal(body, &res)
	return res.Referral.ID
}

func extractTQID(body []byte) string {
	var res struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	json.Unmarshal(body, &res)
	if len(res.Data) > 0 {
		return res.Data[0].ID
	}
	return ""
}

func main() {
	fmt.Println("Starting E2E Tests...")
	fmt.Println("| Step                           | Method | Path                                                         | Exp  | Act  | Pass |")
	fmt.Println("|--------------------------------|--------|--------------------------------------------------------------|------|------|------|")

	// Step 0: Health Check
	code, body := doReq("GET", "/health", "", nil)
	logStep("0. Health Check", "GET", "/health", 200, code, string(body))

	// Step 1: System Admin Config
	adminToken := login("superadmin@moh.gov.et")
	code, body = doReq("GET", "/api/v1/admin/config", adminToken, nil)
	logStep("1. Get Config", "GET", "/api/v1/admin/config", 200, code, string(body))

	code, body = doReq("PUT", "/api/v1/admin/config", adminToken, map[string]interface{}{"buffer_days": "3", "aging_factor": "1.2"})
	logStep("1. Put Config", "PUT", "/api/v1/admin/config", 200, code, string(body))

	// Step 2: Doctor Create Referral
	drTAToken := login("doctor.ta@hospital.et")
	code, body = doReq("GET", "/api/v1/doctor/stats", drTAToken, nil)
	logStep("2. TA Doctor Stats", "GET", "/api/v1/doctor/stats", 200, code, string(body))

	// Create REF_TA_1 (Abebe to TA Cardiology)
	refTA1Body := map[string]interface{}{
		"patient_id":                   "e0000000-0000-0000-0000-000000000001",
		"target_hospital_id":           "a1000000-0000-0000-0000-000000000001",
		"target_dept_id":               "b1000000-0000-0000-0000-000000000001",
		"sender_hospital_id":           "a1000000-0000-0000-0000-000000000001",
		"liaison_officer_id":           "d1000000-0000-0000-0000-000000000002",
		"clinical_summary":             "Severe chest pain",
		"patient_history":              "Hypertension since 2015",
		"reason_of_referral":           "Suspected ACS",
		"reason_for_referral_category": "ROUTINE",
		"condition_at_referral":        "unstable",
		"diagnoses":                    []map[string]interface{}{{"icd_code": "I21.9", "is_primary": true, "diagnosis_certainty": "SUSPECTED"}},
		"status":                       "DRAFT",
	}
	code, body = doReq("POST", "/api/v1/doctor/referrals", drTAToken, refTA1Body)
	logStep("2. TA Create Ref 1", "POST", "/api/v1/doctor/referrals", 201, code, string(body))
	refTA1 := extractRefID(body)

	if refTA1 != "" {
		// MUST NOT provide "status" but MUST provide "diagnoses"
		submitBody := make(map[string]interface{})
		for k, v := range refTA1Body {
			if k != "status" {
				submitBody[k] = v
			}
		}
		code, body = doReq("PUT", "/api/v1/doctor/referrals/"+refTA1+"/submit", drTAToken, submitBody)
		logStep("2. TA Submit Ref 1", "PUT", "/api/v1/doctor/referrals/.../submit", 200, code, string(body))
	}

	// Create REF_TA_2 (Meseret to BL Pediatrics, critical)
	refTA2Body := map[string]interface{}{
		"patient_id":                   "e0000000-0000-0000-0000-000000000002",
		"target_hospital_id":           "a3000000-0000-0000-0000-000000000003",
		"target_dept_id":               "b5000000-0000-0000-0000-000000000005",
		"sender_hospital_id":           "a1000000-0000-0000-0000-000000000001",
		"liaison_officer_id":           "d1000000-0000-0000-0000-000000000002",
		"clinical_summary":             "Unconscious",
		"patient_history":              "Trauma victim",
		"reason_of_referral":           "Higher treatment",
		"reason_for_referral_category": "EMERGENCY",
		"condition_at_referral":        "critical",
		"diagnoses":                    []map[string]interface{}{{"icd_code": "S06.9X9A", "is_primary": true, "diagnosis_certainty": "CONFIRMED"}},
		"status":                       "DRAFT",
	}
	code, body = doReq("POST", "/api/v1/doctor/referrals", drTAToken, refTA2Body)
	logStep("2. TA Create Ref 2", "POST", "/api/v1/doctor/referrals", 201, code, string(body))
	refTA2 := extractRefID(body)

	if refTA2 != "" {
		submitBody := make(map[string]interface{})
		for k, v := range refTA2Body {
			if k != "status" {
				submitBody[k] = v
			}
		}
		code, body = doReq("PUT", "/api/v1/doctor/referrals/"+refTA2+"/submit", drTAToken, submitBody)
		logStep("2. TA Submit Ref 2", "PUT", "/api/v1/doctor/referrals/.../submit", 200, code, string(body))
	}

	// Step 3: Liaison (TA handles both outgoing)
	liaTAToken := login("liaison.ta@hospital.et")
	if refTA1 != "" {
		code, body = doReq("POST", "/api/v1/liaison/referrals/"+refTA1+"/read", liaTAToken, nil)
		logStep("3. TA Liaison Read 1", "POST", "/api/v1/liaison/referrals/.../read", 200, code, string(body))
		code, body = doReq("POST", "/api/v1/liaison/referrals/"+refTA1+"/forward", liaTAToken, nil)
		logStep("3. TA Liaison Forward 1", "POST", "/api/v1/liaison/referrals/.../forward", 200, code, string(body))
	}
	if refTA2 != "" {
		code, body = doReq("POST", "/api/v1/liaison/referrals/"+refTA2+"/read", liaTAToken, nil)
		logStep("3. TA Liaison Read 2", "POST", "/api/v1/liaison/referrals/.../read", 200, code, string(body))
		code, body = doReq("POST", "/api/v1/liaison/referrals/"+refTA2+"/forward", liaTAToken, nil)
		logStep("3. TA Liaison Forward 2", "POST", "/api/v1/liaison/referrals/.../forward", 200, code, string(body))
	}

	// Step 4: Specialist
	specTAToken := login("specialist.ta@hospital.et")
	if refTA1 != "" {
		code, body = doReq("POST", "/api/v1/specialist/referrals/"+refTA1+"/read", specTAToken, nil)
		logStep("4. TA Spec Read 1", "POST", "/api/v1/specialist/referrals/.../read", 200, code, string(body))
		code, body = doReq("POST", "/api/v1/specialist/referrals/"+refTA1+"/triage-severity", specTAToken, map[string]interface{}{"score": 85, "justification": "High risk"})
		logStep("4. TA Spec Triage 1", "POST", "/api/v1/specialist/referrals/.../triage", 200, code, string(body))
		code, body = doReq("POST", "/api/v1/specialist/referrals/"+refTA1+"/accept", specTAToken, nil)
		logStep("4. TA Spec Accept 1", "POST", "/api/v1/specialist/referrals/.../accept", 200, code, string(body))
	}

	specBLToken := login("specialist.bl@hospital.et")
	if refTA2 != "" {
		code, body = doReq("POST", "/api/v1/specialist/referrals/"+refTA2+"/read", specBLToken, nil)
		logStep("4. BL Spec Read 2", "POST", "/api/v1/specialist/referrals/.../read", 200, code, string(body))
		code, body = doReq("POST", "/api/v1/specialist/referrals/"+refTA2+"/triage-severity", specBLToken, map[string]interface{}{"score": 95, "justification": "Critical"})
		logStep("4. BL Spec Triage 2", "POST", "/api/v1/specialist/referrals/.../triage", 200, code, string(body))
		code, body = doReq("POST", "/api/v1/specialist/referrals/"+refTA2+"/accept", specBLToken, nil)
		logStep("4. BL Spec Accept 2", "POST", "/api/v1/specialist/referrals/.../accept", 200, code, string(body))
	}

	// Step 5: Dept Head
	headTAToken := login("depthead.ta@hospital.et")
	code, body = doReq("POST", "/api/v1/department-head/schedule/batch", headTAToken, nil)
	logStep("5. TA Head Batch", "POST", "/api/v1/department-head/schedule/batch", 200, code, string(body))

	// Step 6: Receptionist
	recTAToken := login("reception.ta@hospital.et")
	code, body = doReq("GET", "/api/v1/receptionist/referrals/schedule", recTAToken, nil)
	logStep("6. TA Rec Sched", "GET", "/api/v1/receptionist/referrals/schedule", 200, code, string(body))

	// Get TQ ID for TA
	code, body = doReq("GET", "/api/v1/specialist/referrals/triage-queue", specTAToken, nil)
	tqTA1 := extractTQID(body)
	if tqTA1 != "" {
		code, body = doReq("POST", "/api/v1/receptionist/referrals/"+tqTA1+"/arrive", recTAToken, nil)
		logStep("6. TA Rec Arrive", "POST", "/api/v1/receptionist/referrals/.../arrive", 200, code, string(body))
	}

	// Step 7: Clinical
	if refTA1 != "" {
		code, body = doReq("POST", "/api/v1/referrals/"+refTA1+"/clinical/outcome", specTAToken, map[string]interface{}{"outcome": "improved", "outcome_notes": "Discharged", "was_referral_appropriate": true, "length_of_stay_days": 2})
		logStep("7. TA Spec Outcome", "POST", "/api/v1/referrals/.../clinical/outcome", 200, code, string(body))
	}

	// Step 8: Notifications
	code, body = doReq("POST", "/api/v1/internal/notifications/send", adminToken, nil)
	logStep("8. Admin Notif Send", "POST", "/api/v1/internal/notifications/send", 200, code, string(body))

	// Step 9: Sharded Scheduler Cycle
	code, body = doReq("POST", "/api/v1/internal/jobs/run-scheduler-cycle", adminToken, map[string]string{"lease_holder": "e2e-tester-1"})
	logStep("9. Run Scheduler Cycle", "POST", "/api/v1/internal/jobs/run-scheduler-cycle", 200, code, string(body))

	// Write report to file
	f, _ := os.OpenFile("e2e_report.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	defer f.Close()
	f.WriteString("\n--- E2E Run at " + time.Now().Format(time.RFC3339) + " ---\n")
	for _, line := range report {
		f.WriteString(line + "\n")
	}

	fmt.Println("E2E test suite completed. Report appended to e2e_report.txt")
}
func init() {}
