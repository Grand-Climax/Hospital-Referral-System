package cache

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type PasswordResetOTPStore interface {
	SetChallenge(ctx context.Context, userID uuid.UUID, challenge *OTPChallenge, ttl time.Duration) error
	GetChallenge(ctx context.Context, userID uuid.UUID) (*OTPChallenge, error)
	DeleteChallenge(ctx context.Context, userID uuid.UUID) error
}

type redisPasswordResetOTPStore struct {
	client *redis.Client
}

func NewRedisPasswordResetOTPStore(client *redis.Client) PasswordResetOTPStore {
	return &redisPasswordResetOTPStore{client: client}
}

func (s *redisPasswordResetOTPStore) key(userID uuid.UUID) string {
	return "pwd_reset:otp:" + userID.String()
}

func (s *redisPasswordResetOTPStore) SetChallenge(ctx context.Context, userID uuid.UUID, challenge *OTPChallenge, ttl time.Duration) error {
	payload, err := json.Marshal(challenge)
	if err != nil {
		return err
	}
	return s.client.Set(ctx, s.key(userID), payload, ttl).Err()
}

func (s *redisPasswordResetOTPStore) GetChallenge(ctx context.Context, userID uuid.UUID) (*OTPChallenge, error) {
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

func (s *redisPasswordResetOTPStore) DeleteChallenge(ctx context.Context, userID uuid.UUID) error {
	return s.client.Del(ctx, s.key(userID)).Err()
}
