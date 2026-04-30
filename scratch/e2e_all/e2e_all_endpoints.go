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
	baseURL = "http://localhost:8080"
	report  = []string{}
)

func init() {
	if url := os.Getenv("BASE_URL"); url != "" {
		baseURL = url
	}
}

type StepResult struct {
	Step      string
	Method    string
	Path      string
	ExpStatus int
	ActStatus int
	Pass      bool
	Details   string
}

func logStep(res StepResult) {
	passStr := "FAIL"
	if res.Pass {
		passStr = "PASS"
	}

	resultLine := fmt.Sprintf("| %-45s | %-6s | %-70s | %d | %d | %s |", res.Step, res.Method, res.Path, res.ExpStatus, res.ActStatus, passStr)
	fmt.Println(resultLine)
	report = append(report, resultLine)

	if !res.Pass && res.Details != "" {
		det := res.Details
		if len(det) > 1000 {
			det = det[:1000] + "... [truncated]"
		}
		fmt.Printf("   Error/Response: %s\n", det)
		report = append(report, "   Error: "+det)
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
		Data struct {
			AccessToken string `json:"access_token"`
		} `json:"data"`
		AccessToken string `json:"access_token"`
	}
	json.Unmarshal(body, &res)
	if res.Data.AccessToken != "" {
		return res.Data.AccessToken
	}
	return res.AccessToken
}

func getUserIDByEmail(token, email string) string {
	code, body := doReq("GET", "/api/v1/system-admin/users", token, nil)
	if code != 200 { return "" }
	var res struct { Data []map[string]interface{} }
	json.Unmarshal(body, &res)
	for _, u := range res.Data {
		if u["email"] == email {
			return u["id"].(string)
		}
	}
	return ""
}

func extractID(body []byte, keyPath ...string) string {
	var data interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		return ""
	}

	curr := data
	for _, k := range keyPath {
		m, ok := curr.(map[string]interface{})
		if !ok {
			return ""
		}
		curr, ok = m[k]
		if !ok {
			return ""
		}
	}

	if s, ok := curr.(string); ok {
		return s
	}
	return ""
}

func main() {
	fmt.Println("Starting Comprehensive End-to-End Test for All Endpoints...")
	fmt.Println("| Step                                          | Method | Path                                                                   | Exp  | Act  | Pass |")
	fmt.Println("|-----------------------------------------------|--------|------------------------------------------------------------------------|------|------|------|")

	// ---------------------------------------------------------
	// 0. HEALTH & PUBLIC
	// ---------------------------------------------------------
	code, body := doReq("GET", "/health", "", nil)
	logStep(StepResult{Step: "Health Check", Method: "GET", Path: "/health", ExpStatus: 200, ActStatus: code, Pass: code == 200, Details: string(body)})

	code, body = doReq("GET", "/", "", nil)
	logStep(StepResult{Step: "Home Page", Method: "GET", Path: "/", ExpStatus: 200, ActStatus: code, Pass: code == 200})

	code, body = doReq("GET", "/swagger/index.html", "", nil)
	logStep(StepResult{Step: "Swagger Docs", Method: "GET", Path: "/swagger/index.html", ExpStatus: 200, ActStatus: code, Pass: code == 200})

	// ---------------------------------------------------------
	// 1. AUTHENTICATION
	// ---------------------------------------------------------
	var authRes struct {
		Data struct {
			RefreshToken string `json:"refresh_token"`
		} `json:"data"`
		RefreshToken string `json:"refresh_token"`
	}

	code, body = doReq("POST", "/api/v1/auth/login", "", map[string]string{"email": "superadmin@moh.gov.et", "password": "password123"})
	json.Unmarshal(body, &authRes)
	rt := authRes.Data.RefreshToken
	if rt == "" { rt = authRes.RefreshToken }

	code, body = doReq("POST", "/api/v1/auth/refresh", "", map[string]string{"refresh_token": rt})
	logStep(StepResult{Step: "Auth Refresh", Method: "POST", Path: "/api/v1/auth/refresh", ExpStatus: 200, ActStatus: code, Pass: code == 200, Details: string(body)})

	adminToken := login("superadmin@moh.gov.et")
	code, body = doReq("POST", "/api/v1/auth/logout", adminToken, map[string]string{"refresh_token": rt})
	logStep(StepResult{Step: "Auth Logout", Method: "POST", Path: "/api/v1/auth/logout", ExpStatus: 200, ActStatus: code, Pass: code == 200, Details: string(body)})

	// Re-login for subsequent tests
	adminToken = login("superadmin@moh.gov.et")
	drTAToken := login("doctor.ta@hospital.et")
	liaTAToken := login("liaison.ta@hospital.et")
	specTAToken := login("specialist.ta@hospital.et")
	recTAToken := login("reception.ta@hospital.et")
	headTAToken := login("depthead.ta@hospital.et")
	spAdminToken := login("admin.specialized@hospital.et")

	specTAUserID := getUserIDByEmail(adminToken, "specialist.ta@hospital.et")
	fmt.Printf("Extracted Specialist TA UserID: %s\n", specTAUserID)

	// ---------------------------------------------------------
	// 2. ADMIN CONFIG
	// ---------------------------------------------------------
	code, body = doReq("GET", "/api/v1/admin/config", adminToken, nil)
	logStep(StepResult{Step: "Get Admin Config", Method: "GET", Path: "/api/v1/admin/config", ExpStatus: 200, ActStatus: code, Pass: code == 200, Details: string(body)})

	code, body = doReq("PUT", "/api/v1/admin/config", adminToken, map[string]string{"buffer_days": "3"})
	logStep(StepResult{Step: "Update Admin Config", Method: "PUT", Path: "/api/v1/admin/config", ExpStatus: 200, ActStatus: code, Pass: code == 200, Details: string(body)})

	// Negative: Non-admin update config
	code, body = doReq("PUT", "/api/v1/admin/config", drTAToken, map[string]string{"buffer_days": "5"})
	logStep(StepResult{Step: "Update Config (NEG - Role)", Method: "PUT", Path: "/api/v1/admin/config", ExpStatus: 403, ActStatus: code, Pass: code == 403, Details: string(body)})

	// ---------------------------------------------------------
	// 3. DOCTOR & REFERRAL LIFECYCLE
	// ---------------------------------------------------------
	code, body = doReq("GET", "/api/v1/doctor/stats", drTAToken, nil)
	logStep(StepResult{Step: "Doctor Stats", Method: "GET", Path: "/api/v1/doctor/stats", ExpStatus: 200, ActStatus: code, Pass: code == 200})

	code, body = doReq("GET", "/api/v1/doctor/latest-pending", drTAToken, nil)
	logStep(StepResult{Step: "Doctor Latest Pending", Method: "GET", Path: "/api/v1/doctor/latest-pending", ExpStatus: 200, ActStatus: code, Pass: code == 200})

	// Create Patient
	patientReq := map[string]interface{}{
		"first_name":   "Abebe",
		"last_name":    "Kebede",
		"phone_number": "+251911000001",
		"sex":          "male",
		"national_id":  "NAT-E2E-001",
	}
	code, body = doReq("POST", "/api/v1/patients", drTAToken, patientReq)
	logStep(StepResult{Step: "Create Patient", Method: "POST", Path: "/api/v1/patients", ExpStatus: 201, ActStatus: code, Pass: code == 201, Details: string(body)})

	// Create Referral Draft
	refReq := map[string]interface{}{
		"patient_id":                   "e0000000-0000-0000-0000-000000000001",
		"target_hospital_id":           "a1000000-0000-0000-0000-000000000001",
		"target_dept_id":               "b1000000-0000-0000-0000-000000000001",
		"sender_hospital_id":           "a1000000-0000-0000-0000-000000000001",
		"liaison_officer_id":           "d1000000-0000-0000-0000-000000000002",
		"clinical_summary":             "Test summary",
		"reason_for_referral_category": "ROUTINE",
		"condition_at_referral":        "stable",
		"patient_history":              "None",
		"reason_of_referral":           "Specialist review",
		"diagnoses":                    []map[string]interface{}{{"icd_code": "I21.9", "is_primary": true, "diagnosis_certainty": "CONFIRMED"}},
		"status":                       "DRAFT",
	}
	code, body = doReq("POST", "/api/v1/doctor/referrals", drTAToken, refReq)
	refID := extractID(body, "data", "id")
	if refID == "" { refID = extractID(body, "referral", "id") }
	logStep(StepResult{Step: "Create Referral Draft", Method: "POST", Path: "/api/v1/doctor/referrals", ExpStatus: 201, ActStatus: code, Pass: code == 201 && refID != "", Details: string(body)})

	// Update Draft
	delete(refReq, "status")
	refReq["clinical_summary"] = "Updated summary"
	code, body = doReq("PUT", "/api/v1/doctor/referrals/"+refID, drTAToken, refReq)
	logStep(StepResult{Step: "Update Referral Draft", Method: "PUT", Path: "/api/v1/doctor/referrals/:id", ExpStatus: 200, ActStatus: code, Pass: code == 200, Details: string(body)})

	// Submit Referral
	code, body = doReq("PUT", "/api/v1/doctor/referrals/"+refID+"/submit", drTAToken, refReq)
	logStep(StepResult{Step: "Submit Referral", Method: "PUT", Path: "/api/v1/doctor/referrals/:id/submit", ExpStatus: 200, ActStatus: code, Pass: code == 200, Details: string(body)})

	code, body = doReq("GET", "/api/v1/doctor/referrals", drTAToken, nil)
	logStep(StepResult{Step: "Doctor List Referrals", Method: "GET", Path: "/api/v1/doctor/referrals", ExpStatus: 200, ActStatus: code, Pass: code == 200})

	code, body = doReq("GET", "/api/v1/doctor/referrals/"+refID, drTAToken, nil)
	logStep(StepResult{Step: "Doctor Get Referral", Method: "GET", Path: "/api/v1/doctor/referrals/:id", ExpStatus: 200, ActStatus: code, Pass: code == 200})

	// ---------------------------------------------------------
	// 4. LIAISON
	// ---------------------------------------------------------
	code, body = doReq("GET", "/api/v1/liaison/referrals/", liaTAToken, nil)
	logStep(StepResult{Step: "Liaison List Referrals", Method: "GET", Path: "/api/v1/liaison/referrals/", ExpStatus: 200, ActStatus: code, Pass: code == 200})

	code, body = doReq("POST", "/api/v1/liaison/referrals/"+refID+"/read", liaTAToken, nil)
	logStep(StepResult{Step: "Liaison Read Referral", Method: "POST", Path: "/api/v1/liaison/referrals/:id/read", ExpStatus: 200, ActStatus: code, Pass: code == 200})

	code, body = doReq("POST", "/api/v1/liaison/referrals/"+refID+"/forward", liaTAToken, nil)
	logStep(StepResult{Step: "Liaison Forward Referral", Method: "POST", Path: "/api/v1/liaison/referrals/:id/forward", ExpStatus: 200, ActStatus: code, Pass: code == 200})

	// ---------------------------------------------------------
	// 5. SPECIALIST
	// ---------------------------------------------------------
	code, body = doReq("GET", "/api/v1/specialist/referrals", specTAToken, nil)
	logStep(StepResult{Step: "Specialist List Referrals", Method: "GET", Path: "/api/v1/specialist/referrals", ExpStatus: 200, ActStatus: code, Pass: code == 200})

	code, body = doReq("POST", "/api/v1/specialist/referrals/"+refID+"/read", specTAToken, nil)
	logStep(StepResult{Step: "Specialist Read Referral", Method: "POST", Path: "/api/v1/specialist/referrals/:id/read", ExpStatus: 200, ActStatus: code, Pass: code == 200})

	code, body = doReq("POST", "/api/v1/specialist/referrals/"+refID+"/triage-severity", specTAToken, map[string]interface{}{"score": 75, "justification": "Moderate"})
	logStep(StepResult{Step: "Specialist Triage Severity", Method: "POST", Path: "/api/v1/specialist/referrals/:id/triage-severity", ExpStatus: 200, ActStatus: code, Pass: code == 200})

	code, body = doReq("POST", "/api/v1/specialist/referrals/"+refID+"/accept", specTAToken, nil)
	logStep(StepResult{Step: "Specialist Accept Referral", Method: "POST", Path: "/api/v1/specialist/referrals/:id/accept", ExpStatus: 200, ActStatus: code, Pass: code == 200})

	code, body = doReq("GET", "/api/v1/specialist/referrals/triage-queue", specTAToken, nil)
	logStep(StepResult{Step: "Specialist Get Triage Queue", Method: "GET", Path: "/api/v1/specialist/referrals/triage-queue", ExpStatus: 200, ActStatus: code, Pass: code == 200})

	// ---------------------------------------------------------
	// 6. DEPT HEAD
	// ---------------------------------------------------------
	code, body = doReq("POST", "/api/v1/department-head/schedule/batch", headTAToken, nil)
	logStep(StepResult{Step: "Dept Head Batch Schedule", Method: "POST", Path: "/api/v1/department-head/schedule/batch", ExpStatus: 200, ActStatus: code, Pass: code == 200})

	// ---------------------------------------------------------
	// 7. RECEPTIONIST
	// ---------------------------------------------------------
	code, body = doReq("GET", "/api/v1/receptionist/referrals/schedule", recTAToken, nil)
	logStep(StepResult{Step: "Receptionist Get Schedule", Method: "GET", Path: "/api/v1/receptionist/referrals/schedule", ExpStatus: 200, ActStatus: code, Pass: code == 200})
	
	// Extract queueID from schedule for our referral
	queueID := ""
	var schedResp struct { Data []map[string]interface{} }
	json.Unmarshal(body, &schedResp)
	for _, q := range schedResp.Data {
		if q["referral_id"] == refID {
			queueID = q["id"].(string)
			break
		}
	}
	
	if queueID == "" {
		// Fallback: Check Specialist Triage Queue
		code, body = doReq("GET", "/api/v1/specialist/referrals/triage-queue", specTAToken, nil)
		var qResp struct { Data []map[string]interface{} }
		json.Unmarshal(body, &qResp)
		for _, q := range qResp.Data {
			if q["referral_id"] == refID {
				queueID = q["id"].(string)
				break
			}
		}
	}

	if queueID != "" {
		code, body = doReq("POST", "/api/v1/receptionist/referrals/"+queueID+"/arrive", recTAToken, nil)
		logStep(StepResult{Step: "Receptionist Confirm Arrival", Method: "POST", Path: "/api/v1/receptionist/:id/arrive", ExpStatus: 200, ActStatus: code, Pass: code == 200})

		code, body = doReq("POST", "/api/v1/receptionist/referrals/"+queueID+"/assign-doctor", recTAToken, map[string]interface{}{"doctor_id": specTAUserID})
		logStep(StepResult{Step: "Receptionist Assign Doctor", Method: "POST", Path: "/api/v1/receptionist/:id/assign-doctor", ExpStatus: 200, ActStatus: code, Pass: code == 200})
	} else {
		fmt.Printf("Warning: queueID not found for refID %s in schedule or triage queue\n", refID)
	}

	// ---------------------------------------------------------
	// 8. CLINICAL & COMPLETION
	// ---------------------------------------------------------
	code, body = doReq("POST", "/api/v1/referrals/"+refID+"/clinical/updates", specTAToken, map[string]interface{}{
		"update_reason": "CONDITION_CHANGE",
		"clinical_notes": "Patient seen, stable condition.",
	})
	logStep(StepResult{Step: "Add Clinical Update", Method: "POST", Path: "/api/v1/referrals/:id/clinical/updates", ExpStatus: 200, ActStatus: code, Pass: code == 200})

	code, body = doReq("POST", "/api/v1/referrals/"+refID+"/clinical/outcome", specTAToken, map[string]interface{}{
		"outcome": "discharged",
		"outcome_notes": "Prescribed medication, follow up in 2 weeks.",
	})
	logStep(StepResult{Step: "Complete Referral (Outcome)", Method: "POST", Path: "/api/v1/referrals/:id/clinical/outcome", ExpStatus: 200, ActStatus: code, Pass: code == 200})

	// ---------------------------------------------------------
	// 9. SYSTEM ADMIN
	// ---------------------------------------------------------
	code, body = doReq("GET", "/api/v1/system-admin/users", adminToken, nil)
	logStep(StepResult{Step: "SysAdmin List Users", Method: "GET", Path: "/api/v1/system-admin/users", ExpStatus: 200, ActStatus: code, Pass: code == 200})

	code, body = doReq("GET", "/api/v1/hospitals", adminToken, nil)
	logStep(StepResult{Step: "List Hospitals", Method: "GET", Path: "/api/v1/hospitals", ExpStatus: 200, ActStatus: code, Pass: code == 200})

	code, body = doReq("GET", "/api/v1/departments", adminToken, nil)
	logStep(StepResult{Step: "List Departments", Method: "GET", Path: "/api/v1/departments", ExpStatus: 200, ActStatus: code, Pass: code == 200})

	// ---------------------------------------------------------
	// 9. REFERENCE & PATIENT
	// ---------------------------------------------------------
	code, body = doReq("GET", "/api/v1/reference/hospitals", drTAToken, nil)
	logStep(StepResult{Step: "Ref Hospitals", Method: "GET", Path: "/api/v1/reference/hospitals", ExpStatus: 200, ActStatus: code, Pass: code == 200})

	code, body = doReq("GET", "/api/v1/reference/icd-codes", drTAToken, nil)
	logStep(StepResult{Step: "Ref ICD Codes", Method: "GET", Path: "/api/v1/reference/icd-codes", ExpStatus: 200, ActStatus: code, Pass: code == 200})

	code, body = doReq("GET", "/api/v1/patients/lookup?national_id=NAT-E2E-001", drTAToken, nil)
	logStep(StepResult{Step: "Patient Lookup", Method: "GET", Path: "/api/v1/patients/lookup", ExpStatus: 200, ActStatus: code, Pass: code == 200})

	// ---------------------------------------------------------
	// 10. ATTACHMENTS
	// ---------------------------------------------------------
	code, body = doReq("GET", "/api/v1/attachments/signature", drTAToken, nil)
	logStep(StepResult{Step: "Get Upload Signature", Method: "GET", Path: "/api/v1/attachments/signature", ExpStatus: 200, ActStatus: code, Pass: code == 200})

	// ---------------------------------------------------------
	// 11. CLINICAL
	// ---------------------------------------------------------
	code, body = doReq("GET", "/api/v1/referrals/"+refID+"/clinical/history", specTAToken, nil)
	logStep(StepResult{Step: "Clinical History", Method: "GET", Path: "/api/v1/referrals/:id/clinical/history", ExpStatus: 200, ActStatus: code, Pass: code == 200})

	// ---------------------------------------------------------
	// 12. HOSPITAL ADMIN
	// ---------------------------------------------------------
	code, body = doReq("GET", "/api/v1/hospital-admin/referrals-log", spAdminToken, nil)
	logStep(StepResult{Step: "HospAdmin Referrals Log", Method: "GET", Path: "/api/v1/hospital-admin/referrals-log", ExpStatus: 200, ActStatus: code, Pass: code == 200})

	// ---------------------------------------------------------
	// 13. NEGATIVE TESTS
	// ---------------------------------------------------------
	code, body = doReq("DELETE", "/api/v1/hospitals/invalid-uuid", adminToken, nil)
	logStep(StepResult{Step: "Delete Invalid Hosp (NEG)", Method: "DELETE", Path: "/api/v1/hospitals/invalid-uuid", ExpStatus: 400, ActStatus: code, Pass: code == 400})

	// Final Report Save
	f, _ := os.Create("e2e_all_endpoints_report.txt")
	defer f.Close()
	f.WriteString("COMPREHENSIVE ALL-ENDPOINT E2E REPORT\n")
	f.WriteString(fmt.Sprintf("Run Date: %s\n", time.Now().Format(time.RFC3339)))
	for _, line := range report {
		f.WriteString(line + "\n")
	}

	fmt.Println("\nAll-endpoint E2E test suite completed. Report saved to e2e_all_endpoints_report.txt")
}
