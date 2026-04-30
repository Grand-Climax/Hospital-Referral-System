package dto

type CreateOverrideRequest struct {
	Date     string `json:"target_date" binding:"required"`
	NewLimit int    `json:"new_limit" binding:"required,min=0"`
	Reason   string `json:"reason" binding:"required"`
}

type UpdateOverrideRequest struct {
	NewLimit int    `json:"new_limit" binding:"required,min=0"`
	Reason   string `json:"reason" binding:"required"`
}

type CapacityOverrideResponse struct {
	ID         string `json:"id"`
	TargetDate string `json:"target_date"`
	NewLimit   int    `json:"new_limit"`
	Reason     string `json:"reason"`
	IsActive   bool   `json:"is_active"`
}
