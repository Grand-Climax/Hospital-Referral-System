# Frontend Integration Guide — Triage Queue (Role-Aware)

> **Audience:** Next.js frontend team integrating the new role-aware
> triage queue endpoints exposed by the backend.

> **Single most important contract change:**
> **All triage-detail endpoints now use `referral_id` in the path,
> for every role (specialist, receptionist, department head).**
> If anywhere in the FE you were planning to use `queue_id` in a URL
> (especially for the dept-head detail), switch it to `referral_id`.
> The list response still gives you both fields, but only
> `referral_id` is used as the URL path parameter.

---

## Table of contents

1. [Migration checklist](#1-migration-checklist-read-first)
2. [Endpoint matrix](#2-endpoint-matrix)
3. [Query parameters (list endpoints)](#3-query-parameters-shared-by-all-three-list-endpoints)
4. [List response shape](#4-list-response-shape-dtotriagelistenvelope)
5. [Detail response shapes per role](#5-detail-response-shapes-per-role)
6. [Role permission matrix](#6-role-permission-matrix)
7. [available_actions wiring → backend action endpoints](#7-available_actions-→-action-endpoints-wiring)
8. [UI/UX patterns](#8-uiux-patterns)
   - 8.1 [List page layout](#81-list-page-layout-all-roles)
   - 8.2 [Specialist detail panel](#82-specialist-detail-panel)
   - 8.3 [Receptionist detail panel](#83-receptionist-detail-panel)
   - 8.4 [Department Head detail panel](#84-department-head-detail-panel)
   - 8.5 [Visual treatment cookbook](#85-visual-treatment-cookbook-chips-pills-colors)
   - 8.6 [Empty, loading, and error states](#86-empty-loading-and-error-states)
   - 8.7 [Keyboard, accessibility, and i18n](#87-keyboard-accessibility-and-i18n)
9. [TanStack Query hooks](#9-tanstack-query-hooks)
10. [WebSocket invalidation](#10-websocket-invalidation)
11. [Sidebar navigation suggestion](#11-sidebar-navigation-suggestion)
12. [Postman / Insomnia reference card](#12-postman--insomnia-reference-card)
13. [Out of scope / open items](#13-out-of-scope--open-items)

---

## 1. Migration checklist (read first)

If your FE already had pre-release scaffolding for the dept-head detail
endpoint, here's the diff you need to apply:

- **URL change:** anywhere you build
  `'/api/v1/department-head/triage-queue/' + queueId`, replace with
  `'/api/v1/department-head/triage-queue/' + referralId`.
- **Component prop change:** `<TriageDetail role="department-head"
  id={queueId}>` → `<TriageDetail role="department-head" id={referralId}>`.
- **Cache key change:** any TanStack key like
  `['triage-detail', 'department-head', queueId]` → use `referralId`.
- **Server payload still includes both** `queue_id` and `referral_id`
  in every list row, so you can choose which one to render in the UI
  (e.g. an "Audit ID" tooltip can still show `queue_id`) — but only
  `referral_id` is used in URLs.

| What changed | Before                           | After                          |
| ------------ | -------------------------------- | ------------------------------ |
| Specialist detail URL    | `/specialist/referrals/{referral_id}/triage-detail` | unchanged |
| Receptionist detail URL  | `/receptionist/referrals/{referral_id}/triage-detail` | unchanged |
| Dept-head detail URL     | `/department-head/triage-queue/{queue_id}` (pre-release) | **`/department-head/triage-queue/{referral_id}`** |

Result: **one mental model — referral_id is the only id the FE puts
into URLs**.

---

## 2. Endpoint matrix

| # | Method | Path                                                       | Role                  | `{id}` in path |
| - | ------ | ---------------------------------------------------------- | --------------------- | -------------- |
| 1 | GET    | `/api/v1/specialist/referrals/triage-queue`                | RECEIVING_SPECIALIST  | —              |
| 2 | GET    | `/api/v1/specialist/referrals/{id}/triage-detail`          | RECEIVING_SPECIALIST  | **referral_id** |
| 3 | GET    | `/api/v1/receptionist/referrals/triage-queue`              | RECEPTIONIST          | —              |
| 4 | GET    | `/api/v1/receptionist/referrals/{id}/triage-detail`        | RECEPTIONIST          | **referral_id** |
| 5 | GET    | `/api/v1/department-head/triage-queue`                     | DEPT_HEAD             | —              |
| 6 | GET    | `/api/v1/department-head/triage-queue/{id}`                | DEPT_HEAD             | **referral_id** |

All list endpoints share the same filter/sort matrix (next section).
All detail endpoints share the same id semantics (referral_id).

---

## 3. Query parameters (shared by all three list endpoints)

| Param                 | Type   | Default            | Notes                                                                                            |
| --------------------- | ------ | ------------------ | ------------------------------------------------------------------------------------------------ |
| `limit`               | int    | 20                 | clamped to `1..100`                                                                              |
| `page`                | int    | 1                  | 1-based                                                                                          |
| `department_id`       | UUID   | —                  | **Ignored on dept-head** (forced to caller's scope)                                              |
| `arrival_status`      | csv    | —                  | `EXPECTED,ARRIVED,ADMITTED,MISSED`                                                               |
| `referral_status`     | csv    | —                  | `ACCEPTED,SCHEDULED` only — terminal statuses live behind `include_terminal`                     |
| `has_doctor_assigned` | bool   | —                  | filters by treating-doctor presence                                                              |
| `patient_id`          | UUID   | —                  | filters to one patient                                                                           |
| `national_id`         | string | —                  | sent in cleartext over HTTPS; HMAC-hashed server-side before SQL lookup                          |
| `sort_by`             | string | `composite_score`  | one of `composite_score`, `appointment_date`, `created_at`                                       |
| `sort_order`          | string | `desc`             | `asc` or `desc`                                                                                  |
| `include_terminal`    | bool   | `false`            | set true only for audit views; default excludes `COMPLETED/DECEASED/CANCELLED/REJECTED_*/REDIRECTED` |

**Forward-compat rules:** unknown enum values are silently dropped
rather than rejected; numeric overrides clamp instead of erroring; a
bad `sort_by` falls back to `composite_score`. This is intentional so
older FE builds keep working when the backend grows new values.

### Example URLs

```
# default specialist view
GET /api/v1/specialist/referrals/triage-queue

# "Waiting only" (no appointment yet)
GET /api/v1/specialist/referrals/triage-queue?referral_status=ACCEPTED&arrival_status=EXPECTED

# "Missed today, sort by score desc"
GET /api/v1/specialist/referrals/triage-queue?arrival_status=MISSED

# Receptionist: unassigned arrived patients (need a doctor)
GET /api/v1/receptionist/referrals/triage-queue?arrival_status=ARRIVED&has_doctor_assigned=false

# Audit: show every row including terminal referrals
GET /api/v1/department-head/triage-queue?include_terminal=true

# Lookup a specific patient by national ID
GET /api/v1/specialist/referrals/triage-queue?national_id=ETH-1234567
```

---

## 4. List response shape (`dto.TriageListEnvelope`)

```json
{
  "success": true,
  "data": [
    {
      "queue_id":             "1d8c4c1a-...",
      "referral_id":          "f9b18b00-...",
      "patient_id":           "9c33fa55-...",
      "patient_name":         "Hanan Tadesse",
      "composite_score":      78.4,
      "appointment_date":     "2026-05-26T10:00:00Z",
      "arrival_status":       "EXPECTED",
      "referral_status":      "SCHEDULED",
      "condition_at_referral":"critical",
      "department_id":        "...",
      "department_name":      "Cardiology",
      "has_doctor_assigned":  true,
      "assigned_doctor_id":   "...",
      "assigned_doctor_name": "Dr. Selamawit Bekele",
      "created_at":           "2026-05-22T08:00:00Z"
    }
  ],
  "total":   42,
  "page":    1,
  "limit":   20,
  "has_more": true
}
```

**Field semantics:**

- `referral_id` — use this in URLs (every detail endpoint).
- `queue_id` — purely advisory; show only if you need an audit identifier.
- `composite_score` — the same number the scheduling engine uses;
  render as a colored chip. Higher = more urgent.
- `appointment_date` — `null` while the row is still **waiting**.
- `condition_at_referral` — `stable | urgent | critical`.
- `arrival_status` — `EXPECTED | ARRIVED | ADMITTED | MISSED`.
- `referral_status` — `ACCEPTED | SCHEDULED` (excludes terminal by default).

---

## 5. Detail response shapes per role

All three detail endpoints accept a referral UUID and return a
`{ success: true, data: { … } }` envelope. The `data` block differs
per role; below is the full inventory.

### 5.1 Specialist — `TriageDetailSpecialistResponse`

```jsonc
{
  "success": true,
  "data": {
    "queue_id":              "1d8c4c1a-...",
    "referral_id":           "f9b18b00-...",
    "arrival_status":        "EXPECTED",
    "referral_status":       "ACCEPTED",
    "condition_at_referral": "critical",
    "composite_score":       82.5,
    "ml_severity_score":     0.73,
    "triage_status":         "AUTO_SCORED",
    "appointment_date":      null,
    "department_id":         "...",
    "department_name":       "Cardiology",
    "created_at":            "2026-05-22T08:00:00Z",

    "patient": {
      "id":           "9c33fa55-...",
      "full_name":    "Hanan Tadesse",
      "first_name":   "Hanan",
      "middle_name":  "",
      "last_name":    "Tadesse",
      "sex":          "female",
      "age_years":    34,
      "home_region":  "Oromia",
      "national_id":  "ETH-1234567",          // specialist + receptionist only
      "phone_number": "+251911223344"          // specialist + receptionist only
    },

    "vitals": {
      "recorded_at":      "2026-05-22T07:50:00Z",
      "systolic_bp":      150, "diastolic_bp": 95,
      "heart_rate":       102, "sp_o2": 94.0,
      "temperature":      37.6, "respiratory_rate": 22, "gcs_score": 15
    },

    "diagnoses": [
      { "icd_code": "I21.9", "description": "Acute MI, unspecified",
        "is_primary": true, "diagnosis_certainty": "SUSPECTED" }
    ],

    "clinical_summary":     "Chest pain on exertion, sweating, dyspnea.",
    "reason_of_referral":   "Suspected ACS, request cath lab evaluation.",
    "investigation_results":"ECG: ST-elevation in II/III/aVF. Trop-I 1.2 ng/mL.",

    "treating_doctor": {
      "user_id":     "u-1",
      "full_name":   "Dr. Selamawit Bekele",
      "email":       "selamawit@hosp.gov.et",
      "access_type": "TREATING_DOCTOR",
      "granted_at":  "2026-05-22T08:30:00Z",
      "granted_by":  "u-9"
    },

    "consulting_doctors": [
      {
        "user_id": "u-2", "full_name": "Dr. Yonas Bekele",
        "access_type": "CONSULTED_DOCTOR",
        "granted_at": "2026-05-23T10:00:00Z",
        "granted_by": "u-1",
        "revoked_at": null, "revoke_reason": ""
      }
    ],

    "referral_access_list": [ /* every grant ever made, active + revoked */ ],

    "arrival_history": [
      { "at": "2026-05-22T08:10:00Z", "event": "ACCEPT_REFERRAL",
        "description": "Referral accepted into triage",
        "actor_id": "u-9", "actor_name": "Dr. Aster Tesfaye" }
    ],

    "available_actions": {
      "schedule":           true,
      "emergency_schedule": true,
      "return_to_triage":   false,
      "mark_arrived":       false,
      "mark_missed":        false,
      "assign_doctor":      false,
      "revoke_doctor":      false
    }
  }
}
```

### 5.2 Receptionist — `TriageDetailReceptionistResponse`

```jsonc
{
  "success": true,
  "data": {
    "queue_id":         "...",
    "referral_id":      "...",
    "arrival_status":   "EXPECTED",
    "referral_status":  "SCHEDULED",
    "appointment_date": "2026-05-26T10:00:00Z",
    "department_id":    "...",
    "department_name":  "Cardiology",
    "patient":          { /* same shape, national_id + phone INCLUDED */ },
    "assigned_doctor":  { /* same shape as specialist treating_doctor, or null */ },
    "arrived_at":       null,
    "miss_reason":      "",
    "arrival_history":  [ /* same shape */ ],
    "available_actions": {
      "schedule":           false,
      "emergency_schedule": false,
      "return_to_triage":   false,
      "mark_arrived":       true,
      "mark_missed":        true,
      "assign_doctor":      false,
      "revoke_doctor":      false
    }
  }
}
```

Receptionists do **not** receive: `clinical_summary`, `vitals`,
`diagnoses`, `ml_severity_score`, `triage_status`, `consulting_doctors`,
`referral_access_list`.

### 5.3 Department Head — `TriageDetailDeptHeadResponse`

```jsonc
{
  "success": true,
  "data": {
    "queue_id":              "...",
    "referral_id":           "...",
    "arrival_status":        "EXPECTED",
    "referral_status":       "SCHEDULED",
    "condition_at_referral": "urgent",
    "composite_score":       61.2,
    "appointment_date":      "2026-05-26T10:00:00Z",
    "department_id":         "...",
    "department_name":       "Cardiology",
    "has_doctor_assigned":   true,
    "assigned_doctor":       { /* name + email, no access dates */ },
    "patient":               { /* NO national_id, NO phone */ },
    "arrival_history":       [ /* same shape */ ],
    "available_actions":     { /* all-false: dept-head is read-only */ },
    "created_at":            "2026-05-22T08:00:00Z"
  }
}
```

---

## 6. Role permission matrix

| Capability                      | Specialist | Receptionist | Dept Head |
| ------------------------------- | :--------: | :----------: | :-------: |
| List queue                      | ✅         | ✅           | ✅ (own dept only) |
| Filter by `department_id`       | ✅         | ✅           | ❌ (forced) |
| See patient name                | ✅         | ✅           | ✅        |
| See `national_id` + `phone`     | ✅         | ✅           | ❌        |
| See vitals / ICD / clinical text| ✅         | ❌           | ❌        |
| See `ml_severity_score`         | ✅         | ❌           | ❌        |
| See `referral_access_list`      | ✅         | ❌           | ❌        |
| Trigger `emergency_schedule`    | ✅         | ❌           | ❌        |
| Trigger `return_to_triage`      | ✅         | ✅           | ❌        |
| `mark_arrived` / `mark_missed`  | ❌         | ✅           | ❌        |
| `assign_doctor` / `revoke_doctor`| ❌        | ✅           | ❌        |

> **`available_actions` is the source of truth.** Even when the matrix
> above says ✅, disable the button when its action key is `false`. The
> server enforces the same guards in the action endpoints, so a
> wrongly-enabled button will 400 on click — and the spinner-then-toast
> experience is worse than a greyed-out button.

---

## 7. `available_actions` → action endpoints wiring

| `available_actions` key   | Action endpoint                                                          | Method | Roles allowed |
| ------------------------- | ------------------------------------------------------------------------ | ------ | ------------- |
| `schedule`                | `/api/v1/specialist/referrals/{id}/schedule`                              | POST   | Specialist |
| `emergency_schedule`      | `/api/v1/specialist/referrals/{id}/emergency-schedule`                    | POST   | Specialist |
| `return_to_triage`        | `/api/v1/specialist/referrals/{id}/return-to-triage` <br>or<br> `/api/v1/receptionist/referrals/{id}/return-to-triage` | POST | Specialist + Receptionist |
| `mark_arrived`            | `/api/v1/receptionist/referrals/{id}/arrive`                              | POST   | Receptionist |
| `mark_missed`             | `/api/v1/receptionist/referrals/{id}/miss`                                | POST   | Receptionist |
| `assign_doctor`           | `/api/v1/receptionist/referrals/{id}/assign-doctor`                       | POST   | Receptionist |
| `revoke_doctor`           | `/api/v1/receptionist/referrals/{id}/revoke-doctor`                       | POST   | Receptionist |

`{id}` is always `referral_id`. After a successful action, invalidate
both the list and detail queries (see §10).

---

## 8. UI/UX patterns

### 8.1 List page layout (all roles)

```
┌─ Page header ───────────────────────────────────────────────────────┐
│ Triage Queue                          Last refresh · 12s ago  [↻]   │
│ 42 patients · 8 waiting · 30 scheduled · 4 missed                   │
├─ Filter bar (sticky) ───────────────────────────────────────────────┤
│ [Department ▾] [chips: Expected · Arrived · Admitted · Missed]       │
│ [Status: All / Accepted / Scheduled] [Assigned: All / Yes / No]      │
│ [Sort by ▾]  [🔍 search national_id (debounce 300ms)]   [Audit ☐]   │
├─ Result list / virtualized table ───────────────────────────────────┤
│ Score |  Patient            | Dept       | Appt time  | Status | Dr │
│ 91.2  | Hanan Tadesse       | Cardiology | tomorrow ◷ | SCHED  | ✓ │
│ 77.0  | Yonas Bekele        | Cardiology | —          | ACPT   | – │
│ …                                                                     │
├─ Footer pagination ─────────────────────────────────────────────────┤
│ Showing 1–20 of 42        [Load more ▾]                              │
└─────────────────────────────────────────────────────────────────────┘
```

**Behavior notes**

- **Sticky filter bar** at top of viewport while scrolling list.
- **Composite score chip** is the dominant visual cue (color-coded —
  see §8.5).
- **Patient name** is the only place we render decrypted PII at the
  list level.
- **Click row** opens the detail in a right-side slide-over panel
  (75 % viewport on desktop, full-screen on mobile).
- **Selection key for tan-stack rows** must be `referral_id`, not
  `queue_id`, so the row-id matches what the detail call expects.
- **Skeleton rows** render while paginating (avoid layout shift).

### 8.2 Specialist detail panel

The specialist detail is the **richest** view. Use a tabbed layout
because there's a lot of content:

```
┌──────────────────────────────────────────────────────────────────────┐
│  ← back     Hanan Tadesse · F · 34yo · Oromia               [⋯ menu] │
│  composite 82.5 · ML 0.73 · ⚠ Critical · ACCEPTED                    │
│                                                                       │
│  Action bar:                                                          │
│  [ Schedule ]  [ Emergency Schedule ⚠ ]  [ Return to triage ⤺ ]      │
│                                                                       │
│  ┌─ Tabs ───────────────────────────────────────────────────────────┐│
│  │ Clinical  ·  Doctors  ·  Timeline                                ││
│  └──────────────────────────────────────────────────────────────────┘│
│                                                                       │
│  CLINICAL                                                             │
│  ┌─ Vitals (recorded 2h ago) ───────┐  ┌─ ML Severity ──────────────┐│
│  │ BP   150/95  ●  HR   102          │  │ ████████░░  0.73           ││
│  │ SpO₂  94 %   ●  Temp 37.6 °C      │  │ Tier: HIGH                 ││
│  │ RR    22     ●  GCS  15           │  │ Last predicted 12 min ago  ││
│  └───────────────────────────────────┘  └────────────────────────────┘│
│                                                                       │
│  Diagnoses                                                            │
│  ┌──────────────┬────────────────────────────────┬──────────────────┐│
│  │ Code         │ Description                    │ Certainty        ││
│  │ I21.9 ⭐      │ Acute MI, unspecified          │ SUSPECTED        ││
│  └──────────────┴────────────────────────────────┴──────────────────┘│
│                                                                       │
│  Clinical Summary                                                     │
│  ┌──────────────────────────────────────────────────────────────────┐│
│  │ Chest pain on exertion, sweating, dyspnea.                       ││
│  └──────────────────────────────────────────────────────────────────┘│
│  Reason of referral · Investigation results · same expandable cards. │
└──────────────────────────────────────────────────────────────────────┘
```

**DOCTORS tab**

```
Treating doctor
┌──────────────────────────────────────────────────────────────────────┐
│  ● Dr. Selamawit Bekele · selamawit@hosp.gov.et                      │
│    Granted 2 days ago by Dr. Aster Tesfaye                           │
└──────────────────────────────────────────────────────────────────────┘

Consulting doctors (2)
┌──────────────────────────────────────────────────────────────────────┐
│  ● Dr. Yonas Bekele · CONSULTED · active · granted yesterday          │
│  ○ Dr. Tigist Kassa  · CONSULTED · revoked · "patient stable"         │
└──────────────────────────────────────────────────────────────────────┘
```

Active grants in solid dot, revoked in hollow dot + strikethrough.

**TIMELINE tab**

Vertical stepper bound to `arrival_history`. Each event:

```
○ 22 May · 08:10  · ACCEPT_REFERRAL
    "Referral accepted into triage"
    — Dr. Aster Tesfaye

│
○ 22 May · 08:30  · ASSIGN_DOCTOR
    "Treating doctor assigned"
    — Dr. Aster Tesfaye

│
● 24 May · 14:00  · CONFIRM_ARRIVAL
    "Patient arrived and was checked in"
    — Liya (receptionist)
```

Use the icon palette in §8.5 (✅ for arrival, ⚠ for missed, etc.).

### 8.3 Receptionist detail panel

Receptionists need to act quickly at the desk. Single-pane layout, no
tabs, action buttons big and obvious:

```
┌──────────────────────────────────────────────────────────────────────┐
│  ← back     Hanan Tadesse · F · 34yo                                 │
│  Appt: Tue 26 May · 10:00  ·  Cardiology  ·  Status: EXPECTED        │
│                                                                       │
│  Big action buttons (single row, primary color):                     │
│  ┌──────────────┐ ┌──────────────┐ ┌──────────────┐                 │
│  │  ✅ Arrived  │ │  ⚠ Missed    │ │  ⤺ Return    │                 │
│  └──────────────┘ └──────────────┘ └──────────────┘                 │
│                                                                       │
│  Treating doctor (or assign)                                          │
│  ┌──────────────────────────────────────────────────────────────────┐│
│  │  Dr. Selamawit Bekele                       [ Revoke ]            ││
│  └──────────────────────────────────────────────────────────────────┘│
│  ── or, if unassigned ──                                              │
│  ┌──────────────────────────────────────────────────────────────────┐│
│  │  No doctor assigned          [ Assign doctor… ]                  ││
│  └──────────────────────────────────────────────────────────────────┘│
│                                                                       │
│  Contact                                                              │
│  📞 +251 911 22 33 44   ·   🆔 ETH-1234567                            │
│                                                                       │
│  Recent activity (last 5 events from arrival_history)                │
│  …                                                                    │
└──────────────────────────────────────────────────────────────────────┘
```

**Behavior notes**

- Tap `Arrived` → optimistic update of `arrival_status` to `ARRIVED`,
  POST `/receptionist/referrals/{id}/arrive`. On 400 from server,
  rollback and toast the error.
- `Assign doctor…` opens a modal that calls
  `/receptionist/doctors` for the dropdown source. After assignment,
  invalidate detail + list.
- `Return` button only renders when `available_actions.return_to_triage`
  is true (i.e. the row is `MISSED`).
- Receptionists should be able to **call** the phone number directly —
  wrap it in a `tel:` link.

### 8.4 Department Head detail panel

Read-only ops view. Compact card, no action buttons, designed to be
embedded in a side-by-side dashboard layout next to a capacity widget:

```
┌─── Triage detail ───────────────────────────────────────────────────┐
│  Hanan Tadesse  ·  F  ·  34yo  ·  Oromia                            │
│                                                                      │
│  Score 61.2 · Urgent · SCHEDULED                                    │
│  Appt: Tue 26 May 10:00  ·  Cardiology                              │
│  Doctor: Dr. Selamawit Bekele (assigned)                            │
│                                                                      │
│  ── Timeline (recent 5) ──                                          │
│  • Accept · 22 May 08:10 · Dr. Aster                                │
│  • Assign · 22 May 08:30 · Dr. Aster                                │
│  • Arrive · 24 May 14:00 · Liya                                     │
│                                                                      │
│  [Open full audit]  [Open in batch schedule]                        │
└─────────────────────────────────────────────────────────────────────┘
```

Dept-head detail page can show two of these cards side by side, or a
single one in a sticky right panel while the queue list scrolls on the
left.

### 8.5 Visual treatment cookbook (chips, pills, colors)

#### Composite score chip

| Score range  | Color (Tailwind) | Label  |
| ------------ | ---------------- | ------ |
| `>= 80`      | `bg-red-100 text-red-700 border-red-300`   | "Critical" |
| `60–79.99`   | `bg-orange-100 text-orange-700 border-orange-300` | "High" |
| `40–59.99`   | `bg-amber-100 text-amber-700 border-amber-300` | "Medium" |
| `< 40`       | `bg-emerald-100 text-emerald-700 border-emerald-300` | "Low" |

```tsx
function ScoreChip({ score }: { score: number }) {
  const cls =
    score >= 80
      ? "bg-red-100 text-red-700 border-red-300"
      : score >= 60
      ? "bg-orange-100 text-orange-700 border-orange-300"
      : score >= 40
      ? "bg-amber-100 text-amber-700 border-amber-300"
      : "bg-emerald-100 text-emerald-700 border-emerald-300";
  return (
    <span className={`inline-flex items-center gap-1 rounded-full border px-2 py-0.5 text-xs font-medium ${cls}`}>
      {score.toFixed(1)}
    </span>
  );
}
```

#### Condition pill (`condition_at_referral`)

| Value      | Color                                                                  |
| ---------- | ---------------------------------------------------------------------- |
| `critical` | `bg-red-600 text-white`                                                |
| `urgent`   | `bg-orange-500 text-white`                                             |
| `stable`   | `bg-emerald-500 text-white`                                            |

#### Arrival status badge

| Value      | Icon | Color                                |
| ---------- | ---- | ------------------------------------ |
| `EXPECTED` | ◷    | `bg-slate-100 text-slate-700`        |
| `ARRIVED`  | ✅   | `bg-emerald-100 text-emerald-700`    |
| `ADMITTED` | 🏥   | `bg-sky-100 text-sky-700`            |
| `MISSED`   | ⚠    | `bg-red-100 text-red-700`            |

#### Referral status badge

| Value       | Color                                |
| ----------- | ------------------------------------ |
| `ACCEPTED`  | `bg-indigo-100 text-indigo-700`      |
| `SCHEDULED` | `bg-violet-100 text-violet-700`      |

#### Timeline event icons

| `event`                  | Icon |
| ------------------------ | ---- |
| `ACCEPT_REFERRAL`        | 📥   |
| `APPROVE_REFERRAL`       | 👍   |
| `CONFIRM_ARRIVAL`        | ✅   |
| `MARK_MISSED`            | ⚠    |
| `ASSIGN_DOCTOR`          | 👨‍⚕️   |
| `UNASSIGN_DOCTOR`        | ⤴    |
| `MANUAL_EMERGENCY_SCHEDULE` | 🚨 |
| `BATCH_SCHEDULE_RUN`     | 📦   |
| `GRANT_CONSULT_ACCESS`   | 🔓   |
| `REVOKE_CONSULT_ACCESS`  | 🔒   |
| `RECORD_OUTCOME`         | 📑   |

### 8.6 Empty, loading, and error states

| State                      | Render                                                                            |
| -------------------------- | --------------------------------------------------------------------------------- |
| List loading (first page)  | 6 skeleton rows with shimmer (animate-pulse).                                     |
| List loading (next page)   | Inline spinner under "Load more" button, keep existing rows.                       |
| List empty + no filters    | Friendly "No patients on the triage queue right now — enjoy the moment ☕" card. |
| List empty + filters set   | "No results match your filters." + "[Clear filters]" button.                      |
| Detail 404                 | "This referral is not on the triage queue. It may have been completed or redirected." + "[Back to queue]". |
| Detail 401                 | Should redirect to login (your global axios/fetch interceptor's job).             |
| Detail 500                 | Error card + "[Retry]". Log to Sentry / observability.                            |
| Stale data (WS down)       | Small "⚠ Live updates paused" banner above the list. Manual refresh button.       |

### 8.7 Keyboard, accessibility, and i18n

- **Row keyboard nav** — `↑` / `↓` to move selection, `Enter` to open
  detail, `Esc` to close detail slide-over.
- **ARIA** — table is `role="table"`, each chip has `aria-label`
  (e.g. "Composite score: 82.5, critical"). Action buttons disabled
  via `aria-disabled` + actual `disabled` attr.
- **Color independence** — never rely on color alone for status; pair
  every chip with text and an icon.
- **i18n strings** — wrap every label in `t(...)` from your i18n
  provider. The backend sends raw enum values (`EXPECTED`, `MISSED`,
  etc.) — translate on the FE.
- **Tabular nums** — use `font-variant-numeric: tabular-nums` for the
  composite score column to avoid jitter.

---

## 9. TanStack Query hooks

### 9.1 Types

```ts
// src/lib/triage-types.ts
export type ArrivalStatus = "EXPECTED" | "ARRIVED" | "ADMITTED" | "MISSED";
export type ReferralStatus = "ACCEPTED" | "SCHEDULED";
export type Condition = "stable" | "urgent" | "critical";
export type SortBy = "composite_score" | "appointment_date" | "created_at";
export type SortOrder = "asc" | "desc";

export type TriageFilter = {
  limit?: number;
  departmentId?: string;
  arrivalStatus?: ArrivalStatus[];
  referralStatus?: ReferralStatus[];
  hasDoctorAssigned?: boolean;
  patientId?: string;
  nationalId?: string;
  sortBy?: SortBy;
  sortOrder?: SortOrder;
  includeTerminal?: boolean;
};

export type TriageListItem = {
  queue_id: string;
  referral_id: string;
  patient_id: string;
  patient_name: string;
  composite_score: number;
  appointment_date: string | null;
  arrival_status: ArrivalStatus;
  referral_status: ReferralStatus;
  condition_at_referral: Condition | "";
  department_id: string;
  department_name: string;
  has_doctor_assigned: boolean;
  assigned_doctor_id?: string;
  assigned_doctor_name?: string;
  created_at: string;
};

export type TriageListEnvelope = {
  success: boolean;
  data: TriageListItem[];
  total: number;
  page: number;
  limit: number;
  has_more: boolean;
};

export type Role = "specialist" | "receptionist" | "department-head";
```

### 9.2 List hook (infinite scroll)

```ts
// src/hooks/useTriageQueue.ts
import { useInfiniteQuery } from "@tanstack/react-query";
import type { Role, TriageFilter, TriageListEnvelope } from "@/lib/triage-types";

const basePath: Record<Role, string> = {
  specialist:        "/api/v1/specialist/referrals/triage-queue",
  receptionist:      "/api/v1/receptionist/referrals/triage-queue",
  "department-head": "/api/v1/department-head/triage-queue",
};

function buildSearchParams(f: TriageFilter, page: number): URLSearchParams {
  const p = new URLSearchParams();
  p.set("page", String(page));
  if (f.limit) p.set("limit", String(f.limit));
  if (f.departmentId) p.set("department_id", f.departmentId);
  if (f.arrivalStatus?.length) p.set("arrival_status", f.arrivalStatus.join(","));
  if (f.referralStatus?.length) p.set("referral_status", f.referralStatus.join(","));
  if (typeof f.hasDoctorAssigned === "boolean")
    p.set("has_doctor_assigned", String(f.hasDoctorAssigned));
  if (f.patientId) p.set("patient_id", f.patientId);
  if (f.nationalId) p.set("national_id", f.nationalId);
  if (f.sortBy) p.set("sort_by", f.sortBy);
  if (f.sortOrder) p.set("sort_order", f.sortOrder);
  if (f.includeTerminal) p.set("include_terminal", "true");
  return p;
}

export function useTriageQueue(role: Role, filter: TriageFilter = {}) {
  return useInfiniteQuery({
    queryKey: ["triage-queue", role, filter],
    initialPageParam: 1,
    queryFn: async ({ pageParam }) => {
      const url = `${basePath[role]}?${buildSearchParams(filter, pageParam).toString()}`;
      const res = await fetch(url, { credentials: "include" });
      if (!res.ok) throw new Error(`triage queue: ${res.status}`);
      return (await res.json()) as TriageListEnvelope;
    },
    getNextPageParam: (last) => (last.has_more ? last.page + 1 : undefined),
    staleTime: 30_000,
  });
}
```

### 9.3 Detail hook (referral_id everywhere)

```ts
// src/hooks/useTriageDetail.ts
import { useQuery } from "@tanstack/react-query";
import type { Role } from "@/lib/triage-types";

const detailPath: Record<Role, (refId: string) => string> = {
  specialist:        (id) => `/api/v1/specialist/referrals/${id}/triage-detail`,
  receptionist:      (id) => `/api/v1/receptionist/referrals/${id}/triage-detail`,
  "department-head": (id) => `/api/v1/department-head/triage-queue/${id}`,
};

export function useTriageDetail<T = unknown>(role: Role, referralId?: string) {
  return useQuery<T | null>({
    queryKey: ["triage-detail", role, referralId],
    enabled: !!referralId,
    queryFn: async () => {
      const res = await fetch(detailPath[role](referralId!), { credentials: "include" });
      if (res.status === 404) return null;
      if (!res.ok) throw new Error(`triage detail: ${res.status}`);
      return (await res.json()) as T;
    },
    staleTime: 15_000,
  });
}
```

### 9.4 Action mutations with optimistic UI

```ts
// src/hooks/useTriageActions.ts
import { useMutation, useQueryClient } from "@tanstack/react-query";

export function useMarkArrived() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (referralId: string) => {
      const res = await fetch(
        `/api/v1/receptionist/referrals/${referralId}/arrive`,
        { method: "POST", credentials: "include" }
      );
      if (!res.ok) throw new Error(await res.text());
      return res.json();
    },
    onMutate: async (referralId) => {
      // Optimistic: flip arrival_status in detail cache + matching list row.
      const detailKey = ["triage-detail", "receptionist", referralId];
      const previous = qc.getQueryData<any>(detailKey);
      qc.setQueryData(detailKey, (old: any) =>
        old ? { ...old, data: { ...old.data, arrival_status: "ARRIVED" } } : old
      );
      return { previous, detailKey };
    },
    onError: (_err, _id, ctx) => {
      if (ctx) qc.setQueryData(ctx.detailKey, ctx.previous);
    },
    onSettled: (_data, _err, referralId) => {
      qc.invalidateQueries({ queryKey: ["triage-queue"] });
      qc.invalidateQueries({ queryKey: ["triage-detail"], exact: false });
    },
  });
}

// Same pattern for: useMarkMissed, useReturnToTriage, useAssignDoctor,
// useEmergencySchedule. Each one POSTs to the URL in §7 and invalidates
// both ["triage-queue"] and ["triage-detail"].
```

---

## 10. WebSocket invalidation

Backend in-app notification events that affect the queue or detail:

| In-app event                | Invalidate                                                                  |
| --------------------------- | --------------------------------------------------------------------------- |
| `PATIENT_ARRIVED`           | `["triage-queue"]`, `["triage-detail", *, referral_id]`                    |
| `PATIENT_MISSED`            | same                                                                        |
| `RETURNED_TO_TRIAGE`        | same                                                                        |
| `EMERGENCY_SCHEDULE_USED`   | `["triage-queue"]`, `["triage-detail", *, referral_id]`                    |
| `BATCH_SCHEDULE_COMPLETED`  | `["triage-queue"]`                                                          |
| `DOCTOR_ASSIGNED`           | `["triage-detail", *, referral_id]`                                         |
| `DOCTOR_REVOKED`            | `["triage-detail", *, referral_id]`                                         |
| `DAILY_CAPACITY_UPDATED`    | `["triage-queue"]` (priority ordering may shift)                            |

```ts
ws.on("PATIENT_ARRIVED", ({ referral_id }) => {
  qc.invalidateQueries({ queryKey: ["triage-queue"] });
  qc.invalidateQueries({
    queryKey: ["triage-detail"],
    predicate: (q) => q.queryKey.includes(referral_id),
  });
});
```

If WS is unavailable, the `staleTime: 30_000` on the list query gives
you eventual consistency at the cost of up to 30 s lag.

---

## 11. Sidebar navigation suggestion

```
Triage
├─ Active queue       → /triage
├─ Missed             → /triage?arrival_status=MISSED
├─ Waiting only       → /triage?referral_status=ACCEPTED&arrival_status=EXPECTED
└─ Audit (everything) → /triage?include_terminal=true
```

For dept heads, add a "By condition" pivot driven by
`/department-head/triage-queue/buckets` (separate dashboard endpoint —
returns counts per condition tier).

---

## 12. Postman / Insomnia reference card

```http
# Specialist: list with full filter matrix
GET {{baseURL}}/api/v1/specialist/referrals/triage-queue
    ?arrival_status=EXPECTED,MISSED
    &referral_status=ACCEPTED,SCHEDULED
    &has_doctor_assigned=false
    &sort_by=composite_score&sort_order=desc
    &limit=20&page=1
Authorization: Bearer <jwt>
```

```http
# Specialist: detail (referral_id in path)
GET {{baseURL}}/api/v1/specialist/referrals/{referral_id}/triage-detail
Authorization: Bearer <jwt>
```

```http
# Receptionist: detail (referral_id in path)
GET {{baseURL}}/api/v1/receptionist/referrals/{referral_id}/triage-detail
Authorization: Bearer <jwt>
```

```http
# Department Head: detail (referral_id in path — NEW: was queue_id)
GET {{baseURL}}/api/v1/department-head/triage-queue/{referral_id}
Authorization: Bearer <jwt>
```

```http
# Receptionist: mark arrived
POST {{baseURL}}/api/v1/receptionist/referrals/{referral_id}/arrive
Authorization: Bearer <jwt>
```

---

## 13. Out of scope / open items

- **`mark-admitted` endpoint** — the `ADMITTED` enum is filterable but
  no action endpoint exists yet. Filter chip will return zero results
  until that's wired.
- **Audit snapshot on terminal transitions** — `MarkDeceased`,
  `RejectAfterSend`, `RedirectReferral` currently hard-delete the
  triage row. `include_terminal=true` only surfaces rows that *stayed*
  on the queue. A future migration could keep a tombstone.
- **Legacy receptionist endpoints** (`/upcoming`, `/missed`,
  `/offline-data`) are unchanged. Plan to migrate to the new
  `/triage-queue` model in a follow-up release once the new UI is
  fully cut over.
- **`queue_id` in URLs** — never. Always `referral_id`. The `queue_id`
  field is kept in the list response purely as an audit identifier.
