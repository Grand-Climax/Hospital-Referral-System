package usecase

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"Hospital-Referral-System/internal/delivery/http/dto"
	"Hospital-Referral-System/internal/domain/entity"
	irepository "Hospital-Referral-System/internal/domain/interfaces/repository"
	iusecase "Hospital-Referral-System/internal/domain/interfaces/usecase"
)

type networkUseCase struct {
	networkRepo  irepository.NetworkRepository
	hospitalRepo irepository.HospitalRepository
}

func NewNetworkUseCase(repo irepository.NetworkRepository, hospitalRepo irepository.HospitalRepository) iusecase.NetworkUseCase {
	return &networkUseCase{networkRepo: repo, hospitalRepo: hospitalRepo}
}

func (u *networkUseCase) CreateRoute(ctx context.Context, req dto.CreateNetworkRouteRequest) (*entity.ReferralNetwork, error) {
	// Self-referencing validation
	if req.SenderHospitalID == req.ReceiverHospitalID {
		return nil, errors.New("cannot create a self-referring network route")
	}

	// Tier-based validation: PRIMARY hospitals can only route to SECONDARY
	if u.hospitalRepo != nil {
		senderRaw, err := u.hospitalRepo.FindByID(ctx, req.SenderHospitalID)
		if err != nil || senderRaw == nil {
			return nil, errors.New("sender hospital not found")
		}
		if senderRaw.TierLevel == entity.PrimaryHosp {
			receiverRaw, err := u.hospitalRepo.FindByID(ctx, req.ReceiverHospitalID)
			if err != nil || receiverRaw == nil {
				return nil, errors.New("receiver hospital not found")
			}
			if receiverRaw.TierLevel != entity.SecondaryHosp {
				return nil, errors.New("primary hospitals can only create networks to secondary hospitals")
			}
		}
	}

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
