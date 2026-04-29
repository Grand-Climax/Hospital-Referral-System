package dto

type HospitalAdminUpdateHospitalProfileRequest struct {
	Name         *string `json:"name" example:"Jimma University Medical Center"`
	ContactPhone *string `json:"contact_phone" example:"+251 47 111 1458"`
	Address      *string `json:"address" example:"Jimma, Oromia, Ethiopia"`
}

type HospitalAdminLinkDepartmentRequest struct {
	DepartmentID string `json:"department_id" binding:"required" example:"dfc2b777-a5d5-424b-911a-976b2e8d8614"`
	DailyLimit   int    `json:"daily_limit" example:"20"`
}

type HospitalAdminSetDepartmentActiveRequest struct {
	IsActive *bool `json:"is_active" binding:"required" example:"true"`
}

type HospitalAdminAssignDepartmentHeadRequest struct {
	StaffID string `json:"staff_id" binding:"required" example:"711fd40a-1083-4445-b5bd-aaa2c113de20"`
}
