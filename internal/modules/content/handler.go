package content

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

// GetSitemap 是公开接口，供站点地图聚合公开内容；只暴露编号与更新时间。
func (h *Handler) GetSitemap(w http.ResponseWriter, r *http.Request) {
	limit := maxSitemapEntries
	if raw := r.URL.Query().Get("limit"); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 1 || value > maxSitemapEntries {
			h.writeError(w, r, &ValidationError{Message: "条目数量不正确"})
			return
		}
		limit = value
	}
	items, err := h.service.Sitemap(r.Context(), limit)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	// 公开只读数据，允许 CDN 缓存 5 分钟；内容变更由 Worker 递增版本使其失效。
	w.Header().Set("Cache-Control", "public, max-age=300")
	httpserver.JSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *Handler) ListTopics(w http.ResponseWriter, r *http.Request) {
	page, pageSize, err := pagination(r)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	result, err := h.service.ListTopics(r.Context(), r.URL.Query().Get("sort"), page, pageSize)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	httpserver.JSON(w, http.StatusOK, result)
}

func (h *Handler) GetTopic(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	topic, err := h.service.GetTopic(r.Context(), id)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	httpserver.JSON(w, http.StatusOK, map[string]any{"topic": topic})
}

func (h *Handler) CreateTopic(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		CoverKey    string `json:"cover_key"`
	}
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

func (h *Handler) ListQuestions(w http.ResponseWriter, r *http.Request) {
	var topicID *int64
	if raw := r.URL.Query().Get("topic_id"); raw != "" {
		value, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || value <= 0 {
			h.writeError(w, r, &ValidationError{Message: "话题参数不正确"})
			return
		}
		topicID = &value
	}
	page, pageSize, err := pagination(r)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	result, err := h.service.ListQuestions(r.Context(), topicID, r.URL.Query().Get("sort"), page, pageSize)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	httpserver.JSON(w, http.StatusOK, result)
}

func (h *Handler) GetQuestion(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	question, err := h.service.GetQuestion(r.Context(), id)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	httpserver.JSON(w, http.StatusOK, map[string]any{"question": question})
}

func (h *Handler) CreateQuestion(w http.ResponseWriter, r *http.Request) {
	principal, ok := auth.PrincipalFromContext(r.Context())
	if !ok {
		httpserver.Error(w, r, http.StatusUnauthorized, "authentication_required", "请先登录")
		return
	}
	var input struct {
		Title           string  `json:"title"`
		ContentMarkdown string  `json:"content_markdown"`
		TopicIDs        []int64 `json:"topic_ids"`
	}
	if err := decodeJSON(w, r, &input); err != nil {
		h.writeError(w, r, &ValidationError{Message: "请求格式不正确"})
		return
	}
	question, err := h.service.CreateQuestion(r.Context(), principal.UserID, input.Title, input.ContentMarkdown, input.TopicIDs)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	httpserver.JSON(w, http.StatusCreated, map[string]any{"question": question})
}

func (h *Handler) ListAnswers(w http.ResponseWriter, r *http.Request) {
	questionID, err := pathID(r)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	page, pageSize, err := pagination(r)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	result, err := h.service.ListAnswers(r.Context(), questionID, page, pageSize)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	httpserver.JSON(w, http.StatusOK, result)
}

func (h *Handler) CreateAnswer(w http.ResponseWriter, r *http.Request) {
	principal, ok := auth.PrincipalFromContext(r.Context())
	if !ok {
		httpserver.Error(w, r, http.StatusUnauthorized, "authentication_required", "请先登录")
		return
	}
	questionID, err := pathID(r)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	var input struct {
		ContentMarkdown string `json:"content_markdown"`
	}
	if err := decodeJSON(w, r, &input); err != nil {
		h.writeError(w, r, &ValidationError{Message: "请求格式不正确"})
		return
	}
	answer, err := h.service.CreateAnswer(r.Context(), principal.UserID, questionID, input.ContentMarkdown)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	httpserver.JSON(w, http.StatusCreated, map[string]any{"answer": answer})
}

func (h *Handler) writeError(w http.ResponseWriter, r *http.Request, err error) {
	var validation *ValidationError
	var permission *PermissionError
	switch {
	case errors.As(err, &validation):
		httpserver.Error(w, r, http.StatusBadRequest, "invalid_content", validation.Message)
	case errors.As(err, &permission):
		httpserver.Error(w, r, http.StatusForbidden, "permission_denied", permission.Message)
	case errors.Is(err, ErrTopicExists):
		httpserver.Error(w, r, http.StatusConflict, "topic_exists", "话题名称已存在")
	case errors.Is(err, ErrTopicNotFound):
		httpserver.Error(w, r, http.StatusNotFound, "topic_not_found", "话题不存在")
	case errors.Is(err, ErrNotFound):
		httpserver.Error(w, r, http.StatusNotFound, "content_not_found", "内容不存在")
	default:
		h.logger.Error("content operation failed", "request_id", httpserver.RequestIDFromContext(r.Context()), "error", err)
		httpserver.Error(w, r, http.StatusInternalServerError, "content_error", "内容操作未完成")
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

func pathID(r *http.Request) (int64, error) {
	value, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || value <= 0 {
		return 0, &ValidationError{Message: "内容编号不正确"}
	}
	return value, nil
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 2<<20)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(target)
}
