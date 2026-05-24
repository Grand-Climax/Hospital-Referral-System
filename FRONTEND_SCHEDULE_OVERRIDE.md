# Specialist – Schedule Override (Triage Detail)

A focused integration guide for the **Schedule Override** action that lives inside
the Specialist Triage Detail page. Covers the two backend endpoints, the request
/ response contracts, when to surface what, the UI/UX patterns, and every edge
case the backend already enforces (so the FE never has to second-guess).

> Audience: Next.js frontend (App Router) using TanStack Query + Tailwind.

---

## ⛔ Hard rule first — when the override (and routine schedule) MUST be hidden

The two scheduling buttons must **not** be rendered when the patient has
physically reached the hospital. The triage row's `arrival_status` is the only
source of truth — use these **exact** entity values (see
`internal/domain/entity/triage_queue.go`):

| `arrival_status` (entity name) | Wire value | Show schedule? | Show override? | UI hint |
|---|---|---|---|---|
| `ArrivalExpected` | `"EXPECTED"` | ✅ | ✅ | normal flow |
| `ArrivalMissed` | `"MISSED"` | ✅ (rescue) | ✅ (rescue) | label both buttons as *"Rescue & reschedule"* |
| `ArrivalArrived` | `"ARRIVED"` | ❌ | ❌ | patient is on-site — replace the action rail with a *"Patient already arrived"* info card and link to the receptionist flow |
| `ArrivalAdmitted` | `"ADMITTED"` | ❌ | ❌ | same as ARRIVED — visit is in progress / finished |

> ⚠️ The FE does NOT compute this itself — the triage detail response already
> ships `available_actions.schedule` and `available_actions.emergency_schedule`
> as booleans that follow the rule above. **Use the flags, do not re-derive.**

The backend enforces the same rule defensively: a POST to either endpoint when
`arrival_status ∈ {ARRIVED, ADMITTED}` returns
`400 cannot schedule appointment: patient has already arrived or been admitted`.
So even if the FE misses, the server won't.

---

## 1. What "Schedule Override" actually means

There are **two** scheduling endpoints, and the FE must show them as **separate
buttons with different visual weight**:

| Action | Endpoint | When it's allowed | Capacity rule |
|---|---|---|---|
| **Schedule** (routine) | `POST /api/v1/specialist/referrals/{referralId}/schedule` | always when `referral_status ∈ {ACCEPTED, SCHEDULED}` and `arrival_status ∈ {EXPECTED, MISSED}` | rejects when `booked >= max_slots` (does NOT touch overbook) |
| **Schedule Override** (emergency) | `POST /api/v1/specialist/referrals/{referralId}/emergency-schedule` | same eligibility | rejects only when `booked >= max_slots + overbook_limit` (the only path that consumes overbook) |

Both endpoints handle **three scenarios** identically:

1. **First booking** – `ACCEPTED` → `SCHEDULED`, sets `appointment_date`.
2. **Rescheduling a SCHEDULED patient** – overwrites `appointment_date`; the
   backend now also refreshes the **old date's** `daily_schedules.booked_slots`
   snapshot (this was previously stale; fixed).
3. **Rescuing a MISSED patient** – flips `arrival_status: MISSED → EXPECTED`,
   sets the new `appointment_date`, keeps `referral_status = SCHEDULED`.

In all three cases the backend:
- writes / updates the `daily_schedules` row for the chosen date
  (creates it on first booking of the day, increments `booked_slots` after),
- writes an audit log (`MANUAL_EMERGENCY_SCHEDULE` for override, `OVERRIDE_QUEUE`
  for routine — yes, that label is legacy),
- queues an SMS + an in-app event (`APPOINTMENT_SCHEDULED` or
  `MISSED_APPOINTMENT_RESCHEDULED`),
- for override, also emits `EMERGENCY_SCHEDULE_USED` so the dept head dashboard
  badge updates in real time.

---

## 2. Request / Response DTOs

### 2.1 Routine schedule

```ts
// POST /api/v1/specialist/referrals/{referralId}/schedule
type SchedulingRequest = {
  appointment_date: string; // RFC3339, e.g. "2026-05-30T00:00:00Z"
  notes?: string;
  override?: boolean;       // ignored by the backend on this endpoint - kept for compat
};

type SchedulingResponse = {
  success: true;
  message: string;
  rescheduled_from_missed: boolean; // true when the patient was MISSED before this call
};
```

### 2.2 Schedule Override (emergency)

```ts
// POST /api/v1/specialist/referrals/{referralId}/emergency-schedule
type ManualEmergencyScheduleRequest = {
  appointment_date: string; // "YYYY-MM-DD" - NOT a full RFC3339 timestamp
  justification: string;    // required UNLESS condition_at_referral is "critical"
};

type SchedulingResponse = {
  success: true;
  message: string;
  rescheduled_from_missed: boolean;
};
```

> **Date format gotcha.** Routine uses `time.Time` (ISO 8601 / RFC3339),
> emergency uses `"YYYY-MM-DD"`. Don't mix them up. We picked the simpler date
> string for override because dept heads sanity-check it as a calendar day.

### 2.3 Standard error envelope

```ts
type ErrorResponse = { success: false; error: string };
```

Status codes:

| HTTP | When | What FE should show |
|---|---|---|
| 400 | invalid id / invalid date / past date / missing justification on non-critical / patient already ARRIVED or ADMITTED | inline form error, do NOT close the modal |
| 401 | session expired | redirect to login |
| 500 | "capacity reached for this date - emergency override required" (routine) | swap CTA to **Schedule Override** with a tooltip explaining why |
| 500 | "even overbook capacity is full for this date" (override) | toast: "No room even with overbook on this date — pick another date" |
| 500 | DB / unexpected | generic toast + Sentry breadcrumb |

> Yes, the capacity-exhausted error currently returns 500 with a human-readable
> message. The FE can string-match on `"capacity reached"` and `"overbook"` to
> disambiguate without a server change.

---

## 3. When each button is enabled

The triage detail (specialist view) response already gives you the precomputed
flags – use them, don't recompute on the FE:

```ts
type TriageDetailSpecialistAvailableActions = {
  schedule: boolean;
  emergency_schedule: boolean;
  return_to_triage: boolean;
};
```

Mapping:

| flag = true when | Show |
|---|---|
| `schedule` | `referral_status ∈ {ACCEPTED, SCHEDULED}` AND `arrival_status ∉ {ARRIVED, ADMITTED}` |
| `emergency_schedule` | same as above (override is allowed everywhere routine is) |
| `return_to_triage` | `arrival_status == MISSED` |

Rendering rule:

```tsx
{actions.schedule && <ScheduleButton ... />}
{actions.emergency_schedule && <ScheduleOverrideButton ... />}
{actions.return_to_triage && <ReturnToTriageButton ... />}
```

If neither `schedule` nor `emergency_schedule` is true (patient already ARRIVED
or ADMITTED), show a **muted info card** instead of the buttons:

> "Patient has already arrived. Use the reception flow to manage their visit."

---

## 4. UI/UX recipe – Schedule Override section

The override action sits in the **right-side action rail** of the Triage Detail
page, *below* the routine **Schedule** button. The visual hierarchy must signal
that this is a power-user action, not the default.

### 4.1 Section layout

```
┌─────────────────────────────────────────────────────────┐
│ Schedule                                                │
│ ──────────────────────────────────────────────────────  │
│  [ Schedule patient ▸ ]   ← primary, indigo            │
│                                                         │
│  ╔═══════════════════════════════════════════════════╗ │
│  ║ ⚠ Schedule Override                               ║ │
│  ║                                                   ║ │
│  ║ Only use this when routine scheduling is blocked  ║ │
│  ║ by capacity AND the patient cannot wait. This is  ║ │
│  ║ the only path that uses the department's overbook ║ │
│  ║ buffer ({overbook_limit} extra slots/day).        ║ │
│  ║                                                   ║ │
│  ║  [ Use override ▸ ]   ← amber, outline           ║ │
│  ╚═══════════════════════════════════════════════════╝ │
└─────────────────────────────────────────────────────────┘
```

Tailwind sketch:

```tsx
<section className="rounded-lg border border-gray-200 p-4 space-y-4">
  <h3 className="text-sm font-semibold text-gray-700">Schedule</h3>

  <Button onClick={openScheduleModal} variant="primary" disabled={!actions.schedule}>
    Schedule patient
  </Button>

  {actions.emergency_schedule && (
    <div className="rounded-md border border-amber-300 bg-amber-50 p-3 space-y-2">
      <div className="flex items-center gap-2">
        <AlertTriangleIcon className="h-4 w-4 text-amber-600" />
        <span className="text-sm font-semibold text-amber-900">
          Schedule Override
        </span>
      </div>
      <p className="text-xs text-amber-800 leading-relaxed">
        Only use when routine scheduling is blocked by capacity <em>and</em>
        the patient cannot wait. Consumes the department's overbook buffer
        ({overbookLimit} extra slot{overbookLimit === 1 ? "" : "s"} per day).
        Every use is logged and the department head is notified.
      </p>
      <Button onClick={openOverrideModal} variant="warning-outline" size="sm">
        Use override
      </Button>
    </div>
  )}
</section>
```

### 4.2 Override modal

```
┌── Schedule Override ────────────────────────────[x]──┐
│                                                       │
│  Patient: Almaz Tesfaye                               │
│  Condition at referral: critical  ← chip, red         │
│  Current state: SCHEDULED · expected · 2026-05-30     │
│                                                       │
│  Appointment date *                                   │
│  [ 📅 2026-05-25 ▾ ]                                  │
│                                                       │
│  Capacity on 2026-05-25:                              │
│  ███████████████████░░░  19 / 20 (overbook: 4 free)   │
│                                                       │
│  Reason *                                             │
│  ○ Patient cannot wait (clinical urgency)             │
│  ● Rescue a missed appointment                        │
│  ○ Reschedule at patient request                      │
│  ○ Other (explain below)                              │
│                                                       │
│  Justification * (≥ 10 chars; required unless         │
│                   condition_at_referral == "critical")│
│  ┌─────────────────────────────────────────────────┐  │
│  │ Patient missed Mon appointment after kidney     │  │
│  │ episode; clinical team requested earlier slot.  │  │
│  └─────────────────────────────────────────────────┘  │
│                                                       │
│  ⚠ This will consume an overbook slot. The            │
│    department head will be notified.                  │
│                                                       │
│              [ Cancel ]   [ Confirm override ▸ ]      │
└───────────────────────────────────────────────────────┘
```

Rules baked into the modal:

- **Reason selector** is FE-only — concat to `justification` before POST:
  ```ts
  const justification = `${reasonLabel}: ${freeText.trim()}`;
  ```
- **Skip justification requirement** when the referral's `condition_at_referral`
  (already in the detail response) is `"critical"` (case-insensitive). Show a
  pill: *"Justification optional — patient is critical"*.
- **Past-date guard on the date input**: disable today's previous days.
- **Live capacity readout** uses
  `GET /api/v1/specialist/referrals/{referralId}/schedule-options?days=14`
  so the user sees real numbers before they pick.

### 4.3 Confirm + post-success

After a successful 200:

- close modal,
- invalidate `["triage", "detail", referralId]`,
- invalidate `["triage", "list", ...]` (any filter),
- invalidate `["schedule-options", referralId]`,
- toast: `Scheduled for {date}` or `Rescued missed appointment → {date}`
  depending on `rescheduled_from_missed`.

### 4.4 Inline state machine the modal must respect

```
referral_status     arrival_status     button label                       success toast
─────────────────── ────────────────── ────────────────────────────────── ──────────────────────────────────
ACCEPTED            EXPECTED           "Schedule emergency"               "Scheduled for {date}"
SCHEDULED           EXPECTED           "Reschedule (override)"            "Rescheduled to {date}"
SCHEDULED           MISSED             "Rescue & reschedule (override)"   "Missed appointment rescheduled to {date}"
ACCEPTED            MISSED             "Rescue & reschedule (override)"   "Missed appointment rescheduled to {date}"
ACCEPTED|SCHEDULED  ARRIVED|ADMITTED   button hidden                      —
```

---

## 5. TanStack Query hooks

```ts
// hooks/useEmergencySchedule.ts
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/lib/api";

type Vars = {
  referralId: string;
  appointment_date: string;  // YYYY-MM-DD
  justification: string;     // already includes reason prefix
};

export function useEmergencySchedule() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (vars: Vars) => {
      const { data } = await api.post(
        `/specialist/referrals/${vars.referralId}/emergency-schedule`,
        { appointment_date: vars.appointment_date, justification: vars.justification }
      );
      return data as { success: true; message: string; rescheduled_from_missed: boolean };
    },
    onSuccess: (_, vars) => {
      qc.invalidateQueries({ queryKey: ["triage", "detail", vars.referralId] });
      qc.invalidateQueries({ queryKey: ["triage", "list"] });
      qc.invalidateQueries({ queryKey: ["schedule-options", vars.referralId] });
    },
  });
}
```

```ts
// hooks/useRoutineSchedule.ts - same shape but the body is { appointment_date: ISO, notes? }
// and the URL ends in /schedule (no -override).
```

For the **capacity preview** inside the modal, reuse the existing endpoint:

```ts
// GET /api/v1/specialist/referrals/{id}/schedule-options?days=14
type ScheduleOption = {
  date: string;            // YYYY-MM-DD
  max_slots: number;
  booked_slots: number;
  available_slots: number; // max_slots - booked_slots, floored at 0
  overbook_limit: number;
  has_override: boolean;   // a CapacityOverride is active for that day
};
```

Use this to render the calendar dots / availability badges in the date picker.
For a *single* date the user typed in manually (not in the next-N-days window),
fall back to a simple capacity probe — there's no dedicated endpoint, so just
attempt the post and let the 500 message route the user back to override.

---

## 6. Edge cases & how the backend already handles them

| Case | Backend behavior | FE responsibility |
|---|---|---|
| Past date | 400 `appointment date cannot be in the past` | disable past days in the picker |
| Patient already ARRIVED | 400 `cannot schedule appointment: patient has already arrived or been admitted` | hide buttons (use `available_actions`) |
| Patient already ADMITTED | same | same |
| Referral already COMPLETED / CANCELLED / REJECTED / REDIRECTED / DECEASED | 400 `only accepted or scheduled referrals can be ...` | the triage list already filters these out by default |
| Non-critical patient, blank justification | 400 `manual emergency schedule requires a critical condition or explicit justification` | enforce a min length (≥10 chars) before allowing submit |
| Capacity full, routine | 500 `capacity reached for this date - emergency override required` | swap CTA to override |
| Capacity full, even overbook | 500 `even overbook capacity is full for this date` | toast: try another day; show next 3 dates with `available_slots > 0` from `schedule-options` |
| MISSED rescue | flips ArrivalStatus to EXPECTED, sends `MISSED_APPOINTMENT_RESCHEDULED` SMS + in-app | render `rescheduled_from_missed` flag in toast |
| Reschedule SCHEDULED → new date | **NEW (fixed in this PR)**: old date's `daily_schedules.booked_slots` is also recounted, so calendar / dept-head views stay accurate | none |
| `condition_at_referral` stored as `"Critical"` / `"CRITICAL"` | **NEW (fixed)**: now matched case-insensitively | none |
| Double-click / duplicate submit | backend will happily run twice → two audits, two SMS | **FE must disable the button while the mutation is pending**; debounce 1s |
| Active `CapacityOverride` on the date | `max_slots` for capacity check uses the override's `NewLimit` | display `has_override` flag in the picker so user knows why a day suddenly has more room |
| Day boundary / timezone | backend stores `appointment_date` as `DATE`; the past-date check uses server local time | always send `YYYY-MM-DD` to the override endpoint — do NOT send a timestamp |
| Race: two specialists override the same slot at the same second | possible to exceed overbook by 1 in theory (capacity check is outside the transaction); acceptable for v1 | if you start seeing it in metrics we'll switch to a `SELECT ... FOR UPDATE` |
| Patient has CONSULTING doctors granted | unaffected — scheduling does not touch access | render the access list read-only in the detail page |
| Sending notification fails | swallowed by the use case (booking still commits) | don't surface — the dept head dashboard will reconcile |
| Dept head **deleted** today's `daily_schedules` row manually | next booking will recreate it with the current `EffectiveCapacity` limits | none |

---

## 7. Suggested copy for warning text

Short, sharp, dispenser-style. Pick one set and keep it consistent across the app.

> **Tooltip on the override CTA:**
> *"Override consumes the overbook buffer. Use only when routine capacity is exhausted and the patient cannot wait. The dept head is notified every time."*

> **Modal subtitle:**
> *"You are booking outside normal capacity. Justification is required unless the patient is marked critical."*

> **Toast after success (override, non-rescue):**
> *"Override scheduled for {date} — dept head has been notified."*

> **Toast after success (override + missed rescue):**
> *"Missed appointment rescued and rescheduled for {date}."*

---

## 8. Accessibility & micro-UX

- The amber override panel must have `role="region"` and `aria-labelledby` so
  screen readers announce "Schedule Override – warning region".
- The "Use override" button gets `aria-describedby` pointing at the warning
  paragraph.
- Modal traps focus; ESC cancels; ENTER on the date input does NOT submit (must
  click Confirm — too easy to misfire otherwise).
- Disable Confirm while `isPending`. Show a spinner inline, not a full-screen
  overlay.
- If `condition_at_referral === "critical"`, render the justification textarea
  as **optional** (`required={false}`) and surface a green pill —
  *"Justification optional — patient is critical"*.

---

## 9. Open items for later

- Server-side idempotency key on POST so a network retry doesn't double-book.
- Structured `reason_code` enum on the request body so dashboards can group by
  reason. Today the FE prefixes the reason into `justification` as a free string.
- A dedicated `ActionScheduleAppointment` audit action for routine bookings
  (currently piggybacks on `OVERRIDE_QUEUE`). This also fixes a gap where
  routine schedules don't appear in the triage detail timeline.
- Real-time capacity push (WebSocket) so the modal's availability bar reflects
  someone else's booking without a refetch.

---

## 10. Quick checklist for the FE PR

- [ ] Two distinct buttons (routine + override) wired to the right endpoints.
- [ ] Override button is amber-outlined, has a warning paragraph and a tooltip.
- [ ] Modal pulls `schedule-options` for date suggestions and live capacity.
- [ ] Justification is required UNLESS `condition_at_referral.toLowerCase().trim() === "critical"`.
- [ ] Reason selector concatenated into `justification` before POST.
- [ ] Submit disabled while pending.
- [ ] On success: invalidate `triage detail`, `triage list`, `schedule-options`.
- [ ] On 500 `"capacity reached"` from routine: auto-suggest "try override".
- [ ] On 500 `"overbook ... full"` from override: surface next 3 days with availability.
- [ ] `rescheduled_from_missed === true` → "rescued" wording in the toast.
- [ ] Hide both buttons when `arrival_status ∈ {ARRIVED, ADMITTED}`.
