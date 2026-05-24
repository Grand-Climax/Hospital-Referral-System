package handlers

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"Hospital-Referral-System/internal/delivery/http/dto"
	"Hospital-Referral-System/internal/domain/entity"
)

// parseTriageListFilter builds a dto.TriageListFilter from the query
// string for any of the three role-aware triage list endpoints. The
// caller is responsible for injecting HospitalID and (for dept-head)
// forcing DepartmentID from the JWT scope BEFORE calling the use case.
//
// Validation philosophy: unknown values are silently dropped (not an
// error) so a FE that adds a new filter value never breaks an older
// client. Numeric clamps (limit, page) are applied here to keep handlers
// thin.
func parseTriageListFilter(c *gin.Context) dto.TriageListFilter {
	out := dto.TriageListFilter{}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}
	out.Limit = limit
	out.Offset = (page - 1) * limit

	if deptStr := strings.TrimSpace(c.Query("department_id")); deptStr != "" {
		if id, err := uuid.Parse(deptStr); err == nil {
			out.DepartmentID = &id
		}
	}
	if patStr := strings.TrimSpace(c.Query("patient_id")); patStr != "" {
		if id, err := uuid.Parse(patStr); err == nil {
			out.PatientID = &id
		}
	}
	if nid := strings.TrimSpace(c.Query("national_id")); nid != "" {
		out.NationalID = nid
	}

	out.ArrivalStatuses = parseArrivalStatusCSV(c.Query("arrival_status"))
	out.ReferralStatuses = parseReferralStatusCSV(c.Query("referral_status"))

	if hd := strings.TrimSpace(c.Query("has_doctor_assigned")); hd != "" {
		if b, err := strconv.ParseBool(hd); err == nil {
			out.HasDoctorAssigned = &b
		}
	}

	if it := strings.TrimSpace(c.Query("include_terminal")); it != "" {
		if b, err := strconv.ParseBool(it); err == nil {
			out.IncludeTerminal = b
		}
	}

	out.SortBy = normalizeSortBy(c.Query("sort_by"))
	out.SortOrder = normalizeSortOrder(c.Query("sort_order"))

	return out
}

// validArrivalStatuses is the whitelist tolerated on the wire. Anything
// outside this set is dropped rather than causing a 400 so older clients
// keep working when the enum grows.
var validArrivalStatuses = map[string]entity.ArrivalStatus{
	"EXPECTED": entity.ArrivalExpected,
	"ARRIVED":  entity.ArrivalArrived,
	"ADMITTED": entity.ArrivalAdmitted,
	"MISSED":   entity.ArrivalMissed,
}

func parseArrivalStatusCSV(raw string) []entity.ArrivalStatus {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	seen := make(map[entity.ArrivalStatus]struct{}, len(parts))
	out := make([]entity.ArrivalStatus, 0, len(parts))
	for _, p := range parts {
		key := strings.ToUpper(strings.TrimSpace(p))
		if status, ok := validArrivalStatuses[key]; ok {
			if _, dup := seen[status]; !dup {
				seen[status] = struct{}{}
				out = append(out, status)
			}
		}
	}
	return out
}

// validReferralStatuses is intentionally limited to the two values an
// active triage queue cares about; terminal statuses are handled via the
// include_terminal flag instead.
var validReferralStatuses = map[string]entity.ReferralStatus{
	"ACCEPTED":  entity.StatusAccepted,
	"SCHEDULED": entity.StatusScheduled,
}

func parseReferralStatusCSV(raw string) []entity.ReferralStatus {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	seen := make(map[entity.ReferralStatus]struct{}, len(parts))
	out := make([]entity.ReferralStatus, 0, len(parts))
	for _, p := range parts {
		key := strings.ToUpper(strings.TrimSpace(p))
		if status, ok := validReferralStatuses[key]; ok {
			if _, dup := seen[status]; !dup {
				seen[status] = struct{}{}
				out = append(out, status)
			}
		}
	}
	return out
}

func normalizeSortBy(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "composite_score", "appointment_date", "created_at":
		return strings.ToLower(strings.TrimSpace(raw))
	default:
		return "composite_score"
	}
}

func normalizeSortOrder(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "asc":
		return "asc"
	default:
		return "desc"
	}
}

// pageFromOffset converts (limit, offset) back to a 1-based page number
// for the response envelope. Returns 1 when limit is non-positive.
func pageFromOffset(limit, offset int) int {
	if limit <= 0 {
		return 1
	}
	return offset/limit + 1
}

// hasMorePage reports whether another page exists given the current
// offset, page size, and total count.
func hasMorePage(offset, limit int, total int64) bool {
	if limit <= 0 {
		return false
	}
	return int64(offset+limit) < total
}
