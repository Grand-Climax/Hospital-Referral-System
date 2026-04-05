package dto

type HospitalResponse struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	TierLevel    string `json:"tier_level"`
	Region       string `json:"region"`
	Address      string `json:"address,omitempty"`
	ContactPhone string `json:"contact_phone,omitempty"`
	IsActive     bool   `json:"is_active"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
	BaseResponse
}

type HospitalListResponse struct {
	Data  []HospitalResponse `json:"data"`
	Total int64              `json:"total"`
	Page  int                `json:"page"`
	BaseResponse
}
