package content

import (
	"net/http"
	"strconv"

	"github.com/voidxo/voidxo/internal/modules/auth"
	"github.com/voidxo/voidxo/internal/platform/httpserver"
)

func (h *Handler) ListArticles(w http.ResponseWriter, r *http.Request) {
	topicID, err := optionalTopicID(r)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	page, pageSize, err := pagination(r)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	result, err := h.service.ListArticles(r.Context(), topicID, r.URL.Query().Get("sort"), page, pageSize)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	httpserver.JSON(w, http.StatusOK, result)
}

func (h *Handler) GetArticle(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	article, err := h.service.GetArticle(r.Context(), id)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	httpserver.JSON(w, http.StatusOK, map[string]any{"article": article})
}

func (h *Handler) CreateArticle(w http.ResponseWriter, r *http.Request) {
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
	article, err := h.service.CreateArticle(r.Context(), principal.UserID, input.Title, input.ContentMarkdown, input.TopicIDs)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	httpserver.JSON(w, http.StatusCreated, map[string]any{"article": article})
}

func (h *Handler) UpdateArticle(w http.ResponseWriter, r *http.Request) {
	principal, ok := auth.PrincipalFromContext(r.Context())
	if !ok {
		httpserver.Error(w, r, http.StatusUnauthorized, "authentication_required", "请先登录")
		return
	}
	id, err := pathID(r)
	if err != nil {
		h.writeError(w, r, err)
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
	article, err := h.service.UpdateArticle(r.Context(), principal, id, input.Title, input.ContentMarkdown, input.TopicIDs)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	httpserver.JSON(w, http.StatusOK, map[string]any{"article": article})
}

func (h *Handler) DeleteArticle(w http.ResponseWriter, r *http.Request) {
	principal, ok := auth.PrincipalFromContext(r.Context())
	if !ok {
		httpserver.Error(w, r, http.StatusUnauthorized, "authentication_required", "请先登录")
		return
	}
	id, err := pathID(r)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	if err := h.service.DeleteArticle(r.Context(), principal, id); err != nil {
		h.writeError(w, r, err)
		return
	}
	httpserver.JSON(w, http.StatusOK, map[string]any{"deleted": true})
}

func (h *Handler) UpdateQuestion(w http.ResponseWriter, r *http.Request) {
	principal, ok := auth.PrincipalFromContext(r.Context())
	if !ok {
		httpserver.Error(w, r, http.StatusUnauthorized, "authentication_required", "请先登录")
		return
	}
	id, err := pathID(r)
	if err != nil {
		h.writeError(w, r, err)
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
	question, err := h.service.UpdateQuestion(r.Context(), principal, id, input.Title, input.ContentMarkdown, input.TopicIDs)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	httpserver.JSON(w, http.StatusOK, map[string]any{"question": question})
}

func (h *Handler) DeleteQuestion(w http.ResponseWriter, r *http.Request) {
	principal, ok := auth.PrincipalFromContext(r.Context())
	if !ok {
		httpserver.Error(w, r, http.StatusUnauthorized, "authentication_required", "请先登录")
		return
	}
	id, err := pathID(r)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	if err := h.service.DeleteQuestion(r.Context(), principal, id); err != nil {
		h.writeError(w, r, err)
		return
	}
	httpserver.JSON(w, http.StatusOK, map[string]any{"deleted": true})
}

func (h *Handler) UpdateAnswer(w http.ResponseWriter, r *http.Request) {
	principal, ok := auth.PrincipalFromContext(r.Context())
	if !ok {
		httpserver.Error(w, r, http.StatusUnauthorized, "authentication_required", "请先登录")
		return
	}
	id, err := answerPathID(r)
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
	answer, err := h.service.UpdateAnswer(r.Context(), principal, id, input.ContentMarkdown)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	httpserver.JSON(w, http.StatusOK, map[string]any{"answer": answer})
}

func (h *Handler) DeleteAnswer(w http.ResponseWriter, r *http.Request) {
	principal, ok := auth.PrincipalFromContext(r.Context())
	if !ok {
		httpserver.Error(w, r, http.StatusUnauthorized, "authentication_required", "请先登录")
		return
	}
	id, err := answerPathID(r)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	if err := h.service.DeleteAnswer(r.Context(), principal, id); err != nil {
		h.writeError(w, r, err)
		return
	}
	httpserver.JSON(w, http.StatusOK, map[string]any{"deleted": true})
}

func optionalTopicID(r *http.Request) (*int64, error) {
	raw := r.URL.Query().Get("topic_id")
	if raw == "" {
		return nil, nil
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || value <= 0 {
		return nil, &ValidationError{Message: "话题参数不正确"}
	}
	return &value, nil
}

func answerPathID(r *http.Request) (int64, error) {
	value, err := strconv.ParseInt(r.PathValue("answer_id"), 10, 64)
	if err != nil || value <= 0 {
		return 0, &ValidationError{Message: "回答编号不正确"}
	}
	return value, nil
}

func (h *Handler) writePermissionError(w http.ResponseWriter, r *http.Request, message string) {
	httpserver.Error(w, r, http.StatusForbidden, "permission_denied", message)
}
