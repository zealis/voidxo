package media

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/voidxo/voidxo/internal/modules/auth"
	"github.com/voidxo/voidxo/internal/platform/httpserver"
	"github.com/voidxo/voidxo/internal/platform/imageproc"
)

const multipartMemory = 8 << 20

// ProfileUpdater 让头像/封面在上传成功后同步到用户资料。
type ProfileUpdater interface {
	SetAvatar(ctx context.Context, userID int64, key string) error
	SetCover(ctx context.Context, userID int64, key string) error
}

type Handler struct {
	service  *Service
	profiles ProfileUpdater
	logger   *slog.Logger
}

func NewHandler(service *Service, profiles ProfileUpdater, logger *slog.Logger) *Handler {
	return &Handler{service: service, profiles: profiles, logger: logger}
}

func (h *Handler) UploadContentImage(w http.ResponseWriter, r *http.Request) {
	h.upload(w, r, "image", KindContent, nil)
}

func (h *Handler) UploadAvatar(w http.ResponseWriter, r *http.Request) {
	h.upload(w, r, "avatar", KindAvatar, func(ctx context.Context, userID int64, key string) error {
		if h.profiles == nil {
			return errors.New("profile updater not configured")
		}
		return h.profiles.SetAvatar(ctx, userID, key)
	})
}

func (h *Handler) UploadCover(w http.ResponseWriter, r *http.Request) {
	h.upload(w, r, "cover", KindCover, func(ctx context.Context, userID int64, key string) error {
		if h.profiles == nil {
			return errors.New("profile updater not configured")
		}
		return h.profiles.SetCover(ctx, userID, key)
	})
}

func (h *Handler) upload(w http.ResponseWriter, r *http.Request, field, kind string, after func(context.Context, int64, string) error) {
	principal, ok := auth.PrincipalFromContext(r.Context())
	if !ok {
		httpserver.Error(w, r, http.StatusUnauthorized, "authentication_required", "请先登录")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, multipartMemory)
	if err := r.ParseMultipartForm(multipartMemory); err != nil {
		httpserver.Error(w, r, http.StatusBadRequest, "invalid_upload", "上传内容不正确")
		return
	}
	file, header, err := r.FormFile(field)
	if err != nil {
		httpserver.Error(w, r, http.StatusBadRequest, "invalid_upload", "请选择要上传的图片")
		return
	}
	defer file.Close()

	data, err := imageproc.ReadAllLimited(file, contentMaxBytes)
	if err != nil {
		httpserver.Error(w, r, http.StatusRequestEntityTooLarge, "image_too_large", "图片不能超过 5 MB")
		return
	}

	result, err := h.service.Upload(r.Context(), principal.UserID, kind, header.Filename, data)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	if after != nil {
		if err := after(r.Context(), principal.UserID, result.Key); err != nil {
			h.logger.Error("failed to attach image to profile", "error", err)
			httpserver.Error(w, r, http.StatusInternalServerError, "image_attach_failed", "图片已上传，但资料更新失败")
			return
		}
	}
	httpserver.JSON(w, http.StatusCreated, map[string]any{"image": result})
}

// Serve 输出图片；键与规格都只作为目录/文件名使用，扩展名由服务端决定。
func (h *Handler) Serve(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	variant := r.PathValue("variant")
	if !validKey(key) || variant == "" || strings.ContainsAny(variant, "/\\.") {
		http.NotFound(w, r)
		return
	}
	data, contentType, err := h.service.Open(r.Context(), key, variant)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		h.logger.Error("failed to read image", "key", key, "variant", variant, "error", err)
		http.Error(w, "image unavailable", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	w.Header().Set("ETag", `"`+key+"-"+variant+`"`)
	if match := r.Header.Get("If-None-Match"); match != "" && strings.Contains(match, key+"-"+variant) {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	_, _ = w.Write(data)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	principal, ok := auth.PrincipalFromContext(r.Context())
	if !ok {
		httpserver.Error(w, r, http.StatusUnauthorized, "authentication_required", "请先登录")
		return
	}
	key := r.PathValue("key")
	if !validKey(key) {
		httpserver.Error(w, r, http.StatusBadRequest, "invalid_image", "图片标识不正确")
		return
	}
	isManager := principal.HasPermission("content.manage") || principal.HasPermission("content.moderate")
	if err := h.service.Delete(r.Context(), principal.UserID, isManager, key); err != nil {
		h.writeError(w, r, err)
		return
	}
	httpserver.JSON(w, http.StatusOK, map[string]any{"deleted": true})
}

func (h *Handler) writeError(w http.ResponseWriter, r *http.Request, err error) {
	var validation *ValidationError
	switch {
	case errors.As(err, &validation):
		httpserver.Error(w, r, http.StatusBadRequest, "invalid_image", validation.Message)
	case errors.Is(err, ErrTooLarge):
		httpserver.Error(w, r, http.StatusRequestEntityTooLarge, "image_too_large", "图片超过大小限制")
	case errors.Is(err, ErrInvalidImage):
		httpserver.Error(w, r, http.StatusBadRequest, "invalid_image", "仅支持 jpg、png、gif 格式的真实图片")
	case errors.Is(err, ErrForbidden):
		httpserver.Error(w, r, http.StatusForbidden, "permission_denied", "没有操作该图片的权限")
	case errors.Is(err, ErrNotFound):
		httpserver.Error(w, r, http.StatusNotFound, "image_not_found", "图片不存在")
	default:
		h.logger.Error("image operation failed", "request_id", httpserver.RequestIDFromContext(r.Context()), "error", err)
		httpserver.Error(w, r, http.StatusInternalServerError, "image_error", "图片操作未完成")
	}
}

func validKey(key string) bool {
	if len(key) != 32 {
		return false
	}
	for _, char := range key {
		if (char < '0' || char > '9') && (char < 'a' || char > 'f') {
			return false
		}
	}
	return true
}
