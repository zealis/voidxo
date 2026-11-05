package email

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/voidxo/voidxo/internal/platform/httpserver"
)

type Handler struct {
	service *Service
	logger  *slog.Logger
}

func NewHandler(service *Service, logger *slog.Logger) *Handler {
	return &Handler{service: service, logger: logger}
}

// SendCode 是公开接口：注册与找回密码都需要它，靠验证码本身的限流防滥用。
func (h *Handler) SendCode(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Email   string `json:"email"`
		Purpose string `json:"purpose"`
	}
	if err := decodeJSON(w, r, &input); err != nil {
		httpserver.Error(w, r, http.StatusBadRequest, "invalid_request", "请求格式不正确")
		return
	}
	ttl, err := h.service.SendCode(r.Context(), input.Email, input.Purpose,
		httpserver.ClientIPFromContext(r.Context()))
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	// 统一返回 202，不透露邮箱是否已注册，避免账号枚举。
	httpserver.JSON(w, http.StatusAccepted, map[string]any{
		"expires_in": int(ttl.Seconds()),
		"message":    "如果该邮箱可用于当前操作，验证码会在几分钟内送达",
	})
}

func (h *Handler) writeError(w http.ResponseWriter, r *http.Request, err error) {
	var validation *ValidationError
	switch {
	case errors.As(err, &validation):
		httpserver.Error(w, r, http.StatusBadRequest, "invalid_email_request", validation.Message)
	case errors.Is(err, ErrRateLimited):
		httpserver.Error(w, r, http.StatusTooManyRequests, "email_code_rate_limited", "验证码发送过于频繁，请稍后再试")
	default:
		h.logger.Error("email code request failed", "error", err)
		httpserver.Error(w, r, http.StatusInternalServerError, "internal_error", "服务暂时不可用")
	}
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 8<<10)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(target)
}
