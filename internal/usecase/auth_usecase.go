package usecase

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	"Hospital-Referral-System/internal/domain/entity"
	"Hospital-Referral-System/internal/infrastructure/cache"
	"Hospital-Referral-System/internal/pkg/auth"
	"Hospital-Referral-System/internal/repository"
)

type AuthUseCase interface {
	Login(ctx context.Context, email, password, ipAddress, userAgent string) (*auth.TokenPair, error)
	Refresh(ctx context.Context, refreshToken, ipAddress, userAgent string) (*auth.TokenPair, error)
	Logout(ctx context.Context, accessToken, refreshToken string) error
}

type authUseCase struct {
	repo      repository.AuthRepository
	blacklist cache.TokenBlacklist
	sessions  cache.SessionStore
}

func NewAuthUseCase(repo repository.AuthRepository, blacklist cache.TokenBlacklist, sessions cache.SessionStore) AuthUseCase {
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
	// Validate token structure using JWT lib but ignoring expiration first, or strictly trusting it.
	// Actually, the refresh token doesn't have custom claims to be validated, it is just checked via DB lookup.
	// Let's perform DB lookup.
	
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

	// Generate New Pair
	tokenPair, refreshExp, err := auth.GenerateTokenPair(user)
	if err != nil {
		return nil, err
	}

	// Revoke old session (Refresh Token Rotation)
	now := time.Now()
	session.RevokedAt = &now
	_ = u.repo.UpdateSession(ctx, session)
	_ = u.sessions.DeleteSession(ctx, refreshHash)

	// Create new session
	newSession := &entity.Session{
		UserID:           user.ID,
		RefreshTokenHash: hashToken(tokenPair.RefreshToken),
		IPAddress:        &ipAddress,
		UserAgent:        &userAgent,
		ExpiresAt:        refreshExp,
	}
	if err := u.repo.CreateSession(ctx, newSession); err != nil {
		return nil, err
	}
	
	_ = u.sessions.SetSession(ctx, newSession)

	return tokenPair, nil
}

func (u *authUseCase) Logout(ctx context.Context, accessToken, refreshToken string) error {
	// 1. Blacklist the Access Token (for remaining valid time, default to 15 mins for safety)
	// We use 15 minutes as safety net, though technically we could parse the exact token expiration.
	_ = u.blacklist.Add(ctx, accessToken, 15*time.Minute)

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
