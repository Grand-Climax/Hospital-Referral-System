package usecase

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"Hospital-Referral-System/internal/delivery/http/dto"
	"Hospital-Referral-System/internal/domain/entity"
	irepository "Hospital-Referral-System/internal/domain/interfaces/repository"
	iusecase "Hospital-Referral-System/internal/domain/interfaces/usecase"
	"Hospital-Referral-System/internal/infrastructure/crypto"
)

type triageUseCase struct {
	db           *gorm.DB
	referralRepo irepository.ReferralRepository
	triageRepo   irepository.TriageQueueRepository
	mlRepo       irepository.MLPredictionRepository
	configRepo   irepository.SystemConfigRepository
	auditRepo    irepository.AuditLogRepository
	accessRepo   irepository.ReferralAccessRepository
	cryptoSvc    *crypto.PatientCryptoService
	mlUC         iusecase.MLUseCase
}

func NewTriageUseCase(
	db *gorm.DB,
	rRepo irepository.ReferralRepository,
	tRepo irepository.TriageQueueRepository,
	mlRepo irepository.MLPredictionRepository,
	cfgRepo irepository.SystemConfigRepository,
	auditRepo irepository.AuditLogRepository,
	accessRepo irepository.ReferralAccessRepository,
	cryptoSvc *crypto.PatientCryptoService,
	mlUC iusecase.MLUseCase,
) iusecase.TriageUseCase {
	return &triageUseCase{
		db:           db,
		referralRepo: rRepo,
		triageRepo:   tRepo,
		mlRepo:       mlRepo,
		configRepo:   cfgRepo,
		auditRepo:    auditRepo,
		accessRepo:   accessRepo,
		cryptoSvc:    cryptoSvc,
		mlUC:         mlUC,
	}
}

func (u *triageUseCase) LandInQueue(ctx context.Context, referralID uuid.UUID) error {
	ref, err := u.referralRepo.GetReferralByID(ctx, referralID)
	if err != nil {
		return err
	}

	queue := &entity.TriageQueue{
		ReferralID:      referralID,
		HospitalID:      ref.TargetHospitalID,
		DepartmentID:    ref.TargetDeptID,
		ArrivalStatus:   entity.ArrivalExpected,
		AppointmentDate: nil,
	}

	score, _ := u.CalculateCompositeScore(ctx, referralID)
	queue.CompositeScore = score

	return u.triageRepo.Create(ctx, queue)
}

func (u *triageUseCase) CalculateCompositeScore(ctx context.Context, referralID uuid.UUID) (float64, error) {
	ref, err := u.referralRepo.GetReferralByID(ctx, referralID)
	if err != nil {
		return 0, err
	}
	if ref.ReferralForm == nil {
		return 0, errors.New("referral form missing")
	}

	var triageScore float64
	switch strings.ToLower(strings.TrimSpace(ref.ReferralForm.ConditionAtReferral)) {
	case "critical":
		triageScore = 100
	case "urgent":
		triageScore = 70
	case "stable":
		triageScore = 30
	default:
		triageScore = 10
	}

	mlSeverity := 50.0
	pred, err := u.mlRepo.GetLatestByReferralID(ctx, referralID)
	if err == nil && pred != nil {
		mlSeverity = pred.OutputScore
	}

	agingFactor := 1.0
	cfg, err := u.configRepo.GetByKey(ctx, "aging_factor")
	if err == nil && cfg != nil {
		af, _ := strconv.ParseFloat(cfg.Value, 64)
		if af > 0 {
			agingFactor = af
		}
	}
	daysWait := time.Since(ref.CreatedAt).Hours() / 24
	agingBonus := daysWait * agingFactor

	composite := (triageScore * 0.6) + (mlSeverity * 0.3) + agingBonus
	return composite, nil
}

// ListForTriage is the legacy hospital-wide list kept for backwards
// compatibility with old callers. New code should use ListTriageFiltered
// for filterable, paginated, single-query access with role scoping.
func (u *triageUseCase) ListForTriage(ctx context.Context, hospitalID uuid.UUID, limit, offset int) ([]dto.TriageListResponse, int64, error) {
	queues, count, err := u.triageRepo.ListForTriage(ctx, hospitalID, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	resp := make([]dto.TriageListResponse, 0, len(queues))
	for _, q := range queues {
		ref, _ := u.referralRepo.GetReferralByID(ctx, q.ReferralID)
		resp = append(resp, u.mapQueueToListItem(&q, ref))
	}
	return resp, count, nil
}

// ListForTriageByDepartment is the legacy department-scoped list, kept
// for backwards compatibility. New code should use ListTriageFiltered.
func (u *triageUseCase) ListForTriageByDepartment(ctx context.Context, hospitalID, deptID uuid.UUID, limit, offset int) ([]dto.TriageListResponse, int64, error) {
	queues, count, err := u.triageRepo.FindByHospitalAndDept(ctx, hospitalID, deptID, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	resp := make([]dto.TriageListResponse, 0, len(queues))
	for _, q := range queues {
		ref, _ := u.referralRepo.GetReferralByID(ctx, q.ReferralID)
		resp = append(resp, u.mapQueueToListItem(q, ref))
	}
	return resp, count, nil
}

// ListTriageFiltered is the single role-aware list path used by every
// triage queue endpoint. It accepts a rich filter, hashes national_id
// at this boundary so the repo never sees plaintext PII, batches one
// JOIN+preload query through the repository, and projects to the rich
// TriageListItem DTO including decrypted patient names.
func (u *triageUseCase) ListTriageFiltered(ctx context.Context, filter dto.TriageListFilter) ([]dto.TriageListItem, int64, error) {
	repoFilter := irepository.TriageQueueFilter{
		HospitalID:        filter.HospitalID,
		DepartmentID:      filter.DepartmentID,
		ArrivalStatuses:   filter.ArrivalStatuses,
		ReferralStatuses:  filter.ReferralStatuses,
		PatientID:         filter.PatientID,
		HasDoctorAssigned: filter.HasDoctorAssigned,
		IncludeTerminal:   filter.IncludeTerminal,
		SortBy:            filter.SortBy,
		SortOrder:         filter.SortOrder,
		Limit:             filter.Limit,
		Offset:            filter.Offset,
	}

	// Hash national_id at the use-case boundary so plaintext PII never
	// crosses into the repository layer. Empty hash means "not provided".
	if filter.NationalID != "" && u.cryptoSvc != nil {
		hash := u.cryptoSvc.GenerateHMAC(filter.NationalID)
		repoFilter.NationalIDHash = &hash
	}

	queues, total, err := u.triageRepo.ListTriageQueueFiltered(ctx, repoFilter)
	if err != nil {
		return nil, 0, err
	}

	items := make([]dto.TriageListItem, 0, len(queues))
	for i := range queues {
		q := &queues[i]
		items = append(items, u.mapQueueToListItem(q, q.Referral))
	}
	return items, total, nil
}

// mapQueueToListItem builds the rich TriageListItem from a queue row +
// (optionally) its preloaded referral graph. Tolerates partial preloads
// so both the new JOIN path and the legacy per-row fetch path work.
func (u *triageUseCase) mapQueueToListItem(q *entity.TriageQueue, ref *entity.Referral) dto.TriageListItem {
	item := dto.TriageListItem{
		QueueID:         q.ID,
		ReferralID:      q.ReferralID,
		CompositeScore:  q.CompositeScore,
		AppointmentDate: q.AppointmentDate,
		ArrivalStatus:   string(q.ArrivalStatus),
		DepartmentID:    q.DepartmentID,
		CreatedAt:       q.AssignedAt,
		PatientName:     "Unknown",
	}
	if q.Department != nil && q.Department.Name != "" {
		item.DepartmentName = q.Department.Name
	}
	if q.AssignedDoctorID != nil {
		item.HasDoctorAssigned = true
		item.AssignedDoctorID = q.AssignedDoctorID
		if q.AssignedDoctor != nil && q.AssignedDoctor.ID != uuid.Nil {
			item.AssignedDoctorName = composeUserName(q.AssignedDoctor)
		}
	}
	if ref != nil {
		item.ReferralStatus = string(ref.Status)
		if ref.Patient != nil {
			item.PatientID = ref.Patient.ID
			_ = ref.Patient.DecryptFields(u.cryptoSvc)
			name := strings.TrimSpace(ref.Patient.FirstNamePlain + " " + ref.Patient.LastNamePlain)
			if name != "" {
				item.PatientName = name
			}
		}
		if ref.ReferralForm != nil {
			item.ConditionAtReferral = strings.ToLower(strings.TrimSpace(ref.ReferralForm.ConditionAtReferral))
		}
	}
	return item
}

// composeUserName builds "First Last" from a User entity, handling the
// missing-middle-name case gracefully.
func composeUserName(u *entity.User) string {
	if u == nil {
		return ""
	}
	return strings.TrimSpace(strings.Join([]string{u.FirstName, u.LastName}, " "))
}

func (u *triageUseCase) ReviewTriage(ctx context.Context, referralID, userID uuid.UUID, req dto.TriageReviewRequest) error {
	queue, err := u.triageRepo.GetByReferralID(ctx, referralID)
	if err != nil {
		return err
	}

	switch req.Action {
	case "APPROVE":
		// logic
	case "OVERRIDE":
		if req.CompositeScore != nil {
			queue.CompositeScore = *req.CompositeScore
		}
	case "REJECT":
		return errors.New("rejection logic not yet implemented")
	}

	if err := u.triageRepo.Update(ctx, queue); err != nil {
		return err
	}

	return u.auditRepo.LogWithContext(ctx, userID, entity.ActionOverrideQueue, &referralID, nil, req)
}

func (u *triageUseCase) ListScheduledInRange(ctx context.Context, hospitalID, deptID uuid.UUID, start, end time.Time) ([]entity.TriageQueue, error) {
	return u.triageRepo.ListScheduledInRange(ctx, hospitalID, deptID, start, end)
}

// ───────────────────────────────────────────────────────────────────────────
// Detail endpoints (role-aware projections)
// ───────────────────────────────────────────────────────────────────────────

// timelineActions is the curated audit_log action set rendered as the
// arrival_history timeline. Keep this list explicit so unrelated audit
// rows (e.g. login, view) never leak into the per-referral timeline.
var timelineActions = []entity.ActionType{
	entity.ActionAcceptReferral,
	entity.ActionApproveReferral,
	entity.ActionConfirmArrival,
	entity.ActionMarkMissed,
	entity.ActionAssignDoctor,
	entity.ActionUnassignDoctor,
	entity.ActionEmergencySchedule,
	entity.ActionBatchSchedule,
	entity.ActionGrantConsultAccess,
	entity.ActionRevokeConsultAccess,
	entity.ActionRecordOutcome,
}

// triageContext is the shared bundle the three detail methods build
// once and then project differently per role. Loading it through one
// helper keeps the per-role methods focused on field selection only.
type triageContext struct {
	queue    *entity.TriageQueue
	referral *entity.Referral
	access   []entity.ReferralAccess
	timeline []entity.AuditLog
}

// loadTriageContextByReferral fetches everything a detail projection
// needs given a Referral UUID. All three role-aware detail endpoints
// (specialist, receptionist, dept-head) flow through this loader so
// the FE only ever needs to know one id (referral_id).
func (u *triageUseCase) loadTriageContextByReferral(ctx context.Context, referralID uuid.UUID) (*triageContext, error) {
	queue, err := u.triageRepo.GetByReferralID(ctx, referralID)
	if err != nil {
		return nil, err
	}
	return u.loadTriageContextForQueue(ctx, queue)
}

func (u *triageUseCase) loadTriageContextForQueue(ctx context.Context, queue *entity.TriageQueue) (*triageContext, error) {
	if queue == nil {
		return nil, errors.New("triage queue row not found")
	}
	ref, err := u.referralRepo.GetReferralByID(ctx, queue.ReferralID)
	if err != nil {
		return nil, err
	}
	// Best-effort: decrypt patient fields here so each projection just
	// reads FirstNamePlain / LastNamePlain.
	if ref != nil && ref.Patient != nil {
		_ = ref.Patient.DecryptFields(u.cryptoSvc)
	}

	access, _ := u.accessRepo.ListAllByReferral(ctx, queue.ReferralID)
	timeline, _ := u.auditRepo.ListByReferralAndActions(ctx, queue.ReferralID, timelineActions, 100)

	return &triageContext{
		queue:    queue,
		referral: ref,
		access:   access,
		timeline: timeline,
	}, nil
}

// computeAvailableActions encodes the role-aware action bitmap. The
// invariants come directly from the existing guards:
//   - emergency_schedule requires referral_status ∈ {ACCEPTED, SCHEDULED}
//     AND arrival_status NOT IN {ARRIVED, ADMITTED}  -> scheduling_usecase.go
//   - return_to_triage requires arrival_status = MISSED -> arrival_usecase.go
//   - mark_arrived requires arrival_status = EXPECTED AND appointment_date set
//   - mark_missed requires arrival_status = EXPECTED AND appointment is past
func computeAvailableActions(role entity.UserRole, q *entity.TriageQueue, ref *entity.Referral) dto.TriageDetailAvailableActions {
	actions := dto.TriageDetailAvailableActions{}
	if q == nil || ref == nil {
		return actions
	}

	arrival := q.ArrivalStatus
	status := ref.Status
	hasAppt := q.AppointmentDate != nil
	notArrivedOrAdmitted := arrival != entity.ArrivalArrived && arrival != entity.ArrivalAdmitted
	statusSchedulable := status == entity.StatusAccepted || status == entity.StatusScheduled

	switch role {
	case entity.RoleReceivingSpecialist:
		actions.Schedule = statusSchedulable && notArrivedOrAdmitted
		actions.EmergencySchedule = statusSchedulable && notArrivedOrAdmitted
		actions.ReturnToTriage = arrival == entity.ArrivalMissed

	case entity.RoleReceptionist:
		actions.MarkArrived = arrival == entity.ArrivalExpected && hasAppt
		actions.MarkMissed = arrival == entity.ArrivalExpected && hasAppt
		actions.AssignDoctor = arrival == entity.ArrivalArrived || arrival == entity.ArrivalAdmitted
		actions.RevokeDoctor = q.AssignedDoctorID != nil
		actions.ReturnToTriage = arrival == entity.ArrivalMissed

	case entity.RoleDeptHead:
		// Dept head is read-only on the triage row in this view; the
		// actions field is left empty intentionally. The FE can still
		// link to the batch-schedule and override endpoints elsewhere.
	}

	return actions
}

// buildPatientBlock projects an entity.Patient into the role-aware
// detail DTO. Sensitive fields (national_id, phone) are populated only
// when includeSensitive is true (currently: specialist + receptionist).
func buildPatientBlock(p *entity.Patient, includeSensitive bool) dto.TriageDetailPatient {
	if p == nil {
		return dto.TriageDetailPatient{}
	}
	full := strings.TrimSpace(p.FirstNamePlain + " " + p.LastNamePlain)
	out := dto.TriageDetailPatient{
		ID:         p.ID,
		FullName:   full,
		FirstName:  p.FirstNamePlain,
		MiddleName: p.MiddleNamePlain,
		LastName:   p.LastNamePlain,
		Sex:        p.Sex,
	}
	if p.DateOfBirth != nil {
		age := int(time.Since(*p.DateOfBirth).Hours() / (24 * 365.25))
		if age >= 0 && age <= 130 {
			out.AgeYears = &age
		}
	}
	if p.HomeRegion != nil {
		out.HomeRegion = string(*p.HomeRegion)
	}
	if includeSensitive {
		out.NationalID = p.NationalIDPlain
		out.PhoneNumber = p.PhonePlain
	}
	return out
}

// buildDoctorCard projects a ReferralAccess + User into a doctor card.
func buildDoctorCard(a *entity.ReferralAccess) dto.TriageDetailDoctor {
	card := dto.TriageDetailDoctor{
		AccessType: string(a.AccessType),
	}
	granted := a.GrantedAt
	card.GrantedAt = &granted
	g := a.GrantedBy
	card.GrantedBy = &g
	if a.RevokedAt != nil {
		card.RevokedAt = a.RevokedAt
	}
	if a.RevokeReason != nil {
		card.RevokeReason = *a.RevokeReason
	}
	if a.User != nil && a.User.ID != uuid.Nil {
		card.UserID = a.User.ID
		card.FullName = composeUserName(a.User)
		card.Email = a.User.Email
	} else {
		card.UserID = a.UserID
	}
	return card
}

// buildTimeline maps audit_log rows to TriageDetailTimelineEvent
// entries with a short human-readable description per event.
func buildTimeline(logs []entity.AuditLog) []dto.TriageDetailTimelineEvent {
	out := make([]dto.TriageDetailTimelineEvent, 0, len(logs))
	for _, l := range logs {
		ev := dto.TriageDetailTimelineEvent{
			At:          l.Timestamp,
			Event:       string(l.ActionType),
			Description: describeAction(l.ActionType),
			ActorID:     l.UserID,
		}
		if l.User != nil {
			ev.ActorName = composeUserName(l.User)
		}
		out = append(out, ev)
	}
	return out
}

func describeAction(a entity.ActionType) string {
	switch a {
	case entity.ActionAcceptReferral:
		return "Referral accepted into triage"
	case entity.ActionApproveReferral:
		return "Referral approved by liaison"
	case entity.ActionConfirmArrival:
		return "Patient arrived and was checked in"
	case entity.ActionMarkMissed:
		return "Appointment marked as missed"
	case entity.ActionAssignDoctor:
		return "Treating doctor assigned"
	case entity.ActionUnassignDoctor:
		return "Doctor assignment revoked"
	case entity.ActionEmergencySchedule:
		return "Emergency schedule applied"
	case entity.ActionBatchSchedule:
		return "Scheduled via batch run"
	case entity.ActionGrantConsultAccess:
		return "Consulting doctor granted access"
	case entity.ActionRevokeConsultAccess:
		return "Consulting doctor access revoked"
	case entity.ActionRecordOutcome:
		return "Outcome recorded"
	default:
		return string(a)
	}
}

func findTreatingDoctor(access []entity.ReferralAccess) *entity.ReferralAccess {
	for i := range access {
		if access[i].AccessType == entity.AccessTreatingDoctor && access[i].RevokedAt == nil {
			return &access[i]
		}
	}
	return nil
}

func filterConsultingDoctors(access []entity.ReferralAccess) []entity.ReferralAccess {
	out := make([]entity.ReferralAccess, 0, len(access))
	for i := range access {
		if access[i].AccessType == entity.AccessConsultedDoctor {
			out = append(out, access[i])
		}
	}
	return out
}

// GetTriageDetailForSpecialist is the rich view a Receiving Specialist
// uses to triage one referral: full PII, clinical text, diagnoses,
// vitals, ML severity, access list, arrival timeline.
func (u *triageUseCase) GetTriageDetailForSpecialist(ctx context.Context, referralID, userID uuid.UUID) (*dto.TriageDetailSpecialistResponse, error) {
	tc, err := u.loadTriageContextByReferral(ctx, referralID)
	if err != nil {
		return nil, err
	}
	_ = userID
	out := &dto.TriageDetailSpecialistResponse{Success: true}
	d := &out.Data
	d.QueueID = tc.queue.ID
	d.ReferralID = tc.queue.ReferralID
	d.ArrivalStatus = string(tc.queue.ArrivalStatus)
	d.AppointmentDate = tc.queue.AppointmentDate
	d.CompositeScore = tc.queue.CompositeScore
	d.DepartmentID = tc.queue.DepartmentID
	d.CreatedAt = tc.queue.AssignedAt

	if tc.referral != nil {
		d.ReferralStatus = string(tc.referral.Status)
		if tc.referral.ReferralForm != nil {
			d.ConditionAtReferral = strings.ToLower(strings.TrimSpace(tc.referral.ReferralForm.ConditionAtReferral))
			d.ClinicalSummary = tc.referral.ReferralForm.ClinicalSummary
			d.ReasonOfReferral = tc.referral.ReferralForm.ReasonOfReferral
			if tc.referral.ReferralForm.InvestigationResults != nil {
				d.InvestigationResult = *tc.referral.ReferralForm.InvestigationResults
			}
		}
		d.Patient = buildPatientBlock(tc.referral.Patient, true)
		if len(tc.referral.Vitals) > 0 {
			v := tc.referral.Vitals[0]
			d.Vitals = &dto.TriageDetailVitals{
				RecordedAt:      &v.RecordedAt,
				SystolicBP:      v.SystolicBP,
				DiastolicBP:     v.DiastolicBP,
				HeartRate:       v.HeartRate,
				SpO2:            v.SpO2,
				Temperature:     v.Temperature,
				RespiratoryRate: v.RespiratoryRate,
				GCSScore:        v.GCSScore,
			}
		}
		d.Diagnoses = make([]dto.TriageDetailDiagnosis, 0, len(tc.referral.Diagnoses))
		for _, diag := range tc.referral.Diagnoses {
			desc := ""
			if diag.CodeInfo != nil {
				desc = diag.CodeInfo.Description
			}
			d.Diagnoses = append(d.Diagnoses, dto.TriageDetailDiagnosis{
				ICDCode:            diag.ICDCode,
				Description:        desc,
				IsPrimary:          diag.IsPrimary,
				DiagnosisCertainty: string(diag.DiagnosisCertainty),
			})
		}
	}

	if pred, err := u.mlRepo.GetLatestByReferralID(ctx, referralID); err == nil && pred != nil {
		score := pred.OutputScore
		d.MLSeverityScore = &score
	}
	// Triage status is read from the most recent override audit if any.
	d.TriageStatus = string(entity.TriageAutoScored)

	// Access list + treating + consulting.
	if treating := findTreatingDoctor(tc.access); treating != nil {
		card := buildDoctorCard(treating)
		d.TreatingDoctor = &card
	}
	consultRows := filterConsultingDoctors(tc.access)
	d.ConsultingDoctors = make([]dto.TriageDetailDoctor, 0, len(consultRows))
	for i := range consultRows {
		d.ConsultingDoctors = append(d.ConsultingDoctors, buildDoctorCard(&consultRows[i]))
	}
	d.ReferralAccessList = make([]dto.TriageDetailDoctor, 0, len(tc.access))
	for i := range tc.access {
		d.ReferralAccessList = append(d.ReferralAccessList, buildDoctorCard(&tc.access[i]))
	}
	if tc.queue.Department != nil && tc.queue.Department.Name != "" {
		d.DepartmentName = tc.queue.Department.Name
	}

	d.ArrivalHistory = buildTimeline(tc.timeline)
	d.AvailableActions = computeAvailableActions(entity.RoleReceivingSpecialist, tc.queue, tc.referral)
	return out, nil
}

// GetTriageDetailForReceptionist returns the redacted detail: patient
// name + appointment + arrival status + assigned doctor only.
func (u *triageUseCase) GetTriageDetailForReceptionist(ctx context.Context, referralID, userID uuid.UUID) (*dto.TriageDetailReceptionistResponse, error) {
	tc, err := u.loadTriageContextByReferral(ctx, referralID)
	if err != nil {
		return nil, err
	}
	_ = userID
	out := &dto.TriageDetailReceptionistResponse{Success: true}
	d := &out.Data
	d.QueueID = tc.queue.ID
	d.ReferralID = tc.queue.ReferralID
	d.ArrivalStatus = string(tc.queue.ArrivalStatus)
	d.AppointmentDate = tc.queue.AppointmentDate
	d.ArrivedAt = tc.queue.ArrivedAt
	if tc.queue.MissReason != nil {
		d.MissReason = string(*tc.queue.MissReason)
	}
	d.DepartmentID = tc.queue.DepartmentID
	if tc.queue.Department != nil && tc.queue.Department.Name != "" {
		d.DepartmentName = tc.queue.Department.Name
	}
	if tc.referral != nil {
		d.ReferralStatus = string(tc.referral.Status)
		d.Patient = buildPatientBlock(tc.referral.Patient, true)
	}
	if treating := findTreatingDoctor(tc.access); treating != nil {
		card := buildDoctorCard(treating)
		d.AssignedDoctor = &card
	} else if tc.queue.AssignedDoctor != nil && tc.queue.AssignedDoctorID != nil {
		// Fallback path: assignment may exist on the queue row even if
		// the access grant lookup raced or returned empty.
		d.AssignedDoctor = &dto.TriageDetailDoctor{
			UserID:     *tc.queue.AssignedDoctorID,
			FullName:   composeUserName(tc.queue.AssignedDoctor),
			AccessType: string(entity.AccessTreatingDoctor),
		}
	}
	d.ArrivalHistory = buildTimeline(tc.timeline)
	d.AvailableActions = computeAvailableActions(entity.RoleReceptionist, tc.queue, tc.referral)
	return out, nil
}

// GetTriageDetailForDeptHead returns the operations view: capacity-
// relevant fields, no clinical text or ML internals. Lookup uses the
// REFERRAL UUID so the FE has a single id model across all three
// role-aware detail endpoints. The list endpoint still surfaces both
// queue_id and referral_id; callers should pass referral_id here.
func (u *triageUseCase) GetTriageDetailForDeptHead(ctx context.Context, referralID, userID uuid.UUID) (*dto.TriageDetailDeptHeadResponse, error) {
	tc, err := u.loadTriageContextByReferral(ctx, referralID)
	if err != nil {
		return nil, err
	}
	_ = userID
	out := &dto.TriageDetailDeptHeadResponse{Success: true}
	d := &out.Data
	d.QueueID = tc.queue.ID
	d.ReferralID = tc.queue.ReferralID
	d.ArrivalStatus = string(tc.queue.ArrivalStatus)
	d.AppointmentDate = tc.queue.AppointmentDate
	d.CompositeScore = tc.queue.CompositeScore
	d.DepartmentID = tc.queue.DepartmentID
	d.CreatedAt = tc.queue.AssignedAt
	if tc.queue.Department != nil && tc.queue.Department.Name != "" {
		d.DepartmentName = tc.queue.Department.Name
	}
	if tc.queue.AssignedDoctorID != nil {
		d.HasDoctorAssigned = true
		if tc.queue.AssignedDoctor != nil {
			d.AssignedDoctor = &dto.TriageDetailDoctor{
				UserID:   *tc.queue.AssignedDoctorID,
				FullName: composeUserName(tc.queue.AssignedDoctor),
				Email:    tc.queue.AssignedDoctor.Email,
			}
		}
	}
	if tc.referral != nil {
		d.ReferralStatus = string(tc.referral.Status)
		if tc.referral.ReferralForm != nil {
			d.ConditionAtReferral = strings.ToLower(strings.TrimSpace(tc.referral.ReferralForm.ConditionAtReferral))
		}
		// Dept head sees patient identity (name + age + region) but not
		// national_id / phone.
		d.Patient = buildPatientBlock(tc.referral.Patient, false)
	}
	d.ArrivalHistory = buildTimeline(tc.timeline)
	d.AvailableActions = computeAvailableActions(entity.RoleDeptHead, tc.queue, tc.referral)
	return out, nil
}
