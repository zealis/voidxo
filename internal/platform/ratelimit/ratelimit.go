// Package ratelimit 提供基于 Redis 的固定窗口限流。
// 不同用途通过 prefix 隔离键空间，避免搜索、私信等互相影响。
package ratelimit

import (
	"context"
	"fmt"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

var windowScript = goredis.NewScript(`
local value = redis.call('INCR', KEYS[1])
if value == 1 then
  redis.call('PEXPIRE', KEYS[1], ARGV[1])
end
return value
`)

type RedisLimiter struct {
	client *goredis.Client
	prefix string
	limit  int64
	window time.Duration
}

func NewRedisLimiter(client *goredis.Client, prefix string, limit int64, window time.Duration) *RedisLimiter {
	return &RedisLimiter{client: client, prefix: prefix, limit: limit, window: window}
}

// Allow 返回本次请求是否仍在配额内；计数窗口在首次请求时设置过期时间。
func (l *RedisLimiter) Allow(ctx context.Context, subject string) (bool, error) {
	return l.AllowWithin(ctx, subject, l.limit, l.window)
}

// AllowWithin 允许调用方按场景指定配额，例如同一邮箱的「每分钟 1 次」与「每小时 5 次」。
// 不同窗口必须使用不同的 subject，否则计数会互相干扰。
func (l *RedisLimiter) AllowWithin(ctx context.Context, subject string, limit int64, window time.Duration) (bool, error) {
	key := "voidxo:limit:" + l.prefix + ":" + subject
	value, err := windowScript.Run(ctx, l.client, []string{key}, window.Milliseconds()).Int64()
	if err != nil {
		return false, fmt.Errorf("%s limiter unavailable: %w", l.prefix, err)
	}
	return value <= limit, nil
}
