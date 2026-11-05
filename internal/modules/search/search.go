package search

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/voidxo/voidxo/internal/modules/content"
)

var (
	ErrRateLimited = errors.New("search rate limited")
)

type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}

const (
	defaultPageSize = 10
	maxPageSize     = 30
	maxQueryRune    = 50
)

// Result 按内容类型分组返回，未查询的类型保持空数组而不是 null。
type Result struct {
	Query     string                         `json:"query"`
	Questions content.Page[content.Question] `json:"questions"`
	Articles  content.Page[content.Article]  `json:"articles"`
	Topics    content.Page[content.Topic]    `json:"topics"`
}

type Kinds struct {
	Questions bool
	Articles  bool
	Topics    bool
}

type Repository interface {
	SearchQuestions(ctx context.Context, pattern string, limit, offset int) ([]content.Question, bool, error)
	SearchArticles(ctx context.Context, pattern string, limit, offset int) ([]content.Article, bool, error)
	SearchTopics(ctx context.Context, pattern string, limit, offset int) ([]content.Topic, bool, error)
}

// Limiter 用于限制单个用户的搜索频率，未注入时不做限流。
type Limiter interface {
	Allow(ctx context.Context, key string) (bool, error)
}

type Service struct {
	repo    Repository
	limiter Limiter
}

func NewService(repo Repository, limiters ...Limiter) *Service {
	service := &Service{repo: repo}
	if len(limiters) > 0 {
		service.limiter = limiters[0]
	}
	return service
}

// EscapeLike 转义 MySQL LIKE 的通配符，避免用户输入被当作模式匹配。
func EscapeLike(value string) string {
	replacer := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return replacer.Replace(value)
}

func (s *Service) Search(ctx context.Context, userID int64, query, kind string, page, pageSize int) (Result, error) {
	query = strings.TrimSpace(query)
	if count := utf8.RuneCountInString(query); count < 1 || count > maxQueryRune {
		return Result{}, &ValidationError{Message: "搜索关键词需为 1 至 50 个字符"}
	}
	kinds, ok := parseKind(kind)
	if !ok {
		return Result{}, &ValidationError{Message: "搜索类型不正确"}
	}
	page, pageSize = normalizePage(page, pageSize)

	if s.limiter != nil {
		allowed, err := s.limiter.Allow(ctx, "user:"+strconv.FormatInt(userID, 10))
		if err != nil {
			return Result{}, err
		}
		if !allowed {
			return Result{}, ErrRateLimited
		}
	}

	pattern := "%" + EscapeLike(query) + "%"
	offset := (page - 1) * pageSize
	result := Result{
		Query:     query,
		Questions: content.Page[content.Question]{Items: []content.Question{}, Page: page, PageSize: pageSize},
		Articles:  content.Page[content.Article]{Items: []content.Article{}, Page: page, PageSize: pageSize},
		Topics:    content.Page[content.Topic]{Items: []content.Topic{}, Page: page, PageSize: pageSize},
	}

	if kinds.Questions {
		items, hasMore, err := s.repo.SearchQuestions(ctx, pattern, pageSize, offset)
		if err != nil {
			return Result{}, err
		}
		result.Questions = content.Page[content.Question]{Items: ensure(items), Page: page, PageSize: pageSize, HasMore: hasMore}
	}
	if kinds.Articles {
		items, hasMore, err := s.repo.SearchArticles(ctx, pattern, pageSize, offset)
		if err != nil {
			return Result{}, err
		}
		result.Articles = content.Page[content.Article]{Items: ensure(items), Page: page, PageSize: pageSize, HasMore: hasMore}
	}
	if kinds.Topics {
		items, hasMore, err := s.repo.SearchTopics(ctx, pattern, pageSize, offset)
		if err != nil {
			return Result{}, err
		}
		result.Topics = content.Page[content.Topic]{Items: ensure(items), Page: page, PageSize: pageSize, HasMore: hasMore}
	}
	return result, nil
}

func parseKind(kind string) (Kinds, bool) {
	switch kind {
	case "", "all":
		return Kinds{Questions: true, Articles: true, Topics: true}, true
	case "question":
		return Kinds{Questions: true}, true
	case "article":
		return Kinds{Articles: true}, true
	case "topic":
		return Kinds{Topics: true}, true
	}
	return Kinds{}, false
}

func normalizePage(page, pageSize int) (int, int) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = defaultPageSize
	}
	if pageSize > maxPageSize {
		pageSize = maxPageSize
	}
	return page, pageSize
}

func ensure[T any](items []T) []T {
	if items == nil {
		return []T{}
	}
	return items
}
