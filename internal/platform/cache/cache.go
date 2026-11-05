package cache

import (
	"context"
	"time"
)

// Cache 是只读路径上的 JSON 缓存边界。
// 采用「作用域版本号」失效：写入方只需 Bump，旧版本键自然过期，
// 避免按前缀扫描删除大量键。
type Cache interface {
	Get(ctx context.Context, scope, key string, target any) (bool, error)
	Set(ctx context.Context, scope, key string, value any, ttl time.Duration) error
	Bump(ctx context.Context, scope string) error
	Version(ctx context.Context, scope string) (int64, error)
}

// Noop 用于未配置 Redis 或测试场景，始终未命中且不报错。
type Noop struct{}

func (Noop) Get(context.Context, string, string, any) (bool, error)        { return false, nil }
func (Noop) Set(context.Context, string, string, any, time.Duration) error { return nil }
func (Noop) Bump(context.Context, string) error                            { return nil }
func (Noop) Version(context.Context, string) (int64, error)                { return 0, nil }
