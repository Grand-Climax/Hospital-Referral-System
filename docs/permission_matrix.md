# Hospital Referral Hub - Permission Matrix

This document maps the seven core user roles to their allowed system actions, based on the **ActionType** enum and the underlying table structures (such as `ReferralAccess`, `ClinicalUpdate`, `TriageQueue`, and `AuditLog`).

## Role Definitions Mapping to Database Enums

*   **Referring Doctor**: Maps to `REFERRING_DOCTOR`. Initiates the referral process at the sender hospital.
*   **Liaison Officer**: Maps to `LIAISON_OFFICER`. Coordinates patient transfers and logistical redirections.
*   **Specialist**: Maps to `RECEIVING_SPECIALIST`. Reviews, accepts, rejects, or redirects incoming referrals at the receiving hospital. Treats the patient.
*   **Receiving Hospital Admin**: Maps to `DEPT_HEAD` and `RECEPTIONIST`. Approves outgoing/incoming referrals requiring admin review, manages department capacity, and coordinates patient arrival.
*   **Ministry Official**: Maps to `MOH_ANALYST`. Monitors system-wide analytics, quality metrics, and performance dashboards.
*   **System Admin**: Maps to `SYSTEM_ADMIN`. Manages underlying infrastructure, users, hospitals, and system configurations.
*   **Patient**: No direct `UserRole` table entry, but interacts via secure channels (e.g., OTP links, SMS tracker) to view their own referral status and details.

---

## Allowed Actions Matrix

| Action / Capability | Referring Doctor | Liaison Officer | Specialist | Receiving Hospital Admin | Ministry Official | System Admin | Patient |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **Authentication** | | | | | | | |
| `LOGIN` & `LOGOUT` | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ❌ (OTP/Link) |
| **Referral Lifecycle** | | | | | | | |
| `CREATE_REFERRAL` | ✅ | ✅ (on behalf) | ❌ | ❌ | ❌ | ❌ | ❌ |
| `APPROVE_REFERRAL` | ❌ | ❌ | ❌ | ✅ (Dept Head) | ❌ | ❌ | ❌ |
| `ACCEPT_REFERRAL` | ❌ | ❌ | ✅ | ❌ | ❌ | ❌ | ❌ |
| `REJECT_REFERRAL` | ❌ | ❌ | ✅ | ❌ | ❌ | ❌ | ❌ |
| `REDIRECT_REFERRAL` | ❌ | ✅ | ✅ | ❌ | ❌ | ❌ | ❌ |
| **Clinical & Triage Management** | | | | | | | |
| `OVERRIDE_ML_SCORE` | ❌ | ❌ | ✅ | ❌ | ❌ | ❌ | ❌ |
| Add `ClinicalUpdate` | ✅ | ❌ | ✅ | ❌ | ❌ | ❌ | ❌ |
| Manage `TriageQueue` (Arrivals) | ❌ | ❌ | ❌ | ✅ (Receptionist) | ❌ | ❌ | ❌ |
| Manage `CapacityOverride` | ❌ | ❌ | ❌ | ✅ (Dept Head) | ❌ | ❌ | ❌ |
| Record `ReferralOutcome` | ❌ | ❌ | ✅ | ❌ | ❌ | ❌ | ❌ |
| **Data Access & Privacy** | | | | | | | |
| `VIEW_PATIENT_DATA` | ✅ (Own Patients)| ✅ (Sender Hosp)| ✅ (Assigned) | ✅ (Dept/Hospital) | ✅ (Anonymized) | ✅ (For Audit) | ✅ (Own Data) |
| Grant `ReferralAccess` | ✅ (To Consult)| ❌ | ✅ (To Consult)| ✅ (Receptionist) | ❌ | ❌ | ❌ |
| `EXPORT_DATA` | ❌ | ❌ | ❌ | ✅ (Dept Only) | ✅ (System-wide) | ✅ (System-wide)| ❌ |
| View `AuditLog` | ❌ | ❌ | ❌ | ❌ | ❌ | ✅ | ❌ |

---

## Detailed Action Breakdown

### 1. Referring Doctor (`REFERRING_DOCTOR`)
**Primary Duty:** Evaluate patients and initiate outbound referrals.
*   **Create:** Can generate `DRAFT` and submit referrals.
*   **Update:** Can add preliminary vitals and initial clinical summaries. Can append `ClinicalUpdate` notes if the patient's condition changes while waiting.
*   **Access:** Can only view data for patients they have actively referred or treat. Temporarily holds `TREATING_DOCTOR` access type.

### 2. Liaison Officer (`LIAISON_OFFICER`)
**Primary Duty:** Coordinate logistical transfers between sender and receiver locations.
*   **Manage:** Can draft referrals on behalf of doctors (with doctor sign-off).
*   **Routing:** Handles `REDIRECT_REFERRAL` if the intended hospital lacks capacity or transportation logistics change. Can track ambulances or manage emergency justifications.

### 3. Specialist (`RECEIVING_SPECIALIST`)
**Primary Duty:** Evaluate incoming referrals, provide clinical guidance, and treat arriving patients.
*   **Triage:** Can `ACCEPT_REFERRAL`, `REJECT_REFERRAL` (requires reason), or `REDIRECT_REFERRAL`.
*   **ML Interaction:** Empowered to `OVERRIDE_ML_SCORE` with justification if the AI prediction does not match clinical judgment.
*   **Clinical:** Can append `ClinicalUpdate` records, review full `ReferralForm`, and log the final `ReferralOutcome`.
*   **Access:** Granted `TREATING_DOCTOR` or `CONSULTED_DOCTOR` via the `ReferralAccess` table.

### 4. Receiving Hospital Admin (`DEPT_HEAD` / `RECEPTIONIST`)
**Primary Duty:** Manage hospital workflows, queuing, and administrative approvals.
*   **Approvals:** `DEPT_HEAD` triggers `APPROVE_REFERRAL` for cross-tier transfers (e.g., Primary to Specialized) requiring validation.
*   **Capacity:** `DEPT_HEAD` manages `CapacityOverride` and `HospitalDepartment` daily limits.
*   **Reception:** `RECEPTIONIST` updates `ArrivalStatus` on the `TriageQueue`, marks no-shows (`MISSED`), and assigns the treating `User` (generating a `ReferralAccess` record).

### 5. Ministry Official (`MOH_ANALYST`)
**Primary Duty:** Monitor national or regional healthcare trends and referal network efficiency.
*   **Analytics:** Can `EXPORT_DATA` across the entire system.
*   **Privacy:** Views patient data that has been stripped of direct identifiers (or securely hashed), focusing on `ICDCode` aggregates, `length_of_stay_days`, `wait_time_hours`, and overriding patterns.

### 6. System Admin (`SYSTEM_ADMIN`)
**Primary Duty:** IT operations, user assignment, system integrity, and security auditing.
*   **Management:** Handles updates to the `Hospital`, `Department`, and `User` tables (e.g., onboarding a new hospital or resetting MFA secrets).
*   **Auditing:** Exclusive sweeping access to view the immutable `AuditLog` table specifically mapping user actions against referrals.
*   **Exports:** Full database backup generation and raw SQL report querying.

### 7. Patient 
**Primary Duty:** Track their distinct transition between care facilities.
*   **Tracking:** Accesses their individual referral tracker via securely generated SMS links (`phone_number`).
*   **Visibility:** Can securely query their `ReferralStatus`, target `Hospital` destination, and `appointment_date` from the `TriageQueue`. Cannot access internal notes, ML scoring, or audit logs.
