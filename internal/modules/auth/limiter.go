package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

var limitScript = goredis.NewScript(`
local value = redis.call('INCR', KEYS[1])
if value == 1 then
  redis.call('PEXPIRE', KEYS[1], ARGV[1])
end
return value
`)

type LoginLimiter interface {
	Allow(ctx context.Context, ip, login string) (bool, error)
	Reset(ctx context.Context, ip, login string) error
}

type RedisLoginLimiter struct {
	client *goredis.Client
	limit  int64
	window time.Duration
}

func NewRedisLoginLimiter(client *goredis.Client, limit int64, window time.Duration) *RedisLoginLimiter {
	return &RedisLoginLimiter{client: client, limit: limit, window: window}
}

func (l *RedisLoginLimiter) Allow(ctx context.Context, ip, login string) (bool, error) {
	key := limiterKey(ip, login)
	value, err := limitScript.Run(ctx, l.client, []string{key}, l.window.Milliseconds()).Int64()
	if err != nil {
		return false, fmt.Errorf("%w: %v", ErrSessionStoreUnavailable, err)
	}
	return value <= l.limit, nil
}

func (l *RedisLoginLimiter) Reset(ctx context.Context, ip, login string) error {
	if err := l.client.Del(ctx, limiterKey(ip, login)).Err(); err != nil {
		return fmt.Errorf("%w: %v", ErrSessionStoreUnavailable, err)
	}
	return nil
}

func limiterKey(ip, login string) string {
	digest := sha256.Sum256([]byte(ip + "\x00" + login))
	return "voidxo:limit:login:" + hex.EncodeToString(digest[:])
}
