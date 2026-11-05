package auth

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/voidxo/voidxo/internal/modules/user"
)

type fakeUserRepo struct {
	accounts       map[int64]user.Account
	roles          []string
	permissions    []string
	loginRecorded  bool
	passwordSet    bool
	created        []user.CreateParams
	duplicateOnAdd bool
	nextID         int64
}

func (r *fakeUserRepo) FindByLogin(_ context.Context, login string) (user.Account, error) {
	for _, account := range r.accounts {
		if account.Username == login || account.Email == login {
			return account, nil
		}
	}
	return user.Account{}, user.ErrNotFound
}

func (r *fakeUserRepo) FindByID(_ context.Context, id int64) (user.Account, error) {
	account, ok := r.accounts[id]
	if !ok {
		return user.Account{}, user.ErrNotFound
	}
	return account, nil
}

func (r *fakeUserRepo) Create(_ context.Context, params user.CreateParams) (user.Account, error) {
	if r.duplicateOnAdd {
		return user.Account{}, user.ErrDuplicate
	}
	for _, account := range r.accounts {
		if account.Username == params.Username || account.Email == params.Email {
			return user.Account{}, user.ErrDuplicate
		}
	}
	r.nextID++
	account := user.Account{
		ID: r.nextID, Username: params.Username, Email: params.Email,
		PasswordHash: params.PasswordHash, Status: 1,
	}
	r.accounts[account.ID] = account
	r.created = append(r.created, params)
	return account, nil
}

func (r *fakeUserRepo) SetPassword(_ context.Context, id int64, hash string) error {
	account := r.accounts[id]
	account.PasswordHash = hash
	r.accounts[id] = account
	r.passwordSet = true
	return nil
}

func (r *fakeUserRepo) RecordLogin(context.Context, int64, string) error {
	r.loginRecorded = true
	return nil
}

func (r *fakeUserRepo) UpdateProfile(context.Context, int64, user.UpdateProfileParams) (user.Account, error) {
	return user.Account{}, errors.New("not implemented")
}

func (r *fakeUserRepo) Permissions(context.Context, int64) ([]string, error) {
	return r.permissions, nil
}

func (r *fakeUserRepo) Roles(context.Context, int64) ([]string, error) {
	return r.roles, nil
}

type fakeSessionStore struct {
	data        map[string]Session
	unavailable bool
}

func newFakeSessionStore() *fakeSessionStore {
	return &fakeSessionStore{data: make(map[string]Session)}
}

func (s *fakeSessionStore) Create(_ context.Context, hash string, session Session, _ time.Duration) error {
	if s.unavailable {
		return ErrSessionStoreUnavailable
	}
	s.data[hash] = session
	return nil
}

func (s *fakeSessionStore) Get(_ context.Context, hash string) (Session, error) {
	if s.unavailable {
		return Session{}, ErrSessionStoreUnavailable
	}
	session, ok := s.data[hash]
	if !ok {
		return Session{}, ErrSessionNotFound
	}
	return session, nil
}

func (s *fakeSessionStore) Touch(_ context.Context, hash string, _ time.Duration) error {
	if s.unavailable {
		return ErrSessionStoreUnavailable
	}
	if _, ok := s.data[hash]; !ok {
		return ErrSessionNotFound
	}
	return nil
}

func (s *fakeSessionStore) Delete(_ context.Context, hash string) error {
	if s.unavailable {
		return ErrSessionStoreUnavailable
	}
	delete(s.data, hash)
	return nil
}

type fakeSessionRepo struct {
	records map[string]SessionRecord
	revoked map[string]bool
}

func newFakeSessionRepo() *fakeSessionRepo {
	return &fakeSessionRepo{records: make(map[string]SessionRecord), revoked: make(map[string]bool)}
}

func (r *fakeSessionRepo) Create(_ context.Context, record SessionRecord) error {
	r.records[record.TokenHash] = record
	return nil
}

func (r *fakeSessionRepo) ActiveHashes(_ context.Context, _ int64) ([]string, error) {
	hashes := make([]string, 0, len(r.records))
	for hash := range r.records {
		if !r.revoked[hash] {
			hashes = append(hashes, hash)
		}
	}
	return hashes, nil
}

func (r *fakeSessionRepo) Revoke(_ context.Context, hash, _ string) error {
	r.revoked[hash] = true
	return nil
}

func (r *fakeSessionRepo) RevokeAll(_ context.Context, _ int64, _ string) ([]string, error) {
	var hashes []string
	for hash := range r.records {
		if !r.revoked[hash] {
			r.revoked[hash] = true
			hashes = append(hashes, hash)
		}
	}
	return hashes, nil
}

type fakeLimiter struct {
	allowed bool
	err     error
	reset   bool
}

func (l *fakeLimiter) Allow(context.Context, string, string) (bool, error) {
	return l.allowed, l.err
}
func (l *fakeLimiter) Reset(context.Context, string, string) error {
	l.reset = true
	return nil
}

type authHarness struct {
	service     *Service
	users       *fakeUserRepo
	sessions    *fakeSessionStore
	sessionRepo *fakeSessionRepo
	limiter     *fakeLimiter
	now         time.Time
	nextTokenID int
}

func newAuthHarness(t *testing.T) *authHarness {
	t.Helper()
	hasher := testHasher()
	passwordHash, err := hasher.Hash("correct-password")
	if err != nil {
		t.Fatal(err)
	}
	h := &authHarness{
		users: &fakeUserRepo{
			accounts: map[int64]user.Account{1: {
				ID: 1, Username: "alice", Email: "alice@example.com", PasswordHash: passwordHash,
				Status: 1, CreatedAt: time.Now(), UpdatedAt: time.Now(),
			}},
			roles: []string{"moderator"}, permissions: []string{"content.moderate"},
		},
		sessions: newFakeSessionStore(), sessionRepo: newFakeSessionRepo(),
		limiter: &fakeLimiter{allowed: true}, now: time.Date(2026, 11, 5, 0, 0, 0, 0, time.UTC),
	}
	service, err := NewService(
		h.users, hasher, h.sessions, h.sessionRepo, h.limiter, 7*24*time.Hour, 30*24*time.Hour,
		slog.New(slog.NewTextHandler(io.Discard, nil)),
	)
	if err != nil {
		t.Fatal(err)
	}
	service.now = func() time.Time { return h.now }
	service.newToken = func() (string, string, error) {
		h.nextTokenID++
		token := fmt.Sprintf("test-session-token-%d", h.nextTokenID)
		return token, HashToken(token), nil
	}
	h.service = service
	return h
}

func (h *authHarness) login(t *testing.T) LoginResult {
	t.Helper()
	result, err := h.service.Login(context.Background(), LoginInput{
		Login: "alice", Password: "correct-password", DeviceID: "test-device", IPAddress: "203.0.113.9",
	})
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func TestLoginCreatesSessionAndAuthenticates(t *testing.T) {
	h := newAuthHarness(t)
	result := h.login(t)
	if result.Token == "" || result.Principal.UserID != 1 {
		t.Fatalf("unexpected login result: %#v", result)
	}
	if !h.users.loginRecorded || !h.limiter.reset {
		t.Fatal("login side effects were not recorded")
	}
	principal, err := h.service.Authenticate(context.Background(), result.Token)
	if err != nil {
		t.Fatal(err)
	}
	if !principal.HasPermission("content.moderate") {
		t.Fatalf("permissions missing: %#v", principal)
	}
}

func TestLoginRejectsInvalidPasswordAndDisabledAccount(t *testing.T) {
	h := newAuthHarness(t)
	if _, err := h.service.Login(context.Background(), LoginInput{Login: "alice", Password: "wrong-password"}); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("wrong password returned %v", err)
	}
	account := h.users.accounts[1]
	account.Status = 0
	h.users.accounts[1] = account
	if _, err := h.service.Login(context.Background(), LoginInput{Login: "alice", Password: "correct-password"}); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("disabled account returned %v", err)
	}
}

func TestLoginRateLimitAndSessionExpiry(t *testing.T) {
	h := newAuthHarness(t)
	h.limiter.allowed = false
	if _, err := h.service.Login(context.Background(), LoginInput{Login: "alice", Password: "correct-password"}); !errors.Is(err, ErrTooManyAttempts) {
		t.Fatalf("rate limit returned %v", err)
	}
	h.limiter.allowed = true
	result := h.login(t)
	h.now = result.ExpiresAt.Add(time.Second)
	if _, err := h.service.Authenticate(context.Background(), result.Token); !errors.Is(err, ErrAuthenticationNeeded) {
		t.Fatalf("expired session returned %v", err)
	}
	if len(h.sessions.data) != 0 {
		t.Fatal("expired session was not removed")
	}
}

func TestLogoutRevokesSession(t *testing.T) {
	h := newAuthHarness(t)
	result := h.login(t)
	if err := h.service.Logout(context.Background(), result.Token); err != nil {
		t.Fatal(err)
	}
	if _, err := h.service.Authenticate(context.Background(), result.Token); !errors.Is(err, ErrAuthenticationNeeded) {
		t.Fatalf("logged out session still valid: %v", err)
	}
	if !h.sessionRepo.revoked[HashToken(result.Token)] {
		t.Fatal("session record was not revoked")
	}
}

func TestChangePasswordRevokesAllSessions(t *testing.T) {
	h := newAuthHarness(t)
	result := h.login(t)
	if err := h.service.ChangePassword(context.Background(), 1, "correct-password", "new-long-password"); err != nil {
		t.Fatal(err)
	}
	if !h.users.passwordSet {
		t.Fatal("password was not updated")
	}
	if len(h.sessions.data) != 0 {
		t.Fatal("active redis sessions remain after password change")
	}
	if !h.sessionRepo.revoked[HashToken(result.Token)] {
		t.Fatal("database session was not revoked")
	}
	if _, err := h.service.Authenticate(context.Background(), result.Token); !errors.Is(err, ErrAuthenticationNeeded) {
		t.Fatalf("old session still valid: %v", err)
	}
}

func (r *fakeUserRepo) SetAvatar(context.Context, int64, string) error { return nil }
func (r *fakeUserRepo) EmailExists(_ context.Context, email string) (bool, error) {
	for _, account := range r.accounts {
		if account.Email == email {
			return true, nil
		}
	}
	return false, nil
}

func (r *fakeUserRepo) FindByEmail(_ context.Context, email string) (user.Account, error) {
	for _, account := range r.accounts {
		if account.Email == email {
			return account, nil
		}
	}
	return user.Account{}, user.ErrNotFound
}
func (r *fakeUserRepo) SetCover(context.Context, int64, string) error { return nil }

func (r *fakeUserRepo) FindPublic(context.Context, int64) (user.PublicProfile, error) {
	return user.PublicProfile{}, user.ErrNotFound
}

func (r *fakeUserRepo) ListPublic(context.Context, string, int, int) ([]user.PublicProfile, bool, error) {
	return nil, false, nil
}
