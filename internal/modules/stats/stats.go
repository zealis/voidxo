// Package stats 提供站点公开统计，供首页与页脚展示。
// 只暴露聚合数量，不包含任何用户级数据。
package stats

import (
	"context"
	"time"

	"github.com/voidxo/voidxo/internal/platform/cache"
)

type Stats struct {
	Users     int `json:"users"`
	Questions int `json:"questions"`
	Answers   int `json:"answers"`
	Articles  int `json:"articles"`
	Comments  int `json:"comments"`
	Topics    int `json:"topics"`
}

type Repository interface {
	Counts(ctx context.Context) (Stats, error)
}

type Service struct {
	repo  Repository
	cache cache.Cache
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) SetCache(c cache.Cache) {
	s.cache = c
}

// 与内容域共用缓存作用域，内容事件递增版本后统计缓存自然失效。
const cacheScope = "content"

func (s *Service) Public(ctx context.Context) (Stats, error) {
	if s.cache != nil {
		var cached Stats
		if ok, _ := s.cache.Get(ctx, cacheScope, "stats", &cached); ok {
			return cached, nil
		}
	}
	counts, err := s.repo.Counts(ctx)
	if err != nil {
		return Stats{}, err
	}
	if s.cache != nil {
		_ = s.cache.Set(ctx, cacheScope, "stats", counts, 60*time.Second)
	}
	return counts, nil
}
