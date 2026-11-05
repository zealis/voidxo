package email

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"regexp"
	"strings"
	"testing"
	"time"
)

type storedCode struct {
	id       int64
	email    string
	purpose  string
	hash     string
	attempts int
	expires  time.Time
	consumed bool
}

type repoStub struct {
	records  []*storedCode
	lastMail Mail
	nextID   int64
}

func (r *repoStub) CreateCode(_ context.Context, address, purpose, codeHash string, expiresAt time.Time, mail Mail) error {
	r.nextID++
	r.records = append(r.records, &storedCode{
		id: r.nextID, email: address, purpose: purpose, hash: codeHash, expires: expiresAt,
	})
	r.lastMail = mail
	return nil
}

func (r *repoStub) LatestCode(_ context.Context, address, purpose string) (Code, error) {
	for index := len(r.records) - 1; index >= 0; index-- {
		record := r.records[index]
		if record.email == address && record.purpose == purpose && !record.consumed {
			return Code{ID: record.id, CodeHash: record.hash, Attempts: record.attempts, ExpiresAt: record.expires}, nil
		}
	}
	return Code{}, ErrCodeExpired
}

func (r *repoStub) IncrementAttempts(_ context.Context, id int64) error {
	for _, record := range r.records {
		if record.id == id {
			record.attempts++
		}
	}
	return nil
}

func (r *repoStub) Consume(_ context.Context, id int64) error {
	for _, record := range r.records {
		if record.id == id {
			if record.consumed {
				return ErrCodeExpired
			}
			record.consumed = true
		}
	}
	return nil
}

type lookupStub struct {
	exists bool
	err    error
}

func (l lookupStub) EmailExists(context.Context, string) (bool, error) {
	return l.exists, l.err
}

type limiterStub struct {
	allowed bool
	err     error
	calls   int
}

func (l *limiterStub) AllowWithin(context.Context, string, int64, time.Duration) (bool, error) {
	l.calls++
	return l.allowed, l.err
}

func newTestService(repo Repository, exists bool) *Service {
	return NewService(repo, lookupStub{exists: exists}, &limiterStub{allowed: true}, "pepper",
		slog.New(slog.NewTextHandler(io.Discard, nil)))
}

func TestSendCodeRejectsInvalidInput(t *testing.T) {
	service := newTestService(&repoStub{}, false)
	if _, err := service.SendCode(context.Background(), "not-an-email", PurposeRegister, ""); err == nil {
		t.Fatal("invalid email must be rejected")
	}
	if _, err := service.SendCode(context.Background(), "a@b.com", "unknown", ""); err == nil {
		t.Fatal("invalid purpose must be rejected")
	}
}

func TestSendCodeSkipsRegisterForExistingEmail(t *testing.T) {
	repo := &repoStub{}
	service := newTestService(repo, true)

	if _, err := service.SendCode(context.Background(), "a@b.com", PurposeRegister, ""); err != nil {
		t.Fatalf("send failed: %v", err)
	}
	if len(repo.records) != 0 {
		t.Fatal("register code must not be issued for an existing email")
	}
}

func TestSendCodeSkipsResetForUnknownEmail(t *testing.T) {
	repo := &repoStub{}
	service := newTestService(repo, false)

	if _, err := service.SendCode(context.Background(), "a@b.com", PurposeReset, ""); err != nil {
		t.Fatalf("send failed: %v", err)
	}
	if len(repo.records) != 0 {
		t.Fatal("reset code must not be issued for an unknown email")
	}
}

func TestSendCodeStoresHashAndQueuesMail(t *testing.T) {
	repo := &repoStub{}
	service := newTestService(repo, false)

	if _, err := service.SendCode(context.Background(), "New@Example.com", PurposeRegister, "127.0.0.1"); err != nil {
		t.Fatalf("send failed: %v", err)
	}
	if len(repo.records) != 1 {
		t.Fatalf("expected one code, got %d", len(repo.records))
	}
	record := repo.records[0]
	if record.email != "new@example.com" {
		t.Fatalf("email not normalized: %q", record.email)
	}
	if strings.Contains(record.hash, repo.lastMail.Text) && record.hash == repo.lastMail.Text {
		t.Fatal("code must not be stored in plaintext")
	}
	if len(record.hash) != 64 {
		t.Fatalf("expected sha256 hex, got %q", record.hash)
	}
	if !strings.Contains(repo.lastMail.Subject, "注册") {
		t.Fatalf("unexpected subject: %q", repo.lastMail.Subject)
	}
	if !strings.Contains(repo.lastMail.Text, "验证码") {
		t.Fatalf("mail body missing code text: %q", repo.lastMail.Text)
	}
}

func TestSendCodeRateLimited(t *testing.T) {
	repo := &repoStub{}
	service := NewService(repo, lookupStub{}, &limiterStub{allowed: false}, "", slog.New(slog.NewTextHandler(io.Discard, nil)))
	if _, err := service.SendCode(context.Background(), "a@b.com", PurposeRegister, ""); !errors.Is(err, ErrRateLimited) {
		t.Fatalf("expected rate limit, got %v", err)
	}
	if len(repo.records) != 0 {
		t.Fatal("rate limited request must not create a code")
	}
}

func TestSendCodeAllowsWhenLimiterUnavailable(t *testing.T) {
	repo := &repoStub{}
	service := NewService(repo, lookupStub{}, &limiterStub{err: errors.New("redis down")}, "",
		slog.New(slog.NewTextHandler(io.Discard, nil)))
	if _, err := service.SendCode(context.Background(), "a@b.com", PurposeRegister, ""); err != nil {
		t.Fatalf("limiter failure must not block sending: %v", err)
	}
	if len(repo.records) != 1 {
		t.Fatal("code should still be created")
	}
}

func TestVerifyConsumesCorrectCode(t *testing.T) {
	repo := &repoStub{}
	service := newTestService(repo, false)
	if _, err := service.SendCode(context.Background(), "a@b.com", PurposeRegister, ""); err != nil {
		t.Fatal(err)
	}
	code := extractCode(repo.lastMail.Text)

	if err := service.Verify(context.Background(), "a@b.com", PurposeRegister, code); err != nil {
		t.Fatalf("verify failed: %v", err)
	}
	if !repo.records[0].consumed {
		t.Fatal("code must be consumed after verification")
	}
	// 二次使用应视为过期
	if err := service.Verify(context.Background(), "a@b.com", PurposeRegister, code); !errors.Is(err, ErrCodeExpired) {
		t.Fatalf("reused code must be rejected, got %v", err)
	}
}

func TestVerifyRejectsWrongCodeAndCountsAttempts(t *testing.T) {
	repo := &repoStub{}
	service := newTestService(repo, false)
	if _, err := service.SendCode(context.Background(), "a@b.com", PurposeRegister, ""); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < maxAttempts; i++ {
		if err := service.Verify(context.Background(), "a@b.com", PurposeRegister, "000000"); !errors.Is(err, ErrInvalidCode) {
			t.Fatalf("attempt %d: expected invalid code, got %v", i+1, err)
		}
	}
	if repo.records[0].attempts != maxAttempts {
		t.Fatalf("attempts not counted: %d", repo.records[0].attempts)
	}
	// 达到上限后即使给对码也拒绝
	code := extractCode(repo.lastMail.Text)
	if err := service.Verify(context.Background(), "a@b.com", PurposeRegister, code); !errors.Is(err, ErrTooManyAttempts) {
		t.Fatalf("expected too many attempts, got %v", err)
	}
}

func TestVerifyRejectsExpiredCode(t *testing.T) {
	repo := &repoStub{}
	service := newTestService(repo, false)
	if _, err := service.SendCode(context.Background(), "a@b.com", PurposeRegister, ""); err != nil {
		t.Fatal(err)
	}
	repo.records[0].expires = time.Now().Add(-time.Minute)
	code := extractCode(repo.lastMail.Text)
	if err := service.Verify(context.Background(), "a@b.com", PurposeRegister, code); !errors.Is(err, ErrCodeExpired) {
		t.Fatalf("expected expired, got %v", err)
	}
}

func TestVerifyRejectsCrossPurpose(t *testing.T) {
	repo := &repoStub{}
	service := newTestService(repo, false)
	if _, err := service.SendCode(context.Background(), "a@b.com", PurposeRegister, ""); err != nil {
		t.Fatal(err)
	}
	code := extractCode(repo.lastMail.Text)
	if err := service.Verify(context.Background(), "a@b.com", PurposeReset, code); !errors.Is(err, ErrInvalidCode) && !errors.Is(err, ErrCodeExpired) {
		t.Fatalf("cross purpose code must be rejected, got %v", err)
	}
}

var codePattern = regexp.MustCompile(`\b\d{6}\b`)

// extractCode 从邮件正文里取出 6 位验证码，模拟用户读信。
func extractCode(text string) string {
	return codePattern.FindString(text)
}
