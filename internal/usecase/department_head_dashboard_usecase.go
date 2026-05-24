package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"Hospital-Referral-System/internal/delivery/http/dto"
	"Hospital-Referral-System/internal/domain/entity"
	irepository "Hospital-Referral-System/internal/domain/interfaces/repository"
	iusecase "Hospital-Referral-System/internal/domain/interfaces/usecase"
)

type departmentHeadDashboardUseCase struct {
	capacityUC   iusecase.CapacityManagementUseCase
	schedulingUC iusecase.SchedulingUseCase
	triageRepo   irepository.TriageQueueRepository
	referralRepo irepository.ReferralRepository
	overrideRepo irepository.CapacityOverrideRepository
	userRepo     irepository.UserRepository
	deptRepo     irepository.DepartmentRepository
	auditRepo    irepository.AuditLogRepository
}

// NewDepartmentHeadDashboardUseCase wires the dept-head dashboard
// aggregator. It deliberately depends on existing use cases (capacity
// manager, scheduling) so that dashboard numbers always match the
// numbers the booking engine sees.
func NewDepartmentHeadDashboardUseCase(
	capacityUC iusecase.CapacityManagementUseCase,
	schedulingUC iusecase.SchedulingUseCase,
	triageRepo irepository.TriageQueueRepository,
	referralRepo irepository.ReferralRepository,
	overrideRepo irepository.CapacityOverrideRepository,
	userRepo irepository.UserRepository,
	deptRepo irepository.DepartmentRepository,
	auditRepo irepository.AuditLogRepository,
) iusecase.DepartmentHeadDashboardUseCase {
	return &departmentHeadDashboardUseCase{
		capacityUC:   capacityUC,
		schedulingUC: schedulingUC,
		triageRepo:   triageRepo,
		referralRepo: referralRepo,
		overrideRepo: overrideRepo,
		userRepo:     userRepo,
		deptRepo:     deptRepo,
		auditRepo:    auditRepo,
	}
}

func todayBounds() (time.Time, time.Time) {
	now := time.Now()
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	return start, start.Add(24 * time.Hour).Add(-time.Nanosecond)
}

// GetDashboardStats composes the landing-page summary. Errors on
// individual sub-queries are logged-via-nil-coalescing rather than
// aborting the whole call: a single failing repo should degrade one
// widget, not the entire dashboard.
func (u *departmentHeadDashboardUseCase) GetDashboardStats(ctx context.Context, hospitalID, deptID uuid.UUID) (*dto.DepartmentHeadDashboardStats, error) {
	if hospitalID == uuid.Nil || deptID == uuid.Nil {
		return nil, errors.New("hospital_id and department_id are required")
	}

	today, _ := todayBounds()

	stats := &dto.DepartmentHeadDashboardStats{}

	if detail, err := u.capacityUC.GetCapacityDetail(ctx, hospitalID, deptID, today); err == nil {
		stats.TodayCapacity = detail
	}

	if waiting, err := u.triageRepo.FindWaitingByHospitalAndDept(ctx, hospitalID, deptID); err == nil {
		stats.WaitingQueueSize = len(waiting)
	}

	if d, err := u.triageRepo.OldestWaitingDaysByDept(ctx, hospitalID, deptID); err == nil {
		stats.OldestWaitingDays = d
	}

	if c, err := u.triageRepo.CountByDeptAndDate(ctx, hospitalID, deptID, today); err == nil {
		stats.ScheduledToday = c
	}

	next7Start := today.AddDate(0, 0, 1)
	next7End := today.AddDate(0, 0, 7)
	if c, err := u.triageRepo.CountScheduledByDeptInRange(ctx, hospitalID, deptID, next7Start, next7End); err == nil {
		stats.ScheduledNext7Days = c
	}

	missedStart := today.AddDate(0, 0, -7)
	missedEnd := today.AddDate(0, 0, -1)
	if c, err := u.triageRepo.CountMissedByDeptInRange(ctx, hospitalID, deptID, missedStart, missedEnd); err == nil {
		stats.MissedLast7Days = c
	}

	if rows, err := u.referralRepo.CountByTargetDeptAndStatuses(
		ctx, hospitalID, deptID,
		[]entity.ReferralStatus{
			entity.StatusAccepted,
			entity.StatusScheduled,
			entity.StatusCompleted,
			entity.StatusUnderSpecialistReview,
			entity.StatusForwarded,
			entity.StatusRejectedBySpecialist,
		},
		nil, nil,
	); err == nil {
		stats.StatusCounts = make([]dto.DeptReferralStatusItem, 0, len(rows))
		for _, r := range rows {
			stats.StatusCounts = append(stats.StatusCounts, dto.DeptReferralStatusItem{
				Status: string(r.Status),
				Count:  r.Count,
			})
			switch r.Status {
			case entity.StatusAccepted:
				stats.PendingReferrals = r.Count
			}
		}
	}

	completedStart := today.AddDate(0, 0, -30)
	if rows, err := u.referralRepo.CountByTargetDeptAndStatuses(
		ctx, hospitalID, deptID,
		[]entity.ReferralStatus{entity.StatusCompleted},
		&completedStart, nil,
	); err == nil && len(rows) > 0 {
		stats.CompletedLast30Days = rows[0].Count
	}

	hospStr := hospitalID.String()
	deptStr := deptID.String()
	active := true
	// In this system "REFERRING_DOCTOR" is the universal doctor role
	// (named that way for legacy reasons; it applies to receiver-side
	// doctors and assigned treating doctors too, not just senders).
	// Together with receptionists they form the dept's active staff.
	if users, _, err := u.userRepo.ListUsers(ctx, irepository.UserListFilter{
		Roles:        []entity.UserRole{entity.RoleReferringDoctor, entity.RoleReceptionist},
		HospitalID:   &hospStr,
		DepartmentID: &deptStr,
		IsActive:     &active,
		Page:         1,
		PageSize:     500,
	}); err == nil {
		stats.ActiveStaff = len(users)
	}

	if overrides, err := u.overrideRepo.ListByDept(ctx, hospitalID, deptID); err == nil {
		stats.ActiveOverrides = len(overrides)
	}

	return stats, nil
}

// GetTrends walks `days` calendar days ending today (inclusive) and
// reuses EffectiveCapacity per day so the chart numbers are identical to
// the calendar rollup.
func (u *departmentHeadDashboardUseCase) GetTrends(ctx context.Context, hospitalID, deptID uuid.UUID, days int) ([]dto.DepartmentHeadTrendPoint, error) {
	if days <= 0 {
		days = 14
	}
	if days > 90 {
		days = 90
	}

	today, _ := todayBounds()
	out := make([]dto.DepartmentHeadTrendPoint, 0, days)

	for i := days - 1; i >= 0; i-- {
		date := today.AddDate(0, 0, -i)
		maxSlots, overbook, booked, err := u.schedulingUC.EffectiveCapacity(ctx, hospitalID, deptID, date)
		if err != nil {
			continue
		}
		available := maxSlots - int(booked)
		if available < 0 {
			available = 0
		}
		var utilization float64
		if maxSlots > 0 {
			utilization = float64(booked) / float64(maxSlots)
		}
		hasOverride := false
		if ov, e := u.overrideRepo.GetActive(ctx, hospitalID, deptID, date); e == nil && ov != nil {
			hasOverride = true
		}
		out = append(out, dto.DepartmentHeadTrendPoint{
			Date:           date.Format("2006-01-02"),
			MaxSlots:       maxSlots,
			BookedSlots:    booked,
			OverbookLimit:  overbook,
			AvailableSlots: available,
			Utilization:    utilization,
			HasOverride:    hasOverride,
		})
	}

	return out, nil
}

// GetPriorityBuckets pulls every ACTIVE triage row (EXPECTED + MISSED,
// scheduled or not, excluding terminal referrals) and buckets in memory.
// Previously this widget used FindWaitingByHospitalAndDept which excluded
// any row with an appointment_date — so a dept head with all patients
// already scheduled would see zeroes across the board. The active set is
// the correct population for the priority distribution view: it covers
// both unscheduled backlog AND upcoming visits.
func (u *departmentHeadDashboardUseCase) GetPriorityBuckets(ctx context.Context, hospitalID, deptID uuid.UUID) (*dto.PriorityBucketResponse, error) {
	queue, err := u.triageRepo.FindActiveByHospitalAndDept(ctx, hospitalID, deptID)
	if err != nil {
		return nil, err
	}

	condBuckets := map[string]int64{"critical": 0, "urgent": 0, "stable": 0, "unspecified": 0}
	sevBuckets := map[string]int64{"HIGH": 0, "MEDIUM": 0, "LOW": 0, "UNKNOWN": 0}

	type ranked struct {
		row entity.TriageQueue
		ref *entity.Referral
	}
	candidates := make([]ranked, 0, len(queue))

	for _, q := range queue {
		ref, _ := u.referralRepo.GetReferralByID(ctx, q.ReferralID)
		candidates = append(candidates, ranked{row: q, ref: ref})

		cond := "unspecified"
		if ref != nil && ref.ReferralForm != nil {
			c := strings.ToLower(strings.TrimSpace(ref.ReferralForm.ConditionAtReferral))
			if _, ok := condBuckets[c]; ok {
				cond = c
			}
		}
		condBuckets[cond]++

		tier := "UNKNOWN"
		if ref != nil && ref.MLSeverityScore != nil {
			switch {
			case *ref.MLSeverityScore >= 75:
				tier = "HIGH"
			case *ref.MLSeverityScore >= 40:
				tier = "MEDIUM"
			default:
				tier = "LOW"
			}
		}
		sevBuckets[tier]++
	}

	resp := &dto.PriorityBucketResponse{
		TotalWaiting: int64(len(queue)),
		ByCondition: []dto.PriorityBucket{
			{Label: "CRITICAL", Count: condBuckets["critical"]},
			{Label: "URGENT", Count: condBuckets["urgent"]},
			{Label: "STABLE", Count: condBuckets["stable"]},
			{Label: "UNSPECIFIED", Count: condBuckets["unspecified"]},
		},
		BySeverity: []dto.PriorityBucket{
			{Label: "HIGH", Count: sevBuckets["HIGH"]},
			{Label: "MEDIUM", Count: sevBuckets["MEDIUM"]},
			{Label: "LOW", Count: sevBuckets["LOW"]},
			{Label: "UNKNOWN", Count: sevBuckets["UNKNOWN"]},
		},
	}

	// sort.Slice in-place by composite score, take top 5
	for i := 0; i < len(candidates); i++ {
		for j := i + 1; j < len(candidates); j++ {
			if candidates[j].row.CompositeScore > candidates[i].row.CompositeScore {
				candidates[i], candidates[j] = candidates[j], candidates[i]
			}
		}
	}
	limit := 5
	if len(candidates) < limit {
		limit = len(candidates)
	}
	resp.TopWaiting = make([]dto.PriorityBucketWaitingItem, 0, limit)
	for i := 0; i < limit; i++ {
		item := dto.PriorityBucketWaitingItem{
			ReferralID:     candidates[i].row.ReferralID.String(),
			CompositeScore: candidates[i].row.CompositeScore,
		}
		if candidates[i].ref != nil {
			item.CreatedAt = candidates[i].ref.CreatedAt
			item.WaitingDays = int(time.Since(candidates[i].ref.CreatedAt).Hours() / 24)
			if candidates[i].ref.Patient != nil {
				item.PatientName = strings.TrimSpace(fmt.Sprintf("%s %s", candidates[i].ref.Patient.FirstNamePlain, candidates[i].ref.Patient.LastNamePlain))
			}
		}
		resp.TopWaiting = append(resp.TopWaiting, item)
	}

	return resp, nil
}

// GetStaffSummary returns per-role active/inactive counts and a roster
// of the (active) doctors + receptionists for the caller's dept. The
// roster is capped at 200 entries via the ListUsers paging cap.
func (u *departmentHeadDashboardUseCase) GetStaffSummary(ctx context.Context, hospitalID, deptID uuid.UUID) (*dto.StaffSummaryResponse, error) {
	hospStr := hospitalID.String()
	deptStr := deptID.String()

	out := &dto.StaffSummaryResponse{}

	if dept, err := u.deptRepo.FindHospitalDepartment(ctx, hospitalID, deptID); err == nil && dept != nil {
		out.StaffCapacityHint = dept.MaxCapacityOfStaff
		out.Department = dept.Department.Name
	}

	listFor := func(role entity.UserRole, isActive bool) []entity.User {
		flag := isActive
		r := role
		users, _, err := u.userRepo.ListUsers(ctx, irepository.UserListFilter{
			Role:         &r,
			HospitalID:   &hospStr,
			DepartmentID: &deptStr,
			IsActive:     &flag,
			Page:         1,
			PageSize:     200,
		})
		if err != nil {
			return nil
		}
		return users
	}

	// REFERRING_DOCTOR is the universal doctor role in this system
	// (used for both sender-side doctors and receiver-side treating
	// doctors). Keep the original filter — assigned-doctor and dept
	// staff lookups elsewhere depend on this same role.
	docActive := listFor(entity.RoleReferringDoctor, true)
	docInactive := listFor(entity.RoleReferringDoctor, false)
	recActive := listFor(entity.RoleReceptionist, true)
	recInactive := listFor(entity.RoleReceptionist, false)

	out.Doctors = dto.StaffRoleCount{
		Active:   len(docActive),
		Inactive: len(docInactive),
		Total:    len(docActive) + len(docInactive),
	}
	out.Receptionists = dto.StaffRoleCount{
		Active:   len(recActive),
		Inactive: len(recInactive),
		Total:    len(recActive) + len(recInactive),
	}

	today, _ := todayBounds()
	if c, err := u.triageRepo.CountAssignedDoctorsByDeptAndDate(ctx, hospitalID, deptID, today); err == nil {
		out.DoctorsAssignedToday = c
	}

	members := make([]dto.StaffMember, 0, len(docActive)+len(recActive))
	for _, list := range [][]entity.User{docActive, recActive} {
		for _, member := range list {
			members = append(members, dto.StaffMember{
				ID:           member.ID.String(),
				FirstName:    member.FirstName,
				LastName:     member.LastName,
				Email:        member.Email,
				Role:         string(member.Role),
				IsActive:     member.IsActive,
				ProfileImage: member.ProfileImageURL,
			})
		}
	}
	out.Members = members

	return out, nil
}

// dashboardActionTypes is the curated set of actions the dept head wants
// to see in the activity stream. Kept here (rather than in entity) so we
// can tune it without touching the audit model.
var dashboardActionTypes = []entity.ActionType{
	entity.ActionBatchSchedule,
	entity.ActionEmergencySchedule,
	entity.ActionOverrideQueue,
	entity.ActionManageCapacity,
	entity.ActionAssignDoctor,
	entity.ActionUnassignDoctor,
	entity.ActionConfirmArrival,
	entity.ActionMarkMissed,
	entity.ActionRecordOutcome,
	entity.ActionAddClinicalUpdate,
	entity.ActionAcceptReferral,
	entity.ActionRejectReferral,
	entity.ActionRedirectReferral,
}

// GetActivity returns recent audit-log rows scoped to the dept head's
// hospital and the curated action set. Department-level filtering is
// applied client-side after the join (audit rows are linked to users,
// not departments, so a per-dept SQL filter would require a wider
// schema change).
func (u *departmentHeadDashboardUseCase) GetActivity(ctx context.Context, hospitalID, deptID uuid.UUID, limit int, startDate, endDate *time.Time) ([]dto.DepartmentHeadActivityItem, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	filter := irepository.AuditLogFilter{
		Page:        1,
		PageSize:    limit * 3, // overfetch so dept filter can prune
		ActionTypes: dashboardActionTypes,
	}
	if startDate != nil {
		s := startDate.Format("2006-01-02")
		filter.StartDate = &s
	}
	if endDate != nil {
		e := endDate.Format("2006-01-02")
		filter.EndDate = &e
	}

	rows, _, err := u.auditRepo.ListByHospital(ctx, hospitalID, filter)
	if err != nil {
		return nil, err
	}

	deptStr := deptID.String()
	out := make([]dto.DepartmentHeadActivityItem, 0, limit)
	for _, r := range rows {
		if len(out) >= limit {
			break
		}
		// Best-effort dept-scoping: keep rows where the actor has the
		// same DepartmentID (when set) or where the audit row has no
		// way to be scoped (we fall back to "hospital-wide").
		if r.User != nil && r.User.DepartmentID != nil && r.User.DepartmentID.String() != deptStr {
			continue
		}

		item := dto.DepartmentHeadActivityItem{
			ID:         r.ID.String(),
			Timestamp:  r.Timestamp,
			ActionType: string(r.ActionType),
			ActorID:    r.UserID.String(),
			Summary:    summarizeAuditAction(r),
		}
		if r.User != nil {
			item.ActorName = strings.TrimSpace(r.User.FirstName + " " + r.User.LastName)
			item.ActorRole = string(r.User.Role)
		}
		if r.ReferralID != nil {
			s := r.ReferralID.String()
			item.ReferralID = &s
		}
		if r.NewValue != nil && *r.NewValue != "" {
			var parsed map[string]interface{}
			if err := json.Unmarshal([]byte(*r.NewValue), &parsed); err == nil {
				item.NewValue = parsed
			}
		}
		out = append(out, item)
	}
	return out, nil
}

// summarizeAuditAction produces a short human-readable phrase per
// action. Kept private so the wording is consistent across the dept
// head dashboard.
func summarizeAuditAction(r entity.AuditLog) string {
	switch r.ActionType {
	case entity.ActionBatchSchedule:
		return "Ran batch scheduling"
	case entity.ActionEmergencySchedule:
		return "Created an emergency schedule"
	case entity.ActionOverrideQueue:
		return "Modified a capacity override"
	case entity.ActionManageCapacity:
		return "Updated capacity settings"
	case entity.ActionAssignDoctor:
		return "Assigned a doctor"
	case entity.ActionUnassignDoctor:
		return "Unassigned a doctor"
	case entity.ActionConfirmArrival:
		return "Confirmed a patient arrival"
	case entity.ActionMarkMissed:
		return "Marked appointment missed"
	case entity.ActionRecordOutcome:
		return "Recorded a clinical outcome"
	case entity.ActionAddClinicalUpdate:
		return "Added a clinical update"
	case entity.ActionAcceptReferral:
		return "Accepted a referral"
	case entity.ActionRejectReferral:
		return "Rejected a referral"
	case entity.ActionRedirectReferral:
		return "Redirected a referral"
	}
	return string(r.ActionType)
}

// Compile-time interface check; avoids a "value not used" complaint if
// we touch helpers in the future and lets editors jump to the impl.
var _ = gorm.ErrRecordNotFound
