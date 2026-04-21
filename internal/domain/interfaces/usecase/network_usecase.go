package interfaces

import (
	"context"

	"github.com/google/uuid"

	"Hospital-Referral-System/internal/delivery/http/dto"
	"Hospital-Referral-System/internal/domain/entity"
)

type NetworkUseCase interface {
	CreateRoute(ctx context.Context, req dto.CreateNetworkRouteRequest) (*entity.ReferralNetwork, error)
	ListRoutes(ctx context.Context, senderID *uuid.UUID) ([]entity.ReferralNetwork, error)
	DeleteRoute(ctx context.Context, id uuid.UUID) error
}
