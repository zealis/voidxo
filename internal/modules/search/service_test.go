package search

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/voidxo/voidxo/internal/modules/content"
)

type repoStub struct {
	questionPattern string
	articleCalls    int
	topicCalls      int
}

func (r *repoStub) SearchQuestions(_ context.Context, pattern string, _, _ int) ([]content.Question, bool, error) {
	r.questionPattern = pattern
	return []content.Question{{ID: 1, Title: "命中问题"}}, false, nil
}

func (r *repoStub) SearchArticles(context.Context, string, int, int) ([]content.Article, bool, error) {
	r.articleCalls++
	return nil, false, nil
}

func (r *repoStub) SearchTopics(context.Context, string, int, int) ([]content.Topic, bool, error) {
	r.topicCalls++
	return nil, false, nil
}

type limiterStub struct {
	allowed bool
	calls   int
	err     error
}

func (l *limiterStub) Allow(context.Context, string) (bool, error) {
	l.calls++
	return l.allowed, l.err
}

func TestEscapeLikeNeutralizesWildcards(t *testing.T) {
	got := EscapeLike(`100%_test\`)
	want := `100\%\_test\\`
	if got != want {
		t.Fatalf("escape mismatch: %q != %q", got, want)
	}
}

func TestSearchValidation(t *testing.T) {
	service := NewService(&repoStub{})
	if _, err := service.Search(context.Background(), 1, "   ", "all", 1, 10); err == nil {
		t.Fatal("blank query accepted")
	}
	if _, err := service.Search(context.Background(), 1, strings.Repeat("字", maxQueryRune+1), "all", 1, 10); err == nil {
		t.Fatal("oversized query accepted")
	}
	if _, err := service.Search(context.Background(), 1, "有效", "user", 1, 10); err == nil {
		t.Fatal("unsupported kind accepted")
	}
}

func TestSearchOnlyQueriesRequestedKind(t *testing.T) {
	repo := &repoStub{}
	service := NewService(repo)
	result, err := service.Search(context.Background(), 1, "标题", "question", 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if repo.articleCalls != 0 || repo.topicCalls != 0 {
		t.Fatalf("unexpected repository calls: article=%d topic=%d", repo.articleCalls, repo.topicCalls)
	}
	if result.Questions.PageSize != defaultPageSize || result.Questions.Page != 1 {
		t.Fatalf("page not normalized: %#v", result.Questions)
	}
	if len(result.Questions.Items) != 1 {
		t.Fatalf("question results missing")
	}
	if result.Articles.Items == nil || result.Topics.Items == nil {
		t.Fatal("empty sections must be empty arrays")
	}
}

func TestSearchRateLimit(t *testing.T) {
	limiter := &limiterStub{allowed: false}
	service := NewService(&repoStub{}, limiter)
	_, err := service.Search(context.Background(), 7, "关键词", "all", 1, 10)
	if !errors.Is(err, ErrRateLimited) {
		t.Fatalf("expected rate limit, got %v", err)
	}
	if limiter.calls != 1 {
		t.Fatalf("limiter not consulted: %d", limiter.calls)
	}
}

func TestSearchPatternEscaped(t *testing.T) {
	repo := &repoStub{}
	service := NewService(repo)
	if _, err := service.Search(context.Background(), 1, "50%_off", "question", 1, 10); err != nil {
		t.Fatal(err)
	}
	if repo.questionPattern != `%50\%\_off%` {
		t.Fatalf("pattern not escaped: %q", repo.questionPattern)
	}
}
