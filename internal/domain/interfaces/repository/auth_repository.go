package repository

import (
	"context"

	"Hospital-Referral-System/internal/domain/entity"
)

type AuthRepository interface {
	BaseRepository[entity.User]
	FindByEmail(ctx context.Context, email string) (*entity.User, error)
	CreateSession(ctx context.Context, session *entity.Session) error
	FindSessionByRefreshTokenHash(ctx context.Context, hash string) (*entity.Session, error)
	UpdateSession(ctx context.Context, session *entity.Session) error
}
