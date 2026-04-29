package interfaces

import (
	"context"

	"Hospital-Referral-System/internal/domain/entity"
	"github.com/google/uuid"
)

type AuthRepository interface {
	BaseRepository[entity.User]
	FindByEmail(ctx context.Context, email string) (*entity.User, error)
	CreateSession(ctx context.Context, session *entity.Session) error
	FindSessionByRefreshTokenHash(ctx context.Context, hash string) (*entity.Session, error)
	UpdateSession(ctx context.Context, session *entity.Session) error
	ListActiveSessionsByHospital(ctx context.Context, hospitalID uuid.UUID, staffID *uuid.UUID, page, pageSize int) ([]entity.Session, int64, error)
	ListActiveSessionsByUser(ctx context.Context, userID uuid.UUID) ([]entity.Session, error)
	RevokeActiveSessionsByUser(ctx context.Context, userID uuid.UUID) (int64, error)
}
