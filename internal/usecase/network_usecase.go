package usecase

import (
	"context"

	"github.com/google/uuid"

	"Hospital-Referral-System/internal/delivery/http/dto"
	"Hospital-Referral-System/internal/domain/entity"
	irepository "Hospital-Referral-System/internal/domain/interfaces/repository"
	iusecase "Hospital-Referral-System/internal/domain/interfaces/usecase"
)

type networkUseCase struct {
	networkRepo irepository.NetworkRepository
}

func NewNetworkUseCase(repo irepository.NetworkRepository) iusecase.NetworkUseCase {
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
