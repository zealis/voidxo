package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

const sessionKeyPrefix = "voidxo:session:"

type RedisSessionStore struct {
	client *goredis.Client
}

func NewRedisSessionStore(client *goredis.Client) *RedisSessionStore {
	return &RedisSessionStore{client: client}
}

func (s *RedisSessionStore) Create(ctx context.Context, tokenHash string, session Session, ttl time.Duration) error {
	payload, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrSessionStoreUnavailable, err)
	}
	if err := s.client.Set(ctx, sessionKeyPrefix+tokenHash, payload, ttl).Err(); err != nil {
		return fmt.Errorf("%w: %v", ErrSessionStoreUnavailable, err)
	}
	return nil
}

func (s *RedisSessionStore) Get(ctx context.Context, tokenHash string) (Session, error) {
	payload, err := s.client.Get(ctx, sessionKeyPrefix+tokenHash).Bytes()
	if errors.Is(err, goredis.Nil) {
		return Session{}, ErrSessionNotFound
	}
	if err != nil {
		return Session{}, fmt.Errorf("%w: %v", ErrSessionStoreUnavailable, err)
	}
	var session Session
	if err := json.Unmarshal(payload, &session); err != nil {
		return Session{}, fmt.Errorf("%w: invalid session payload", ErrSessionStoreUnavailable)
	}
	return session, nil
}

func (s *RedisSessionStore) Touch(ctx context.Context, tokenHash string, ttl time.Duration) error {
	updated, err := s.client.Expire(ctx, sessionKeyPrefix+tokenHash, ttl).Result()
	if err != nil {
		return fmt.Errorf("%w: %v", ErrSessionStoreUnavailable, err)
	}
	if !updated {
		return ErrSessionNotFound
	}
	return nil
}

func (s *RedisSessionStore) Delete(ctx context.Context, tokenHash string) error {
	if err := s.client.Del(ctx, sessionKeyPrefix+tokenHash).Err(); err != nil {
		return fmt.Errorf("%w: %v", ErrSessionStoreUnavailable, err)
	}
	return nil
}
