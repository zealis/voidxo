package auth

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"
	"unicode/utf8"

	"github.com/voidxo/voidxo/internal/platform/httpserver"
)

type Handler struct {
	service       *Service
	secureCookies bool
	logger        *slog.Logger
}

func NewHandler(service *Service, secureCookies bool, logger *slog.Logger) *Handler {
	return &Handler{service: service, secureCookies: secureCookies, logger: logger}
}

func (h *Handler) GetCSRF(w http.ResponseWriter, r *http.Request) {
	token, err := NewCSRFToken()
	if err != nil {
		h.logger.Error("create csrf token", "error", err)
		httpserver.Error(w, r, http.StatusInternalServerError, "csrf_unavailable", "暂时无法创建安全令牌")
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     CSRFCookieName(h.secureCookies),
		Value:    token,
		Path:     "/",
		MaxAge:   12 * 60 * 60,
		Secure:   h.secureCookies,
		HttpOnly: false,
		SameSite: http.SameSiteLaxMode,
	})
	httpserver.JSON(w, http.StatusOK, map[string]string{"csrf_token": token})
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Login    string `json:"login"`
		Password string `json:"password"`
		DeviceID string `json:"device_id"`
	}
	if err := decodeJSON(w, r, &input); err != nil {
		httpserver.Error(w, r, http.StatusBadRequest, "invalid_request", "请求格式不正确")
		return
	}
	if len([]rune(input.DeviceID)) > 128 {
		httpserver.Error(w, r, http.StatusBadRequest, "invalid_device", "设备标识过长")
		return
	}
	result, err := h.service.Login(r.Context(), LoginInput{
		Login: input.Login, Password: input.Password, DeviceID: input.DeviceID,
		IPAddress: httpserver.ClientIPFromContext(r.Context()), UserAgent: truncate(r.UserAgent(), 600),
	})
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	h.setSessionCookie(w, result)
	httpserver.JSON(w, http.StatusOK, map[string]any{"user": userView(result.Principal, result.AvatarKey)})
}

// Register 创建普通账号并直接登录，验证码由邮箱模块校验。
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Username string `json:"username"`
		Email    string `json:"email"`
		Password string `json:"password"`
		Code     string `json:"code"`
		DeviceID string `json:"device_id"`
	}
	if err := decodeJSON(w, r, &input); err != nil {
		httpserver.Error(w, r, http.StatusBadRequest, "invalid_request", "请求格式不正确")
		return
	}
	if len([]rune(input.DeviceID)) > 128 {
		httpserver.Error(w, r, http.StatusBadRequest, "invalid_device", "设备标识过长")
		return
	}
	result, err := h.service.Register(r.Context(), RegisterInput{
		Username: input.Username, Email: input.Email, Password: input.Password, Code: input.Code,
		DeviceID: input.DeviceID, IPAddress: httpserver.ClientIPFromContext(r.Context()),
		UserAgent: truncate(r.UserAgent(), 600),
	})
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	h.setSessionCookie(w, result)
	httpserver.JSON(w, http.StatusCreated, map[string]any{"user": userView(result.Principal, result.AvatarKey)})
}

// ResetPassword 通过邮箱验证码重设密码，成功后需用新密码登录。
func (h *Handler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Email       string `json:"email"`
		Code        string `json:"code"`
		NewPassword string `json:"new_password"`
	}
	if err := decodeJSON(w, r, &input); err != nil {
		httpserver.Error(w, r, http.StatusBadRequest, "invalid_request", "请求格式不正确")
		return
	}
	if err := h.service.ResetPassword(r.Context(), ResetPasswordInput{
		Email: input.Email, Code: input.Code, NewPassword: input.NewPassword,
	}); err != nil {
		h.writeError(w, r, err)
		return
	}
	httpserver.JSON(w, http.StatusOK, map[string]any{"reset": true})
}

func (h *Handler) setSessionCookie(w http.ResponseWriter, result LoginResult) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName(h.secureCookies),
		Value:    result.Token,
		Path:     "/",
		Expires:  result.ExpiresAt,
		MaxAge:   int(time.Until(result.ExpiresAt).Seconds()),
		Secure:   h.secureCookies,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	principal, ok := PrincipalFromContext(r.Context())
	if !ok {
		httpserver.Error(w, r, http.StatusUnauthorized, "authentication_required", "请先登录")
		return
	}
	avatarKey, err := h.service.AvatarKey(r.Context(), principal.UserID)
	if err != nil {
		// 头像只是展示字段，查询失败不应让已登录用户被判定为未登录。
		h.logger.Warn("load avatar key", "request_id", httpserver.RequestIDFromContext(r.Context()), "error", err)
		avatarKey = ""
	}
	httpserver.JSON(w, http.StatusOK, map[string]any{"user": userView(principal, avatarKey)})
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	token, _ := r.Cookie(SessionCookieName(h.secureCookies))
	if token != nil {
		if err := h.service.Logout(r.Context(), token.Value); err != nil {
			h.logger.Error("logout failed", "request_id", httpserver.RequestIDFromContext(r.Context()), "error", err)
			h.clearSessionCookie(w)
			httpserver.Error(w, r, http.StatusServiceUnavailable, "logout_incomplete", "会话暂时无法撤销，请稍后重试")
			return
		}
	}
	h.clearSessionCookie(w)
	httpserver.JSON(w, http.StatusOK, map[string]bool{"logged_out": true})
}

func (h *Handler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	principal, ok := PrincipalFromContext(r.Context())
	if !ok {
		httpserver.Error(w, r, http.StatusUnauthorized, "authentication_required", "请先登录")
		return
	}
	var input struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}
	if err := decodeJSON(w, r, &input); err != nil {
		httpserver.Error(w, r, http.StatusBadRequest, "invalid_request", "请求格式不正确")
		return
	}
	if err := h.service.ChangePassword(r.Context(), principal.UserID, input.CurrentPassword, input.NewPassword); err != nil {
		h.writeError(w, r, err)
		return
	}
	h.clearSessionCookie(w)
	httpserver.JSON(w, http.StatusOK, map[string]bool{"password_changed": true})
}

func (h *Handler) RequireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(SessionCookieName(h.secureCookies))
		if err != nil || cookie.Value == "" {
			httpserver.Error(w, r, http.StatusUnauthorized, "authentication_required", "请先登录")
			return
		}
		principal, err := h.service.Authenticate(r.Context(), cookie.Value)
		if errors.Is(err, ErrAuthenticationNeeded) {
			h.clearSessionCookie(w)
			httpserver.Error(w, r, http.StatusUnauthorized, "authentication_required", "请先登录")
			return
		}
		if err != nil {
			h.logger.Error("session validation failed", "request_id", httpserver.RequestIDFromContext(r.Context()), "error", err)
			httpserver.Error(w, r, http.StatusServiceUnavailable, "auth_unavailable", "登录状态暂时无法验证")
			return
		}
		next(w, r.WithContext(WithPrincipal(r.Context(), principal)))
	}
}

func (h *Handler) RequireCSRF(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(CSRFCookieName(h.secureCookies))
		if err != nil || !ValidCSRF(cookie.Value, r.Header.Get(CSRFHeader)) {
			httpserver.Error(w, r, http.StatusForbidden, "csrf_invalid", "安全令牌无效，请刷新页面后重试")
			return
		}
		next(w, r)
	}
}

func (h *Handler) RequirePermission(code string, next http.HandlerFunc) http.HandlerFunc {
	return h.RequireAuth(func(w http.ResponseWriter, r *http.Request) {
		principal, _ := PrincipalFromContext(r.Context())
		if !principal.HasPermission(code) {
			httpserver.Error(w, r, http.StatusForbidden, "permission_denied", "没有执行该操作的权限")
			return
		}
		next(w, r)
	})
}

func (h *Handler) writeError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, ErrInvalidCredentials):
		httpserver.Error(w, r, http.StatusUnauthorized, "invalid_credentials", "账号或密码错误")
	case errors.Is(err, ErrCurrentPassword):
		httpserver.Error(w, r, http.StatusBadRequest, "current_password_invalid", "当前密码不正确")
	case errors.Is(err, ErrTooManyAttempts):
		w.Header().Set("Retry-After", "60")
		httpserver.Error(w, r, http.StatusTooManyRequests, "too_many_attempts", "尝试次数过多，请稍后重试")
	case errors.Is(err, ErrPasswordInvalid):
		httpserver.Error(w, r, http.StatusBadRequest, "password_invalid", "密码长度需为 8 至 128 个字符")
	case errors.Is(err, ErrUsernameInvalid):
		httpserver.Error(w, r, http.StatusBadRequest, "username_invalid", "用户名需为 3 至 20 个字符，可用中英文、数字、下划线与连字符")
	case errors.Is(err, ErrAccountExists):
		httpserver.Error(w, r, http.StatusConflict, "account_exists", "该用户名或邮箱已被使用")
	case errors.Is(err, ErrCodeInvalid):
		httpserver.Error(w, r, http.StatusBadRequest, "code_invalid", "验证码不正确")
	case errors.Is(err, ErrCodeExpired):
		httpserver.Error(w, r, http.StatusBadRequest, "code_expired", "验证码已过期，请重新获取")
	case errors.Is(err, ErrCodeAttempts):
		w.Header().Set("Retry-After", "60")
		httpserver.Error(w, r, http.StatusTooManyRequests, "code_attempts", "验证码尝试次数过多，请重新获取")
	case errors.Is(err, ErrAuthUnavailable):
		h.logger.Error("auth dependency unavailable", "request_id", httpserver.RequestIDFromContext(r.Context()), "error", err)
		httpserver.Error(w, r, http.StatusServiceUnavailable, "auth_unavailable", "登录服务暂时不可用")
	case errors.Is(err, ErrAuthenticationNeeded):
		httpserver.Error(w, r, http.StatusUnauthorized, "authentication_required", "请先登录")
	default:
		h.logger.Error("auth operation failed", "request_id", httpserver.RequestIDFromContext(r.Context()), "error", err)
		httpserver.Error(w, r, http.StatusInternalServerError, "auth_error", "操作未完成")
	}
}

func (h *Handler) clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name: SessionCookieName(h.secureCookies), Value: "", Path: "/", MaxAge: -1,
		Secure: h.secureCookies, HttpOnly: true, SameSite: http.SameSiteLaxMode,
	})
}

func userView(principal Principal, avatarKey string) map[string]any {
	return map[string]any{
		"id": principal.UserID, "username": principal.Username, "email": principal.Email,
		"avatar_key": avatarKey,
		"roles":      principal.Roles, "permissions": principal.Permissions,
	}
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(target)
}

func truncate(value string, max int) string {
	if utf8.RuneCountInString(value) <= max {
		return value
	}
	runes := []rune(value)
	return string(runes[:max])
}
