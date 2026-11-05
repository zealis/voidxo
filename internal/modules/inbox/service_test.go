package inbox

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"

	"github.com/voidxo/voidxo/internal/platform/markdown"
)

type repoStub struct {
	recipient       Recipient
	recipientErr    error
	created         bool
	lastMarkdown    string
	lastRendered    string
	conversations   []Conversation
	messages        []Message
	markedUpdated   int
	unread          int
	lastCounterpart int64
}

func (r *repoStub) FindRecipientByID(context.Context, int64) (Recipient, error) {
	return r.recipient, r.recipientErr
}

func (r *repoStub) FindRecipientByUsername(context.Context, string) (Recipient, error) {
	return r.recipient, r.recipientErr
}

func (r *repoStub) CreateMessage(_ context.Context, senderID, receiverID int64, source, rendered string) (Message, error) {
	r.created = true
	r.lastMarkdown = source
	r.lastRendered = rendered
	return Message{ID: 1, SenderID: senderID, ReceiverID: receiverID, ContentMarkdown: source, ContentRendered: rendered}, nil
}

func (r *repoStub) ListConversations(context.Context, int64, int, int) ([]Conversation, bool, error) {
	return r.conversations, false, nil
}

func (r *repoStub) ListMessages(_ context.Context, _, counterpartID int64, _, _ int) ([]Message, bool, error) {
	r.lastCounterpart = counterpartID
	return r.messages, false, nil
}

func (r *repoStub) MarkConversationRead(context.Context, int64, int64) (int, error) {
	return r.markedUpdated, nil
}

func (r *repoStub) UnreadCount(context.Context, int64) (int, error) {
	return r.unread, nil
}

type limiterStub struct {
	allowed bool
	err     error
}

func (l limiterStub) Allow(context.Context, string) (bool, error) {
	return l.allowed, l.err
}

func newService(repo Repository) *Service {
	return NewService(repo, markdown.New(), slog.New(slog.NewTextHandler(io.Discard, nil)))
}

func TestSendRejectsSelfMessage(t *testing.T) {
	repo := &repoStub{recipient: Recipient{ID: 2, Active: true}}
	service := newService(repo)

	_, err := service.Send(context.Background(), 2, 2, "你好")
	var validation *ValidationError
	if !errors.As(err, &validation) {
		t.Fatalf("expected validation error, got %v", err)
	}
	if repo.created {
		t.Fatal("self message must not be stored")
	}
}

func TestSendRejectsDisabledRecipient(t *testing.T) {
	repo := &repoStub{recipient: Recipient{ID: 2, Active: false}}
	service := newService(repo)

	_, err := service.Send(context.Background(), 1, 2, "你好")
	var validation *ValidationError
	if !errors.As(err, &validation) {
		t.Fatalf("expected validation error, got %v", err)
	}
}

func TestSendRejectsEmptyAndOverlongContent(t *testing.T) {
	repo := &repoStub{recipient: Recipient{ID: 2, Active: true}}
	service := newService(repo)

	for _, content := range []string{"   ", strings.Repeat("字", maxContentLength+1)} {
		if _, err := service.Send(context.Background(), 1, 2, content); err == nil {
			t.Fatalf("content %d chars must be rejected", len(content))
		}
	}
	if repo.created {
		t.Fatal("invalid content must not be stored")
	}
}

func TestSendRejectsEmptyRenderedContent(t *testing.T) {
	repo := &repoStub{recipient: Recipient{ID: 2, Active: true}}
	service := newService(repo)

	if _, err := service.Send(context.Background(), 1, 2, "<script></script>"); err == nil {
		t.Fatal("content that renders to nothing must be rejected")
	}
}

func TestSendRateLimited(t *testing.T) {
	repo := &repoStub{recipient: Recipient{ID: 2, Active: true}}
	service := newService(repo)
	service.SetLimiter(limiterStub{allowed: false})

	if _, err := service.Send(context.Background(), 1, 2, "你好"); !errors.Is(err, ErrRateLimited) {
		t.Fatalf("expected rate limited, got %v", err)
	}
}

func TestSendAllowsWhenLimiterUnavailable(t *testing.T) {
	repo := &repoStub{recipient: Recipient{ID: 2, Active: true}}
	service := newService(repo)
	service.SetLimiter(limiterStub{err: errors.New("redis down")})

	message, err := service.Send(context.Background(), 1, 2, "你好")
	if err != nil {
		t.Fatalf("limiter failure must not block messaging: %v", err)
	}
	if message.ID != 1 {
		t.Fatalf("unexpected message: %+v", message)
	}
}

func TestSendRendersMarkdown(t *testing.T) {
	repo := &repoStub{recipient: Recipient{ID: 2, Active: true}}
	service := newService(repo)
	service.SetLimiter(limiterStub{allowed: true})

	if _, err := service.Send(context.Background(), 1, 2, "**加粗**"); err != nil {
		t.Fatalf("send failed: %v", err)
	}
	if !strings.Contains(repo.lastRendered, "<strong>加粗</strong>") {
		t.Fatalf("markdown not rendered: %q", repo.lastRendered)
	}
}

func TestMessagesRejectsSelfConversation(t *testing.T) {
	repo := &repoStub{recipient: Recipient{ID: 1, Active: true}}
	service := newService(repo)

	if _, err := service.Messages(context.Background(), 1, 1, 1, 20); err == nil {
		t.Fatal("self conversation must be rejected")
	}
}

func TestMessagesReturnsCounterpartAndEmptyList(t *testing.T) {
	repo := &repoStub{recipient: Recipient{ID: 2, Username: "tester", Active: true}}
	service := newService(repo)

	page, err := service.Messages(context.Background(), 1, 2, 1, 20)
	if err != nil {
		t.Fatal(err)
	}
	if page.Counterpart.Username != "tester" {
		t.Fatalf("counterpart missing: %+v", page.Counterpart)
	}
	if page.Items == nil {
		t.Fatal("empty list must serialize as []")
	}
	if repo.lastCounterpart != 2 {
		t.Fatalf("counterpart id not passed: %d", repo.lastCounterpart)
	}
}

func TestConversationsNormalizesEmptyList(t *testing.T) {
	service := newService(&repoStub{})
	page, err := service.Conversations(context.Background(), 1, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if page.Items == nil {
		t.Fatal("empty list must serialize as []")
	}
	if page.Page != 1 || page.PageSize != defaultPageSize {
		t.Fatalf("page not normalized: %+v", page)
	}
}

func TestMarkReadReturnsUpdatedAndUnread(t *testing.T) {
	repo := &repoStub{markedUpdated: 3, unread: 5}
	service := newService(repo)

	updated, unread, err := service.MarkRead(context.Background(), 1, 2)
	if err != nil {
		t.Fatal(err)
	}
	if updated != 3 || unread != 5 {
		t.Fatalf("unexpected result: %d %d", updated, unread)
	}
}

func TestRecipientByUsernameValidates(t *testing.T) {
	service := newService(&repoStub{recipient: Recipient{ID: 2, Username: "tester", Active: true}})
	if _, err := service.RecipientByUsername(context.Background(), "  "); err == nil {
		t.Fatal("blank username must be rejected")
	}
	recipient, err := service.RecipientByUsername(context.Background(), " tester ")
	if err != nil || recipient.ID != 2 {
		t.Fatalf("unexpected recipient: %+v %v", recipient, err)
	}
}

func TestRecipientByUsernameRejectsDisabled(t *testing.T) {
	service := newService(&repoStub{recipient: Recipient{ID: 2, Username: "tester", Active: false}})
	if _, err := service.RecipientByUsername(context.Background(), "tester"); err == nil {
		t.Fatal("disabled recipient must be rejected")
	}
}
