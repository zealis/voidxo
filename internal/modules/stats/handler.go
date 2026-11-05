package stats

import (
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

// Public 是公开只读接口，允许 CDN 缓存 60 秒。
func (h *Handler) Public(w http.ResponseWriter, r *http.Request) {
	counts, err := h.service.Public(r.Context())
	if err != nil {
		h.logger.Error("load site stats failed", "error", err)
		httpserver.Error(w, r, http.StatusInternalServerError, "internal_error", "服务暂时不可用")
		return
	}
	w.Header().Set("Cache-Control", "public, max-age=60")
	httpserver.JSON(w, http.StatusOK, map[string]any{"stats": counts})
}
