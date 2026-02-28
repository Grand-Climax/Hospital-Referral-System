package cache

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

type TokenBlacklist interface {
	Add(ctx context.Context, token string, expiration time.Duration) error
	IsBlacklisted(ctx context.Context, token string) (bool, error)
}

type redisTokenBlacklist struct {
	client *redis.Client
}

func NewRedisTokenBlacklist(client *redis.Client) TokenBlacklist {
	return &redisTokenBlacklist{
		client: client,
	}
}

// Add stores the token in Redis with an expiration time. The value is structurally irrelevant.
func (b *redisTokenBlacklist) Add(ctx context.Context, token string, expiration time.Duration) error {
	return b.client.Set(ctx, "bl:"+token, "true", expiration).Err()
}

// IsBlacklisted checks if the token exists in Redis.
func (b *redisTokenBlacklist) IsBlacklisted(ctx context.Context, token string) (bool, error) {
	err := b.client.Get(ctx, "bl:"+token).Err()
	if err == redis.Nil {
		return false, nil // Token is not blacklisted
	} else if err != nil {
		return false, err // Redis error
	}
	return true, nil // Token is blacklisted
}
