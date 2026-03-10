package usecase

import (
	"context"

	"github.com/google/uuid"

	"Hospital-Referral-System/internal/delivery/http/dto"
	"Hospital-Referral-System/internal/domain/entity"
	"Hospital-Referral-System/internal/repository"
)

type NetworkUseCase interface {
	CreateRoute(ctx context.Context, req dto.CreateNetworkRouteRequest) (*entity.ReferralNetwork, error)
	ListRoutes(ctx context.Context, senderID *uuid.UUID) ([]entity.ReferralNetwork, error)
	DeleteRoute(ctx context.Context, id uuid.UUID) error
}

type networkUseCase struct {
	networkRepo repository.NetworkRepository
}

func NewNetworkUseCase(repo repository.NetworkRepository) NetworkUseCase {
	return &networkUseCase{networkRepo: repo}
}

func (u *networkUseCase) CreateRoute(ctx context.Context, req dto.CreateNetworkRouteRequest) (*entity.ReferralNetwork, error) {
	route := &entity.ReferralNetwork{
		SenderHospitalID:      req.SenderHospitalID,
		ReceiverHospitalID:    req.ReceiverHospitalID,
		ReferralType:          req.ReferralType,
		RequiresAdminApproval: req.RequiresAdminApproval,
	}

	if route.ReferralType == "" {
		route.ReferralType = "routine"
	}

	if err := u.networkRepo.CreateNetworkRoute(ctx, route); err != nil {
		return nil, err
	}
	return route, nil
}

func (u *networkUseCase) ListRoutes(ctx context.Context, senderID *uuid.UUID) ([]entity.ReferralNetwork, error) {
	return u.networkRepo.ListNetworkRoutes(ctx, senderID)
}

func (u *networkUseCase) DeleteRoute(ctx context.Context, id uuid.UUID) error {
	return u.networkRepo.DeleteNetworkRoute(ctx, id)
}
