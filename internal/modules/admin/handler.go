package admin

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

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

func (h *Handler) Stats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.service.Stats(r.Context())
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	httpserver.JSON(w, http.StatusOK, map[string]any{"stats": stats})
}

func (h *Handler) ListUsers(w http.ResponseWriter, r *http.Request) {
	page, pageSize, err := pagination(r)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	query := r.URL.Query()
	result, err := h.service.ListUsers(r.Context(), UserFilter{
		Query:    query.Get("q"),
		Role:     strings.TrimSpace(query.Get("role")),
		Status:   strings.TrimSpace(query.Get("status")),
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	httpserver.JSON(w, http.StatusOK, result)
}

func (h *Handler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	principal, ok := auth.PrincipalFromContext(r.Context())
	if !ok {
		httpserver.Error(w, r, http.StatusUnauthorized, "authentication_required", "请先登录")
		return
	}
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		h.writeError(w, r, &ValidationError{Message: "用户编号不正确"})
		return
	}
	var input struct {
		Status *uint8  `json:"status"`
		Role   *string `json:"role"`
	}
	if err := decodeJSON(w, r, &input); err != nil {
		h.writeError(w, r, &ValidationError{Message: "请求格式不正确"})
		return
	}
	if input.Status == nil && input.Role == nil {
		h.writeError(w, r, &ValidationError{Message: "没有需要修改的字段"})
		return
	}
	user, err := h.service.UpdateUser(r.Context(), principal, id, UserUpdate{Status: input.Status, Role: input.Role})
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	httpserver.JSON(w, http.StatusOK, map[string]any{"user": user})
}

func (h *Handler) ListContent(w http.ResponseWriter, r *http.Request) {
	page, pageSize, err := pagination(r)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	kind := r.PathValue("kind")
	query := r.URL.Query()
	result, err := h.service.ListContent(r.Context(), kind, ContentFilter{
		Query:    query.Get("q"),
		Deleted:  strings.TrimSpace(query.Get("deleted")),
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	httpserver.JSON(w, http.StatusOK, result)
}

func (h *Handler) DeleteContent(w http.ResponseWriter, r *http.Request) {
	principal, ok := auth.PrincipalFromContext(r.Context())
	if !ok {
		httpserver.Error(w, r, http.StatusUnauthorized, "authentication_required", "请先登录")
		return
	}
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		h.writeError(w, r, &ValidationError{Message: "内容编号不正确"})
		return
	}
	if err := h.service.DeleteContent(r.Context(), principal, r.PathValue("kind"), id); err != nil {
		h.writeError(w, r, err)
		return
	}
	httpserver.JSON(w, http.StatusOK, map[string]any{"deleted": true})
}

func (h *Handler) ListTopics(w http.ResponseWriter, r *http.Request) {
	page, pageSize, err := pagination(r)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	query := r.URL.Query()
	result, err := h.service.ListTopics(r.Context(), query.Get("q"), query.Get("include_deleted") == "1", page, pageSize)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	httpserver.JSON(w, http.StatusOK, result)
}

func (h *Handler) CreateTopic(w http.ResponseWriter, r *http.Request) {
	var input topicInput
	if err := decodeJSON(w, r, &input); err != nil {
		h.writeError(w, r, &ValidationError{Message: "请求格式不正确"})
		return
	}
	topic, err := h.service.CreateTopic(r.Context(), input.Name, input.Description, input.CoverKey)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	httpserver.JSON(w, http.StatusCreated, map[string]any{"topic": topic})
}

func (h *Handler) UpdateTopic(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		h.writeError(w, r, &ValidationError{Message: "话题编号不正确"})
		return
	}
	var input topicInput
	if err := decodeJSON(w, r, &input); err != nil {
		h.writeError(w, r, &ValidationError{Message: "请求格式不正确"})
		return
	}
	topic, err := h.service.UpdateTopic(r.Context(), id, input.Name, input.Description, input.CoverKey)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	httpserver.JSON(w, http.StatusOK, map[string]any{"topic": topic})
}

func (h *Handler) DeleteTopic(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		h.writeError(w, r, &ValidationError{Message: "话题编号不正确"})
		return
	}
	if err := h.service.DeleteTopic(r.Context(), id); err != nil {
		h.writeError(w, r, err)
		return
	}
	httpserver.JSON(w, http.StatusOK, map[string]any{"deleted": true})
}

func (h *Handler) ListOptions(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.ListOptions(r.Context())
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	httpserver.JSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *Handler) UpdateOptions(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Values map[string]string `json:"values"`
	}
	if err := decodeJSON(w, r, &input); err != nil {
		h.writeError(w, r, &ValidationError{Message: "请求格式不正确"})
		return
	}
	if err := h.service.UpdateOptions(r.Context(), input.Values); err != nil {
		h.writeError(w, r, err)
		return
	}
	items, err := h.service.ListOptions(r.Context())
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	httpserver.JSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *Handler) ListImages(w http.ResponseWriter, r *http.Request) {
	page, pageSize, err := pagination(r)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	query := r.URL.Query()
	result, err := h.service.ListImages(r.Context(), ContentFilter{
		Query:    query.Get("q"),
		Deleted:  strings.TrimSpace(query.Get("deleted")),
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	httpserver.JSON(w, http.StatusOK, result)
}

func (h *Handler) DeleteImage(w http.ResponseWriter, r *http.Request) {
	principal, ok := auth.PrincipalFromContext(r.Context())
	if !ok {
		httpserver.Error(w, r, http.StatusUnauthorized, "authentication_required", "请先登录")
		return
	}
	key := strings.TrimSpace(r.PathValue("key"))
	if key == "" {
		h.writeError(w, r, &ValidationError{Message: "图片编号不正确"})
		return
	}
	if err := h.service.DeleteImage(r.Context(), principal, key); err != nil {
		h.writeError(w, r, err)
		return
	}
	httpserver.JSON(w, http.StatusOK, map[string]any{"deleted": true})
}

type topicInput struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	CoverKey    string `json:"cover_key"`
}

// ValidationError 是后台入参错误，与其他模块的错误结构保持一致。
type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string { return e.Message }

func (h *Handler) writeError(w http.ResponseWriter, r *http.Request, err error) {
	var validation *ValidationError
	switch {
	case errors.As(err, &validation):
		httpserver.Error(w, r, http.StatusBadRequest, "invalid_admin_input", validation.Message)
	case errors.Is(err, ErrValidation):
		httpserver.Error(w, r, http.StatusBadRequest, "invalid_admin_input", trimSentinel(err))
	case errors.Is(err, ErrForbidden):
		httpserver.Error(w, r, http.StatusForbidden, "permission_denied", trimSentinel(err))
	case errors.Is(err, ErrNotFound):
		httpserver.Error(w, r, http.StatusNotFound, "admin_resource_not_found", "目标不存在")
	default:
		// 认证与内容模块的错误在装配层已转换成后台错误，其余一律按服务端错误记录。
		h.logger.Error("admin request failed", "error", err, "path", r.URL.Path)
		httpserver.Error(w, r, http.StatusInternalServerError, "internal_error", "服务暂时不可用")
	}
}

// trimSentinel 去掉错误包装前缀，只把可读文案返回给前端。
func trimSentinel(err error) string {
	message := err.Error()
	if index := strings.Index(message, ": "); index >= 0 {
		return message[index+2:]
	}
	return message
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

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(target)
}
