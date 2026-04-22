package interfaces

import (
	"context"

	"Hospital-Referral-System/internal/domain/entity"
)

type HospitalListFilter struct {
	Page     int
	PageSize int
	Tier     *entity.HospitalTier
	Region   *string
	IsActive *bool
	Search   *string
}

type HospitalRepository interface {
	BaseRepository[entity.Hospital]
	ListHospitals(ctx context.Context, filter HospitalListFilter) ([]entity.Hospital, int64, error)
}
