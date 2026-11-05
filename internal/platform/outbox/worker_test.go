package outbox

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"
)

type storeStub struct {
	events    []Event
	completed []int64
	failed    []struct {
		id    int64
		retry time.Duration
	}
	claimErr error
}

func (s *storeStub) Claim(context.Context, int) ([]Event, error) {
	if s.claimErr != nil {
		return nil, s.claimErr
	}
	events := s.events
	s.events = nil
	return events, nil
}

func (s *storeStub) Complete(_ context.Context, id int64) error {
	s.completed = append(s.completed, id)
	return nil
}

func (s *storeStub) Fail(_ context.Context, id int64, _ string, retry time.Duration) error {
	s.failed = append(s.failed, struct {
		id    int64
		retry time.Duration
	}{id: id, retry: retry})
	return nil
}

func newTestWorker(store Store) *Worker {
	return NewWorker(store, slog.New(slog.NewTextHandler(io.Discard, nil)))
}

func TestWorkerDispatchesByPrefix(t *testing.T) {
	store := &storeStub{events: []Event{
		{ID: 1, Type: EventQuestionCreated, Attempts: 1},
		{ID: 2, Type: EventArticleDeleted, Attempts: 1},
	}}
	worker := newTestWorker(store)
	handled := 0
	worker.HandlePrefix("content.", func(context.Context, Event) error {
		handled++
		return nil
	})

	count, err := worker.ProcessOnce(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if count != 2 || handled != 2 {
		t.Fatalf("unexpected counts: count=%d handled=%d", count, handled)
	}
	if len(store.completed) != 2 {
		t.Fatalf("events not completed: %#v", store.completed)
	}
}

func TestWorkerRetriesWithBackoff(t *testing.T) {
	store := &storeStub{events: []Event{{ID: 7, Type: EventCommentCreated, Attempts: 1}}}
	worker := newTestWorker(store)
	worker.HandlePrefix("content.", func(context.Context, Event) error {
		return errors.New("temporary failure")
	})

	if _, err := worker.ProcessOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(store.failed) != 1 || store.failed[0].retry <= 0 {
		t.Fatalf("expected retry, got %#v", store.failed)
	}
	if len(store.completed) != 0 {
		t.Fatal("failed event must not be completed")
	}
}

func TestWorkerGivesUpAfterMaxAttempts(t *testing.T) {
	store := &storeStub{events: []Event{{ID: 9, Type: EventQuestionUpdated, Attempts: maxAttempts}}}
	worker := newTestWorker(store)
	worker.HandlePrefix("content.", func(context.Context, Event) error {
		return errors.New("permanent failure")
	})

	if _, err := worker.ProcessOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(store.failed) != 1 || store.failed[0].retry != 0 {
		t.Fatalf("expected terminal failure, got %#v", store.failed)
	}
}

func TestWorkerCompletesUnknownEvents(t *testing.T) {
	store := &storeStub{events: []Event{{ID: 11, Type: "unknown.event", Attempts: 1}}}
	worker := newTestWorker(store)
	if _, err := worker.ProcessOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(store.completed) != 1 {
		t.Fatalf("unknown event should be completed, got %#v", store.completed)
	}
}

func TestBackoffSchedule(t *testing.T) {
	if backoff(1) != 2*time.Second {
		t.Fatalf("unexpected backoff for attempt 1: %s", backoff(1))
	}
	if backoff(maxAttempts) != 0 {
		t.Fatal("max attempts must stop retrying")
	}
}
