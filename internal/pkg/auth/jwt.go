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
	UserID uuid.UUID       `json:"sub"`
	Role   entity.UserRole `json:"role"`
	HospID *uuid.UUID      `json:"hosp_id,omitempty"`
	DeptID *uuid.UUID      `json:"dept_id,omitempty"`
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

func GenerateTokenPair(user *entity.User) (*TokenPair, time.Time, error) {
	// Access token (e.g. 15 mins)
	accessExpiration := time.Now().Add(15 * time.Minute)
	accessPayload := &TokenPayload{
		UserID: user.ID,
		Role:   user.Role,
		HospID: user.HospitalID,
		DeptID: user.DepartmentID,
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
