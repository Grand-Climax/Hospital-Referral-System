package interfaces

import (
	"context"

	"github.com/google/uuid"
)

type DailyWeightUseCase interface {
	Execute(ctx context.Context, userID uuid.UUID) (string, error)
}
