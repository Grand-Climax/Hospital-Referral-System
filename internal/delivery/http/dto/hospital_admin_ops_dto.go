package dto

import (
	"encoding/json"
	"fmt"
	"strings"
)

type HospitalAdminUpdateHospitalProfileRequest struct {
	Name         *string `json:"name" example:"Jimma University Medical Center"`
	ContactPhone *string `json:"contact_phone" example:"+251 47 111 1458"`
	Address      *string `json:"address" example:"Jimma, Oromia, Ethiopia"`
}

type HospitalAdminLinkDepartmentRequest struct {
	DepartmentID string `json:"department_id" binding:"required" example:"b5000000-0000-0000-0000-000000000005"`
	DailyLimit   int    `json:"daily_limit" example:"20"`
}

type HospitalAdminSetDepartmentActiveRequest struct {
	IsActive      *bool `json:"is_active" example:"true"`
	IsActiveCamel *bool `json:"isActive"`
}

func (r *HospitalAdminSetDepartmentActiveRequest) ResolvedIsActive() *bool {
	if r.IsActive != nil {
		return r.IsActive
	}
	return r.IsActiveCamel
}

// HospitalAdminAssignDepartmentHeadRequest accepts staff id under several JSON keys
// (snake_case is canonical; camelCase is supported for frontend clients).
type HospitalAdminAssignDepartmentHeadRequest struct {
	StaffID               string `json:"staff_id" example:"d3000000-0000-0000-0000-000000000003"`
	StaffIDCamel          string `json:"staffId"`
	UserID                string `json:"user_id"`
	UserIDCamel           string `json:"userId"`
	HeadID                string `json:"head_id"`
	HeadIDCamel           string `json:"headId"`
	DepartmentHeadID      string `json:"department_head_id"`
	DepartmentHeadIDCamel string `json:"departmentHeadId"`
	ID                    string `json:"id"`
}

// ResolvedStaffID returns the first non-empty staff/user id from the request body.
func (r *HospitalAdminAssignDepartmentHeadRequest) ResolvedStaffID() string {
	for _, v := range []string{
		r.StaffID, r.StaffIDCamel,
		r.UserID, r.UserIDCamel,
		r.HeadID, r.HeadIDCamel,
		r.DepartmentHeadID, r.DepartmentHeadIDCamel,
		r.ID,
	} {
		if s := strings.TrimSpace(v); s != "" {
			return s
		}
	}
	return ""
}

// ResolveStaffIDFromBody accepts common JSON shapes for department-head assignment.
func ResolveStaffIDFromBody(raw map[string]interface{}) string {
	req := HospitalAdminAssignDepartmentHeadRequest{
		StaffID:               coerceToString(raw["staff_id"]),
		StaffIDCamel:          coerceToString(raw["staffId"]),
		UserID:                coerceToString(raw["user_id"]),
		UserIDCamel:           coerceToString(raw["userId"]),
		HeadID:                coerceToString(raw["head_id"]),
		HeadIDCamel:           coerceToString(raw["headId"]),
		DepartmentHeadID:      coerceToString(raw["department_head_id"]),
		DepartmentHeadIDCamel: coerceToString(raw["departmentHeadId"]),
		ID:                    coerceToString(raw["id"]),
	}
	if id := req.ResolvedStaffID(); id != "" {
		return id
	}
	for _, nestedKey := range []string{"staff", "user", "head"} {
		if nested, ok := raw[nestedKey].(map[string]interface{}); ok {
			if id := strings.TrimSpace(coerceToString(nested["id"])); id != "" {
				return id
			}
		}
	}
	return ""
}

func coerceToString(v interface{}) string {
	if v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return strings.TrimSpace(t)
	case json.Number:
		return strings.TrimSpace(t.String())
	case fmt.Stringer:
		return strings.TrimSpace(t.String())
	default:
		return strings.TrimSpace(fmt.Sprint(v))
	}
}

type HospitalAdminPersonnelWidgetResponse struct {
	BaseResponse
	TotalPersonnel int64 `json:"total_personnel" example:"150"`
	ActiveDuty     int64 `json:"active_duty" example:"120"`
	Inactive       int64 `json:"inactive" example:"30"`
	AccessRequests int64 `json:"access_requests" example:"0"`
}
