package search

import (
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

func (h *Handler) Search(w http.ResponseWriter, r *http.Request) {
	principal, ok := auth.PrincipalFromContext(r.Context())
	if !ok {
		httpserver.Error(w, r, http.StatusUnauthorized, "authentication_required", "请先登录")
		return
	}
	query := r.URL.Query()
	page, pageSize := 1, defaultPageSize
	if raw := query.Get("page"); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 1 {
			httpserver.Error(w, r, http.StatusBadRequest, "invalid_search", "页码不正确")
			return
		}
		page = value
	}
	if raw := query.Get("page_size"); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 1 || value > maxPageSize {
			httpserver.Error(w, r, http.StatusBadRequest, "invalid_search", "每页数量不正确")
			return
		}
		pageSize = value
	}

	result, err := h.service.Search(r.Context(), principal.UserID, query.Get("q"), query.Get("type"), page, pageSize)
	if err != nil {
		var validation *ValidationError
		switch {
		case errors.As(err, &validation):
			httpserver.Error(w, r, http.StatusBadRequest, "invalid_search", validation.Message)
		case errors.Is(err, ErrRateLimited):
			httpserver.Error(w, r, http.StatusTooManyRequests, "search_rate_limited", "搜索过于频繁，请稍后再试")
		default:
			h.logger.Error("search failed", "request_id", httpserver.RequestIDFromContext(r.Context()), "error", err)
			httpserver.Error(w, r, http.StatusInternalServerError, "search_error", "搜索未完成")
		}
		return
	}
	httpserver.JSON(w, http.StatusOK, result)
}
