package interfaces

import (
	"context"

	"Hospital-Referral-System/internal/pkg/auth"
	"github.com/google/uuid"
)

type LoginResult struct {
	MFAToken         string
	MFASetupRequired bool
}

type AuthUseCase interface {
	Login(ctx context.Context, email, password string) (*LoginResult, error)
	VerifyMFA(ctx context.Context, userID uuid.UUID, code, ipAddress, userAgent string) (*auth.TokenPair, error)
	SetupMFA(ctx context.Context, userID uuid.UUID) (string, error)
	ResetMFA(ctx context.Context, adminUserID, targetUserID uuid.UUID) error
	Refresh(ctx context.Context, refreshToken, ipAddress, userAgent string) (*auth.TokenPair, error)
	Logout(ctx context.Context, accessToken, refreshToken string) error
}
