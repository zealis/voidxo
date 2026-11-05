package inbox

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/voidxo/voidxo/internal/modules/auth"
	"github.com/voidxo/voidxo/internal/platform/httpserver"
)

type Handler struct {
	service *Service
	logger  *slog.Logger
}

func NewHandler(service *Service, logger *slog.Logger) *Handler {
	return &Handler{service: service, logger: logger}
}

func (h *Handler) ListConversations(w http.ResponseWriter, r *http.Request) {
	principal, ok := auth.PrincipalFromContext(r.Context())
	if !ok {
		httpserver.Error(w, r, http.StatusUnauthorized, "authentication_required", "请先登录")
		return
	}
	page, pageSize, err := pagination(r)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	result, err := h.service.Conversations(r.Context(), principal.UserID, page, pageSize)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	httpserver.JSON(w, http.StatusOK, result)
}

func (h *Handler) UnreadCount(w http.ResponseWriter, r *http.Request) {
	principal, ok := auth.PrincipalFromContext(r.Context())
	if !ok {
		httpserver.Error(w, r, http.StatusUnauthorized, "authentication_required", "请先登录")
		return
	}
	count, err := h.service.UnreadCount(r.Context(), principal.UserID)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	httpserver.JSON(w, http.StatusOK, map[string]any{"unread": count})
}

func (h *Handler) Recipient(w http.ResponseWriter, r *http.Request) {
	recipient, err := h.service.RecipientByUsername(r.Context(), r.URL.Query().Get("username"))
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	httpserver.JSON(w, http.StatusOK, map[string]any{"recipient": recipient})
}

func (h *Handler) ListMessages(w http.ResponseWriter, r *http.Request) {
	principal, ok := auth.PrincipalFromContext(r.Context())
	if !ok {
		httpserver.Error(w, r, http.StatusUnauthorized, "authentication_required", "请先登录")
		return
	}
	counterpartID, err := pathUserID(r)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	page, pageSize, err := pagination(r)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	result, err := h.service.Messages(r.Context(), principal.UserID, counterpartID, page, pageSize)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	httpserver.JSON(w, http.StatusOK, result)
}

func (h *Handler) Send(w http.ResponseWriter, r *http.Request) {
	principal, ok := auth.PrincipalFromContext(r.Context())
	if !ok {
		httpserver.Error(w, r, http.StatusUnauthorized, "authentication_required", "请先登录")
		return
	}
	var input struct {
		ReceiverID int64  `json:"receiver_id"`
		Content    string `json:"content_markdown"`
	}
	if err := decodeJSON(w, r, &input); err != nil {
		h.writeError(w, r, &ValidationError{Message: "请求格式不正确"})
		return
	}
	message, err := h.service.Send(r.Context(), principal.UserID, input.ReceiverID, input.Content)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	httpserver.JSON(w, http.StatusCreated, map[string]any{"message": message})
}

func (h *Handler) MarkRead(w http.ResponseWriter, r *http.Request) {
	principal, ok := auth.PrincipalFromContext(r.Context())
	if !ok {
		httpserver.Error(w, r, http.StatusUnauthorized, "authentication_required", "请先登录")
		return
	}
	counterpartID, err := pathUserID(r)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	updated, unread, err := h.service.MarkRead(r.Context(), principal.UserID, counterpartID)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	httpserver.JSON(w, http.StatusOK, map[string]any{"updated": updated, "unread": unread})
}

func (h *Handler) writeError(w http.ResponseWriter, r *http.Request, err error) {
	var validation *ValidationError
	switch {
	case errors.As(err, &validation):
		httpserver.Error(w, r, http.StatusBadRequest, "invalid_inbox_input", validation.Message)
	case errors.Is(err, ErrRecipientNotFound):
		httpserver.Error(w, r, http.StatusNotFound, "recipient_not_found", "收件人不存在")
	case errors.Is(err, ErrRateLimited):
		httpserver.Error(w, r, http.StatusTooManyRequests, "inbox_rate_limited", "发送过于频繁，请稍后再试")
	default:
		h.logger.Error("inbox request failed", "error", err, "path", r.URL.Path)
		httpserver.Error(w, r, http.StatusInternalServerError, "internal_error", "服务暂时不可用")
	}
}

func pagination(r *http.Request) (int, int, error) {
	page := 1
	pageSize := defaultPageSize
	query := r.URL.Query()
	if raw := query.Get("page"); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 1 {
			return 0, 0, &ValidationError{Message: "页码不正确"}
		}
		page = value
	}
	if raw := query.Get("page_size"); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 1 || value > maxPageSize {
			return 0, 0, &ValidationError{Message: "每页数量不正确"}
		}
		pageSize = value
	}
	return page, pageSize, nil
}

func pathUserID(r *http.Request) (int64, error) {
	value, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || value <= 0 {
		return 0, &ValidationError{Message: "用户编号不正确"}
	}
	return value, nil
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 256<<10)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(target)
}
