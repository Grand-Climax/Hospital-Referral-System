package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const baseUrl = "http://localhost:8081"
const password = "password123"

type ApiResponse struct {
	Success bool            `json:"success"`
	Data    json.RawMessage `json:"data"`
	Message string          `json:"message"`
	Error   string          `json:"error"`
}

type LoginResponse struct {
	AccessToken string `json:"access_token"`
}

func getToken(email string) string {
	payload := map[string]string{
		"email":    email,
		"password": password,
	}
	body, _ := json.Marshal(payload)
	resp, err := http.Post(baseUrl+"/api/v1/auth/login", "application/json", bytes.NewBuffer(body))
	if err != nil {
		fmt.Printf("Login error for %s: %v\n", email, err)
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		fmt.Printf("Login failed for %s with status %d: %s\n", email, resp.StatusCode, string(b))
		return ""
	}

	var lr LoginResponse
	json.NewDecoder(resp.Body).Decode(&lr)
	return lr.AccessToken
}

func doRequest(method, path, token string, payload interface{}) (int, string) {
	var bodyReader io.Reader
	if payload != nil {
		b, _ := json.Marshal(payload)
		bodyReader = bytes.NewBuffer(b)
	}

	req, _ := http.NewRequest(method, baseUrl+path, bodyReader)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return 0, err.Error()
	}
	defer resp.Body.Close()

	b, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(b)
}

func main() {
	var status int
	var content string

	fmt.Println("Step 0: Health Check")
	hresp, err := http.Get(baseUrl + "/health")
	if err != nil {
		fmt.Printf("Health check error: %v\n", err)
		return
	}
	fmt.Printf("Status: %d\n", hresp.StatusCode)

	fmt.Println("\nStep 1: System Admin Config")
	adminToken := getToken("superadmin@moh.gov.et")
	status, content = doRequest("GET", "/api/v1/admin/config", adminToken, nil)
	fmt.Printf("GET /admin/config: %d\n", status)

	status, content = doRequest("PUT", "/api/v1/admin/config", adminToken, map[string]string{
		"buffer_days":  "3",
		"aging_factor": "1.5",
	})
	fmt.Printf("PUT /admin/config: %d\n", status)

	fmt.Println("\nStep 2: Doctor Create Referral")
	doctorToken := getToken("doc.primary@hospital.et")
	status, content = doRequest("GET", "/api/v1/reference/networked-hospitals", doctorToken, nil)
	fmt.Printf("GET /reference/networked-hospitals status: %d\n", status)
	var hospResp struct {
		Data []map[string]interface{} `json:"data"`
	}
	json.Unmarshal([]byte(content), &hospResp)
	hospitals := hospResp.Data
	
	if len(hospitals) == 0 {
		fmt.Println("CRITICAL: No networked hospitals found for doctor!")
		return
	}

	targetHosp := hospitals[0]
	targetHospId := targetHosp["id"].(string)
	targetHospName := targetHosp["name"].(string)
	fmt.Printf("Target Hospital: %s (%s)\n", targetHospName, targetHospId)

	// Determine user emails based on hospital
	specialistEmail := "specialist.cardio@hospital.et"
	deptHeadEmail := "head.cardio@hospital.et"
	receptionEmail := "reception.spec@hospital.et"
	hospAdminEmail := "admin.specialized@hospital.et"

	if targetHospName == "Adama General Hospital" {
		specialistEmail = "specialist.general@hospital.et"
		deptHeadEmail = "head.general@hospital.et"
		receptionEmail = "reception.general@hospital.et"
		hospAdminEmail = "admin.general@hospital.et"
	}

	status, content = doRequest("GET", "/api/v1/reference/hospitals/"+targetHospId+"/departments", doctorToken, nil)
	var deptResp struct {
		Data []map[string]interface{} `json:"data"`
	}
	json.Unmarshal([]byte(content), &deptResp)
	depts := deptResp.Data
	var targetDeptId string
	for _, d := range depts {
		if d["name"] == "Cardiology" {
			targetDeptId = d["id"].(string)
			break
		}
	}
	if targetDeptId == "" && len(depts) > 0 {
		targetDeptId = depts[0]["id"].(string)
		fmt.Printf("Fallback Target Dept: %s\n", depts[0]["name"])
	}

	patientReq := map[string]string{
		"first_name":    "John",
		"last_name":     "Doe",
		"middle_name":  "Middle",
		"national_id":   fmt.Sprintf("NID-%d", time.Now().UnixNano()),
		"phone_number":  fmt.Sprintf("+2519%d", time.Now().Unix()%100000000),
		"sex":           "male",
		"date_of_birth": "1990-01-01T00:00:00Z",
	}
	status, content = doRequest("POST", "/api/v1/patients", doctorToken, patientReq)
	fmt.Printf("POST /api/v1/patients status: %d\n", status)
	var patient map[string]interface{}
	if err := json.Unmarshal([]byte(content), &patient); err != nil {
		fmt.Printf("Unmarshal error: %v, content: %s\n", err, content)
	}
	patientIdRaw, ok := patient["id"]
	if !ok || patientIdRaw == nil {
		fmt.Printf("CRITICAL: Failed to create patient. Response: %s\n", content)
		return
	}
	patientId := patientIdRaw.(string)
	fmt.Printf("Created Patient ID: %s\n", patientId)

	status, content = doRequest("GET", "/api/v1/reference/liaisons", doctorToken, nil)
	fmt.Printf("GET /reference/liaisons status: %d\n", status)
	var liaisonResp struct {
		Data []map[string]interface{} `json:"data"`
	}
	json.Unmarshal([]byte(content), &liaisonResp)
	var liaisonId string
	if len(liaisonResp.Data) > 0 {
		liaisonId = liaisonResp.Data[0]["id"].(string)
	}

	referralReq := map[string]interface{}{
		"patient_id":                    patientId,
		"target_hospital_id":            targetHospId,
		"target_dept_id":                targetDeptId,
		"liaison_officer_id":           liaisonId,
		"priority":                      "URGENT",
		"status":                        "SUBMITTED",
		"clinical_summary":              "Chest pain",
		"patient_history":               "Smoker",
		"physical_examination_findings": "Stable",
		"reason_of_referral":            "Specialist care",
		"reason_for_referral_category":  "EMERGENCY",
		"condition_at_referral":         "Stable",
		"diagnoses": []map[string]interface{}{
			{
				"icd_code":            "I10",
				"is_primary":          true,
				"diagnosis_certainty": "SUSPECTED",
			},
		},
	}
	status, content = doRequest("POST", "/api/v1/doctor/referrals", doctorToken, referralReq)
	fmt.Printf("POST /api/v1/doctor/referrals status: %d\n", status)
	var referral map[string]interface{}
	json.Unmarshal([]byte(content), &referral)
	
	// ReferralCreationResponse might wrap referral in a "referral" field
	var refId string
	if r, ok := referral["referral"].(map[string]interface{}); ok {
		refId = r["id"].(string)
	} else if id, ok := referral["id"].(string); ok {
		refId = id
	}

	if refId == "" {
		fmt.Printf("CRITICAL: Failed to create referral. Response: %s\n", content)
		return
	}
	referralId := refId
	fmt.Printf("Created Referral ID: %s\n", referralId)

	status, _ = doRequest("PUT", "/api/v1/doctor/referrals/"+referralId+"/submit", doctorToken, nil)
	fmt.Printf("Submit Referral: %d\n", status)

	fmt.Println("\nStep 3: Liaison Review")
	liaisonToken := getToken("liaison@moh.gov.et")
	status, _ = doRequest("POST", "/api/v1/liaison/referrals/"+referralId+"/read", liaisonToken, nil)
	fmt.Printf("Liaison Read: %d\n", status)
	status, _ = doRequest("POST", "/api/v1/liaison/referrals/"+referralId+"/forward", liaisonToken, map[string]string{"comment": "Forwarding"})
	fmt.Printf("Liaison Forward: %d\n", status)

	fmt.Println("\nStep 4: Specialist Accept/Triage")
	specialistToken := getToken(specialistEmail)
	status, _ = doRequest("POST", "/api/v1/specialist/referrals/"+referralId+"/read", specialistToken, nil)
	fmt.Printf("Specialist Read: %d\n", status)
	status, _ = doRequest("POST", "/api/v1/specialist/referrals/"+referralId+"/accept", specialistToken, nil)
	fmt.Printf("Specialist Accept: %d\n", status)
	status, _ = doRequest("POST", "/api/v1/specialist/referrals/"+referralId+"/triage-severity", specialistToken, map[string]interface{}{
		"score":         85,
		"justification": "High priority",
	})
	fmt.Printf("Triage Severity: %d\n", status)

	fmt.Println("\nStep 5: Specialist Scheduling")
	appDate := time.Now().Add(48 * time.Hour).Format("2006-01-02")
	status, _ = doRequest("POST", "/api/v1/specialist/referrals/"+referralId+"/emergency-schedule", specialistToken, map[string]string{
		"appointment_date": appDate,
		"notes":            "Emergency",
		"justification":    "Critical condition",
	})
	fmt.Printf("Emergency Schedule: %d\n", status)

	status, content = doRequest("GET", "/api/v1/specialist/referrals/triage-queue", specialistToken, nil)
	var queueResp struct {
		Data []map[string]interface{} `json:"data"`
	}
	json.Unmarshal([]byte(content), &queueResp)
	var triageQueueId string
	for _, r := range queueResp.Data {
		if r["referral_id"] == referralId {
			triageQueueId = r["id"].(string) 
			break
		}
	}
	fmt.Printf("Triage Queue ID: %s\n", triageQueueId)

	fmt.Println("\nStep 6: Department Head Capacity")
	deptHeadToken := getToken(deptHeadEmail)
	status, content = doRequest("GET", "/api/v1/department-head/schedule", deptHeadToken, nil)
	fmt.Printf("GET /department-head/schedule: %d\n", status)

	overrideDate := time.Now().Add(120 * time.Hour).Format("2006-01-02")
	status, _ = doRequest("POST", "/api/v1/department-head/capacity/overrides", deptHeadToken, map[string]interface{}{
		"date":      overrideDate,
		"new_limit": 30,
		"reason":    "May Day",
	})
	fmt.Printf("POST /overrides: %d\n", status)

	fmt.Println("\nStep 7: Receptionist Arrival")
	receptionToken := getToken(receptionEmail)
	status, _ = doRequest("POST", "/api/v1/receptionist/"+triageQueueId+"/arrive", receptionToken, nil)
	fmt.Printf("Confirm Arrival: %d\n", status)

	status, content = doRequest("GET", "/api/v1/users", receptionToken, nil)
	var userResp struct {
		Data []map[string]interface{} `json:"data"`
	}
	json.Unmarshal([]byte(content), &userResp)
	var specUserId string
	for _, u := range userResp.Data {
		if u["email"] == specialistEmail {
			specUserId = u["id"].(string)
			break
		}
	}
	status, _ = doRequest("POST", "/api/v1/receptionist/"+triageQueueId+"/assign-doctor", receptionToken, map[string]string{
		"doctor_id": specUserId,
	})
	fmt.Printf("Assign Doctor: %d\n", status)

	fmt.Println("\nStep 8: Clinical Updates")
	status, _ = doRequest("POST", "/api/v1/referrals/"+referralId+"/clinical/updates", specialistToken, map[string]string{
		"update_reason":  "CONDITION_CHANGE",
		"clinical_notes": "Patient condition improved",
	})
	fmt.Printf("Add Clinical Update: %d\n", status)

	status, _ = doRequest("POST", "/api/v1/referrals/"+referralId+"/clinical/outcome", specialistToken, map[string]string{
		"outcome":       "improved",
		"outcome_notes": "Discharged with prescription",
	})
	fmt.Printf("Record Outcome: %d\n", status)

	fmt.Println("\nStep 9: Notifications")
	hospAdminToken := getToken(hospAdminEmail)
	status, _ = doRequest("POST", "/api/v1/internal/notifications/send", hospAdminToken, nil)
	fmt.Printf("Trigger Manual Send: %d\n", status)
	status, _ = doRequest("POST", "/api/v1/internal/notifications/update-status", hospAdminToken, nil)
	fmt.Printf("Update Status: %d\n", status)

	fmt.Println("\nE2E Test Completed!")
}
