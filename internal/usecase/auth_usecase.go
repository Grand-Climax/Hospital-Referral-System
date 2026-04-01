package usecase

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	"Hospital-Referral-System/internal/domain/entity"
	irepository "Hospital-Referral-System/internal/domain/interfaces/repository"
	iusecase "Hospital-Referral-System/internal/domain/interfaces/usecase"
	"Hospital-Referral-System/internal/infrastructure/cache"
	"Hospital-Referral-System/internal/pkg/auth"
)

type authUseCase struct {
	repo      irepository.AuthRepository
	blacklist cache.TokenBlacklist
	sessions  cache.SessionStore
}

func NewAuthUseCase(repo irepository.AuthRepository, blacklist cache.TokenBlacklist, sessions cache.SessionStore) iusecase.AuthUseCase {
	return &authUseCase{repo: repo, blacklist: blacklist, sessions: sessions}
}

var (
	ErrInvalidCredentials  = errors.New("invalid email or password")
	ErrInactiveAccount     = errors.New("account is inactive or deleted")
	ErrInvalidRefreshToken = errors.New("invalid or expired refresh token")
)

func hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

func (u *authUseCase) Login(ctx context.Context, email, password, ipAddress, userAgent string) (*auth.TokenPair, error) {
	// 1. Check user
	user, err := u.repo.FindByEmail(ctx, email)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	if !user.IsActive || user.IsDeleted {
		return nil, ErrInactiveAccount
	}

	// 2. Validate password
	if !auth.CheckPasswordHash(password, user.PasswordHash) {
		return nil, ErrInvalidCredentials
	}

	// 3. Issue Token Pair
	tokenPair, refreshExp, err := auth.GenerateTokenPair(user)
	if err != nil {
		return nil, err
	}

	// 4. Record session
	session := &entity.Session{
		UserID:           user.ID,
		RefreshTokenHash: hashToken(tokenPair.RefreshToken),
		IPAddress:        &ipAddress,
		UserAgent:        &userAgent,
		ExpiresAt:        refreshExp,
	}
	if err := u.repo.CreateSession(ctx, session); err != nil {
		return nil, err
	}

	// Mirrored rapidly to Redis Session Store
	_ = u.sessions.SetSession(ctx, session)

	return tokenPair, nil
}

func (u *authUseCase) Refresh(ctx context.Context, refreshToken, ipAddress, userAgent string) (*auth.TokenPair, error) {
	// Hash the incoming refresh token
	refreshHash := hashToken(refreshToken)

	// Look up session first in Redis cache
	session, err := u.sessions.GetSession(ctx, refreshHash)
	if err != nil || session == nil {
		// Fallback to DB
		session, err = u.repo.FindSessionByRefreshTokenHash(ctx, refreshHash)
		if err != nil {
			return nil, ErrInvalidRefreshToken
		}
	}

	// Check if session is revoked or expired
	if session.RevokedAt != nil || session.ExpiresAt.Before(time.Now()) {
		return nil, ErrInvalidRefreshToken
	}

	// Check user existence/activity
	user, err := u.repo.FindByID(ctx, session.UserID)
	if err != nil || !user.IsActive || user.IsDeleted {
		return nil, ErrInactiveAccount
	}

	// Generate just a new Access Token (Refresh Token Rotation disabled)
	newAccessToken, err := auth.GenerateAccessTokenOnly(user)
	if err != nil {
		return nil, err
	}

	return &auth.TokenPair{
		AccessToken:  newAccessToken,
		RefreshToken: refreshToken,
	}, nil
}

func (u *authUseCase) Logout(ctx context.Context, accessToken, refreshToken string) error {
	// 1. Blacklist the Access Token (for remaining valid time, default to 1 hour for safety)
	_ = u.blacklist.Add(ctx, accessToken, 1*time.Hour)

	// 2. Revoke the Session using the Refresh Token Hash
	refreshHash := hashToken(refreshToken)
	session, err := u.sessions.GetSession(ctx, refreshHash)
	if err != nil || session == nil {
		session, err = u.repo.FindSessionByRefreshTokenHash(ctx, refreshHash)
	}

	if err == nil && session.RevokedAt == nil {
		now := time.Now()
		session.RevokedAt = &now
		_ = u.repo.UpdateSession(ctx, session)
		_ = u.sessions.DeleteSession(ctx, refreshHash)
	}

	return nil
}
