package interfaces

import (
	"context"
	"time"

	"github.com/google/uuid"

	"Hospital-Referral-System/internal/domain/entity"
)

// TriageQueueFilter is the parameter bag for the role-aware triage queue
// list endpoints (specialist, dept-head, receptionist). All fields are
// optional except HospitalID; nil/empty values mean "no filter".
//
// IncludeTerminal=false (the default) excludes referrals whose Status
// is one of: COMPLETED, DECEASED, CANCELLED, REJECTED_BY_LIAISON,
// REJECTED_BY_SPECIALIST, REJECTED_AFTER_SEND, REDIRECTED. Active
// queue views should leave this false; audit views may pass true.
//
// SortBy must be one of "composite_score", "appointment_date",
// "created_at". SortOrder must be "asc" or "desc". The implementation
// applies a whitelist and falls back to composite_score DESC on bad
// input so SQL injection via the sort column is impossible.
type TriageQueueFilter struct {
	HospitalID        uuid.UUID
	DepartmentID      *uuid.UUID
	ArrivalStatuses   []entity.ArrivalStatus
	ReferralStatuses  []entity.ReferralStatus
	PatientID         *uuid.UUID
	NationalIDHash    *string
	HasDoctorAssigned *bool
	IncludeTerminal   bool
	SortBy            string
	SortOrder         string
	Limit             int
	Offset            int
}

// TriageQueueRepository manages the triage priority queue for incoming referrals.
type TriageQueueRepository interface {
	BaseRepository[entity.TriageQueue]
	GetByReferralID(ctx context.Context, referralID uuid.UUID) (*entity.TriageQueue, error)
	DeleteByReferralID(ctx context.Context, referralID uuid.UUID) error
	ListForTriage(ctx context.Context, hospitalID uuid.UUID, limit, offset int) ([]entity.TriageQueue, int64, error)

	// ListTriageQueueFiltered is the single query path behind the
	// role-aware triage list endpoints. It joins referrals + patients,
	// preloads Department + AssignedDoctor + ReferralForm, applies the
	// filter, and returns the page plus the total count.
	ListTriageQueueFiltered(ctx context.Context, filter TriageQueueFilter) ([]entity.TriageQueue, int64, error)
	ListScheduledInRange(ctx context.Context, hospitalID, deptID uuid.UUID, start, end time.Time) ([]entity.TriageQueue, error)
	GetWaitingByDept(ctx context.Context, hospitalID, deptID uuid.UUID) ([]entity.TriageQueue, error)
	FindWaitingByHospitalAndDept(ctx context.Context, hospitalID, departmentID uuid.UUID) ([]entity.TriageQueue, error)
	FindScheduledByHospitalAndDept(ctx context.Context, hospitalID, deptID uuid.UUID, startDate, endDate time.Time) ([]*entity.TriageQueue, error)
	FindByHospitalAndDept(ctx context.Context, hospitalID, deptID uuid.UUID, limit, offset int) ([]*entity.TriageQueue, int64, error)
	FindAppointmentsForReminders(ctx context.Context, date time.Time) ([]*entity.TriageQueue, error)
	IncrementWaitingWeights(ctx context.Context) (int64, error)
	ListMissedByHospital(ctx context.Context, hospitalID uuid.UUID, limit, offset int) ([]*entity.TriageQueue, int64, error)
	FindMissedByDate(ctx context.Context, beforeDate time.Time) ([]entity.TriageQueue, error)

	// CountByDeptAndDate returns the live count of triage queue rows booked
	// for the given (hospital, department, date) that still occupy a slot.
	// Slots are occupied while the patient is Expected/Arrived/Admitted;
	// Missed and Completed rows are excluded so that a missed appointment
	// frees its capacity for a same-day re-book.
	CountByDeptAndDate(ctx context.Context, hospitalID, deptID uuid.UUID, date time.Time) (int64, error)

	// CountAssignedDoctorsByDeptAndDate returns the distinct
	// assigned_doctor_id count on the triage queue for (hospital,
	// department, date), considered only for Arrived/Admitted rows whose
	// referral is in SCHEDULED status. Purely advisory ("staff_assigned"
	// metric on capacity views).
	CountAssignedDoctorsByDeptAndDate(ctx context.Context, hospitalID, deptID uuid.UUID, date time.Time) (int64, error)

	// FindScheduledByDeptAndDate returns the triage rows scheduled for
	// the given date with Referral + Patient eagerly loaded. Matches the
	// same Expected/Arrived/Admitted set as CountByDeptAndDate so the
	// "scheduled patients" list and the capacity counter agree.
	FindScheduledByDeptAndDate(ctx context.Context, hospitalID, deptID uuid.UUID, date time.Time) ([]entity.TriageQueue, error)

	// CountMissedByDeptInRange returns the number of triage rows whose
	// appointment_date falls inside [start, end] and whose ArrivalStatus
	// is MISSED. Used by the dept-head dashboard to surface a missed-
	// appointment KPI without scanning the whole table.
	CountMissedByDeptInRange(ctx context.Context, hospitalID, deptID uuid.UUID, start, end time.Time) (int64, error)

	// CountScheduledByDeptInRange returns the number of triage rows whose
	// appointment_date falls inside [start, end] and whose ArrivalStatus
	// is Expected/Arrived/Admitted (i.e. still counts toward capacity).
	CountScheduledByDeptInRange(ctx context.Context, hospitalID, deptID uuid.UUID, start, end time.Time) (int64, error)

	// OldestWaitingDaysByDept returns the integer number of days since the
	// oldest still-waiting (appointment_date IS NULL, arrival_status =
	// EXPECTED) triage row in the dept was created. 0 when the queue is
	// empty.
	OldestWaitingDaysByDept(ctx context.Context, hospitalID, deptID uuid.UUID) (int, error)
}
