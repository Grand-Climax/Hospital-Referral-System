package repository

import (
	"context"

	"gorm.io/gorm"

	"Hospital-Referral-System/internal/domain/entity"
)

type AuthRepository interface {
	BaseRepository[entity.User]
	FindByEmail(ctx context.Context, email string) (*entity.User, error)
	CreateSession(ctx context.Context, session *entity.Session) error
	FindSessionByRefreshTokenHash(ctx context.Context, hash string) (*entity.Session, error)
	UpdateSession(ctx context.Context, session *entity.Session) error
}

type authRepository struct {
	BaseRepository[entity.User]
	db *gorm.DB
}

func NewAuthRepository(db *gorm.DB) AuthRepository {
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
