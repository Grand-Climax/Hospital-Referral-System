package cache

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type OTPChallenge struct {
	CodeHash    string    `json:"code_hash"`
	Channel     string    `json:"channel"`
	Attempts    int       `json:"attempts"`
	MaxAttempts int       `json:"max_attempts"`
	ExpiresAt   time.Time `json:"expires_at"`
	CreatedAt   time.Time `json:"created_at"`
}

type MFAOTPStore interface {
	SetChallenge(ctx context.Context, userID uuid.UUID, challenge *OTPChallenge, ttl time.Duration) error
	GetChallenge(ctx context.Context, userID uuid.UUID) (*OTPChallenge, error)
	DeleteChallenge(ctx context.Context, userID uuid.UUID) error
}

type redisMFAOTPStore struct {
	client *redis.Client
}

func NewRedisMFAOTPStore(client *redis.Client) MFAOTPStore {
	return &redisMFAOTPStore{client: client}
}

func (s *redisMFAOTPStore) key(userID uuid.UUID) string {
	return "mfa:otp:" + userID.String()
}

func (s *redisMFAOTPStore) SetChallenge(ctx context.Context, userID uuid.UUID, challenge *OTPChallenge, ttl time.Duration) error {
	payload, err := json.Marshal(challenge)
	if err != nil {
		return err
	}
	return s.client.Set(ctx, s.key(userID), payload, ttl).Err()
}

func (s *redisMFAOTPStore) GetChallenge(ctx context.Context, userID uuid.UUID) (*OTPChallenge, error) {
	data, err := s.client.Get(ctx, s.key(userID)).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var challenge OTPChallenge
	if err := json.Unmarshal(data, &challenge); err != nil {
		return nil, err
	}
	return &challenge, nil
}

func (s *redisMFAOTPStore) DeleteChallenge(ctx context.Context, userID uuid.UUID) error {
	return s.client.Del(ctx, s.key(userID)).Err()
}
