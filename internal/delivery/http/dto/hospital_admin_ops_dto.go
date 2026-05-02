package dto

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

type HospitalAdminAssignDepartmentHeadRequest struct {
	StaffID string `json:"staff_id" binding:"required" example:"d3000000-0000-0000-0000-000000000003"`
}
