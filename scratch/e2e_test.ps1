# E2E Integration Test for Hospital Referral Hub API

$baseUrl = "http://localhost:8081"
$password = "password123"

function Get-Token($email) {
    $loginUrl = "$baseUrl/api/v1/auth/login"
    $body = "{`"email`":`"$email`",`"password`":`"$password`"}"
    
    $response = curl.exe -s -w "\n%{http_code}" -X POST $loginUrl -H "Content-Type: application/json" -d $body
    $lines = $response -split "\n"
    $statusCode = $lines[-1].Trim()
    $content = ($lines[0..($lines.Length - 2)] -join "\n").Trim()
    
    if ($statusCode -ne 200) {
        Write-Host "Error: Login failed for $email with status $statusCode. Response: $content" -ForegroundColor Red
        return $null
    }
    
    $json = $content | ConvertFrom-Json
    return $json.access_token
}

function Test-Endpoint($method, $path, $token, $body = $null) {
    $url = "$baseUrl$path"
    $headers = @("-H", "Authorization: Bearer $token", "-H", "Content-Type: application/json")
    
    if ($null -eq $body) {
        $resp = curl.exe -s -w "\n%{http_code}" -X $method $url $headers
    } else {
        $jsonBody = $body | ConvertTo-Json -Depth 10
        $resp = curl.exe -s -w "\n%{http_code}" -X $method $url $headers -d $jsonBody
    }
    
    # Split output by last newline to get status code
    $lines = $resp -split "\n"
    $statusCode = $lines[-1].Trim()
    $content = ($lines[0..($lines.Length - 2)] -join "\n").Trim()
    
    if ($statusCode -ge 400) {
        Write-Host "Warning: $method $path returned $statusCode. Response: $content" -ForegroundColor Yellow
    }
    
    return @{
        status = [int]$statusCode
        content = $content
    }
}

# --- 0. Health Check ---
Write-Host "Step 0: Health Check"
$health = curl.exe -s -w "%{http_code}" -X GET "$baseUrl/health"
Write-Host "Status: $($health.Substring($health.Length-3))"

# --- 1. System Admin ---
Write-Host "`nStep 1: System Admin Config"
$adminToken = Get-Token "superadmin@moh.gov.et"
$config = Test-Endpoint "GET" "/api/v1/admin/config" $adminToken
Write-Host "GET /admin/config: $($config.status)"

$updateConfig = Test-Endpoint "PUT" "/api/v1/admin/config" $adminToken @{
    buffer_days = "3"
    aging_factor = "1.5"
}
Write-Host "PUT /admin/config: $($updateConfig.status)"

# --- 2. Doctor ---
Write-Host "`nStep 2: Doctor Create Referral"
$doctorToken = Get-Token "doc.primary@hospital.et"
$stats = Test-Endpoint "GET" "/api/v1/doctor/stats" $doctorToken
Write-Host "GET /doctor/stats: $($stats.status)"

# Need a hospital and department ID
$hospitals = Test-Endpoint "GET" "/api/v1/reference/hospitals" $doctorToken
$hospJson = $hospitals.content | ConvertFrom-Json
# Find Jimma University Medical Center (Specialized)
$targetHosp = $hospJson | Where-Object { $_.name -like "*Specialized*" -or $_.name -like "*University*" } | Select-Object -First 1
$targetHospId = $targetHosp.id
Write-Host "Target Hospital: $($targetHosp.name) ($targetHospId)"

# Find Cardiology department
$depts = Test-Endpoint "GET" "/api/v1/reference/hospitals/$targetHospId/departments" $doctorToken
$deptJson = $depts.content | ConvertFrom-Json
$targetDept = $deptJson | Where-Object { $_.name -eq "Cardiology" } | Select-Object -First 1
$targetDeptId = $targetDept.id
Write-Host "Target Department: $($targetDept.name) ($targetDeptId)"

# Need a patient
$patientReq = @{
    first_name = "John"
    last_name = "Doe"
    middle_name = "Middle"
    national_id = "NID-" + (Get-Random)
    phone = "0911223344"
    gender = "MALE"
    date_of_birth = "1990-01-01"
}
$patient = Test-Endpoint "POST" "/api/v1/patients" $doctorToken $patientReq
$patientId = ($patient.content | ConvertFrom-Json).id
Write-Host "Created Patient ID: $patientId"

$referralReq = @{
    patient_id = $patientId
    receiver_hospital_id = $targetHospId
    target_department_id = $targetDeptId
    priority = "URGENT"
    referral_type = "EMERGENCY"
    clinical_summary = "Chest pain"
    patient_history = "Smoker"
    physical_examination_findings = "Stable"
    reason_of_referral = "Specialist care"
    reason_for_referral_category = "Evaluation"
    condition_at_referral = "Stable"
}
$referral = Test-Endpoint "POST" "/api/v1/doctor/referrals" $doctorToken $referralReq
$referralId = ($referral.content | ConvertFrom-Json).id
Write-Host "Created Referral ID: $referralId, Status: $($referral.status)"

$submit = Test-Endpoint "PUT" "/api/v1/doctor/referrals/$referralId/submit" $doctorToken
Write-Host "Submit Referral: $($submit.status)"

# --- 3. Liaison ---
Write-Host "`nStep 3: Liaison Review"
$liaisonToken = Get-Token "liaison@moh.gov.et"
$liaisonRead = Test-Endpoint "POST" "/api/v1/liaison/referrals/$referralId/read" $liaisonToken
Write-Host "Liaison Read: $($liaisonRead.status)"

$liaisonForward = Test-Endpoint "POST" "/api/v1/liaison/referrals/$referralId/forward" $liaisonToken @{
    comment = "Forwarding to specialist"
}
Write-Host "Liaison Forward: $($liaisonForward.status)"

# --- 4. Specialist ---
Write-Host "`nStep 4: Specialist Accept/Triage"
$specialistToken = Get-Token "specialist.cardio@hospital.et"
$specRead = Test-Endpoint "POST" "/api/v1/specialist/referrals/$referralId/read" $specialistToken
Write-Host "Specialist Read: $($specRead.status)"

$specAccept = Test-Endpoint "POST" "/api/v1/specialist/referrals/$referralId/accept" $specialistToken
Write-Host "Specialist Accept: $($specAccept.status)"

$triageSeverity = Test-Endpoint "POST" "/api/v1/specialist/referrals/$referralId/triage-severity" $specialistToken @{
    score = 85
    justification = "High priority case"
}
Write-Host "Triage Severity: $($triageSeverity.status)"

# --- 5. Specialist Scheduling ---
Write-Host "`nStep 5: Specialist Scheduling"
$appDate = (Get-Date).AddDays(2).ToString("yyyy-MM-dd")
$schedulingReq = @{
    appointment_date = $appDate
    notes = "Emergency slot"
}
$schedule = Test-Endpoint "POST" "/api/v1/specialist/referrals/$referralId/emergency-schedule" $specialistToken $schedulingReq
Write-Host "Emergency Schedule: $($schedule.status)"

# Get Triage Queue ID
$queue = Test-Endpoint "GET" "/api/v1/specialist/referrals/triage-queue" $specialistToken
$queueJson = $queue.content | ConvertFrom-Json
# referrals is an array of objects
$triageQueueId = ($queueJson.referrals | Where-Object { $_.referral_id -eq $referralId })[0].id
Write-Host "Triage Queue ID: $triageQueueId"

# --- 6. Department Head ---
Write-Host "`nStep 6: Department Head Capacity"
$deptHeadToken = Get-Token "head.cardio@hospital.et"
$deptSchedule = Test-Endpoint "GET" "/api/v1/department-head/schedule" $deptHeadToken
Write-Host "GET /department-head/schedule: $($deptSchedule.status)"
$scheduleId = ($deptSchedule.content | ConvertFrom-Json).data[0].id

$overrideDate = (Get-Date).AddDays(5).ToString("yyyy-MM-dd")
$override = Test-Endpoint "POST" "/api/v1/department-head/capacity/overrides" $deptHeadToken @{
    date = $overrideDate
    new_limit = 30
    reason = "May Day"
}
$overrideId = ($override.content | ConvertFrom-Json).data.id
Write-Host "POST /overrides: $($override.status)"

# --- 7. Receptionist ---
Write-Host "`nStep 7: Receptionist Arrival"
$receptionToken = Get-Token "reception.spec@hospital.et"
$arrive = Test-Endpoint "POST" "/api/v1/receptionist/referrals/$triageQueueId/arrive" $receptionToken
Write-Host "Confirm Arrival: $($arrive.status)"

# Get Specialist User ID
$users = Test-Endpoint "GET" "/api/v1/users" $receptionToken
$specUserId = ($users.content | ConvertFrom-Json).data | Where-Object { $_.email -eq "specialist.cardio@hospital.et" } | Select-Object -ExpandProperty id

$assign = Test-Endpoint "POST" "/api/v1/receptionist/referrals/$triageQueueId/assign-doctor" $receptionToken @{
    doctor_id = $specUserId
}
Write-Host "Assign Doctor: $($assign.status)"

# --- 8. Clinical ---
Write-Host "`nStep 8: Clinical Updates"
$clinicalUpdate = Test-Endpoint "POST" "/api/v1/referrals/$referralId/clinical/updates" $specialistToken @{
    update_reason = "CONDITION_CHANGE"
    clinical_notes = "Patient condition improved"
}
Write-Host "Add Clinical Update: $($clinicalUpdate.status)"

$outcome = Test-Endpoint "POST" "/api/v1/referrals/$referralId/clinical/outcome" $specialistToken @{
    outcome = "improved"
    outcome_notes = "Discharged with prescription"
}
Write-Host "Record Outcome: $($outcome.status)"

# --- 9. Notifications ---
Write-Host "`nStep 9: Notifications"
$hospAdminToken = Get-Token "admin.specialized@hospital.et"
$sendNotif = Test-Endpoint "POST" "/api/v1/internal/notifications/send" $hospAdminToken
Write-Host "Trigger Manual Send: $($sendNotif.status)"

$updateNotif = Test-Endpoint "POST" "/api/v1/internal/notifications/update-status" $hospAdminToken
Write-Host "Update Status: $($updateNotif.status)"

Write-Host "`nE2E Test Completed!"
