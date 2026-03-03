package cache

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"

	"Hospital-Referral-System/internal/domain/entity"
)

type SessionStore interface {
	SetSession(ctx context.Context, session *entity.Session) error
	GetSession(ctx context.Context, refreshHash string) (*entity.Session, error)
	DeleteSession(ctx context.Context, refreshHash string) error
}

type redisSessionStore struct {
	client *redis.Client
}

func NewRedisSessionStore(client *redis.Client) SessionStore {
	return &redisSessionStore{client: client}
}

func (s *redisSessionStore) SetSession(ctx context.Context, session *entity.Session) error {
	key := "session:" + session.RefreshTokenHash
	data, err := json.Marshal(session)
	if err != nil {
		return err
	}
	
	duration := time.Until(session.ExpiresAt)
	if duration <= 0 {
		return nil
	}
	return s.client.Set(ctx, key, data, duration).Err()
}

func (s *redisSessionStore) GetSession(ctx context.Context, refreshHash string) (*entity.Session, error) {
	key := "session:" + refreshHash
	data, err := s.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, nil // Not found
	} else if err != nil {
		return nil, err
	}

	var session entity.Session
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, err
	}
	return &session, nil
}

func (s *redisSessionStore) DeleteSession(ctx context.Context, refreshHash string) error {
	key := "session:" + refreshHash
	return s.client.Del(ctx, key).Err()
}
