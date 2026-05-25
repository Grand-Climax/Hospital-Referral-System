package usecase

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"Hospital-Referral-System/internal/delivery/http/dto"
	"Hospital-Referral-System/internal/domain/entity"
	irepository "Hospital-Referral-System/internal/domain/interfaces/repository"
	iusecase "Hospital-Referral-System/internal/domain/interfaces/usecase"
)

type schedulingUseCase struct {
	db                *gorm.DB
	referralRepo      irepository.ReferralRepository
	triageRepo        irepository.TriageQueueRepository
	scheduleRepo      irepository.DailyScheduleRepository
	overrideRepo      irepository.CapacityOverrideRepository
	deptRepo          irepository.DepartmentRepository
	configRepo        irepository.SystemConfigRepository
	auditRepo         irepository.AuditLogRepository
	notifUC           iusecase.NotificationUseCase
	inAppNotifUC      iusecase.InAppNotificationUseCase
	jobCheckpointRepo irepository.JobCheckpointRepository
	clinicalRepo      irepository.ClinicalUpdateRepository
	checkpointRepo    irepository.SchedulerCheckpointRepository
	// triageUC owns the composite-score formula so BatchSchedule can
	// recompute waiting rows just before booking them. Optional - nil
	// means "skip the recompute step" (useful in tests / older wiring).
	triageUC iusecase.TriageUseCase
}

func NewSchedulingUseCase(
	db *gorm.DB,
	rRepo irepository.ReferralRepository,
	tRepo irepository.TriageQueueRepository,
	sRepo irepository.DailyScheduleRepository,
	ovRepo irepository.CapacityOverrideRepository,
	deptRepo irepository.DepartmentRepository,
	configRepo irepository.SystemConfigRepository,
	auditRepo irepository.AuditLogRepository,
	notifUC iusecase.NotificationUseCase,
	inAppNotifUC iusecase.InAppNotificationUseCase,
	jRepo irepository.JobCheckpointRepository,
	cRepo irepository.ClinicalUpdateRepository,
	checkpointRepo irepository.SchedulerCheckpointRepository,
	triageUC iusecase.TriageUseCase,
) iusecase.SchedulingUseCase {
	return &schedulingUseCase{
		db:                db,
		referralRepo:      rRepo,
		triageRepo:        tRepo,
		scheduleRepo:      sRepo,
		overrideRepo:      ovRepo,
		deptRepo:          deptRepo,
		configRepo:        configRepo,
		auditRepo:         auditRepo,
		notifUC:           notifUC,
		inAppNotifUC:      inAppNotifUC,
		jobCheckpointRepo: jRepo,
		clinicalRepo:      cRepo,
		checkpointRepo:    checkpointRepo,
		triageUC:          triageUC,
	}
}

// defaultAppointmentHour is the clinic's standard start-of-day. We use it
// to substitute the hour component when a caller only supplied a date
// (i.e. midnight) so patient-facing SMS read "at 08:00" instead of
// "at 00:00". 8AM is the policy default; if a department later wants a
// per-department opening hour, swap this for a lookup against
// HospitalDepartment / SystemConfig.
const defaultAppointmentHour = 8

// normalizeAppointmentTime moves a midnight timestamp forward to the
// clinic's default opening hour. Any explicit time-of-day from the caller
// is preserved (it's only the date-only "00:00:00" case that we rewrite).
func normalizeAppointmentTime(t time.Time) time.Time {
	if t.Hour() == 0 && t.Minute() == 0 && t.Second() == 0 && t.Nanosecond() == 0 {
		return time.Date(t.Year(), t.Month(), t.Day(), defaultAppointmentHour, 0, 0, 0, t.Location())
	}
	return t
}

// EffectiveCapacity is the single source of truth for "is there room?".
// It returns the effective maxSlots and overbookLimit for a given
// (hospital, department, date) plus the live booked count.
//
//   - booked         : live count from TriageQueue (Expected/Arrived/Admitted)
//   - maxSlots       : HospitalDepartment.StandardDailyLimit, overridden by
//     CapacityOverride.NewLimit if an active override exists
//   - overbookLimit  : HospitalDepartment.OverbookLimit (department-level only)
//
// Capacity decisions (Routine / Batch reject when booked >= maxSlots;
// Emergency rejects when booked >= maxSlots + overbookLimit) live in the
// caller so each path can apply its own rule. Exported so the capacity
// manager (detail/calendar) and specialist schedule-options can reuse it.
func (u *schedulingUseCase) EffectiveCapacity(ctx context.Context, hospitalID, deptID uuid.UUID, date time.Time) (maxSlots, overbookLimit int, booked int64, err error) {
	booked, err = u.triageRepo.CountByDeptAndDate(ctx, hospitalID, deptID, date)
	if err != nil {
		return 0, 0, 0, err
	}

	dept, err := u.deptRepo.FindHospitalDepartment(ctx, hospitalID, deptID)
	if err != nil {
		return 0, 0, 0, err
	}
	maxSlots = dept.StandardDailyLimit
	overbookLimit = dept.OverbookLimit

	if ov, e := u.overrideRepo.GetActive(ctx, hospitalID, deptID, date); e == nil && ov != nil {
		maxSlots = ov.NewLimit
	}
	return maxSlots, overbookLimit, booked, nil
}

// ListScheduleOptions returns the next `days` calendar days starting at
// today + system_configs.buffer_days, each annotated with the live
// available capacity (max - booked, floored at 0), overbook limit, and
// whether an active override is in effect. Days where booked >= max are
// skipped; the specialist UI uses the returned dates as routine booking
// suggestions for the given referral.
func (u *schedulingUseCase) ListScheduleOptions(ctx context.Context, referralID uuid.UUID, days int) ([]dto.ScheduleOption, error) {
	if days <= 0 {
		days = 14
	}
	if days > 60 {
		days = 60
	}

	ref, err := u.referralRepo.GetReferralByID(ctx, referralID)
	if err != nil {
		return nil, err
	}

	bufferDays := 2
	if cfg, err := u.configRepo.GetByKey(ctx, "buffer_days"); err == nil && cfg != nil {
		if val, err := strconv.Atoi(cfg.Value); err == nil {
			bufferDays = val
		}
	}

	start := time.Now().AddDate(0, 0, bufferDays)
	out := make([]dto.ScheduleOption, 0, days)
	for i := 0; i < days; i++ {
		date := start.AddDate(0, 0, i)
		maxSlots, overbook, booked, err := u.EffectiveCapacity(ctx, ref.TargetHospitalID, ref.TargetDeptID, date)
		if err != nil {
			continue
		}
		available := maxSlots - int(booked)
		if available < 0 {
			available = 0
		}
		if available == 0 {
			continue
		}
		hasOverride := false
		if ov, e := u.overrideRepo.GetActive(ctx, ref.TargetHospitalID, ref.TargetDeptID, date); e == nil && ov != nil {
			hasOverride = true
		}
		out = append(out, dto.ScheduleOption{
			Date:           date.Format("2006-01-02"),
			MaxSlots:       maxSlots,
			BookedSlots:    booked,
			AvailableSlots: available,
			OverbookLimit:  overbook,
			HasOverride:    hasOverride,
		})
	}
	return out, nil
}

// snapshotDailySchedule writes / updates the immutable history log row for
// the given (hospital, department, date). It is called inside the booking
// transaction AFTER the queue has been saved, so the recount reflects the
// new booking. The log is non-critical: any error is logged but does not
// fail the booking.
func (u *schedulingUseCase) snapshotDailySchedule(ctx context.Context, hospitalID, deptID uuid.UUID, date time.Time, maxSlots, overbookLimit int) {
	count, err := u.triageRepo.CountByDeptAndDate(ctx, hospitalID, deptID, date)
	if err != nil {
		log.Printf("snapshotDailySchedule: recount failed: %v", err)
		return
	}

	existing, err := u.scheduleRepo.FindByDeptAndDate(ctx, hospitalID, deptID, date)
	if errors.Is(err, gorm.ErrRecordNotFound) || existing == nil {
		newLog := &entity.DailySchedule{
			HospitalID:    hospitalID,
			DepartmentID:  deptID,
			ScheduleDate:  date,
			BookedSlots:   int(count),
			MaxSlots:      maxSlots,
			OverbookLimit: overbookLimit,
		}
		if e := u.scheduleRepo.CreateLog(ctx, newLog); e != nil {
			log.Printf("snapshotDailySchedule: create log failed: %v", e)
		}
		return
	}
	if err != nil {
		log.Printf("snapshotDailySchedule: find log failed: %v", err)
		return
	}
	if e := u.scheduleRepo.UpdateBookedSlots(ctx, existing.ID, int(count)); e != nil {
		log.Printf("snapshotDailySchedule: update booked failed: %v", e)
	}
}

// refreshOldDateSnapshotIfRescheduled recounts and persists the booked
// snapshot for the OLD appointment date when a triage row is moved to a
// new date. Without this, the daily_schedules row for the old date keeps
// counting the patient that no longer occupies a slot there. A no-op
// when there was no prior date (first booking) or the date is unchanged.
// The recompute pulls effective capacity for the old date so that a
// missing row is recreated with the correct max_slots / overbook_limit
// (e.g. after a manual cleanup wiped daily_schedules historically).
func (u *schedulingUseCase) refreshOldDateSnapshotIfRescheduled(ctx context.Context, hospitalID, deptID uuid.UUID, oldDate *time.Time, newDate time.Time) {
	if oldDate == nil {
		return
	}
	oldDay := oldDate.Truncate(24 * time.Hour)
	newDay := newDate.Truncate(24 * time.Hour)
	if oldDay.Equal(newDay) {
		return
	}
	oldMax, oldOverbook, _, err := u.EffectiveCapacity(ctx, hospitalID, deptID, oldDay)
	if err != nil {
		log.Printf("refreshOldDateSnapshotIfRescheduled: effective capacity failed: %v", err)
		return
	}
	u.snapshotDailySchedule(ctx, hospitalID, deptID, oldDay, oldMax, oldOverbook)
}

// GetCapacityStatus returns a forward-looking capacity view for the given
// hospital/department. Each entry is computed live from
// EffectiveCapacity - no DailySchedule rows are created.
func (u *schedulingUseCase) GetCapacityStatus(ctx context.Context, hospitalID, deptID uuid.UUID, dateRangeDays int) ([]dto.CapacityStatusResponse, error) {
	var resp []dto.CapacityStatusResponse
	for i := 0; i < dateRangeDays; i++ {
		date := time.Now().AddDate(0, 0, i+1)
		maxSlots, overbookLimit, booked, err := u.EffectiveCapacity(ctx, hospitalID, deptID, date)
		if err != nil {
			continue
		}
		resp = append(resp, dto.CapacityStatusResponse{
			Date:          date,
			TotalCapacity: maxSlots,
			OverbookLimit: overbookLimit,
			BookedSlots:   int(booked),
			IsFull:        booked >= int64(maxSlots+overbookLimit),
		})
	}
	return resp, nil
}

// ScheduleAppointment books or reschedules a referral for a specific date
// under the routine (non-emergency) rule: booked < maxSlots. Returns
// wasMissed = true when this booking rescues an earlier no-show.
func (u *schedulingUseCase) ScheduleAppointment(ctx context.Context, referralID, userID uuid.UUID, req dto.SchedulingRequest) (bool, error) {
	ref, err := u.referralRepo.GetReferralByID(ctx, referralID)
	if err != nil {
		return false, err
	}

	if ref.Status != entity.StatusAccepted && ref.Status != entity.StatusScheduled {
		return false, errors.New("only accepted or scheduled referrals can be scheduled")
	}

	if req.AppointmentDate.Before(time.Now().Truncate(24 * time.Hour)) {
		return false, errors.New("appointment date cannot be in the past")
	}

	// Most callers submit a date-only payload (YYYY-MM-DD), which JSON
	// decodes to 00:00 UTC. Patients shouldn't be told "your appointment
	// is at 00:00" - normalize to the clinic's standard 08:00 start so
	// the SMS reads naturally. Any explicit time-of-day from the caller
	// is preserved.
	req.AppointmentDate = normalizeAppointmentTime(req.AppointmentDate)

	queue, err := u.triageRepo.GetByReferralID(ctx, referralID)
	if err != nil {
		return false, err
	}

	if queue.ArrivalStatus != entity.ArrivalExpected && queue.ArrivalStatus != entity.ArrivalMissed {
		return false, errors.New("cannot schedule appointment: patient has already arrived or been admitted")
	}

	wasMissed := queue.ArrivalStatus == entity.ArrivalMissed
	// Remember the OLD appointment date so we can refresh that day's
	// daily_schedules.booked_slots snapshot if this is a reschedule.
	// Without this, a patient moved from X -> Y leaves X's snapshot
	// stale (still counting them as booked on X).
	oldAppointmentDate := queue.AppointmentDate

	maxSlots, overbookLimit, booked, err := u.EffectiveCapacity(ctx, ref.TargetHospitalID, ref.TargetDeptID, req.AppointmentDate)
	if err != nil {
		return false, err
	}
	if booked >= int64(maxSlots) {
		return false, errors.New("capacity reached for this date - emergency override required")
	}

	err = u.db.Transaction(func(tx *gorm.DB) error {
		if wasMissed {
			queue.ArrivalStatus = entity.ArrivalExpected
		}
		queue.AppointmentDate = &req.AppointmentDate
		if err := tx.Save(queue).Error; err != nil {
			return err
		}

		ref.Status = entity.StatusScheduled
		if err := tx.Save(ref).Error; err != nil {
			return err
		}

		if err := u.auditRepo.LogWithContext(ctx, userID, entity.ActionOverrideQueue, &referralID, nil, req); err != nil {
			return err
		}

		u.snapshotDailySchedule(ctx, ref.TargetHospitalID, ref.TargetDeptID, req.AppointmentDate, maxSlots, overbookLimit)
		u.refreshOldDateSnapshotIfRescheduled(ctx, ref.TargetHospitalID, ref.TargetDeptID, oldAppointmentDate, req.AppointmentDate)
		return nil
	})
	if err != nil {
		return false, err
	}

	// Fire SMS + in-app notification AFTER commit. Doing this inside
	// the transaction caused QueueNotification's own GetByReferralID
	// lookup (which uses the parent connection, not `tx`) to read the
	// pre-update row and miss `appointment_date`, leaving {{Date}}
	// literally in the SMS. Post-commit also means we never SMS a
	// patient about a booking that ultimately rolled back.
	notifType := entity.NotifyScheduling
	eventType := "APPOINTMENT_SCHEDULED"
	if wasMissed {
		notifType = entity.NotifyMissedReschedule
		eventType = "MISSED_APPOINTMENT_RESCHEDULED"
	}
	_ = u.notifUC.QueueNotification(ctx, referralID, notifType, "")
	_ = u.inAppNotifUC.CreateForEvent(ctx, eventType, referralID, userID)

	return wasMissed, nil
}

// ManualEmergencySchedule books a referral as an emergency: it is the only
// path that may consume overbook capacity. Requires either a critical
// referral or an explicit justification. Returns wasMissed = true when
// this booking rescues an earlier no-show.
func (u *schedulingUseCase) ManualEmergencySchedule(ctx context.Context, referralID uuid.UUID, appointmentDate time.Time, justification string, userID uuid.UUID) (bool, error) {
	ref, err := u.referralRepo.GetReferralByID(ctx, referralID)
	if err != nil {
		return false, err
	}

	if ref.Status != entity.StatusAccepted && ref.Status != entity.StatusScheduled {
		return false, errors.New("only accepted or scheduled referrals can be emergency-scheduled")
	}

	if appointmentDate.Before(time.Now().Truncate(24 * time.Hour)) {
		return false, errors.New("appointment date cannot be in the past")
	}

	// Same 08:00 default as ScheduleAppointment - the emergency-schedule
	// handler parses YYYY-MM-DD into 00:00 UTC, which is a poor message
	// for the SMS template.
	appointmentDate = normalizeAppointmentTime(appointmentDate)

	queue, err := u.triageRepo.GetByReferralID(ctx, referralID)
	if err != nil {
		return false, err
	}

	if queue.ArrivalStatus != entity.ArrivalExpected && queue.ArrivalStatus != entity.ArrivalMissed {
		return false, errors.New("cannot schedule appointment: patient has already arrived or been admitted")
	}

	if ref.MLSeverityScore == nil {
		defaultScore := 50.0
		ref.MLSeverityScore = &defaultScore
	}

	// Normalize condition_at_referral so "Critical" / "CRITICAL" / " critical "
	// all qualify the same as "critical". The DB value is free-text from the
	// referral form so case + whitespace drift is real.
	isCritical := false
	if ref.ReferralForm != nil {
		cond := strings.ToLower(strings.TrimSpace(ref.ReferralForm.ConditionAtReferral))
		isCritical = cond == "critical"
	}
	if !isCritical && strings.TrimSpace(justification) == "" {
		return false, errors.New("manual emergency schedule requires a critical condition or explicit justification")
	}

	wasMissed := queue.ArrivalStatus == entity.ArrivalMissed
	oldAppointmentDate := queue.AppointmentDate

	maxSlots, overbookLimit, booked, err := u.EffectiveCapacity(ctx, ref.TargetHospitalID, ref.TargetDeptID, appointmentDate)
	if err != nil {
		return false, err
	}
	// Critical patients bypass the max_slots + overbook ceiling entirely:
	// the whole point of "critical" is that the system must accommodate
	// them regardless of normal capacity. Non-critical emergencies still
	// have to fit inside the overbook buffer.
	if !isCritical && booked >= int64(maxSlots+overbookLimit) {
		return false, errors.New("even overbook capacity is full for this date")
	}

	err = u.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if wasMissed {
			queue.ArrivalStatus = entity.ArrivalExpected
		}
		queue.AppointmentDate = &appointmentDate
		if err := tx.Save(queue).Error; err != nil {
			return err
		}

		ref.Status = entity.StatusScheduled
		if err := tx.Save(ref).Error; err != nil {
			return err
		}

		if err := u.auditRepo.LogWithContext(ctx, userID, entity.ActionEmergencySchedule, &referralID, nil, map[string]interface{}{
			"appointment_date":      appointmentDate.Format("2006-01-02"),
			"justification":         justification,
			"is_critical":           isCritical,
			"booked_at_decision":    booked,
			"max_slots":             maxSlots,
			"overbook_limit":        overbookLimit,
			"bypassed_overbook_cap": isCritical && booked >= int64(maxSlots+overbookLimit),
		}); err != nil {
			return err
		}

		u.snapshotDailySchedule(ctx, ref.TargetHospitalID, ref.TargetDeptID, appointmentDate, maxSlots, overbookLimit)
		u.refreshOldDateSnapshotIfRescheduled(ctx, ref.TargetHospitalID, ref.TargetDeptID, oldAppointmentDate, appointmentDate)
		return nil
	})
	if err != nil {
		return false, err
	}

	// Notifications fire AFTER the commit so QueueNotification's own
	// GetByReferralID/GetByReferralID lookups see the new appointment
	// date instead of the pre-update snapshot (which would leave
	// {{Date}} literally in the SMS body).
	notifType := entity.NotifyScheduling
	eventType := "APPOINTMENT_SCHEDULED"
	if wasMissed {
		notifType = entity.NotifyMissedReschedule
		eventType = "MISSED_APPOINTMENT_RESCHEDULED"
	}
	_ = u.notifUC.QueueNotification(ctx, referralID, notifType, "")
	_ = u.inAppNotifUC.CreateForEvent(ctx, eventType, referralID, userID)
	// Emergency bookings consume the overbook buffer; the dept head
	// should know whenever that happens so they can re-evaluate
	// capacity overrides for the affected date.
	_ = u.inAppNotifUC.CreateForEvent(ctx, "EMERGENCY_SCHEDULE_USED", referralID, userID)

	return wasMissed, nil
}

// BatchSchedule walks the waiting queue of a department and books the
// earliest available slot for each patient under the routine rule (no
// overbooking). Each booking re-evaluates capacity through
// EffectiveCapacity, so concurrent emergency bookings are respected.
//
// A soft 5-minute lease is acquired through SchedulerCheckpointRepository
// before any work begins so that an accidental double-click (or a second
// dept head clicking "Run batch" while the first is still in progress)
// does not produce racing bookings. When the lease cannot be acquired
// the call returns a friendly result (no error) with WaitingCount and
// ScheduledCount left at zero and a Message describing the conflict;
// the handler maps that to HTTP 200 so the UI can surface a toast.
func (u *schedulingUseCase) BatchSchedule(ctx context.Context, hospitalID, departmentID, userID uuid.UUID, sendNotifications bool) (*dto.BatchScheduleResult, error) {
	if u.checkpointRepo != nil {
		holder := "manual-batch:" + userID.String()
		ok, lerr := u.checkpointRepo.AcquireLease(ctx, hospitalID, departmentID, holder, 5*time.Minute)
		if lerr != nil {
			return nil, lerr
		}
		if !ok {
			return &dto.BatchScheduleResult{
				Message: "Batch already running for this department; try again in a few minutes",
			}, nil
		}
		defer func() {
			_ = u.checkpointRepo.UpdateLastProcessed(context.Background(), hospitalID, departmentID, holder)
		}()
	}

	waiting, err := u.triageRepo.FindWaitingByHospitalAndDept(ctx, hospitalID, departmentID)
	if err != nil {
		return nil, err
	}

	// ── Composite recompute pass ────────────────────────────────────────
	// CalculateCompositeScore is only run at LandInQueue time, which
	// means the aging bonus is fixed at zero on every stored row. We
	// refresh the score for every waiting row in THIS dept (no others)
	// before booking, so that:
	//   * aging actually moves the queue (e.g. a 30-day waiter outranks
	//     a fresh patient with the same clinical inputs),
	//   * patients that don't get a slot this run keep their fresh
	//     score for the next batch,
	//   * dashboard / list reads see live numbers immediately.
	//
	// Cost: one CalculateCompositeScore per row + N UPDATEs wrapped in a
	// single transaction. For typical dept queue sizes (<100) this adds
	// well under 100ms. The recompute is scoped strictly to the
	// (hospital, department) being batch-processed — no other depts'
	// rows are touched.
	recomputedCount := 0
	scoreDeltaTotal := 0.0
	if u.triageUC != nil && len(waiting) > 0 {
		_ = u.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			for i := range waiting {
				newScore, scErr := u.triageUC.CalculateCompositeScore(ctx, waiting[i].ReferralID)
				if scErr != nil {
					continue
				}
				delta := newScore - waiting[i].CompositeScore
				if upErr := tx.Model(&entity.TriageQueue{}).
					Where("id = ?", waiting[i].ID).
					Update("composite_score", newScore).Error; upErr != nil {
					continue
				}
				waiting[i].CompositeScore = newScore
				scoreDeltaTotal += delta
				recomputedCount++
			}
			return nil
		})
		// Sort in-memory by the fresh scores so the booking loop honors
		// the new ordering without a re-fetch round trip.
		sort.SliceStable(waiting, func(a, b int) bool {
			return waiting[a].CompositeScore > waiting[b].CompositeScore
		})
	}
	// ────────────────────────────────────────────────────────────────────

	dept, err := u.deptRepo.FindHospitalDepartment(ctx, hospitalID, departmentID)
	if err != nil {
		return nil, err
	}

	bufferDays := 2
	if cfg, err := u.configRepo.GetByKey(ctx, "buffer_days"); err == nil && cfg != nil {
		if val, err := strconv.Atoi(cfg.Value); err == nil {
			bufferDays = val
		}
	}

	horizonDays := 30
	if cfg, err := u.configRepo.GetByKey(ctx, "max_horizon_days"); err == nil && cfg != nil {
		if val, err := strconv.Atoi(cfg.Value); err == nil {
			horizonDays = val
		}
	}

	startDate := time.Now().AddDate(0, 0, bufferDays)

	result := &dto.BatchScheduleResult{
		WaitingCount:   len(waiting),
		ScheduledCount: 0,
	}

	for _, q := range waiting {
		for i := 0; i < horizonDays; i++ {
			targetDate := startDate.AddDate(0, 0, i)

			maxSlots, overbookLimit, booked, err := u.EffectiveCapacity(ctx, hospitalID, departmentID, targetDate)
			if err != nil {
				continue
			}

			if booked >= int64(maxSlots) {
				continue
			}

			err = u.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
				q.AppointmentDate = &targetDate
				q.ArrivalStatus = entity.ArrivalExpected
				if err := tx.Save(&q).Error; err != nil {
					return err
				}

				ref, err := u.referralRepo.GetReferralByID(ctx, q.ReferralID)
				if err != nil {
					return err
				}

				oldStatus := ref.Status
				ref.Status = entity.StatusScheduled
				if err := tx.Save(ref).Error; err != nil {
					return err
				}

				if err := u.referralRepo.CreateStatusHistory(ctx, &entity.ReferralStatusHistory{
					ReferralID:  q.ReferralID,
					ChangedByID: userID,
					FromStatus:  &oldStatus,
					ToStatus:    entity.StatusScheduled,
				}); err != nil {
					return err
				}

				u.snapshotDailySchedule(ctx, hospitalID, departmentID, targetDate, maxSlots, overbookLimit)

				if sendNotifications {
					hospitalName := "the hospital"
					deptName := "the department"
					if dept.Hospital.Name != "" {
						hospitalName = dept.Hospital.Name
					}
					if dept.Department.Name != "" {
						deptName = dept.Department.Name
					}
					message := fmt.Sprintf("Your appointment at %s, %s is confirmed for %s.", hospitalName, deptName, targetDate.Format("2006-01-02"))
					_ = u.notifUC.QueueNotification(ctx, q.ReferralID, entity.NotifyScheduling, message)
				}

				return nil
			})

			if err == nil {
				result.ScheduledCount++
				result.WaitingCount--
				break
			}
		}
	}

	u.auditRepo.LogWithContext(ctx, userID, entity.ActionBatchSchedule, nil, nil, map[string]interface{}{
		"hospital_id":       hospitalID,
		"department_id":     departmentID,
		"recomputed_count":  recomputedCount,
		"score_delta_total": scoreDeltaTotal,
		"result":            result,
	})

	if result.ScheduledCount > 0 {
		_ = u.inAppNotifUC.CreateForEvent(ctx, "BATCH_SCHEDULE_COMPLETED", uuid.Nil, userID)
	} else if result.WaitingCount > 0 {
		// The waiting queue had patients but nothing was placed, almost
		// always because every viable date is at capacity. Surface this
		// to the dept head so they can decide whether to file a
		// CapacityOverride for the affected dates.
		_ = u.inAppNotifUC.CreateForEvent(ctx, "BATCH_SCHEDULE_FAILED", uuid.Nil, userID)
	}

	return result, nil
}

// ProcessMissedAppointments is the daily sweep that flips Expected rows
// whose appointment_date is already in the past to Missed. It does NOT
// touch DailySchedule - capacity is recomputed live so missed slots
// become available automatically.
func (u *schedulingUseCase) ProcessMissedAppointments(ctx context.Context) error {
	enabled, err := u.configRepo.GetBool(ctx, "enable_cron_jobs", false)
	if err != nil || !enabled {
		return nil
	}

	today := time.Now().Truncate(24 * time.Hour)

	queues, err := u.triageRepo.FindMissedByDate(ctx, today)
	if err != nil {
		return err
	}

	for _, q := range queues {
		q.ArrivalStatus = entity.ArrivalMissed
		_ = u.triageRepo.Update(ctx, &q)

		_ = u.inAppNotifUC.CreateForEvent(ctx, "PATIENT_MISSED", q.ReferralID, uuid.Nil)
		_ = u.notifUC.QueueNotification(ctx, q.ReferralID, entity.NotifyReschedule, "You have missed your scheduled appointment. Your case has been flagged for review.")
	}

	if u.jobCheckpointRepo != nil {
		_ = u.jobCheckpointRepo.UpdateLastRun(ctx, "missed_appointments", time.Now())
	}

	return nil
}
