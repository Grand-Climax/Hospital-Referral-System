package interfaces

import (
	"context"

	"Hospital-Referral-System/internal/pkg/auth"
)

type AuthUseCase interface {
	Login(ctx context.Context, email, password, ipAddress, userAgent string) (*auth.TokenPair, error)
	Refresh(ctx context.Context, refreshToken, ipAddress, userAgent string) (*auth.TokenPair, error)
	Logout(ctx context.Context, accessToken, refreshToken string) error
}
