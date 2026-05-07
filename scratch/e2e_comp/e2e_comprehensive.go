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
	Asserts   []string
}

func logStep(res StepResult) {
	passStr := "FAIL"
	if res.Pass {
		passStr = "PASS"
	}

	resultLine := fmt.Sprintf("| %-35s | %-6s | %-60s | %d | %d | %s |", res.Step, res.Method, res.Path, res.ExpStatus, res.ActStatus, passStr)
	fmt.Println(resultLine)
	report = append(report, resultLine)

	for _, a := range res.Asserts {
		fmt.Printf("   - %s\n", a)
		report = append(report, "   - "+a)
	}

	if !res.Pass && res.Details != "" {
		// Truncate details if too long
		det := res.Details
		if len(det) > 500 {
			det = det[:500] + "... [truncated]"
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
		AccessToken string `json:"access_token"` // Fallback
	}
	json.Unmarshal(body, &res)
	if res.Data.AccessToken != "" {
		return res.Data.AccessToken
	}
	return res.AccessToken
}

// extractID extracts an ID from a generic JSON map
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

// findQueueID searches a list response for a specific referral_id
func findQueueID(body []byte, refID string) string {
	var res struct {
		Data []map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(body, &res); err == nil && len(res.Data) > 0 {
		for _, item := range res.Data {
			if item["referral_id"] == refID {
				if id, ok := item["id"].(string); ok {
					return id
				}
			}
		}
	}

	// Try without data wrapper
	var arr []map[string]interface{}
	if err := json.Unmarshal(body, &arr); err == nil {
		for _, item := range arr {
			if item["referral_id"] == refID {
				if id, ok := item["id"].(string); ok {
					return id
				}
			}
		}
	}
	return ""
}

func main() {
	fmt.Println("Starting Comprehensive E2E Tests...")
	fmt.Println("| Step                                | Method | Path                                                         | Exp  | Act  | Pass |")
	fmt.Println("|-------------------------------------|--------|--------------------------------------------------------------|------|------|------|")

	// 0. Health Check
	code, body := doReq("GET", "/health", "", nil)
	logStep(StepResult{Step: "0. Health Check", Method: "GET", Path: "/health", ExpStatus: 200, ActStatus: code, Pass: code == 200, Details: string(body)})

	// 1. System Admin - Config
	adminToken := login("superadmin@moh.gov.et")
	
	// GET Config
	code, body = doReq("GET", "/api/v1/admin/config", adminToken, nil)
	pass := code == 200
	var configRes struct {
		Data map[string]interface{} `json:"data"`
	}
	json.Unmarshal(body, &configRes)
	asserts := []string{}
	if pass {
		if _, ok := configRes.Data["buffer_days"]; ok {
			asserts = append(asserts, "PASS: buffer_days key exists")
		} else {
			asserts = append(asserts, "FAIL: buffer_days key missing")
			pass = false
		}
	}
	logStep(StepResult{Step: "1. GET Config", Method: "GET", Path: "/api/v1/admin/config", ExpStatus: 200, ActStatus: code, Pass: pass, Asserts: asserts})

	// PUT Config
	code, body = doReq("PUT", "/api/v1/admin/config", adminToken, map[string]string{"buffer_days": "3", "aging_factor": "1.2", "max_horizon_days": "14", "overbook_limit_default": "2"})
	logStep(StepResult{Step: "1. PUT Config", Method: "PUT", Path: "/api/v1/admin/config", ExpStatus: 200, ActStatus: code, Pass: code == 200, Details: string(body)})

	// 2. Doctor
	drTAToken := login("doctor.ta@hospital.et")
	
	// Create Ref 1 (Abebe, TA Cardio, unstable)
	ref1Req := map[string]interface{}{
		"patient_id":                   "e0000000-0000-0000-0000-000000000001",
		"target_hospital_id":           "a1000000-0000-0000-0000-000000000001",
		"target_dept_id":               "b1000000-0000-0000-0000-000000000001",
		"sender_hospital_id":           "a1000000-0000-0000-0000-000000000001",
		"liaison_officer_id":           "d1000000-0000-0000-0000-000000000002",
		"clinical_summary":             "Chest pain",
		"reason_for_referral_category": "ROUTINE",
		"condition_at_referral":        "unstable",
		"patient_history":              "Hypertension",
		"reason_of_referral":           "Further evaluation",
		"diagnoses":                    []map[string]interface{}{{"icd_code": "I21.9", "is_primary": true, "diagnosis_certainty": "SUSPECTED"}},
		"status":                       "DRAFT",
	}
	code, body = doReq("POST", "/api/v1/doctor/referrals", drTAToken, ref1Req)
	refTA1 := extractID(body, "data", "id")
	if refTA1 == "" { refTA1 = extractID(body, "referral", "id") }
	logStep(StepResult{Step: "2. Create Ref 1", Method: "POST", Path: "/api/v1/doctor/referrals", ExpStatus: 201, ActStatus: code, Pass: code == 201 && refTA1 != "", Details: string(body)})

	// Submit Ref 1
	delete(ref1Req, "status")
	code, body = doReq("PUT", "/api/v1/doctor/referrals/"+refTA1+"/submit", drTAToken, ref1Req)
	logStep(StepResult{Step: "2. Submit Ref 1", Method: "PUT", Path: "/api/v1/doctor/referrals/.../submit", ExpStatus: 200, ActStatus: code, Pass: code == 200, Details: string(body)})

	// Create Ref 2 (Meseret, BL Peds, critical)
	ref2Req := map[string]interface{}{
		"patient_id":                   "e0000000-0000-0000-0000-000000000002",
		"target_hospital_id":           "a3000000-0000-0000-0000-000000000003",
		"target_dept_id":               "b5000000-0000-0000-0000-000000000005",
		"sender_hospital_id":           "a1000000-0000-0000-0000-000000000001",
		"liaison_officer_id":           "d1000000-0000-0000-0000-000000000002",
		"clinical_summary":             "Unconscious trauma",
		"reason_for_referral_category": "EMERGENCY",
		"condition_at_referral":        "critical",
		"patient_history":              "Accident",
		"reason_of_referral":           "Immediate surgery",
		"diagnoses":                    []map[string]interface{}{{"icd_code": "S06.9X9A", "is_primary": true, "diagnosis_certainty": "CONFIRMED"}},
		"status":                       "DRAFT",
	}
	code, body = doReq("POST", "/api/v1/doctor/referrals", drTAToken, ref2Req)
	refTA2 := extractID(body, "data", "id")
	if refTA2 == "" { refTA2 = extractID(body, "referral", "id") }
	logStep(StepResult{Step: "2. Create Ref 2", Method: "POST", Path: "/api/v1/doctor/referrals", ExpStatus: 201, ActStatus: code, Pass: code == 201 && refTA2 != "", Details: string(body)})

	delete(ref2Req, "status")
	code, body = doReq("PUT", "/api/v1/doctor/referrals/"+refTA2+"/submit", drTAToken, ref2Req)
	logStep(StepResult{Step: "2. Submit Ref 2", Method: "PUT", Path: "/api/v1/doctor/referrals/.../submit", ExpStatus: 200, ActStatus: code, Pass: code == 200, Details: string(body)})

	// 3. Liaison (Tikur Anbessa)
	liaTAToken := login("liaison.ta@hospital.et")
	
	// Ref TA1
	code, body = doReq("POST", "/api/v1/liaison/referrals/"+refTA1+"/read", liaTAToken, nil)
	logStep(StepResult{Step: "3. Liaison Read TA1", Method: "POST", Path: "/api/v1/liaison/referrals/.../read", ExpStatus: 200, ActStatus: code, Pass: code == 200, Details: string(body)})
	code, body = doReq("POST", "/api/v1/liaison/referrals/"+refTA1+"/forward", liaTAToken, nil)
	logStep(StepResult{Step: "3. Liaison Forward TA1", Method: "POST", Path: "/api/v1/liaison/referrals/.../forward", ExpStatus: 200, ActStatus: code, Pass: code == 200, Details: string(body)})

	// Ref TA2
	code, body = doReq("POST", "/api/v1/liaison/referrals/"+refTA2+"/read", liaTAToken, nil)
	logStep(StepResult{Step: "3. Liaison Read TA2", Method: "POST", Path: "/api/v1/liaison/referrals/.../read", ExpStatus: 200, ActStatus: code, Pass: code == 200, Details: string(body)})
	code, body = doReq("POST", "/api/v1/liaison/referrals/"+refTA2+"/forward", liaTAToken, nil)
	logStep(StepResult{Step: "3. Liaison Forward TA2", Method: "POST", Path: "/api/v1/liaison/referrals/.../forward", ExpStatus: 200, ActStatus: code, Pass: code == 200, Details: string(body)})

	// 4. Specialist
	specTAToken := login("specialist.ta@hospital.et")

	code, body = doReq("POST", "/api/v1/specialist/referrals/"+refTA1+"/read", specTAToken, nil)
	logStep(StepResult{Step: "4. TA Spec Read TA1", Method: "POST", Path: "/api/v1/specialist/referrals/.../read", ExpStatus: 200, ActStatus: code, Pass: code == 200, Details: string(body)})

	// Negative Test: Accept without severity
	code, body = doReq("POST", "/api/v1/specialist/referrals/"+refTA1+"/accept", specTAToken, nil)
	logStep(StepResult{Step: "4. Accept without severity (NEG)", Method: "POST", Path: "/api/v1/specialist/referrals/.../accept", ExpStatus: 400, ActStatus: code, Pass: code == 400, Details: string(body)})

	// Set Severity
	code, body = doReq("POST", "/api/v1/specialist/referrals/"+refTA1+"/triage-severity", specTAToken, map[string]interface{}{"score": 85, "justification": "High risk"})
	logStep(StepResult{Step: "4. Set Severity TA1", Method: "POST", Path: "/api/v1/specialist/referrals/.../triage-severity", ExpStatus: 200, ActStatus: code, Pass: code == 200, Details: string(body)})

	// Accept
	code, body = doReq("POST", "/api/v1/specialist/referrals/"+refTA1+"/accept", specTAToken, nil)
	logStep(StepResult{Step: "4. Accept TA1", Method: "POST", Path: "/api/v1/specialist/referrals/.../accept", ExpStatus: 200, ActStatus: code, Pass: code == 200, Details: string(body)})

	// Verify TA1 Status
	code, body = doReq("GET", "/api/v1/specialist/referrals/"+refTA1, specTAToken, nil)
	refTA1Status := extractID(body, "status")
	if refTA1Status == "" { refTA1Status = extractID(body, "data", "status") }
	if refTA1Status == "" { refTA1Status = extractID(body, "referral", "status") }
	logStep(StepResult{Step: "4. Verify TA1 ACCEPTED", Method: "GET", Path: "/api/v1/specialist/referrals/"+refTA1, ExpStatus: 200, ActStatus: code, Pass: refTA1Status == "ACCEPTED", Asserts: []string{"Status == ACCEPTED"}, Details: string(body)})

	// Extract Queue ID for TA1 before it gets batch scheduled
	code, body = doReq("GET", "/api/v1/specialist/referrals/triage-queue", specTAToken, nil)
	tqTA1 := findQueueID(body, refTA1)
	logStep(StepResult{Step: "4. Extract TA1 Queue ID", Method: "GET", Path: "/api/v1/specialist/referrals/triage-queue", ExpStatus: 200, ActStatus: code, Pass: tqTA1 != "", Asserts: []string{"Queue ID found: " + tqTA1}, Details: string(body)})

	// BL Specialist
	specBLToken := login("specialist.bl@hospital.et")
	code, body = doReq("POST", "/api/v1/specialist/referrals/"+refTA2+"/read", specBLToken, nil)
	code, body = doReq("POST", "/api/v1/specialist/referrals/"+refTA2+"/triage-severity", specBLToken, map[string]interface{}{"score": 95, "justification": "Critical"})
	code, body = doReq("POST", "/api/v1/specialist/referrals/"+refTA2+"/accept", specBLToken, nil)
	logStep(StepResult{Step: "4. BL Spec Accept TA2", Method: "POST", Path: "/api/v1/specialist/referrals/.../accept", ExpStatus: 200, ActStatus: code, Pass: code == 200, Details: string(body)})

	// Emergency Schedule TA2
	tomorrow := time.Now().AddDate(0, 0, 1).Format("2006-01-02")
	code, body = doReq("POST", "/api/v1/specialist/referrals/"+refTA2+"/emergency-schedule", specBLToken, map[string]string{"appointment_date": tomorrow, "justification": "Critical condition"})
	logStep(StepResult{Step: "4. Emergency Schedule TA2", Method: "POST", Path: "/api/v1/specialist/referrals/.../emergency-schedule", ExpStatus: 200, ActStatus: code, Pass: code == 200, Details: string(body)})

	// 5. Dept Head (Tikur Anbessa)
	headTAToken := login("depthead.ta@hospital.et")
	
	code, body = doReq("POST", "/api/v1/department-head/schedule/batch", headTAToken, nil)
	logStep(StepResult{Step: "5. TA Head Batch", Method: "POST", Path: "/api/v1/department-head/schedule/batch", ExpStatus: 200, ActStatus: code, Pass: code == 200, Details: string(body)})

	// Verify TA1 Appointment Date (Optional: Can check via /schedule later)
	// We already have tqTA1 from step 4.

	// 6. Receptionist (Aster @ TA)
	recTAToken := login("reception.ta@hospital.et")
	
	if tqTA1 != "" {
		code, body = doReq("POST", "/api/v1/receptionist/referrals/"+tqTA1+"/arrive", recTAToken, nil)
		logStep(StepResult{Step: "6. TA Rec Arrive", Method: "POST", Path: "/api/v1/receptionist/referrals/.../arrive", ExpStatus: 200, ActStatus: code, Pass: code == 200, Details: string(body)})
		
		// Assign Doctor (Yohannes ID: d1000000-0000-0000-0000-000000000003)
		code, body = doReq("POST", "/api/v1/receptionist/referrals/"+tqTA1+"/assign-doctor", recTAToken, map[string]string{"doctor_id": "d1000000-0000-0000-0000-000000000003"})
		logStep(StepResult{Step: "6. TA Rec Assign", Method: "POST", Path: "/api/v1/receptionist/referrals/.../assign-doctor", ExpStatus: 200, ActStatus: code, Pass: code == 200, Details: string(body)})
		
		// Mark Missed
		code, body = doReq("POST", "/api/v1/receptionist/referrals/"+tqTA1+"/miss", recTAToken, map[string]string{"miss_reason": "PATIENT_NO_SHOW"})
		logStep(StepResult{Step: "6. TA Rec Miss", Method: "POST", Path: "/api/v1/receptionist/referrals/.../miss", ExpStatus: 200, ActStatus: code, Pass: code == 200, Details: string(body)})
	}

	// BL Receptionist (Walk-in for Ref 2)
	recBLToken := login("reception.bl@hospital.et")
	code, body = doReq("POST", "/api/v1/receptionist/referrals/walk-in", recBLToken, map[string]string{"referral_id": refTA2})
	logStep(StepResult{Step: "6. BL Rec Walk-in", Method: "POST", Path: "/api/v1/receptionist/referrals/walk-in", ExpStatus: 200, ActStatus: code, Pass: code == 200, Details: string(body)})

	// 7. Clinical Updates & Outcome (TA)
	code, body = doReq("POST", "/api/v1/referrals/"+refTA1+"/clinical/updates", specTAToken, map[string]interface{}{"update_reason": "CONDITION_CHANGE", "clinical_notes": "Patient improved after missing"})
	logStep(StepResult{Step: "7. Add Clinical Update", Method: "POST", Path: "/api/v1/referrals/.../clinical/updates", ExpStatus: 200, ActStatus: code, Pass: code == 200, Details: string(body)})

	code, body = doReq("POST", "/api/v1/referrals/"+refTA1+"/clinical/outcome", specTAToken, map[string]interface{}{"outcome": "improved", "outcome_notes": "Discharged finally"})
	logStep(StepResult{Step: "7. Record Outcome TA1", Method: "POST", Path: "/api/v1/referrals/.../clinical/outcome", ExpStatus: 200, ActStatus: code, Pass: code == 200, Details: string(body)})

	code, body = doReq("POST", "/api/v1/referrals/"+refTA2+"/clinical/outcome", specBLToken, map[string]interface{}{"outcome": "discharged", "outcome_notes": "Emergency handled"})
	logStep(StepResult{Step: "7. Record Outcome TA2", Method: "POST", Path: "/api/v1/referrals/.../clinical/outcome", ExpStatus: 200, ActStatus: code, Pass: code == 200, Details: string(body)})

	// 8. Notifications
	code, body = doReq("POST", "/api/v1/internal/notifications/send", adminToken, nil)
	logStep(StepResult{Step: "8. Admin Notif Send", Method: "POST", Path: "/api/v1/internal/notifications/send", ExpStatus: 200, ActStatus: code, Pass: code == 200, Details: string(body)})

	// 9. In-App Notifications
	fmt.Println("\nVerifying In-App Notifications...")
	
	// List Notifications for Doctor
	code, body = doReq("GET", "/api/v1/me/notifications?limit=10&page=1", drTAToken, nil)
	pass = code == 200
	var notifRes struct {
		Data        []map[string]interface{} `json:"data"`
		UnreadCount int                      `json:"unread_count"`
	}
	json.Unmarshal(body, &notifRes)
	asserts = []string{}
	var firstNotifID string
	if pass {
		asserts = append(asserts, fmt.Sprintf("PASS: Found %d notifications", len(notifRes.Data)))
		if len(notifRes.Data) > 0 {
			firstNotifID = notifRes.Data[0]["id"].(string)
			asserts = append(asserts, "PASS: First notification ID: "+firstNotifID)
		}
	}
	logStep(StepResult{Step: "9. List Notifications", Method: "GET", Path: "/api/v1/me/notifications", ExpStatus: 200, ActStatus: code, Pass: pass, Asserts: asserts, Details: string(body)})

	if firstNotifID != "" {
		// Mark one as read
		code, body = doReq("POST", "/api/v1/me/notifications/"+firstNotifID+"/read", drTAToken, nil)
		logStep(StepResult{Step: "9. Mark Notification Read", Method: "POST", Path: "/api/v1/me/notifications/.../read", ExpStatus: 200, ActStatus: code, Pass: code == 200, Details: string(body)})
	}

	// Get Unread Count
	code, body = doReq("GET", "/api/v1/me/notifications/unread-count", drTAToken, nil)
	var countRes struct {
		UnreadCount int `json:"unread_count"`
	}
	json.Unmarshal(body, &countRes)
	logStep(StepResult{Step: "9. Get Unread Count", Method: "GET", Path: "/api/v1/me/notifications/unread-count", ExpStatus: 200, ActStatus: code, Pass: code == 200, Asserts: []string{fmt.Sprintf("Unread count: %d", countRes.UnreadCount)}, Details: string(body)})

	// Save Report
	f, _ := os.OpenFile("e2e_report.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	defer f.Close()
	f.WriteString("\n--- COMPREHENSIVE E2E RUN at " + time.Now().Format(time.RFC3339) + " ---\n")
	for _, line := range report {
		f.WriteString(line + "\n")
	}

	fmt.Println("\nE2E test suite completed. Report appended to e2e_report.txt")
}
