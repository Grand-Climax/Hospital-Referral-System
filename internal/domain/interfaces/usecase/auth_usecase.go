package interfaces

import (
	"context"

	"Hospital-Referral-System/internal/pkg/auth"
	"github.com/google/uuid"
)

type LoginResult struct {
	MFAToken     string
	Channel      string
	AccessToken  string
	RefreshToken string
}

type PasswordResetVerifyResult struct {
	ResetToken string
}

type AuthUseCase interface {
	Login(ctx context.Context, email, password, requestedChannel string) (*LoginResult, error)
	VerifyOTP(ctx context.Context, userID uuid.UUID, code, ipAddress, userAgent string) (*auth.TokenPair, error)
	Refresh(ctx context.Context, refreshToken, ipAddress, userAgent string) (*auth.TokenPair, error)
	Logout(ctx context.Context, accessToken, refreshToken string) error
	ChangePassword(ctx context.Context, userID uuid.UUID, currentPassword, newPassword, ipAddress, userAgent string) (*auth.TokenPair, error)
	ForgotPassword(ctx context.Context, email string) error
	VerifyForgotPasswordOTP(ctx context.Context, email, code string) (*PasswordResetVerifyResult, error)
	ResetPassword(ctx context.Context, userID uuid.UUID, newPassword, ipAddress, userAgent string) (*auth.TokenPair, error)
}
