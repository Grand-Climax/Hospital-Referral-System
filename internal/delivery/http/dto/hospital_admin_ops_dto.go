package dto

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

type HospitalAdminUpdateHospitalProfileRequest struct {
	Name         *string `json:"name" example:"Jimma University Medical Center"`
	ContactPhone *string `json:"contact_phone" example:"+251 47 111 1458"`
	Address      *string `json:"address" example:"Jimma, Oromia, Ethiopia"`
}

type HospitalAdminLinkDepartmentRequest struct {
	DepartmentID      string `json:"department_id" example:"b5000000-0000-0000-0000-000000000005"`
	DepartmentIDCamel string `json:"departmentId"`
	ID                string `json:"id"`
	DailyLimit        int    `json:"daily_limit" example:"20"`
	DailyLimitCamel   int    `json:"dailyLimit"`
}

func (r *HospitalAdminLinkDepartmentRequest) ResolvedDepartmentID() string {
	for _, v := range []string{r.DepartmentID, r.DepartmentIDCamel, r.ID} {
		if s := strings.TrimSpace(v); s != "" {
			return s
		}
	}
	return ""
}

func (r *HospitalAdminLinkDepartmentRequest) ResolvedDailyLimit() int {
	if r.DailyLimit > 0 {
		return r.DailyLimit
	}
	if r.DailyLimitCamel > 0 {
		return r.DailyLimitCamel
	}
	return 0
}

// ResolveLinkDepartmentFromBody accepts common JSON shapes for linking a department.
func ResolveLinkDepartmentFromBody(raw map[string]interface{}) (departmentID string, dailyLimit int) {
	req := HospitalAdminLinkDepartmentRequest{
		DepartmentID:      coerceToString(raw["department_id"]),
		DepartmentIDCamel: coerceToString(raw["departmentId"]),
		ID:                coerceToString(raw["id"]),
		DailyLimit:        coerceToInt(raw["daily_limit"]),
		DailyLimitCamel:   coerceToInt(raw["dailyLimit"]),
	}
	if id := req.ResolvedDepartmentID(); id != "" {
		return id, req.ResolvedDailyLimit()
	}
	if nested, ok := raw["department"].(map[string]interface{}); ok {
		if id := strings.TrimSpace(coerceToString(nested["id"])); id != "" {
			return id, req.ResolvedDailyLimit()
		}
	}
	return "", req.ResolvedDailyLimit()
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
	if raw == nil {
		return ""
	}
	for _, key := range staffIDBodyKeys {
		if id := staffIDFromValue(raw[key]); id != "" {
			return id
		}
	}
	return ""
}

var staffIDBodyKeys = []string{
	"staff_id", "staffId",
	"user_id", "userId",
	"head_id", "headId",
	"department_head_id", "departmentHeadId",
	"department_head", "departmentHead",
	"dept_head_id", "deptHeadId",
	"dept_head", "deptHead",
	"head_of_department", "headOfDepartment",
	"head_of_department_id", "headOfDepartmentId",
	"employee_id", "employeeId",
	"selected_user_id", "selectedUserId",
	"personnel_id", "personnelId",
	"assignee_id", "assigneeId",
	"head_user_id", "headUserId",
	"staff_member_id", "staffMemberId",
	"staff", "user", "head",
	"id",
}

var staffIDQueryKeys = []string{
	"staff_id", "staffId",
	"user_id", "userId",
	"head_id", "headId",
	"department_head_id", "departmentHeadId",
	"id",
}

var staffIDNestedObjectKeys = []string{
	"staff", "user", "head",
	"department_head", "departmentHead",
	"dept_head", "deptHead",
	"employee", "personnel", "assignee",
}

var staffIDNestedFieldKeys = []string{
	"id", "staff_id", "staffId", "user_id", "userId",
}

// ResolveStaffID resolves a staff/user id from JSON body bytes and optional query params.
func ResolveStaffID(body []byte, query url.Values) string {
	raw, direct := parseJSONObjectOrUUIDString(body)
	if direct != "" {
		return direct
	}
	if id := ResolveStaffIDFromBody(raw); id != "" {
		return id
	}
	if query != nil {
		for _, key := range staffIDQueryKeys {
			if id := strings.TrimSpace(query.Get(key)); id != "" {
				return id
			}
		}
	}
	return ""
}

func parseJSONObjectOrUUIDString(body []byte) (map[string]interface{}, string) {
	trimmed := bytes.TrimSpace(body)
	if len(trimmed) == 0 {
		return nil, ""
	}

	var asString string
	if err := json.Unmarshal(trimmed, &asString); err == nil {
		if id := strings.TrimSpace(asString); id != "" {
			return map[string]interface{}{}, id
		}
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(trimmed, &raw); err == nil {
		return raw, ""
	}
	return nil, ""
}

func staffIDFromValue(v interface{}) string {
	if v == nil {
		return ""
	}
	if nested, ok := v.(map[string]interface{}); ok {
		for _, field := range staffIDNestedFieldKeys {
			if id := strings.TrimSpace(coerceToString(nested[field])); id != "" {
				return id
			}
		}
		return ""
	}
	return strings.TrimSpace(coerceToString(v))
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

func coerceToInt(v interface{}) int {
	if v == nil {
		return 0
	}
	switch t := v.(type) {
	case int:
		return t
	case int32:
		return int(t)
	case int64:
		return int(t)
	case float64:
		return int(t)
	case json.Number:
		n, _ := t.Int64()
		return int(n)
	case string:
		var n int
		fmt.Sscanf(strings.TrimSpace(t), "%d", &n)
		return n
	default:
		return 0
	}
}

type HospitalAdminPersonnelWidgetResponse struct {
	BaseResponse
	TotalPersonnel int64 `json:"total_personnel" example:"150"`
	ActiveDuty     int64 `json:"active_duty" example:"120"`
	Inactive       int64 `json:"inactive" example:"30"`
	AccessRequests int64 `json:"access_requests" example:"0"`
}
