package repository

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"Hospital-Referral-System/internal/domain/entity"
	irepository "Hospital-Referral-System/internal/domain/interfaces/repository"
)

type networkRepository struct {
	db *gorm.DB
}

func NewNetworkRepository(db *gorm.DB) irepository.NetworkRepository {
	return &networkRepository{db: db}
}

func (r *networkRepository) CreateNetworkRoute(ctx context.Context, route *entity.ReferralNetwork) error {
	return r.db.WithContext(ctx).Create(route).Error
}

func (r *networkRepository) ListNetworkRoutes(ctx context.Context, senderID *uuid.UUID) ([]entity.ReferralNetwork, error) {
	var routes []entity.ReferralNetwork
	query := r.db.WithContext(ctx).Preload("SenderHospital").Preload("ReceiverHospital")

	if senderID != nil && *senderID != uuid.Nil {
		query = query.Where("sender_hospital_id = ?", *senderID)
	}

	err := query.Find(&routes).Error
	return routes, err
}

func (r *networkRepository) VerifyNetworkPathway(ctx context.Context, senderID, targetID uuid.UUID) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&entity.ReferralNetwork{}).
		Where("sender_hospital_id = ? AND receiver_hospital_id = ?", senderID, targetID).
		Count(&count).Error
	
	return count > 0, err
}

func (r *networkRepository) GetOutgoingNetworkHospitals(ctx context.Context, senderID uuid.UUID) ([]entity.Hospital, error) {
	var hospitals []entity.Hospital
	err := r.db.WithContext(ctx).
		Joins("JOIN referral_networks ON hospitals.id = referral_networks.receiver_hospital_id").
		Where("referral_networks.sender_hospital_id = ?", senderID).
		Find(&hospitals).Error
	return hospitals, err
}

func (r *networkRepository) DeleteNetworkRoute(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&entity.ReferralNetwork{}, id).Error
}
