package content

import (
	"net/http"
	"strconv"

	"github.com/voidxo/voidxo/internal/modules/auth"
	"github.com/voidxo/voidxo/internal/platform/httpserver"
)

func (h *Handler) ListComments(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	kind := query.Get("type")
	targetID, err := strconv.ParseInt(query.Get("id"), 10, 64)
	if err != nil || targetID <= 0 {
		h.writeError(w, r, &ValidationError{Message: "评论目标不正确"})
		return
	}
	page, pageSize, err := pagination(r)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	result, err := h.service.ListComments(r.Context(), kind, targetID, page, pageSize)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	httpserver.JSON(w, http.StatusOK, result)
}

func (h *Handler) CreateComment(w http.ResponseWriter, r *http.Request) {
	principal, ok := auth.PrincipalFromContext(r.Context())
	if !ok {
		httpserver.Error(w, r, http.StatusUnauthorized, "authentication_required", "请先登录")
		return
	}
	var input struct {
		CommentableType string `json:"commentable_type"`
		CommentableID   int64  `json:"commentable_id"`
		Content         string `json:"content"`
		RootID          *int64 `json:"root_id"`
	}
	if err := decodeJSON(w, r, &input); err != nil {
		h.writeError(w, r, &ValidationError{Message: "请求格式不正确"})
		return
	}
	comment, err := h.service.CreateComment(r.Context(), principal.UserID, input.CommentableType, input.CommentableID, input.Content, input.RootID)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	httpserver.JSON(w, http.StatusCreated, map[string]any{"comment": comment})
}

func (h *Handler) UpdateComment(w http.ResponseWriter, r *http.Request) {
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
		Content string `json:"content"`
	}
	if err := decodeJSON(w, r, &input); err != nil {
		h.writeError(w, r, &ValidationError{Message: "请求格式不正确"})
		return
	}
	comment, err := h.service.UpdateComment(r.Context(), principal, id, input.Content)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	httpserver.JSON(w, http.StatusOK, map[string]any{"comment": comment})
}

func (h *Handler) DeleteComment(w http.ResponseWriter, r *http.Request) {
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
	if err := h.service.DeleteComment(r.Context(), principal, id); err != nil {
		h.writeError(w, r, err)
		return
	}
	httpserver.JSON(w, http.StatusOK, map[string]any{"deleted": true})
}
