package usecase

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"Hospital-Referral-System/internal/domain/entity"
	irepository "Hospital-Referral-System/internal/domain/interfaces/repository"
	iusecase "Hospital-Referral-System/internal/domain/interfaces/usecase"
	"Hospital-Referral-System/internal/infrastructure/cache"
	"Hospital-Referral-System/internal/infrastructure/email"
	"Hospital-Referral-System/internal/infrastructure/sms"
	"Hospital-Referral-System/internal/pkg/auth"
	"github.com/google/uuid"
)

type authUseCase struct {
	repo        irepository.AuthRepository
	configRepo  irepository.SystemConfigRepository
	blacklist   cache.TokenBlacklist
	sessions    cache.SessionStore
	otpStore    cache.MFAOTPStore
	smsClient   sms.SMSClient
	emailClient email.Client
}

func NewAuthUseCase(
	repo irepository.AuthRepository,
	configRepo irepository.SystemConfigRepository,
	blacklist cache.TokenBlacklist,
	sessions cache.SessionStore,
	otpStore cache.MFAOTPStore,
	smsClient sms.SMSClient,
	emailClient email.Client,
) iusecase.AuthUseCase {
	return &authUseCase{
		repo:        repo,
		configRepo:  configRepo,
		blacklist:   blacklist,
		sessions:    sessions,
		otpStore:    otpStore,
		smsClient:   smsClient,
		emailClient: emailClient,
	}
}

var (
	ErrInvalidCredentials  = errors.New("invalid email or password")
	ErrInactiveAccount     = errors.New("account is inactive or deleted")
	ErrInvalidRefreshToken = errors.New("invalid or expired refresh token")
	ErrInvalidMFAChannel   = errors.New("invalid MFA channel; allowed values are email or sms")
	ErrOTPNotRequested     = errors.New("OTP challenge was not requested")
	ErrInvalidOTPCode      = errors.New("invalid OTP code")
	ErrOTPExpired          = errors.New("OTP has expired")
	ErrOTPAttemptsExceeded = errors.New("OTP attempts exceeded")
	ErrMissingSMSContact   = errors.New("SMS OTP is enabled but user has no valid phone number")
	ErrMissingEmailContact = errors.New("Email OTP requires a valid user email")
	ErrMFAOTPResendCooldown = errors.New("OTP resend cooldown active; please wait before requesting another code")
)

func hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

func hashOTPCode(code string) string {
	secret := os.Getenv("OTP_HASH_SECRET")
	if secret == "" {
		secret = os.Getenv("JWT_SECRET")
	}
	h := sha256.Sum256([]byte(secret + ":" + code))
	return hex.EncodeToString(h[:])
}

func generateOTPCode() (string, error) {
	max := 1000000
	buf := make([]byte, 4)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	v := int(buf[0])<<24 | int(buf[1])<<16 | int(buf[2])<<8 | int(buf[3])
	if v < 0 {
		v = -v
	}
	return fmt.Sprintf("%06d", v%max), nil
}

func (u *authUseCase) getBoolConfig(ctx context.Context, key string, defaultValue bool) bool {
	if u.configRepo == nil {
		return defaultValue
	}
	val, err := u.configRepo.GetBool(ctx, key, defaultValue)
	if err != nil {
		return defaultValue
	}
	return val
}

func (u *authUseCase) getIntConfig(ctx context.Context, key string, defaultValue int) int {
	if u.configRepo == nil {
		return defaultValue
	}
	cfg, err := u.configRepo.GetByKey(ctx, key)
	if err != nil || cfg == nil {
		return defaultValue
	}
	n, convErr := strconv.Atoi(strings.TrimSpace(cfg.Value))
	if convErr != nil || n <= 0 {
		return defaultValue
	}
	return n
}

func (u *authUseCase) Login(ctx context.Context, email, password, requestedChannel string) (*iusecase.LoginResult, error) {
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

	// 3. Bypass MFA check
	mfaEnabled := u.getBoolConfig(ctx, "mfa_enabled", true)
	if !mfaEnabled {
		tokenPair, err := u.createFullSessionToken(ctx, user, "", "")
		if err != nil {
			return nil, err
		}
		return &iusecase.LoginResult{
			AccessToken:  tokenPair.AccessToken,
			RefreshToken: tokenPair.RefreshToken,
		}, nil
	}

	// Rate limit check: enforce dynamic resend cooldown if an active challenge exists
	existing, err := u.otpStore.GetChallenge(ctx, user.ID)
	if err == nil && existing != nil {
		cooldownSeconds := u.getIntConfig(ctx, "mfa_otp_resend_cooldown_seconds", 60)
		timePassed := time.Since(existing.CreatedAt)
		if timePassed < time.Duration(cooldownSeconds)*time.Second {
			return nil, ErrMFAOTPResendCooldown
		}
	}

	// 4. Issue MFA intermediate token
	mfaToken, tokenExp, err := auth.GenerateMFAIntermediateToken(user.ID)
	if err != nil {
		return nil, err
	}

	smsEnabled := u.getBoolConfig(ctx, "sms_otp_enabled", false)
	channel := "email"
	requestedChannel = strings.ToLower(strings.TrimSpace(requestedChannel))
	if requestedChannel != "" && requestedChannel != "email" && requestedChannel != "sms" {
		return nil, ErrInvalidMFAChannel
	}
	if smsEnabled {
		if requestedChannel == "sms" {
			channel = "sms"
		} else {
			channel = "email"
		}
	}

	if channel == "sms" {
		if user.PhoneNumber == nil || strings.TrimSpace(*user.PhoneNumber) == "" {
			fallbackEmail := u.getBoolConfig(ctx, "mfa_sms_fallback_email", true)
			if fallbackEmail && strings.TrimSpace(user.Email) != "" {
				channel = "email"
			} else {
				return nil, ErrMissingSMSContact
			}
		}
	}

	otpCode, err := generateOTPCode()
	if err != nil {
		return nil, err
	}
	otpTTLSeconds := u.getIntConfig(ctx, "mfa_otp_ttl_seconds", 300)
	maxAttempts := u.getIntConfig(ctx, "mfa_otp_max_attempts", 5)
	otpTTL := time.Duration(otpTTLSeconds) * time.Second

	if channel == "sms" {
		msg := fmt.Sprintf("Your Hospital Referral OTP is %s. It expires in %d minutes.", otpCode, otpTTLSeconds/60)
		if _, sendErr := u.smsClient.Send(ctx, sms.SendRequest{
			To:      strings.TrimSpace(*user.PhoneNumber),
			Message: msg,
		}); sendErr != nil {
			return nil, sendErr
		}
	} else {
		if strings.TrimSpace(user.Email) == "" {
			return nil, ErrMissingEmailContact
		}
		body := fmt.Sprintf("Your Hospital Referral OTP is %s. It expires in %d minutes.", otpCode, otpTTLSeconds/60)
		if err := u.emailClient.Send(ctx, user.Email, "Your login OTP code", body); err != nil {
			return nil, err
		}
	}

	challenge := &cache.OTPChallenge{
		CodeHash:    hashOTPCode(otpCode),
		Channel:     channel,
		Attempts:    0,
		MaxAttempts: maxAttempts,
		ExpiresAt:   time.Now().Add(otpTTL),
		CreatedAt:   time.Now(),
	}
	if err := u.otpStore.SetChallenge(ctx, user.ID, challenge, otpTTL); err != nil {
		return nil, err
	}

	_ = tokenExp // token expiration is encoded in JWT, challenge has own TTL.
	return &iusecase.LoginResult{
		MFAToken: mfaToken,
		Channel:  channel,
	}, nil
}

func (u *authUseCase) VerifyOTP(ctx context.Context, userID uuid.UUID, code, ipAddress, userAgent string) (*auth.TokenPair, error) {
	user, err := u.repo.FindByID(ctx, userID)
	if err != nil {
		return nil, ErrInvalidCredentials
	}
	if !user.IsActive || user.IsDeleted {
		return nil, ErrInactiveAccount
	}

	challenge, err := u.otpStore.GetChallenge(ctx, userID)
	if err != nil {
		return nil, err
	}
	if challenge == nil {
		return nil, ErrOTPNotRequested
	}
	if time.Now().After(challenge.ExpiresAt) {
		_ = u.otpStore.DeleteChallenge(ctx, userID)
		return nil, ErrOTPExpired
	}
	if challenge.Attempts >= challenge.MaxAttempts {
		_ = u.otpStore.DeleteChallenge(ctx, userID)
		return nil, ErrOTPAttemptsExceeded
	}

	codeHash := hashOTPCode(strings.TrimSpace(code))
	if codeHash != challenge.CodeHash {
		challenge.Attempts++
		remaining := time.Until(challenge.ExpiresAt)
		if remaining < 0 {
			remaining = 0
		}
		_ = u.otpStore.SetChallenge(ctx, userID, challenge, remaining)
		if challenge.Attempts >= challenge.MaxAttempts {
			_ = u.otpStore.DeleteChallenge(ctx, userID)
			return nil, ErrOTPAttemptsExceeded
		}
		return nil, ErrInvalidOTPCode
	}

	_ = u.otpStore.DeleteChallenge(ctx, userID)
	return u.createFullSessionToken(ctx, user, ipAddress, userAgent)
}

func (u *authUseCase) createFullSessionToken(ctx context.Context, user *entity.User, ipAddress, userAgent string) (*auth.TokenPair, error) {
	tokenPair, refreshExp, err := auth.GenerateTokenPair(user)
	if err != nil {
		return nil, err
	}

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
