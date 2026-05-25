package dto

type DepartmentResponse struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
	BaseResponse
}

type DepartmentHeadSummary struct {
	ID       string `json:"id"`
	FullName string `json:"full_name"`
}

type HospitalDepartmentResponse struct {
	ID                 string                 `json:"id"`
	HospitalID         string                 `json:"hospital_id"`
	DepartmentID       string                 `json:"department_id"`
	Department         DepartmentResponse     `json:"department"`
	DepartmentHead     *DepartmentHeadSummary `json:"department_head,omitempty"`
	StandardDailyLimit int                    `json:"standard_daily_limit"`
	IsActive           bool                   `json:"is_active"`
	CreatedAt          string                 `json:"created_at"`
	BaseResponse
}

type DepartmentListResponse struct {
	Data  []DepartmentResponse `json:"data"`
	Total int64                `json:"total"`
	Page  int                  `json:"page"`
	BaseResponse
}

type HospitalDepartmentListResponse struct {
	Data []HospitalDepartmentResponse `json:"data"`
	BaseResponse
}
