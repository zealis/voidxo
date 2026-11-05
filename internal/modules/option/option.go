package option

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/voidxo/voidxo/internal/platform/cache"
)

type kind uint8

const (
	stringKind kind = iota
	boolKind
	intKind
)

var publicDefinitions = map[string]kind{
	"site_name": stringKind, "site_description": stringKind, "site_keywords": stringKind,
	"site_icp_beian": stringKind, "site_gongan_beian": stringKind, "language": stringKind,
	"question_can_edit": boolKind, "question_can_edit_before": intKind,
	"question_can_edit_only_no_answer": boolKind, "question_can_edit_only_no_comment": boolKind,
	"question_can_delete": boolKind, "question_can_delete_before": intKind,
	"question_can_delete_only_no_answer": boolKind, "question_can_delete_only_no_comment": boolKind,
	"answer_can_edit": boolKind, "answer_can_edit_before": intKind, "answer_can_edit_only_no_comment": boolKind,
	"answer_can_delete": boolKind, "answer_can_delete_before": intKind, "answer_can_delete_only_no_comment": boolKind,
	"article_can_edit": boolKind, "article_can_edit_before": intKind, "article_can_edit_only_no_comment": boolKind,
	"article_can_delete": boolKind, "article_can_delete_before": intKind, "article_can_delete_only_no_comment": boolKind,
	"comment_can_edit": boolKind, "comment_can_edit_before": intKind,
	"comment_can_delete": boolKind, "comment_can_delete_before": intKind,
}

// Definition 描述单个配置项的类型与可见性，供后台展示与校验复用。
type Definition struct {
	Type   string
	Public bool
}

func PublicNames() []string {
	names := make([]string, 0, len(publicDefinitions))
	for name := range publicDefinitions {
		names = append(names, name)
	}
	return names
}

// Definitions 返回全部已知配置项的定义；后台据此决定输入控件与分组。
func Definitions() map[string]Definition {
	// 后台可写的内部配置目前与公开配置同源，先按公开集合暴露。
	result := make(map[string]Definition, len(publicDefinitions))
	for name, value := range publicDefinitions {
		result[name] = Definition{Type: value.String(), Public: true}
	}
	return result
}

// AllNames 返回全部已知配置项名称，按字母序，便于稳定展示。
func AllNames() []string {
	names := PublicNames()
	sort.Strings(names)
	return names
}

// Validate 校验配置值是否符合其类型定义；未知名称一律拒绝。
func Validate(name, value string) error {
	definition, ok := publicDefinitions[name]
	if !ok {
		return fmt.Errorf("unknown option %s", name)
	}
	switch definition {
	case boolKind:
		if value != "true" && value != "false" {
			return fmt.Errorf("option %s must be true or false", name)
		}
	case intKind:
		if _, err := strconv.Atoi(value); err != nil {
			return fmt.Errorf("option %s must be an integer", name)
		}
	}
	return nil
}

func (k kind) String() string {
	switch k {
	case boolKind:
		return "bool"
	case intKind:
		return "int"
	default:
		return "string"
	}
}

type Repository interface {
	FindByNames(ctx context.Context, names []string) (map[string]string, error)
}

type Service struct {
	repository Repository
	cache      cache.Cache
}

const cacheScope = "option"

// SetCache 注入公开配置缓存；配置变更频率低，采用 5 分钟 TTL。
func (s *Service) SetCache(c cache.Cache) {
	s.cache = c
}

func NewService(repository Repository) *Service {
	return &Service{repository: repository}
}

func (s *Service) Public(ctx context.Context) (map[string]any, error) {
	if s.cache != nil {
		var cached map[string]any
		if ok, _ := s.cache.Get(ctx, cacheScope, "public", &cached); ok {
			return cached, nil
		}
	}
	stored, err := s.repository.FindByNames(ctx, PublicNames())
	if err != nil {
		return nil, fmt.Errorf("load public options: %w", err)
	}
	result := make(map[string]any, len(stored))
	for name, value := range stored {
		definition, ok := publicDefinitions[name]
		if !ok {
			continue
		}
		switch definition {
		case boolKind:
			result[name] = value == "true"
		case intKind:
			parsed, parseErr := strconv.Atoi(value)
			if parseErr != nil {
				return nil, fmt.Errorf("option %s has invalid integer value", name)
			}
			result[name] = parsed
		default:
			result[name] = value
		}
	}
	if s.cache != nil {
		_ = s.cache.Set(ctx, cacheScope, "public", result, 5*time.Minute)
	}
	return result, nil
}
