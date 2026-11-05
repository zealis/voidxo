package stats

import (
	"context"
	"testing"

	"github.com/voidxo/voidxo/internal/platform/cache"
)

type repoStub struct {
	counts Stats
	calls  int
}

func (r *repoStub) Counts(context.Context) (Stats, error) {
	r.calls++
	return r.counts, nil
}

func TestPublicReturnsCounts(t *testing.T) {
	repo := &repoStub{counts: Stats{Users: 3, Questions: 5, Articles: 2}}
	service := NewService(repo)
	service.SetCache(cache.Noop{})

	result, err := service.Public(context.Background())
	if err != nil {
		t.Fatalf("stats failed: %v", err)
	}
	if result.Users != 3 || result.Questions != 5 || result.Articles != 2 {
		t.Fatalf("unexpected stats: %+v", result)
	}
}

func TestPublicQueriesRepositoryWhenCacheMisses(t *testing.T) {
	repo := &repoStub{}
	service := NewService(repo)
	service.SetCache(cache.Noop{})

	for i := 0; i < 2; i++ {
		if _, err := service.Public(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
	if repo.calls != 2 {
		t.Fatalf("noop cache should always hit the repository, calls=%d", repo.calls)
	}
}
