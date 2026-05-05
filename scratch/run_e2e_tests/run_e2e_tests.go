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
	baseURL = "https://hospital-referral-backend-146237699183.europe-west3.run.app"
	report  = []string{}
)

func logStep(step, method, path string, expected, actual int, errStr string) {
	pass := expected == actual
	passStr := "FAIL"
	if pass {
		passStr = "PASS"
	}
	result := fmt.Sprintf("| %-35s | %-6s | %-55s | %d | %d | %s |", step, method, path, expected, actual, passStr)
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
	fmt.Println("| Step                               | Method | Path                                                    | Exp  | Act  | Pass |")
	fmt.Println("|------------------------------------|--------|---------------------------------------------------------|------|------|------|")

	// Step 0: Health Check
	code, body := doReq("GET", "/health", "", nil)
	logStep("0. Health Check", "GET", "/health", 200, code, string(body))

	// Step 1: System Admin Config
	adminToken := login("superadmin@moh.gov.et")
	code, body = doReq("GET", "/api/v1/admin/config", adminToken, nil)
	logStep("1. Get Config", "GET", "/api/v1/admin/config", 200, code, string(body))
	code, body = doReq("PUT", "/api/v1/admin/config", adminToken, map[string]interface{}{"buffer_days": "3", "aging_factor": "1.2"})
	logStep("1. Put Config", "PUT", "/api/v1/admin/config", 200, code, string(body))

	// Step 2: Doctor Create Referral (TA→TA Cardiology, same hospital for simplicity)
	drTAToken := login("doctor.ta@hospital.et")
	code, body = doReq("GET", "/api/v1/doctor/stats", drTAToken, nil)
	logStep("2. TA Doctor Stats", "GET", "/api/v1/doctor/stats", 200, code, string(body))

	refTA1Body := map[string]interface{}{
		"patient_id":                   "e0000000-0000-0000-0000-000000000001",
		"target_hospital_id":           "a3000000-0000-0000-0000-000000000003", // Black Lion
		"target_dept_id":               "b5000000-0000-0000-0000-000000000005", // Pediatrics
		"sender_hospital_id":           "a1000000-0000-0000-0000-000000000001", // TA
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
		submitBody := make(map[string]interface{})
		for k, v := range refTA1Body {
			if k != "status" {
				submitBody[k] = v
			}
		}
		code, body = doReq("PUT", "/api/v1/doctor/referrals/"+refTA1+"/submit", drTAToken, submitBody)
		logStep("2. TA Submit Ref 1", "PUT", "/api/v1/doctor/referrals/.../submit", 200, code, string(body))
	}

	// Step 3: Liaison Forward
	liaTAToken := login("liaison.ta@hospital.et")
	if refTA1 != "" {
		code, body = doReq("POST", "/api/v1/liaison/referrals/"+refTA1+"/read", liaTAToken, nil)
		logStep("3. TA Liaison Read 1", "POST", "/api/v1/liaison/referrals/.../read", 200, code, string(body))
		code, body = doReq("POST", "/api/v1/liaison/referrals/"+refTA1+"/forward", liaTAToken, nil)
		logStep("3. TA Liaison Forward 1", "POST", "/api/v1/liaison/referrals/.../forward", 200, code, string(body))
	}

	// Step 4: Specialist at Black Lion reads and accepts
	specBLToken := login("specialist.bl@hospital.et")
	if refTA1 != "" {
		code, body = doReq("POST", "/api/v1/specialist/referrals/"+refTA1+"/read", specBLToken, nil)
		logStep("4. BL Spec Read 1", "POST", "/api/v1/specialist/referrals/.../read", 200, code, string(body))
		code, body = doReq("POST", "/api/v1/specialist/referrals/"+refTA1+"/triage-severity", specBLToken, map[string]interface{}{"score": 85, "justification": "High risk"})
		logStep("4. BL Spec Triage 1", "POST", "/api/v1/specialist/referrals/.../triage-severity", 200, code, string(body))
		code, body = doReq("POST", "/api/v1/specialist/referrals/"+refTA1+"/accept", specBLToken, nil)
		logStep("4. BL Spec Accept 1", "POST", "/api/v1/specialist/referrals/.../accept", 200, code, string(body))
	}

	// Step 5: Redirection workflow (BL → TA, dept-locked on Pediatrics)
	if refTA1 != "" {
		// 5a. List redirect options (BL has Pediatrics, so TA which also has... check)
		code, body = doReq("GET", "/api/v1/specialist/referrals/"+refTA1+"/redirect-options", specBLToken, nil)
		logStep("5a. BL Redirect Options", "GET", "/api/v1/specialist/referrals/.../redirect-options", 200, code, string(body))
		fmt.Printf("   Redirect options response: %s\n", string(body))

		// 5b. Change department at BL (BL has Cardiology b1)
		code, body = doReq("PUT", "/api/v1/specialist/referrals/"+refTA1+"/department", specBLToken, map[string]interface{}{
			"department_id": "b1000000-0000-0000-0000-000000000001", // Cardiology - BL has this
		})
		logStep("5b. BL Change Dept→Cardio", "PUT", "/api/v1/specialist/referrals/.../department", 200, code, string(body))

		// 5c. List redirect options again (should now include hospitals with Cardiology)
		code, body = doReq("GET", "/api/v1/specialist/referrals/"+refTA1+"/redirect-options", specBLToken, nil)
		logStep("5c. BL Redirect Options (Cardio)", "GET", "/api/v1/specialist/referrals/.../redirect-options", 200, code, string(body))
		fmt.Printf("   Redirect options after dept change: %s\n", string(body))

		// 5d. Attempt to redirect back to TA (TA sent it, so it should be forbidden/loop)
		loopReq := map[string]interface{}{
			"target_hospital_id": "a1000000-0000-0000-0000-000000000001",
			"reason":             "Redirect back to sender test",
		}
		code, body = doReq("POST", "/api/v1/specialist/referrals/"+refTA1+"/redirect", specBLToken, loopReq)
		if code == 400 || code == 403 || code == 422 {
			logStep("5d. Loop prevention (TA blocked)", "POST", "/api/v1/specialist/referrals/.../redirect", code, code, "")
		} else {
			logStep("5d. Loop prevention FAILED", "POST", "/api/v1/specialist/referrals/.../redirect", 400, code, string(body))
		}

		// 5e. Get redirection history (as admin)
		code, body = doReq("GET", "/api/v1/referrals/"+refTA1+"/redirections", adminToken, nil)
		logStep("5e. Redirection History", "GET", "/api/v1/referrals/.../redirections", 200, code, string(body))
	}

	// Step 6: Network route validation
	// 6a. Self-loop creation should be rejected
	adminHospToken := login("admin.ta@hospital.et")
	code, body = doReq("POST", "/api/v1/admin/network-routes", adminHospToken, map[string]interface{}{
		"sender_hospital_id":   "a1000000-0000-0000-0000-000000000001",
		"receiver_hospital_id": "a1000000-0000-0000-0000-000000000001",
		"referral_type":        "routine",
	})
	if code == 400 || code == 422 {
		logStep("6a. Self-loop rejected", "POST", "/api/v1/admin/network-routes", code, code, "")
	} else {
		logStep("6a. Self-loop NOT rejected", "POST", "/api/v1/admin/network-routes", 400, code, string(body))
	}

	// Step 7: Dept Head & Receptionist (abbreviated)
	headTAToken := login("depthead.ta@hospital.et")
	code, body = doReq("POST", "/api/v1/department-head/schedule/batch", headTAToken, nil)
	logStep("7. TA Head Batch Schedule", "POST", "/api/v1/department-head/schedule/batch", 200, code, string(body))

	recTAToken := login("reception.ta@hospital.et")
	code, body = doReq("GET", "/api/v1/receptionist/referrals/schedule", recTAToken, nil)
	logStep("7. TA Rec Schedule", "GET", "/api/v1/receptionist/referrals/schedule", 200, code, string(body))

	// Step 8: Notifications
	code, body = doReq("POST", "/api/v1/internal/notifications/send", adminToken, nil)
	logStep("8. Notif Send", "POST", "/api/v1/internal/notifications/send", 200, code, string(body))

	// Step 9: Scheduler Cycle
	code, body = doReq("POST", "/api/v1/internal/jobs/run-scheduler-cycle", adminToken, map[string]string{"lease_holder": "e2e-tester-1"})
	logStep("9. Scheduler Cycle", "POST", "/api/v1/internal/jobs/run-scheduler-cycle", 200, code, string(body))

	// Write report
	f, _ := os.OpenFile("e2e_report.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	defer f.Close()
	f.WriteString("\n--- E2E Run at " + time.Now().Format(time.RFC3339) + " ---\n")
	for _, line := range report {
		f.WriteString(line + "\n")
	}
	fmt.Println("\nE2E test suite completed. Report appended to e2e_report.txt")
}
func init() {}
