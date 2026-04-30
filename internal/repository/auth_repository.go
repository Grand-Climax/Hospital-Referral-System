package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"Hospital-Referral-System/internal/domain/entity"
	irepository "Hospital-Referral-System/internal/domain/interfaces/repository"
)

type authRepository struct {
	*BaseRepository[entity.User]
	db *gorm.DB
}

func NewAuthRepository(db *gorm.DB) irepository.AuthRepository {
	return &authRepository{
		BaseRepository: NewBaseRepository[entity.User](db),
		db:             db,
	}
}

func (r *authRepository) FindByEmail(ctx context.Context, email string) (*entity.User, error) {
	var user entity.User
	err := r.db.WithContext(ctx).Where("email = ? AND is_active = true AND is_deleted = false", email).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *authRepository) CreateSession(ctx context.Context, session *entity.Session) error {
	return r.db.WithContext(ctx).Create(session).Error
}

func (r *authRepository) FindSessionByRefreshTokenHash(ctx context.Context, hash string) (*entity.Session, error) {
	var session entity.Session
	err := r.db.WithContext(ctx).Where("refresh_token_hash = ?", hash).First(&session).Error
	if err != nil {
		return nil, err
	}
	return &session, nil
}

func (r *authRepository) UpdateSession(ctx context.Context, session *entity.Session) error {
	return r.db.WithContext(ctx).Save(session).Error
}

func (r *authRepository) ListActiveSessionsByHospital(ctx context.Context, hospitalID uuid.UUID, staffID *uuid.UUID, page, pageSize int) ([]entity.Session, int64, error) {
	var sessions []entity.Session
	var total int64

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	query := r.db.WithContext(ctx).Model(&entity.Session{}).
		Joins("JOIN users ON users.id = sessions.user_id").
		Where("users.hospital_id = ? AND users.is_deleted = false", hospitalID).
		Where("sessions.revoked_at IS NULL AND sessions.expires_at > ?", time.Now())

	if staffID != nil && *staffID != uuid.Nil {
		query = query.Where("sessions.user_id = ?", *staffID)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Order("sessions.created_at DESC").Offset(offset).Limit(pageSize).Find(&sessions).Error; err != nil {
		return nil, 0, err
	}
	return sessions, total, nil
}

func (r *authRepository) ListActiveSessionsByUser(ctx context.Context, userID uuid.UUID) ([]entity.Session, error) {
	var sessions []entity.Session
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND revoked_at IS NULL AND expires_at > ?", userID, time.Now()).
		Order("created_at DESC").
		Find(&sessions).Error
	return sessions, err
}

func (r *authRepository) RevokeActiveSessionsByUser(ctx context.Context, userID uuid.UUID) (int64, error) {
	now := time.Now()
	tx := r.db.WithContext(ctx).Model(&entity.Session{}).
		Where("user_id = ? AND revoked_at IS NULL AND expires_at > ?", userID, now).
		Update("revoked_at", now)
	if tx.Error != nil {
		return 0, tx.Error
	}
	return tx.RowsAffected, nil
}
