package dto

import "strings"

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
	IsActive *bool `json:"is_active" binding:"required" example:"true"`
}

// HospitalAdminAssignDepartmentHeadRequest accepts staff id under several JSON keys
// (snake_case is canonical; camelCase is supported for frontend clients).
type HospitalAdminAssignDepartmentHeadRequest struct {
	StaffID      string `json:"staff_id" example:"d3000000-0000-0000-0000-000000000003"`
	StaffIDCamel string `json:"staffId"`
	UserID       string `json:"user_id"`
	UserIDCamel  string `json:"userId"`
}

// ResolvedStaffID returns the first non-empty staff/user id from the request body.
func (r *HospitalAdminAssignDepartmentHeadRequest) ResolvedStaffID() string {
	for _, v := range []string{r.StaffID, r.StaffIDCamel, r.UserID, r.UserIDCamel} {
		if s := strings.TrimSpace(v); s != "" {
			return s
		}
	}
	return ""
}

type HospitalAdminPersonnelWidgetResponse struct {
	BaseResponse
	TotalPersonnel int64 `json:"total_personnel" example:"150"`
	ActiveDuty     int64 `json:"active_duty" example:"120"`
	Inactive       int64 `json:"inactive" example:"30"`
	AccessRequests int64 `json:"access_requests" example:"0"`
}

