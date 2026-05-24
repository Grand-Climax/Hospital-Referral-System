package interfaces

import (
	"context"
	"time"

	"github.com/google/uuid"

	"Hospital-Referral-System/internal/delivery/http/dto"
)

// DepartmentHeadDashboardUseCase aggregates the read-only views that
// power the dept-head landing page and its sidebar widgets. It owns no
// state of its own - every method composes existing repositories and
// use cases (capacity manager, scheduling, triage, user, audit), so it
// stays consistent with the booking engine and the immutability rules
// from v16 / v17.
type DepartmentHeadDashboardUseCase interface {
	// GetDashboardStats returns the landing-page summary: today's
	// capacity, queue size, scheduled today / next 7 days, missed last
	// 7 days, active staff, active overrides, pending referrals, and a
	// per-status breakdown for the inbound referrals targeted at the
	// caller's department.
	GetDashboardStats(ctx context.Context, hospitalID, deptID uuid.UUID) (*dto.DepartmentHeadDashboardStats, error)

	// GetTrends returns one DepartmentHeadTrendPoint per day in the
	// window ending today (inclusive) and going back `days` days
	// (1 <= days <= 90). Used to render the "capacity utilization"
	// line chart.
	GetTrends(ctx context.Context, hospitalID, deptID uuid.UUID, days int) ([]dto.DepartmentHeadTrendPoint, error)

	// GetPriorityBuckets breaks the dept's waiting queue down by clinical
	// priority (CRITICAL / URGENT / STABLE from the referral form) and
	// by ML severity tier, plus the top 5 highest-composite-score
	// patients still waiting.
	GetPriorityBuckets(ctx context.Context, hospitalID, deptID uuid.UUID) (*dto.PriorityBucketResponse, error)

	// GetStaffSummary returns active/inactive counts split by role for
	// the caller's department, the soft staff-capacity hint, and the
	// distinct count of doctors with bookings for today.
	GetStaffSummary(ctx context.Context, hospitalID, deptID uuid.UUID) (*dto.StaffSummaryResponse, error)

	// GetActivity returns recent audit-log rows scoped to the caller's
	// hospital and a curated list of dept-head-relevant action types
	// (batch scheduling, override mutations, emergency scheduling,
	// arrival confirmations, doctor assignments, etc.). startDate /
	// endDate are inclusive yyyy-mm-dd dates and are optional.
	GetActivity(ctx context.Context, hospitalID, deptID uuid.UUID, limit int, startDate, endDate *time.Time) ([]dto.DepartmentHeadActivityItem, error)
}
