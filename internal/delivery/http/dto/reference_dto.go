package dto

import (
	"Hospital-Referral-System/internal/domain/entity"
)


type ICDCodeListResponse struct {
	Data []entity.ICDCode `json:"data"`
	BaseResponse
}

type PaginatedICDCodeResponse struct {
	BaseResponse
	Data     []entity.ICDCode `json:"data"`
	Total    int64            `json:"total"`
	Page     int              `json:"page"`
	PageSize int              `json:"page_size"`
}

type LiaisonItem struct {
	ID        string `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Email     string `json:"email"`
}

type LiaisonListResponse struct {
	Data []LiaisonItem `json:"data"`
	BaseResponse
}

type RegionListResponse struct {
	Data []string `json:"data"`
	BaseResponse
}

