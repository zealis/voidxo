package engagement

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

func (h *Handler) CreateVote(w http.ResponseWriter, r *http.Request) {
	principal, ok := auth.PrincipalFromContext(r.Context())
	if !ok {
		httpserver.Error(w, r, http.StatusUnauthorized, "authentication_required", "请先登录")
		return
	}
	var input struct {
		VotableType string `json:"votable_type"`
		VotableID   int64  `json:"votable_id"`
		Type        string `json:"type"`
	}
	if err := decodeJSON(w, r, &input); err != nil {
		h.writeError(w, r, &ValidationError{Message: "请求格式不正确"})
		return
	}
	summary, err := h.service.Vote(r.Context(), principal.UserID, input.VotableType, input.VotableID, input.Type)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	httpserver.JSON(w, http.StatusOK, map[string]any{"vote": summary})
}

func (h *Handler) GetVote(w http.ResponseWriter, r *http.Request) {
	principal, ok := auth.PrincipalFromContext(r.Context())
	if !ok {
		httpserver.Error(w, r, http.StatusUnauthorized, "authentication_required", "请先登录")
		return
	}
	kind := r.URL.Query().Get("type")
	targetID, err := strconv.ParseInt(r.URL.Query().Get("id"), 10, 64)
	if err != nil || targetID <= 0 {
		h.writeError(w, r, &ValidationError{Message: "投票目标不正确"})
		return
	}
	summary, err := h.service.VoteState(r.Context(), principal.UserID, kind, targetID)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	httpserver.JSON(w, http.StatusOK, map[string]any{"vote": summary})
}

func (h *Handler) DeleteVote(w http.ResponseWriter, r *http.Request) {
	principal, ok := auth.PrincipalFromContext(r.Context())
	if !ok {
		httpserver.Error(w, r, http.StatusUnauthorized, "authentication_required", "请先登录")
		return
	}
	kind := r.URL.Query().Get("type")
	targetID, err := strconv.ParseInt(r.URL.Query().Get("id"), 10, 64)
	if err != nil || targetID <= 0 {
		h.writeError(w, r, &ValidationError{Message: "投票目标不正确"})
		return
	}
	summary, err := h.service.RemoveVote(r.Context(), principal.UserID, kind, targetID)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	httpserver.JSON(w, http.StatusOK, map[string]any{"vote": summary})
}

func (h *Handler) Follow(w http.ResponseWriter, r *http.Request) {
	principal, ok := auth.PrincipalFromContext(r.Context())
	if !ok {
		httpserver.Error(w, r, http.StatusUnauthorized, "authentication_required", "请先登录")
		return
	}
	var input struct {
		FollowableType string `json:"followable_type"`
		FollowableID   int64  `json:"followable_id"`
	}
	if err := decodeJSON(w, r, &input); err != nil {
		h.writeError(w, r, &ValidationError{Message: "请求格式不正确"})
		return
	}
	summary, err := h.service.Follow(r.Context(), principal.UserID, input.FollowableType, input.FollowableID)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	httpserver.JSON(w, http.StatusOK, map[string]any{"follow": summary})
}

func (h *Handler) GetFollow(w http.ResponseWriter, r *http.Request) {
	principal, ok := auth.PrincipalFromContext(r.Context())
	if !ok {
		httpserver.Error(w, r, http.StatusUnauthorized, "authentication_required", "请先登录")
		return
	}
	kind := r.URL.Query().Get("type")
	// ids 形式用于列表批量查询，避免每张卡片发一次请求。
	if raw := r.URL.Query().Get("ids"); raw != "" {
		parts := strings.Split(raw, ",")
		ids := make([]int64, 0, len(parts))
		for _, part := range parts {
			value, err := strconv.ParseInt(strings.TrimSpace(part), 10, 64)
			if err != nil || value <= 0 {
				h.writeError(w, r, &ValidationError{Message: "关注目标不正确"})
				return
			}
			ids = append(ids, value)
		}
		states, err := h.service.FollowStates(r.Context(), principal.UserID, kind, ids)
		if err != nil {
			h.writeError(w, r, err)
			return
		}
		result := make(map[string]bool, len(states))
		for id, following := range states {
			result[strconv.FormatInt(id, 10)] = following
		}
		httpserver.JSON(w, http.StatusOK, map[string]any{"states": result})
		return
	}
	targetID, err := strconv.ParseInt(r.URL.Query().Get("id"), 10, 64)
	if err != nil || targetID <= 0 {
		h.writeError(w, r, &ValidationError{Message: "关注目标不正确"})
		return
	}
	summary, err := h.service.FollowState(r.Context(), principal.UserID, kind, targetID)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	httpserver.JSON(w, http.StatusOK, map[string]any{"follow": summary})
}

func (h *Handler) Unfollow(w http.ResponseWriter, r *http.Request) {
	principal, ok := auth.PrincipalFromContext(r.Context())
	if !ok {
		httpserver.Error(w, r, http.StatusUnauthorized, "authentication_required", "请先登录")
		return
	}
	kind := r.URL.Query().Get("type")
	targetID, err := strconv.ParseInt(r.URL.Query().Get("id"), 10, 64)
	if err != nil || targetID <= 0 {
		h.writeError(w, r, &ValidationError{Message: "关注目标不正确"})
		return
	}
	summary, err := h.service.Unfollow(r.Context(), principal.UserID, kind, targetID)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	httpserver.JSON(w, http.StatusOK, map[string]any{"follow": summary})
}

func (h *Handler) CreateReport(w http.ResponseWriter, r *http.Request) {
	principal, ok := auth.PrincipalFromContext(r.Context())
	if !ok {
		httpserver.Error(w, r, http.StatusUnauthorized, "authentication_required", "请先登录")
		return
	}
	var input struct {
		ReportableType string `json:"reportable_type"`
		ReportableID   int64  `json:"reportable_id"`
		Reason         string `json:"reason"`
	}
	if err := decodeJSON(w, r, &input); err != nil {
		h.writeError(w, r, &ValidationError{Message: "请求格式不正确"})
		return
	}
	report, err := h.service.CreateReport(r.Context(), principal.UserID, input.ReportableType, input.ReportableID, input.Reason)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	httpserver.JSON(w, http.StatusCreated, map[string]any{"report": report})
}

func (h *Handler) ListReports(w http.ResponseWriter, r *http.Request) {
	page, pageSize, err := pagination(r)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	result, err := h.service.ListReports(r.Context(), r.URL.Query().Get("status"), page, pageSize)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	httpserver.JSON(w, http.StatusOK, result)
}

func (h *Handler) ResolveReport(w http.ResponseWriter, r *http.Request) {
	principal, ok := auth.PrincipalFromContext(r.Context())
	if !ok {
		httpserver.Error(w, r, http.StatusUnauthorized, "authentication_required", "请先登录")
		return
	}
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		h.writeError(w, r, &ValidationError{Message: "举报编号不正确"})
		return
	}
	var input struct {
		Status string `json:"status"`
	}
	if err := decodeJSON(w, r, &input); err != nil {
		h.writeError(w, r, &ValidationError{Message: "请求格式不正确"})
		return
	}
	report, err := h.service.ResolveReport(r.Context(), id, principal.UserID, input.Status)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	httpserver.JSON(w, http.StatusOK, map[string]any{"report": report})
}

func (h *Handler) ListNotifications(w http.ResponseWriter, r *http.Request) {
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
	result, err := h.service.ListNotifications(r.Context(), principal.UserID, page, pageSize)
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

func (h *Handler) MarkNotificationsRead(w http.ResponseWriter, r *http.Request) {
	principal, ok := auth.PrincipalFromContext(r.Context())
	if !ok {
		httpserver.Error(w, r, http.StatusUnauthorized, "authentication_required", "请先登录")
		return
	}
	var input struct {
		IDs []int64 `json:"ids"`
	}
	if err := decodeJSON(w, r, &input); err != nil {
		h.writeError(w, r, &ValidationError{Message: "请求格式不正确"})
		return
	}
	affected, err := h.service.MarkRead(r.Context(), principal.UserID, input.IDs)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	httpserver.JSON(w, http.StatusOK, map[string]any{"updated": affected})
}

func (h *Handler) DeleteNotifications(w http.ResponseWriter, r *http.Request) {
	principal, ok := auth.PrincipalFromContext(r.Context())
	if !ok {
		httpserver.Error(w, r, http.StatusUnauthorized, "authentication_required", "请先登录")
		return
	}
	var input struct {
		IDs []int64 `json:"ids"`
	}
	if err := decodeJSON(w, r, &input); err != nil {
		h.writeError(w, r, &ValidationError{Message: "请求格式不正确"})
		return
	}
	affected, err := h.service.DeleteNotifications(r.Context(), principal.UserID, input.IDs)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	httpserver.JSON(w, http.StatusOK, map[string]any{"deleted": affected})
}

func (h *Handler) writeError(w http.ResponseWriter, r *http.Request, err error) {
	var validation *ValidationError
	var permission *PermissionError
	switch {
	case errors.As(err, &validation):
		httpserver.Error(w, r, http.StatusBadRequest, "invalid_engagement", validation.Message)
	case errors.As(err, &permission):
		httpserver.Error(w, r, http.StatusForbidden, "permission_denied", permission.Message)
	case errors.Is(err, ErrNotFound):
		httpserver.Error(w, r, http.StatusNotFound, "target_not_found", "操作对象不存在")
	case errors.Is(err, ErrDuplicateReport):
		httpserver.Error(w, r, http.StatusConflict, "report_exists", "你已举报过该内容")
	case errors.Is(err, ErrSelfFollow):
		httpserver.Error(w, r, http.StatusBadRequest, "self_follow", "不能关注自己")
	default:
		h.logger.Error("engagement operation failed", "request_id", httpserver.RequestIDFromContext(r.Context()), "error", err)
		httpserver.Error(w, r, http.StatusInternalServerError, "engagement_error", "操作未完成")
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

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(target)
}
