package interfaces

import (
	"context"

	"github.com/google/uuid"
)

type AdminConfigUseCase interface {
	GetConfig(ctx context.Context) (map[string]string, error)
	UpdateConfig(ctx context.Context, updates map[string]string, userID uuid.UUID) error
}
