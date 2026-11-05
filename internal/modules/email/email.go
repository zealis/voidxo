// Package email 负责邮箱验证码的签发与校验。
// 验证码只保存哈希；明文随 Outbox 事件交给 Worker 投递，不在接口层回传。
package email

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"math/big"
	"regexp"
	"strings"
	"time"
)

const (
	PurposeRegister = "register"
	PurposeReset    = "reset"

	codeTTL        = 10 * time.Minute
	maxAttempts    = 5
	codeDigits     = 6
	minuteWindow   = time.Minute
	hourWindow     = time.Hour
	perEmailMinute = 1
	perEmailHour   = 5
	perClientHour  = 20
)

var (
	ErrInvalidEmail    = errors.New("invalid email address")
	ErrInvalidPurpose  = errors.New("invalid email code purpose")
	ErrRateLimited     = errors.New("email code rate limited")
	ErrCodeExpired     = errors.New("email code expired")
	ErrInvalidCode     = errors.New("email code incorrect")
	ErrTooManyAttempts = errors.New("too many email code attempts")
)

var emailPattern = regexp.MustCompile(`^[A-Za-z0-9._%+\-]+@[A-Za-z0-9.\-]+\.[A-Za-z]{2,}$`)

type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string { return e.Message }

// Code 是数据库中的验证码记录，只包含校验所需字段。
type Code struct {
	ID        int64
	CodeHash  string
	Attempts  int
	ExpiresAt time.Time
}

type Repository interface {
	// CreateCode 在同一事务内写入验证码并投递邮件事件。
	CreateCode(ctx context.Context, email, purpose, codeHash string, expiresAt time.Time, mail Mail) error
	LatestCode(ctx context.Context, email, purpose string) (Code, error)
	IncrementAttempts(ctx context.Context, id int64) error
	Consume(ctx context.Context, id int64) error
}

// Mail 是交给 Worker 投递的邮件内容。
type Mail struct {
	To      string
	Subject string
	Text    string
}

// Limiter 由 platform/ratelimit 实现，按调用指定配额。
type Limiter interface {
	AllowWithin(ctx context.Context, subject string, limit int64, window time.Duration) (bool, error)
}

// UserLookup 用于判断邮箱是否已注册：注册要求未注册，重置要求已注册。
type UserLookup interface {
	EmailExists(ctx context.Context, email string) (bool, error)
}

type Service struct {
	repo    Repository
	users   UserLookup
	limiter Limiter
	pepper  string
	logger  *slog.Logger
	now     func() time.Time
}

func NewService(repo Repository, lookup UserLookup, limiter Limiter, pepper string, logger *slog.Logger) *Service {
	return &Service{repo: repo, users: lookup, limiter: limiter, pepper: pepper, logger: logger, now: time.Now}
}

// SendCode 生成并投递验证码。为避免账号枚举，无论邮箱是否存在都返回成功，
// 只有确实需要发送时才写入验证码与邮件事件。
func (s *Service) SendCode(ctx context.Context, address, purpose, clientKey string) (time.Duration, error) {
	address = strings.ToLower(strings.TrimSpace(address))
	if !emailPattern.MatchString(address) {
		return 0, &ValidationError{Message: "邮箱格式不正确"}
	}
	if purpose != PurposeRegister && purpose != PurposeReset {
		return 0, &ValidationError{Message: "验证码用途不正确"}
	}
	if err := s.allow(ctx, "email:1m:"+address, perEmailMinute, minuteWindow); err != nil {
		return 0, err
	}
	if err := s.allow(ctx, "email:1h:"+address, perEmailHour, hourWindow); err != nil {
		return 0, err
	}
	if clientKey != "" {
		if err := s.allow(ctx, "client:1h:"+clientKey, perClientHour, hourWindow); err != nil {
			return 0, err
		}
	}

	exists, err := s.users.EmailExists(ctx, address)
	if err != nil {
		return 0, fmt.Errorf("check email existence: %w", err)
	}
	// 注册只给未注册邮箱发；重置只给已注册邮箱发。两种情况都返回成功。
	if (purpose == PurposeRegister) == exists {
		s.logger.Info("email code skipped by purpose", "purpose", purpose, "email_exists", exists)
		return codeTTL, nil
	}

	code, err := generateCode()
	if err != nil {
		return 0, fmt.Errorf("generate email code: %w", err)
	}
	expiresAt := s.now().Add(codeTTL)
	// 顺手清理一天前过期的记录，失败不影响本次发送。
	if purger, ok := s.repo.(interface{ PurgeExpired(context.Context) error }); ok {
		if err := purger.PurgeExpired(ctx); err != nil {
			s.logger.Warn("purge expired email codes failed", "error", err)
		}
	}
	if err := s.repo.CreateCode(ctx, address, purpose, s.hash(address, purpose, code), expiresAt, Mail{
		To:      address,
		Subject: subjectFor(purpose),
		Text:    textFor(purpose, code),
	}); err != nil {
		return 0, err
	}
	return codeTTL, nil
}

// Verify 校验验证码并立即消费，保证一次性。
func (s *Service) Verify(ctx context.Context, address, purpose, code string) error {
	address = strings.ToLower(strings.TrimSpace(address))
	code = strings.TrimSpace(code)
	if !emailPattern.MatchString(address) {
		return &ValidationError{Message: "邮箱格式不正确"}
	}
	if purpose != PurposeRegister && purpose != PurposeReset {
		return &ValidationError{Message: "验证码用途不正确"}
	}
	if code == "" || len(code) > 12 {
		return &ValidationError{Message: "验证码不正确"}
	}
	record, err := s.repo.LatestCode(ctx, address, purpose)
	if errors.Is(err, ErrCodeExpired) {
		return ErrCodeExpired
	}
	if err != nil {
		return err
	}
	if record.Attempts >= maxAttempts {
		return ErrTooManyAttempts
	}
	if !record.ExpiresAt.After(s.now()) {
		return ErrCodeExpired
	}
	expected := s.hash(address, purpose, code)
	if subtle.ConstantTimeCompare([]byte(expected), []byte(record.CodeHash)) != 1 {
		if err := s.repo.IncrementAttempts(ctx, record.ID); err != nil {
			return err
		}
		return ErrInvalidCode
	}
	return s.repo.Consume(ctx, record.ID)
}

func (s *Service) allow(ctx context.Context, subject string, limit int64, window time.Duration) error {
	if s.limiter == nil {
		return nil
	}
	allowed, err := s.limiter.AllowWithin(ctx, subject, limit, window)
	if err != nil {
		// 限流器故障不阻断找回密码与注册；记录告警即可。
		s.logger.Warn("email code limiter unavailable", "error", err)
		return nil
	}
	if !allowed {
		return ErrRateLimited
	}
	return nil
}

func (s *Service) hash(address, purpose, code string) string {
	digest := sha256.Sum256([]byte(strings.Join([]string{address, purpose, code, s.pepper}, "|")))
	return hex.EncodeToString(digest[:])
}

func generateCode() (string, error) {
	limit := big.NewInt(1000000)
	value, err := rand.Int(rand.Reader, limit)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", value.Int64()), nil
}

func subjectFor(purpose string) string {
	if purpose == PurposeReset {
		return "VOIDXO 重置密码验证码"
	}
	return "VOIDXO 注册验证码"
}

func textFor(purpose, code string) string {
	action := "注册"
	if purpose == PurposeReset {
		action = "重置密码"
	}
	return fmt.Sprintf("你的%s验证码是 %s，10 分钟内有效。\n如果这不是你本人的操作，请忽略这封邮件。", action, code)
}
