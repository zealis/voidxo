package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

type Redis struct {
	client *goredis.Client
}

func NewRedis(client *goredis.Client) *Redis {
	return &Redis{client: client}
}

func versionKey(scope string) string {
	return "voidxo:cache:v:" + scope
}

func dataKey(scope string, version int64, key string) string {
	return "voidxo:cache:" + scope + ":" + strconv.FormatInt(version, 10) + ":" + key
}

func (c *Redis) Get(ctx context.Context, scope, key string, target any) (bool, error) {
	version, err := c.Version(ctx, scope)
	if err != nil {
		return false, err
	}
	raw, err := c.client.Get(ctx, dataKey(scope, version, key)).Bytes()
	if errors.Is(err, goredis.Nil) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("cache get: %w", err)
	}
	if err := json.Unmarshal(raw, target); err != nil {
		// 缓存内容损坏时按未命中处理，并删除脏数据。
		_ = c.client.Del(ctx, dataKey(scope, version, key)).Err()
		return false, nil
	}
	return true, nil
}

func (c *Redis) Set(ctx context.Context, scope, key string, value any, ttl time.Duration) error {
	version, err := c.Version(ctx, scope)
	if err != nil {
		return err
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("cache encode: %w", err)
	}
	if err := c.client.Set(ctx, dataKey(scope, version, key), raw, ttl).Err(); err != nil {
		return fmt.Errorf("cache set: %w", err)
	}
	return nil
}

func (c *Redis) Bump(ctx context.Context, scope string) error {
	if err := c.client.Incr(ctx, versionKey(scope)).Err(); err != nil {
		return fmt.Errorf("cache bump: %w", err)
	}
	return nil
}

func (c *Redis) Version(ctx context.Context, scope string) (int64, error) {
	value, err := c.client.Get(ctx, versionKey(scope)).Int64()
	if errors.Is(err, goredis.Nil) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("cache version: %w", err)
	}
	return value, nil
}
