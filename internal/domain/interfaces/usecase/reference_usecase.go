package interfaces

import (
	"context"

	"github.com/google/uuid"

	"Hospital-Referral-System/internal/domain/entity"
)

type ReferenceUseCase interface {
	GetHospitals(ctx context.Context, tier string) ([]entity.Hospital, error)
	GetDepartments(ctx context.Context) ([]entity.Department, error)
	ListICDCodes(ctx context.Context, search string, category string, page int, pageSize int) ([]entity.ICDCode, int64, error)
	ListICDCategories(ctx context.Context) ([]string, error)
	GetNetworkedHospitals(ctx context.Context, senderHospitalID uuid.UUID) ([]entity.Hospital, error)
	GetHospitalDepartments(ctx context.Context, hospitalID uuid.UUID) ([]entity.Department, error)
	GetLiaisonsByHospital(ctx context.Context, hospitalID uuid.UUID) ([]entity.User, error)
}
