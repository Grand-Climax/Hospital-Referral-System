package interfaces

import (
	"context"

	"github.com/google/uuid"

	"Hospital-Referral-System/internal/domain/entity"
)

type NetworkRepository interface {
	CreateNetworkRoute(ctx context.Context, route *entity.ReferralNetwork) error
	ListNetworkRoutes(ctx context.Context, senderID *uuid.UUID) ([]entity.ReferralNetwork, error)
	VerifyNetworkPathway(ctx context.Context, senderID, targetID uuid.UUID) (bool, error)
	DeleteNetworkRoute(ctx context.Context, id uuid.UUID) error
}
