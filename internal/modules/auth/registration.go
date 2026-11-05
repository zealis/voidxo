package auth

import (
	"context"
	"errors"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/voidxo/voidxo/internal/modules/user"
)

var (
	ErrCodeInvalid     = errors.New("verification code incorrect")
	ErrCodeExpired     = errors.New("verification code expired")
	ErrCodeAttempts    = errors.New("too many verification code attempts")
	ErrAccountExists   = errors.New("account already exists")
	ErrUsernameInvalid = errors.New("username invalid")
)

// CodeVerifier 由邮箱验证码模块实现；auth 只依赖「校验并消费」这一行为。
type CodeVerifier interface {
	Verify(ctx context.Context, address, purpose, code string) error
}

// 验证码用途，与邮箱模块的取值保持一致。
const (
	CodePurposeRegister = "register"
	CodePurposeReset    = "reset"
)

type RegisterInput struct {
	Username  string
	Email     string
	Password  string
	Code      string
	DeviceID  string
	IPAddress string
	UserAgent string
}

type ResetPasswordInput struct {
	Email       string
	Code        string
	NewPassword string
}

func (s *Service) SetCodeVerifier(verifier CodeVerifier) {
	s.codes = verifier
}

// Register 校验邮箱验证码后创建普通账号，并直接签发会话。
func (s *Service) Register(ctx context.Context, input RegisterInput) (LoginResult, error) {
	username := strings.TrimSpace(input.Username)
	address := normalizeEmail(input.Email)
	if !validUsername(username) {
		return LoginResult{}, ErrUsernameInvalid
	}
	if address == "" || strings.TrimSpace(input.Code) == "" {
		return LoginResult{}, ErrCodeInvalid
	}
	if err := ValidateNewPassword(input.Password); err != nil {
		return LoginResult{}, err
	}
	if s.codes == nil {
		return LoginResult{}, ErrAuthUnavailable
	}
	if err := s.codes.Verify(ctx, address, CodePurposeRegister, input.Code); err != nil {
		return LoginResult{}, err
	}

	passwordHash, err := s.hasher.Hash(input.Password)
	if err != nil {
		return LoginResult{}, wrapUnavailable(err)
	}
	account, err := s.users.Create(ctx, user.CreateParams{
		Username:     username,
		Email:        address,
		PasswordHash: passwordHash,
		CreateIP:     input.IPAddress,
		RoleCode:     "user",
	})
	if errors.Is(err, user.ErrDuplicate) {
		return LoginResult{}, ErrAccountExists
	}
	if err != nil {
		return LoginResult{}, wrapUnavailable(err)
	}
	result, err := s.issueSession(ctx, account, input.DeviceID, input.IPAddress, input.UserAgent)
	if err != nil {
		return LoginResult{}, err
	}
	if err := s.users.RecordLogin(ctx, account.ID, input.IPAddress); err != nil {
		s.logger.Warn("failed to record registration login", "user_id", account.ID, "error", err)
	}
	return result, nil
}

// ResetPassword 校验验证码后重设密码，并让该账号的全部旧会话失效。
func (s *Service) ResetPassword(ctx context.Context, input ResetPasswordInput) error {
	address := normalizeEmail(input.Email)
	if address == "" || strings.TrimSpace(input.Code) == "" {
		return ErrCodeInvalid
	}
	if err := ValidateNewPassword(input.NewPassword); err != nil {
		return err
	}
	if s.codes == nil {
		return ErrAuthUnavailable
	}
	if err := s.codes.Verify(ctx, address, CodePurposeReset, input.Code); err != nil {
		return err
	}
	account, err := s.users.FindByEmail(ctx, address)
	if errors.Is(err, user.ErrNotFound) {
		// 验证码已通过说明邮箱确实收到过信；此处按无效请求处理，避免暴露内部状态。
		return ErrCodeInvalid
	}
	if err != nil {
		return wrapUnavailable(err)
	}
	passwordHash, err := s.hasher.Hash(input.NewPassword)
	if err != nil {
		return wrapUnavailable(err)
	}
	if err := s.users.SetPassword(ctx, account.ID, passwordHash); err != nil {
		return wrapUnavailable(err)
	}
	if err := s.RevokeUserSessions(ctx, account.ID, "password_reset"); err != nil {
		return err
	}
	return nil
}

func normalizeEmail(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

// validUsername 允许中英文、数字、下划线与连字符，长度 3 至 20 个字符。
// 不允许出现空白、@ 与控制字符，避免与邮箱登录混淆。
func validUsername(value string) bool {
	count := utf8.RuneCountInString(value)
	if count < 3 || count > 20 {
		return false
	}
	for _, char := range value {
		switch {
		case char >= 'a' && char <= 'z':
		case char >= 'A' && char <= 'Z':
		case char >= '0' && char <= '9':
		case char == '_' || char == '-':
		case unicode.Is(unicode.Han, char):
		default:
			return false
		}
	}
	return true
}
