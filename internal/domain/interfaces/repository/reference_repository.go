package repository

import (
	"context"

	"github.com/google/uuid"

	"Hospital-Referral-System/internal/domain/entity"
)

type ReferenceRepository interface {
	GetHospitals(ctx context.Context, tier string) ([]entity.Hospital, error)
	GetDepartments(ctx context.Context) ([]entity.Department, error)
	ListICDCodes(ctx context.Context) ([]entity.ICDCode, error)
	GetNetworkedHospitals(ctx context.Context, senderHospitalID uuid.UUID) ([]entity.Hospital, error)
	GetHospitalDepartments(ctx context.Context, hospitalID uuid.UUID) ([]entity.Department, error)
	GetLiaisonsByHospital(ctx context.Context, hospitalID uuid.UUID) ([]entity.User, error)
}
