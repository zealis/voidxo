package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/voidxo/voidxo/internal/modules/user"
)

var (
	ErrInvalidCredentials   = errors.New("invalid credentials")
	ErrCurrentPassword      = errors.New("current password is incorrect")
	ErrTooManyAttempts      = errors.New("too many login attempts")
	ErrAuthUnavailable      = errors.New("authentication unavailable")
	ErrAuthenticationNeeded = errors.New("authentication required")
)

type Principal struct {
	UserID      int64
	Username    string
	Email       string
	Roles       []string
	Permissions []string
	DeviceID    string
	ExpiresAt   time.Time
}

func normalizeStrings(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}

// normalize 保证 Roles/Permissions 不为 nil，序列化后始终是数组。
func (p Principal) normalize() Principal {
	if p.Roles == nil {
		p.Roles = []string{}
	}
	if p.Permissions == nil {
		p.Permissions = []string{}
	}
	return p
}

func (p Principal) HasPermission(code string) bool {
	for _, permission := range p.Permissions {
		if permission == code {
			return true
		}
	}
	return false
}

type LoginInput struct {
	Login     string
	Password  string
	DeviceID  string
	IPAddress string
	UserAgent string
}

type LoginResult struct {
	Token     string
	ExpiresAt time.Time
	Principal Principal
	AvatarKey string
}

type Service struct {
	users       user.Repository
	hasher      *PasswordHasher
	sessions    SessionStore
	sessionRepo SessionRepository
	limiter     LoginLimiter
	idleTTL     time.Duration
	absoluteTTL time.Duration
	logger      *slog.Logger
	codes       CodeVerifier
	now         func() time.Time
	newToken    func() (plain, hash string, err error)
	dummyHash   string
}

func NewService(
	users user.Repository,
	hasher *PasswordHasher,
	sessions SessionStore,
	sessionRepo SessionRepository,
	limiter LoginLimiter,
	idleTTL time.Duration,
	absoluteTTL time.Duration,
	logger *slog.Logger,
) (*Service, error) {
	if users == nil || hasher == nil || sessions == nil || sessionRepo == nil || limiter == nil {
		return nil, errors.New("auth service dependencies are incomplete")
	}
	if idleTTL <= 0 || absoluteTTL <= 0 || idleTTL > absoluteTTL {
		return nil, errors.New("invalid session TTL configuration")
	}
	dummyHash, err := hasher.Hash("voidxo-invalid-login-placeholder")
	if err != nil {
		return nil, fmt.Errorf("create dummy password hash: %w", err)
	}
	return &Service{
		users: users, hasher: hasher, sessions: sessions, sessionRepo: sessionRepo, limiter: limiter,
		idleTTL: idleTTL, absoluteTTL: absoluteTTL, logger: logger, now: time.Now,
		newToken: newOpaqueToken, dummyHash: dummyHash,
	}, nil
}

func (s *Service) Login(ctx context.Context, input LoginInput) (LoginResult, error) {
	login := strings.TrimSpace(input.Login)
	if login == "" || input.Password == "" || len(login) > 320 || len(input.Password) > 1024 {
		return LoginResult{}, ErrInvalidCredentials
	}
	normalizedLogin := strings.ToLower(login)
	clientKey := strings.TrimSpace(input.IPAddress)
	if clientKey == "" {
		clientKey = "unknown"
	}
	ipAllowed, err := s.limiter.Allow(ctx, clientKey, "*")
	if err != nil {
		return LoginResult{}, wrapUnavailable(err)
	}
	loginAllowed, err := s.limiter.Allow(ctx, clientKey, normalizedLogin)
	if err != nil {
		return LoginResult{}, wrapUnavailable(err)
	}
	if !ipAllowed || !loginAllowed {
		return LoginResult{}, ErrTooManyAttempts
	}

	account, err := s.users.FindByLogin(ctx, login)
	if errors.Is(err, user.ErrNotFound) {
		_, _ = s.hasher.Verify(s.dummyHash, input.Password)
		return LoginResult{}, ErrInvalidCredentials
	}
	if err != nil {
		return LoginResult{}, wrapUnavailable(err)
	}
	passwordOK, err := s.hasher.Verify(account.PasswordHash, input.Password)
	if err != nil || !passwordOK || !account.Active() {
		return LoginResult{}, ErrInvalidCredentials
	}

	result, err := s.issueSession(ctx, account, input.DeviceID, input.IPAddress, input.UserAgent)
	if err != nil {
		return LoginResult{}, err
	}
	if err := s.users.RecordLogin(ctx, account.ID, input.IPAddress); err != nil {
		s.logger.Warn("failed to record login", "user_id", account.ID, "error", err)
	}
	if err := s.limiter.Reset(ctx, clientKey, "*"); err != nil {
		s.logger.Warn("failed to reset ip login limiter", "error", err)
	}
	if err := s.limiter.Reset(ctx, clientKey, normalizedLogin); err != nil {
		s.logger.Warn("failed to reset login limiter", "error", err)
	}
	return result, nil
}

// issueSession 为已通过身份校验的账号签发会话；登录与注册共用同一套逻辑。
func (s *Service) issueSession(
	ctx context.Context,
	account user.Account,
	deviceID, ipAddress, userAgent string,
) (LoginResult, error) {
	roles, err := s.users.Roles(ctx, account.ID)
	if err != nil {
		return LoginResult{}, wrapUnavailable(err)
	}
	permissions, err := s.users.Permissions(ctx, account.ID)
	if err != nil {
		return LoginResult{}, wrapUnavailable(err)
	}
	token, tokenHash, err := s.newToken()
	if err != nil {
		return LoginResult{}, wrapUnavailable(err)
	}
	now := s.now()
	expiresAt := now.Add(s.absoluteTTL)
	session := Session{
		UserID: account.ID, Username: account.Username, Email: account.Email,
		Roles: normalizeStrings(roles), Permissions: normalizeStrings(permissions), DeviceID: strings.TrimSpace(deviceID),
		CreatedAt: now, LastActiveAt: now, ExpiresAt: expiresAt,
	}
	if err := s.sessionRepo.Create(ctx, SessionRecord{
		UserID: account.ID, TokenHash: tokenHash, DeviceID: session.DeviceID,
		IPAddress: ipAddress, UserAgent: userAgent, CreatedAt: now, ExpiresAt: expiresAt,
	}); err != nil {
		return LoginResult{}, wrapUnavailable(err)
	}
	if err := s.sessions.Create(ctx, tokenHash, session, s.sessionTTL(now, expiresAt)); err != nil {
		if revokeErr := s.sessionRepo.Revoke(ctx, tokenHash, "redis_failure"); revokeErr != nil {
			s.logger.Error("failed to revoke session after redis failure", "error", revokeErr)
		}
		return LoginResult{}, wrapUnavailable(err)
	}
	return LoginResult{
		Token: token, ExpiresAt: expiresAt, AvatarKey: account.AvatarKey,
		Principal: Principal{
			UserID: account.ID, Username: account.Username, Email: account.Email,
			Roles: session.Roles, Permissions: session.Permissions, DeviceID: session.DeviceID, ExpiresAt: expiresAt,
		}.normalize(),
	}, nil
}

func (s *Service) Authenticate(ctx context.Context, token string) (Principal, error) {
	if strings.TrimSpace(token) == "" || len(token) > 1024 {
		return Principal{}, ErrAuthenticationNeeded
	}
	tokenHash := HashToken(token)
	session, err := s.sessions.Get(ctx, tokenHash)
	if errors.Is(err, ErrSessionNotFound) {
		return Principal{}, ErrAuthenticationNeeded
	}
	if err != nil {
		return Principal{}, wrapUnavailable(err)
	}
	now := s.now()
	if session.ExpiresAt.IsZero() || !now.Before(session.ExpiresAt) {
		_ = s.sessions.Delete(ctx, tokenHash)
		return Principal{}, ErrAuthenticationNeeded
	}
	if err := s.sessions.Touch(ctx, tokenHash, s.sessionTTL(now, session.ExpiresAt)); err != nil {
		if errors.Is(err, ErrSessionNotFound) {
			return Principal{}, ErrAuthenticationNeeded
		}
		return Principal{}, wrapUnavailable(err)
	}
	return Principal{
		UserID: session.UserID, Username: session.Username, Email: session.Email,
		Roles: session.Roles, Permissions: session.Permissions, DeviceID: session.DeviceID, ExpiresAt: session.ExpiresAt,
	}.normalize(), nil
}

// AvatarKey 读取用户当前头像标识。头像属于可变更的资料，不写入会话快照，
// 避免用户上传新头像后已有会话仍返回旧值。
func (s *Service) AvatarKey(ctx context.Context, userID int64) (string, error) {
	account, err := s.users.FindByID(ctx, userID)
	if err != nil {
		return "", err
	}
	return account.AvatarKey, nil
}

func (s *Service) Logout(ctx context.Context, token string) error {
	if strings.TrimSpace(token) == "" {
		return nil
	}
	tokenHash := HashToken(token)
	if err := s.sessions.Delete(ctx, tokenHash); err != nil {
		return wrapUnavailable(err)
	}
	if err := s.sessionRepo.Revoke(ctx, tokenHash, "logout"); err != nil {
		return wrapUnavailable(err)
	}
	return nil
}

func (s *Service) ChangePassword(ctx context.Context, userID int64, currentPassword, newPassword string) error {
	if err := ValidateNewPassword(newPassword); err != nil {
		return err
	}
	account, err := s.users.FindByID(ctx, userID)
	if err != nil || !account.Active() {
		return ErrAuthenticationNeeded
	}
	ok, err := s.hasher.Verify(account.PasswordHash, currentPassword)
	if err != nil || !ok {
		return ErrCurrentPassword
	}
	newHash, err := s.hasher.Hash(newPassword)
	if err != nil {
		return err
	}
	hashes, err := s.sessionRepo.ActiveHashes(ctx, userID)
	if err != nil {
		return wrapUnavailable(err)
	}
	for _, tokenHash := range hashes {
		if err := s.sessions.Delete(ctx, tokenHash); err != nil {
			return wrapUnavailable(err)
		}
	}
	if err := s.users.SetPassword(ctx, userID, newHash); err != nil {
		return wrapUnavailable(err)
	}
	if _, err := s.sessionRepo.RevokeAll(ctx, userID, "password_changed"); err != nil {
		return wrapUnavailable(err)
	}
	return nil
}

// RevokeUserSessions 立即让指定用户的所有会话失效，用于角色或状态变更后回收旧权限。
func (s *Service) RevokeUserSessions(ctx context.Context, userID int64, reason string) error {
	hashes, err := s.sessionRepo.ActiveHashes(ctx, userID)
	if err != nil {
		return wrapUnavailable(err)
	}
	for _, tokenHash := range hashes {
		if err := s.sessions.Delete(ctx, tokenHash); err != nil {
			return wrapUnavailable(err)
		}
	}
	if _, err := s.sessionRepo.RevokeAll(ctx, userID, reason); err != nil {
		return wrapUnavailable(err)
	}
	return nil
}

func (s *Service) sessionTTL(now, expiresAt time.Time) time.Duration {
	remaining := expiresAt.Sub(now)
	if remaining <= 0 {
		return time.Second
	}
	if s.idleTTL < remaining {
		return s.idleTTL
	}
	return remaining
}

func wrapUnavailable(err error) error {
	return fmt.Errorf("%w: %v", ErrAuthUnavailable, err)
}

func newOpaqueToken() (string, string, error) {
	value := make([]byte, 32)
	if _, err := rand.Read(value); err != nil {
		return "", "", err
	}
	plain := base64.RawURLEncoding.EncodeToString(value)
	return plain, HashToken(plain), nil
}

func HashToken(token string) string {
	digest := sha256.Sum256([]byte(token))
	return hex.EncodeToString(digest[:])
}
