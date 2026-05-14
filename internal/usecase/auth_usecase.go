package usecase

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"time"

	"Hospital-Referral-System/internal/domain/entity"
	irepository "Hospital-Referral-System/internal/domain/interfaces/repository"
	iusecase "Hospital-Referral-System/internal/domain/interfaces/usecase"
	"Hospital-Referral-System/internal/infrastructure/cache"
	"Hospital-Referral-System/internal/infrastructure/crypto"
	"Hospital-Referral-System/internal/pkg/auth"
	"github.com/google/uuid"
	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
)

type authUseCase struct {
	repo      irepository.AuthRepository
	auditRepo irepository.AuditLogRepository
	blacklist cache.TokenBlacklist
	sessions  cache.SessionStore
	mfaCrypto *crypto.MFASecretCryptoService
	mfaIssuer string
}

func NewAuthUseCase(
	repo irepository.AuthRepository,
	auditRepo irepository.AuditLogRepository,
	blacklist cache.TokenBlacklist,
	sessions cache.SessionStore,
	mfaCrypto *crypto.MFASecretCryptoService,
) iusecase.AuthUseCase {
	issuer := os.Getenv("MFA_TOTP_ISSUER")
	if issuer == "" {
		issuer = "Hospital Referral System"
	}

	return &authUseCase{
		repo:      repo,
		auditRepo: auditRepo,
		blacklist: blacklist,
		sessions:  sessions,
		mfaCrypto: mfaCrypto,
		mfaIssuer: issuer,
	}
}

var (
	ErrInvalidCredentials  = errors.New("invalid email or password")
	ErrInactiveAccount     = errors.New("account is inactive or deleted")
	ErrInvalidRefreshToken = errors.New("invalid or expired refresh token")
	ErrInvalidMFACode      = errors.New("invalid MFA code")
	ErrMFANotConfigured    = errors.New("MFA is not configured for this account")
	ErrMFAAlreadySet       = errors.New("MFA is already configured for this account")
)

func hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

func (u *authUseCase) Login(ctx context.Context, email, password string) (*iusecase.LoginResult, error) {
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

	// 3. Issue short-lived intermediate token for MFA verification
	mfaToken, _, err := auth.GenerateMFAIntermediateToken(user.ID)
	if err != nil {
		return nil, err
	}

	return &iusecase.LoginResult{
		MFAToken:         mfaToken,
		MFASetupRequired: user.MFASecretEnc == nil || *user.MFASecretEnc == "",
	}, nil
}

func (u *authUseCase) createFullSessionToken(ctx context.Context, user *entity.User, ipAddress, userAgent string) (*auth.TokenPair, error) {
	// Issue token pair
	tokenPair, refreshExp, err := auth.GenerateTokenPair(user)
	if err != nil {
		return nil, err
	}

	// Record session
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

func (u *authUseCase) VerifyMFA(ctx context.Context, userID uuid.UUID, code, ipAddress, userAgent string) (*auth.TokenPair, error) {
	user, err := u.repo.FindByID(ctx, userID)
	if err != nil {
		return nil, ErrInvalidCredentials
	}
	if !user.IsActive || user.IsDeleted {
		return nil, ErrInactiveAccount
	}
	if user.MFASecretEnc == nil || *user.MFASecretEnc == "" {
		return nil, ErrMFANotConfigured
	}

	secret, err := u.mfaCrypto.DecryptString(*user.MFASecretEnc)
	if err != nil {
		return nil, ErrInvalidMFACode
	}

	valid, err := totp.ValidateCustom(code, secret, time.Now(), totp.ValidateOpts{
		Period:    30,
		Skew:      1,
		Digits:    otp.DigitsSix,
		Algorithm: otp.AlgorithmSHA1,
	})
	if err != nil || !valid {
		return nil, ErrInvalidMFACode
	}

	return u.createFullSessionToken(ctx, user, ipAddress, userAgent)
}

func (u *authUseCase) SetupMFA(ctx context.Context, userID uuid.UUID) (string, error) {
	user, err := u.repo.FindByID(ctx, userID)
	if err != nil {
		return "", ErrInvalidCredentials
	}
	if !user.IsActive || user.IsDeleted {
		return "", ErrInactiveAccount
	}
	if user.MFASecretEnc != nil && *user.MFASecretEnc != "" {
		return "", ErrMFAAlreadySet
	}

	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      u.mfaIssuer,
		AccountName: user.Email,
		Digits:      otp.DigitsSix,
		Algorithm:   otp.AlgorithmSHA1,
		Period:      30,
	})
	if err != nil {
		return "", err
	}

	encryptedSecret, err := u.mfaCrypto.EncryptString(key.Secret())
	if err != nil {
		return "", err
	}

	user.MFASecretEnc = &encryptedSecret
	if err := u.repo.Update(ctx, user); err != nil {
		return "", err
	}

	return key.URL(), nil
}

func (u *authUseCase) ResetMFA(ctx context.Context, adminUserID, targetUserID uuid.UUID) error {
	user, err := u.repo.FindByID(ctx, targetUserID)
	if err != nil {
		return ErrInvalidCredentials
	}

	hadSecret := user.MFASecretEnc != nil && *user.MFASecretEnc != ""
	user.MFASecretEnc = nil
	if err := u.repo.Update(ctx, user); err != nil {
		return err
	}

	if u.auditRepo != nil {
		_ = u.auditRepo.LogWithContext(
			ctx,
			adminUserID,
			entity.ActionResetMFA,
			nil,
			map[string]bool{"had_mfa_secret": hadSecret},
			map[string]string{"target_user_id": targetUserID.String()},
		)
	}

	return nil
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
