package auth

import (
	"errors"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"Hospital-Referral-System/internal/domain/entity"
)

type TokenPair struct {
	AccessToken  string
	RefreshToken string
}

type TokenPayload struct {
	UserID     uuid.UUID       `json:"sub"`
	Role       entity.UserRole `json:"role,omitempty"`
	HospID     *uuid.UUID      `json:"hosp_id,omitempty"`
	DeptID     *uuid.UUID      `json:"dept_id,omitempty"`
	MFAPending bool            `json:"mfa_pending,omitempty"`
	jwt.RegisteredClaims
}

var (
	ErrInvalidToken = errors.New("invalid or expired token")
)

func getSecret() []byte {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "super-secret-fallback-key-for-local-dev-only"
	}
	return []byte(secret)
}

func GenerateAccessTokenOnly(user *entity.User) (string, error) {
	accessExpiration := time.Now().Add(1 * time.Hour)
	accessPayload := &TokenPayload{
		UserID:     user.ID,
		Role:       user.Role,
		HospID:     user.HospitalID,
		DeptID:     user.DepartmentID,
		MFAPending: false,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(accessExpiration),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, accessPayload).SignedString(getSecret())
}

func GenerateMFAIntermediateToken(userID uuid.UUID) (string, time.Time, error) {
	expiresAt := time.Now().Add(5 * time.Minute)
	payload := &TokenPayload{
		UserID:     userID,
		MFAPending: true,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, payload).SignedString(getSecret())
	if err != nil {
		return "", time.Time{}, err
	}

	return token, expiresAt, nil
}

func GenerateTokenPair(user *entity.User) (*TokenPair, time.Time, error) {
	// Access token (1 hour)
	accessExpiration := time.Now().Add(1 * time.Hour)
	accessPayload := &TokenPayload{
		UserID:     user.ID,
		Role:       user.Role,
		HospID:     user.HospitalID,
		DeptID:     user.DepartmentID,
		MFAPending: false,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(accessExpiration),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	accessToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, accessPayload).SignedString(getSecret())
	if err != nil {
		return nil, time.Time{}, err
	}

	// Refresh token (e.g. 7 days)
	refreshExpiration := time.Now().Add(7 * 24 * time.Hour)
	refreshPayload := &jwt.RegisteredClaims{
		Subject:   user.ID.String(),
		ExpiresAt: jwt.NewNumericDate(refreshExpiration),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
	}
	refreshToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshPayload).SignedString(getSecret())
	if err != nil {
		return nil, time.Time{}, err
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, refreshExpiration, nil
}

func ValidateToken(tokenStr string) (*TokenPayload, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &TokenPayload{}, func(token *jwt.Token) (interface{}, error) {
		return getSecret(), nil
	})

	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*TokenPayload)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}
