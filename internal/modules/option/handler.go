package option

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

func (h *Handler) GetPublic(w http.ResponseWriter, r *http.Request) {
	options, err := h.service.Public(r.Context())
	if err != nil {
		h.logger.Error("load public options", "request_id", httpserver.RequestIDFromContext(r.Context()), "error", err)
		httpserver.Error(w, r, http.StatusInternalServerError, "options_unavailable", "站点配置暂时不可用")
		return
	}
	httpserver.JSON(w, http.StatusOK, options)
}
