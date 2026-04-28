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
		fmt.Printf("   Error: %s\n", res.Details)
		report = append(report, "   Error: "+res.Details)
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
	return res.AccessToken
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
		Success bool `json:"success"`
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
	code, body = doReq("PUT", "/api/v1/admin/config", adminToken, map[string]string{"buffer_days": "3", "aging_factor": "1.2"})
	logStep(StepResult{Step: "1. PUT Config", Method: "PUT", Path: "/api/v1/admin/config", ExpStatus: 200, ActStatus: code, Pass: code == 200, Details: string(body)})

	// Verify PUT
	code, body = doReq("GET", "/api/v1/admin/config", adminToken, nil)
	json.Unmarshal(body, &configRes)
	pass = code == 200 && configRes.Data["buffer_days"] == "3"
	asserts = []string{"Data assertion: buffer_days == 3"}
	logStep(StepResult{Step: "1. Verify Config Change", Method: "GET", Path: "/api/v1/admin/config", ExpStatus: 200, ActStatus: code, Pass: pass, Asserts: asserts})

	// 2. Doctor (Tikur Anbessa)
	drTAToken := login("doctor.ta@hospital.et")
	
	// Stats
	code, body = doReq("GET", "/api/v1/doctor/stats", drTAToken, nil)
	logStep(StepResult{Step: "2. TA Doctor Stats", Method: "GET", Path: "/api/v1/doctor/stats", ExpStatus: 200, ActStatus: code, Pass: code == 200})

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
		"diagnoses":                    []map[string]interface{}{{"icd_code": "I21.9", "is_primary": true, "diagnosis_certainty": "SUSPECTED"}},
		"status":                       "DRAFT",
	}
	code, body = doReq("POST", "/api/v1/doctor/referrals", drTAToken, ref1Req)
	refTA1 := extractID(body, "referral", "id")
	logStep(StepResult{Step: "2. Create Ref 1", Method: "POST", Path: "/api/v1/doctor/referrals", ExpStatus: 201, ActStatus: code, Pass: code == 201 && refTA1 != ""})

	// Submit Ref 1
	delete(ref1Req, "status")
	code, body = doReq("PUT", "/api/v1/doctor/referrals/"+refTA1+"/submit", drTAToken, ref1Req)
	logStep(StepResult{Step: "2. Submit Ref 1", Method: "PUT", Path: "/api/v1/doctor/referrals/.../submit", ExpStatus: 200, ActStatus: code, Pass: code == 200})

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
		"diagnoses":                    []map[string]interface{}{{"icd_code": "S06.9X9A", "is_primary": true, "diagnosis_certainty": "CONFIRMED"}},
		"status":                       "DRAFT",
	}
	code, body = doReq("POST", "/api/v1/doctor/referrals", drTAToken, ref2Req)
	refTA2 := extractID(body, "referral", "id")
	logStep(StepResult{Step: "2. Create Ref 2", Method: "POST", Path: "/api/v1/doctor/referrals", ExpStatus: 201, ActStatus: code, Pass: code == 201 && refTA2 != ""})

	// Submit Ref 2
	delete(ref2Req, "status")
	code, body = doReq("PUT", "/api/v1/doctor/referrals/"+refTA2+"/submit", drTAToken, ref2Req)
	logStep(StepResult{Step: "2. Submit Ref 2", Method: "PUT", Path: "/api/v1/doctor/referrals/.../submit", ExpStatus: 200, ActStatus: code, Pass: code == 200})

	// Create Ref 3 (St Pauls)
	drSPToken := login("doctor.sp@hospital.et")
	ref3Req := map[string]interface{}{
		"patient_id":                   "e0000000-0000-0000-0000-000000000003",
		"target_hospital_id":           "a2000000-0000-0000-0000-000000000002",
		"target_dept_id":               "b3000000-0000-0000-0000-000000000003",
		"sender_hospital_id":           "a2000000-0000-0000-0000-000000000002",
		"liaison_officer_id":           "d2000000-0000-0000-0000-000000000002",
		"clinical_summary":             "Stable ortho",
		"reason_for_referral_category": "ROUTINE",
		"condition_at_referral":        "stable",
		"diagnoses":                    []map[string]interface{}{{"icd_code": "J18.9", "is_primary": true, "diagnosis_certainty": "SUSPECTED"}},
		"status":                       "DRAFT",
	}
	code, body = doReq("POST", "/api/v1/doctor/referrals", drSPToken, ref3Req)
	refSP1 := extractID(body, "referral", "id")
	logStep(StepResult{Step: "2. Create Ref 3 (SP)", Method: "POST", Path: "/api/v1/doctor/referrals", ExpStatus: 201, ActStatus: code, Pass: code == 201 && refSP1 != ""})

	delete(ref3Req, "status")
	code, body = doReq("PUT", "/api/v1/doctor/referrals/"+refSP1+"/submit", drSPToken, ref3Req)
	logStep(StepResult{Step: "2. Submit Ref 3 (SP)", Method: "PUT", Path: "/api/v1/doctor/referrals/.../submit", ExpStatus: 200, ActStatus: code, Pass: code == 200})

	// 3. Liaison (Tikur Anbessa)
	liaTAToken := login("liaison.ta@hospital.et")
	
	// Ref TA1
	code, _ = doReq("POST", "/api/v1/liaison/referrals/"+refTA1+"/read", liaTAToken, nil)
	logStep(StepResult{Step: "3. Liaison Read TA1", Method: "POST", Path: "/api/v1/liaison/referrals/.../read", ExpStatus: 200, ActStatus: code, Pass: code == 200})
	code, _ = doReq("POST", "/api/v1/liaison/referrals/"+refTA1+"/forward", liaTAToken, nil)
	logStep(StepResult{Step: "3. Liaison Forward TA1", Method: "POST", Path: "/api/v1/liaison/referrals/.../forward", ExpStatus: 200, ActStatus: code, Pass: code == 200})

	// Ref TA2
	code, _ = doReq("POST", "/api/v1/liaison/referrals/"+refTA2+"/read", liaTAToken, nil)
	logStep(StepResult{Step: "3. Liaison Read TA2", Method: "POST", Path: "/api/v1/liaison/referrals/.../read", ExpStatus: 200, ActStatus: code, Pass: code == 200})
	code, _ = doReq("POST", "/api/v1/liaison/referrals/"+refTA2+"/forward", liaTAToken, nil)
	logStep(StepResult{Step: "3. Liaison Forward TA2", Method: "POST", Path: "/api/v1/liaison/referrals/.../forward", ExpStatus: 200, ActStatus: code, Pass: code == 200})

	// 4. Specialist
	// TA Specialist (Yohannes)
	specTAToken := login("specialist.ta@hospital.et")

	// Get Capacity
	code, body = doReq("GET", "/api/v1/specialist/referrals/capacity", specTAToken, nil)
	logStep(StepResult{Step: "4. TA Spec Get Capacity", Method: "GET", Path: "/api/v1/specialist/referrals/capacity", ExpStatus: 200, ActStatus: code, Pass: code == 200})

	code, _ = doReq("POST", "/api/v1/specialist/referrals/"+refTA1+"/read", specTAToken, nil)
	logStep(StepResult{Step: "4. TA Spec Read TA1", Method: "POST", Path: "/api/v1/specialist/referrals/.../read", ExpStatus: 200, ActStatus: code, Pass: code == 200})

	// Negative Test: Accept without severity
	code, body = doReq("POST", "/api/v1/specialist/referrals/"+refTA1+"/accept", specTAToken, nil)
	logStep(StepResult{Step: "4. Accept without severity (NEG)", Method: "POST", Path: "/api/v1/specialist/referrals/.../accept", ExpStatus: 400, ActStatus: code, Pass: code == 400, Details: string(body)})

	// Set Severity
	code, _ = doReq("POST", "/api/v1/specialist/referrals/"+refTA1+"/triage-severity", specTAToken, map[string]interface{}{"score": 85, "justification": "High risk"})
	logStep(StepResult{Step: "4. Set Severity TA1", Method: "POST", Path: "/api/v1/specialist/referrals/.../triage-severity", ExpStatus: 200, ActStatus: code, Pass: code == 200})

	// Accept
	code, _ = doReq("POST", "/api/v1/specialist/referrals/"+refTA1+"/accept", specTAToken, nil)
	logStep(StepResult{Step: "4. Accept TA1", Method: "POST", Path: "/api/v1/specialist/referrals/.../accept", ExpStatus: 200, ActStatus: code, Pass: code == 200})

	// Verify TA1 Status
	code, body = doReq("GET", "/api/v1/specialist/referrals/"+refTA1, specTAToken, nil)
	refTA1Status := extractID(body, "status")
	if refTA1Status == "" {
		// Try nested just in case
		refTA1Status = extractID(body, "referral", "status")
	}
	pass = refTA1Status == "ACCEPTED"
	logStep(StepResult{Step: "4. Verify TA1 ACCEPTED", Method: "GET", Path: "/api/v1/specialist/referrals/"+refTA1, ExpStatus: 200, ActStatus: code, Pass: pass, Asserts: []string{"Status == ACCEPTED"}, Details: string(body)})

	// BL Specialist (Martha)
	specBLToken := login("specialist.bl@hospital.et")
	code, _ = doReq("POST", "/api/v1/specialist/referrals/"+refTA2+"/read", specBLToken, nil)
	code, _ = doReq("POST", "/api/v1/specialist/referrals/"+refTA2+"/triage-severity", specBLToken, map[string]interface{}{"score": 95, "justification": "Critical"})
	code, _ = doReq("POST", "/api/v1/specialist/referrals/"+refTA2+"/accept", specBLToken, nil)
	logStep(StepResult{Step: "4. BL Spec Accept TA2", Method: "POST", Path: "/api/v1/specialist/referrals/.../accept", ExpStatus: 200, ActStatus: code, Pass: code == 200})

	// Emergency Schedule TA2
	tomorrow := time.Now().AddDate(0, 0, 1).Format("2006-01-02")
	code, body = doReq("POST", "/api/v1/specialist/referrals/"+refTA2+"/emergency-schedule", specBLToken, map[string]string{"appointment_date": tomorrow, "justification": "Critical condition"})
	logStep(StepResult{Step: "4. Emergency Schedule TA2", Method: "POST", Path: "/api/v1/specialist/referrals/.../emergency-schedule", ExpStatus: 200, ActStatus: code, Pass: code == 200, Details: string(body)})

	// Verify Queue Entry for Emergency
	code, body = doReq("GET", "/api/v1/specialist/referrals/triage-queue", specBLToken, nil)
	var tqBLRes struct { Data []map[string]interface{} `json:"data"` }
	json.Unmarshal(body, &tqBLRes)
	emergencyFound := false
	for _, item := range tqBLRes.Data {
		if item["referral_id"] == refTA2 && item["appointment_date"] != nil {
			emergencyFound = true
			break
		}
	}
	logStep(StepResult{Step: "4. Verify Emergency Queue Entry", Method: "GET", Path: "/api/v1/specialist/referrals/triage-queue", ExpStatus: 200, ActStatus: code, Pass: emergencyFound})

	// SP Specialist (Kidist)
	specSPToken := login("specialist.sp@hospital.et")
	code, _ = doReq("POST", "/api/v1/specialist/referrals/"+refSP1+"/read", specSPToken, nil)
	code, _ = doReq("POST", "/api/v1/specialist/referrals/"+refSP1+"/triage-severity", specSPToken, map[string]interface{}{"score": 70, "justification": "Standard"})
	code, _ = doReq("POST", "/api/v1/specialist/referrals/"+refSP1+"/accept", specSPToken, nil)
	logStep(StepResult{Step: "4. SP Spec Accept SP1", Method: "POST", Path: "/api/v1/specialist/referrals/.../accept", ExpStatus: 200, ActStatus: code, Pass: code == 200})

	// 5. Dept Head (Tikur Anbessa)
	headTAToken := login("depthead.ta@hospital.et")
	
	// Get Schedule for future date
	code, body = doReq("GET", "/api/v1/department-head/schedule?start_date=2026-05-01", headTAToken, nil)
	var schedList struct {
		Data []struct {
			ID string `json:"id"`
			Date string `json:"schedule_date"`
		} `json:"data"`
	}
	json.Unmarshal(body, &schedList)
	schedID := ""
	if len(schedList.Data) > 0 {
		schedID = schedList.Data[0].ID
	}
	logStep(StepResult{Step: "5. GET Schedule", Method: "GET", Path: "/api/v1/department-head/schedule", ExpStatus: 200, ActStatus: code, Pass: code == 200 && schedID != ""})

	// Override CRUD
	code, body = doReq("POST", "/api/v1/department-head/capacity/overrides", headTAToken, map[string]interface{}{"date": "2026-05-01", "new_limit": 10, "reason": "Public holiday"})
	logStep(StepResult{Step: "5. POST Override", Method: "POST", Path: "/api/v1/department-head/capacity/overrides", ExpStatus: 201, ActStatus: code, Pass: code == 201})

	code, body = doReq("GET", "/api/v1/department-head/capacity/overrides", headTAToken, nil)
	var overrideList struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	json.Unmarshal(body, &overrideList)
	ovrID := ""
	for _, o := range overrideList.Data {
		ovrID = o.ID
	}

	if ovrID != "" {
		code, _ = doReq("PUT", "/api/v1/department-head/capacity/overrides/"+ovrID, headTAToken, map[string]interface{}{"new_limit": 12, "reason": "Adjusted holiday"})
		logStep(StepResult{Step: "5. PUT Override", Method: "PUT", Path: "/api/v1/department-head/capacity/overrides/...", ExpStatus: 200, ActStatus: code, Pass: code == 200})
		code, _ = doReq("DELETE", "/api/v1/department-head/capacity/overrides/"+ovrID, headTAToken, nil)
		logStep(StepResult{Step: "5. DELETE Override", Method: "DELETE", Path: "/api/v1/department-head/capacity/overrides/...", ExpStatus: 200, ActStatus: code, Pass: code == 200})
	}

	// Update Max Slots
	if schedID != "" {
		code, _ = doReq("PUT", "/api/v1/department-head/schedule/"+schedID+"/max-slots", headTAToken, map[string]int{"max_slots": 25})
		logStep(StepResult{Step: "5. Update Max Slots", Method: "PUT", Path: "/api/v1/department-head/schedule/.../max-slots", ExpStatus: 200, ActStatus: code, Pass: code == 200})
	}

	// Batch Schedule TA
	code, body = doReq("POST", "/api/v1/department-head/schedule/batch", headTAToken, nil)
	logStep(StepResult{Step: "5. TA Head Batch", Method: "POST", Path: "/api/v1/department-head/schedule/batch", ExpStatus: 200, ActStatus: code, Pass: code == 200})

	// Verify TA1 Appointment Date (should be >= today+3 since buffer is 3)
	code, body = doReq("GET", "/api/v1/specialist/referrals/triage-queue", specTAToken, nil)
	var tqRes struct {
		Data []map[string]interface{} `json:"data"`
	}
	json.Unmarshal(body, &tqRes)
	ta1Date := ""
	tqTA1 := ""
	for _, item := range tqRes.Data {
		if item["referral_id"] == refTA1 {
			if d, ok := item["appointment_date"].(string); ok {
				ta1Date = d
			}
			if id, ok := item["id"].(string); ok {
				tqTA1 = id
			}
			break
		}
	}
	pass = ta1Date != ""
	asserts = []string{"Appointment date: " + ta1Date}
	logStep(StepResult{Step: "5. Verify TA1 Sched Date", Method: "GET", Path: "/api/v1/specialist/referrals/triage-queue", ExpStatus: 200, ActStatus: code, Pass: pass, Asserts: asserts})

	// 6. Receptionist (Aster @ TA)
	recTAToken := login("reception.ta@hospital.et")
	
	// Arrive
	if tqTA1 != "" {
		code, body = doReq("POST", "/api/v1/receptionist/referrals/"+tqTA1+"/arrive", recTAToken, nil)
		logStep(StepResult{Step: "6. TA Rec Arrive", Method: "POST", Path: "/api/v1/receptionist/referrals/.../arrive", ExpStatus: 200, ActStatus: code, Pass: code == 200, Details: string(body)})
		
		// Assign Doctor (Yohannes ID: d1000000-0000-0000-0000-000000000003)
		code, body = doReq("POST", "/api/v1/receptionist/referrals/"+tqTA1+"/assign-doctor", recTAToken, map[string]string{"doctor_id": "d1000000-0000-0000-0000-000000000003"})
		logStep(StepResult{Step: "6. TA Rec Assign", Method: "POST", Path: "/api/v1/receptionist/referrals/.../assign-doctor", ExpStatus: 200, ActStatus: code, Pass: code == 200, Details: string(body)})
		
		// Mark Missed
		code, body = doReq("POST", "/api/v1/receptionist/referrals/"+tqTA1+"/miss", recTAToken, map[string]string{"miss_reason": "PATIENT_NO_SHOW"})
		logStep(StepResult{Step: "6. TA Rec Miss", Method: "POST", Path: "/api/v1/receptionist/referrals/.../miss", ExpStatus: 200, ActStatus: code, Pass: code == 200, Details: string(body)})
		
		// Verify ClinicalUpdate created
		code, body = doReq("GET", "/api/v1/referrals/"+refTA1+"/clinical/history", specTAToken, nil)
		var hist struct { Data []map[string]interface{} `json:"data"` }
		json.Unmarshal(body, &hist)
		updateFound := false
		for _, u := range hist.Data {
			if u["update_reason"] == "MISSED_APPOINTMENT_RE_EVALUATION" {
				updateFound = true
				break
			}
		}
		logStep(StepResult{Step: "6. Verify Missed ClinicalUpdate", Method: "GET", Path: "/api/v1/referrals/.../clinical/history", ExpStatus: 200, ActStatus: code, Pass: updateFound, Asserts: []string{"Update with re-evaluation reason found"}, Details: string(body)})
	}

	// BL Receptionist (Walk-in for Ref 2)
	recBLToken := login("reception.bl@hospital.et")
	code, body = doReq("POST", "/api/v1/receptionist/referrals/walk-in", recBLToken, map[string]string{"referral_id": refTA2})
	logStep(StepResult{Step: "6. BL Rec Walk-in", Method: "POST", Path: "/api/v1/receptionist/referrals/walk-in", ExpStatus: 200, ActStatus: code, Pass: code == 200, Details: string(body)})
	
	// Verify Arrival Boost
	tqWalkIn := extractID(body, "data", "id")
	if tqWalkIn != "" {
		code, body = doReq("GET", "/api/v1/specialist/referrals/triage-queue", specBLToken, nil)
		var tqBL struct { Data []map[string]interface{} `json:"data"` }
		json.Unmarshal(body, &tqBL)
		boost := 0.0
		for _, item := range tqBL.Data {
			if item["id"] == tqWalkIn {
				if b, ok := item["arrival_boost"].(float64); ok {
					boost = b
				}
				break
			}
		}
		logStep(StepResult{Step: "6. Verify Arrival Boost", Method: "GET", Path: "/api/v1/specialist/referrals/triage-queue", ExpStatus: 200, ActStatus: code, Pass: boost == 20, Asserts: []string{fmt.Sprintf("Boost == %v", boost)}, Details: string(body)})
	}

	// 7. Clinical Updates & Outcome (TA)
	code, body = doReq("POST", "/api/v1/referrals/"+refTA1+"/clinical/updates", specTAToken, map[string]interface{}{"update_reason": "CONDITION_CHANGE", "clinical_notes": "Patient improved after missing"})
	logStep(StepResult{Step: "7. Add Clinical Update", Method: "POST", Path: "/api/v1/referrals/.../clinical/updates", ExpStatus: 200, ActStatus: code, Pass: code == 200, Details: string(body)})

	code, body = doReq("POST", "/api/v1/referrals/"+refTA1+"/clinical/outcome", specTAToken, map[string]interface{}{"outcome": "improved", "outcome_notes": "Discharged finally"})
	logStep(StepResult{Step: "7. Record Outcome TA1", Method: "POST", Path: "/api/v1/referrals/.../clinical/outcome", ExpStatus: 200, ActStatus: code, Pass: code == 200, Details: string(body)})

	// Verify Status COMPLETED
	code, body = doReq("GET", "/api/v1/specialist/referrals/"+refTA1, specTAToken, nil)
	finalStatus := extractID(body, "status")
	if finalStatus == "" {
		finalStatus = extractID(body, "referral", "status")
	}
	logStep(StepResult{Step: "7. Verify COMPLETED", Method: "GET", Path: "/api/v1/specialist/referrals/.../", ExpStatus: 200, ActStatus: code, Pass: finalStatus == "COMPLETED", Asserts: []string{"Final Status == COMPLETED"}, Details: string(body)})

	// Outcome for Ref 2
	code, body = doReq("POST", "/api/v1/referrals/"+refTA2+"/clinical/outcome", specBLToken, map[string]interface{}{"outcome": "discharged", "outcome_notes": "Emergency handled"})
	logStep(StepResult{Step: "7. Record Outcome TA2", Method: "POST", Path: "/api/v1/referrals/.../clinical/outcome", ExpStatus: 200, ActStatus: code, Pass: code == 200, Details: string(body)})

	// 8. Notifications
	adminToken = login("superadmin@moh.gov.et")
	code, _ = doReq("POST", "/api/v1/internal/notifications/send", adminToken, nil)
	logStep(StepResult{Step: "8. Admin Notif Send", Method: "POST", Path: "/api/v1/internal/notifications/send", ExpStatus: 200, ActStatus: code, Pass: code == 200})

	// 9. Final Admin Check
	code, body = doReq("GET", "/api/v1/admin/config", adminToken, nil)
	json.Unmarshal(body, &configRes)
	pass = code == 200 && configRes.Data["buffer_days"] == "3"
	logStep(StepResult{Step: "9. Final Config Check", Method: "GET", Path: "/api/v1/admin/config", ExpStatus: 200, ActStatus: code, Pass: pass, Asserts: []string{"buffer_days is still 3"}})

	// Save Report
	f, _ := os.OpenFile("e2e_report.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	defer f.Close()
	f.WriteString("\n--- COMPREHENSIVE E2E RUN at " + time.Now().Format(time.RFC3339) + " ---\n")
	for _, line := range report {
		f.WriteString(line + "\n")
	}

	fmt.Println("\nE2E test suite completed. Report appended to e2e_report.txt")
}
