package profile

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/voidxo/voidxo/internal/modules/auth"
	"github.com/voidxo/voidxo/internal/modules/user"
	"github.com/voidxo/voidxo/internal/platform/httpserver"
)

type Handler struct {
	service *Service
	logger  *slog.Logger
}

func NewHandler(service *Service, logger *slog.Logger) *Handler {
	return &Handler{service: service, logger: logger}
}

func (h *Handler) GetMe(w http.ResponseWriter, r *http.Request) {
	principal, ok := auth.PrincipalFromContext(r.Context())
	if !ok {
		httpserver.Error(w, r, http.StatusUnauthorized, "authentication_required", "请先登录")
		return
	}
	account, err := h.service.Get(r.Context(), principal.UserID)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	httpserver.JSON(w, http.StatusOK, map[string]any{"user": view(account, principal)})
}

func (h *Handler) UpdateMe(w http.ResponseWriter, r *http.Request) {
	principal, ok := auth.PrincipalFromContext(r.Context())
	if !ok {
		httpserver.Error(w, r, http.StatusUnauthorized, "authentication_required", "请先登录")
		return
	}
	var params user.UpdateProfileParams
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&params); err != nil {
		httpserver.Error(w, r, http.StatusBadRequest, "invalid_request", "请求格式不正确")
		return
	}
	account, err := h.service.Update(r.Context(), principal.UserID, params)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	httpserver.JSON(w, http.StatusOK, map[string]any{"user": view(account, principal)})
}

// ListPublic 是公开用户列表接口，供「人脉」页使用。
func (h *Handler) ListPublic(w http.ResponseWriter, r *http.Request) {
	page, pageSize, err := pagination(r)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	result, err := h.service.ListPublic(r.Context(), r.URL.Query().Get("sort"), page, pageSize)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	httpserver.JSON(w, http.StatusOK, result)
}

// GetPublic 是公开资料接口，返回主页可展示的字段。
func (h *Handler) GetPublic(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	result, err := h.service.Public(r.Context(), id)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	httpserver.JSON(w, http.StatusOK, map[string]any{"user": publicView(result)})
}

// ListUserContent 列出用户公开内容，kind 走复数形式（questions/answers/articles/comments）。
func (h *Handler) ListUserContent(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	kind, ok := kindFromPath(r.PathValue("kind"))
	if !ok {
		httpserver.Error(w, r, http.StatusBadRequest, "invalid_kind", "内容类型不支持")
		return
	}
	page, pageSize, err := pagination(r)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	result, err := h.service.UserContent(r.Context(), kind, id, page, pageSize)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	httpserver.JSON(w, http.StatusOK, result)
}

func publicView(profile user.PublicProfile) map[string]any {
	return map[string]any{
		"id": profile.ID, "username": profile.Username,
		"avatar_key": profile.AvatarKey, "cover_key": profile.CoverKey,
		"headline": profile.Headline, "bio": profile.Bio, "blog": profile.Blog,
		"company": profile.Company, "location": profile.Location,
		"follower_count": profile.FollowerCount, "followee_count": profile.FolloweeCount,
		"article_count": profile.ArticleCount, "question_count": profile.QuestionCount,
		"answer_count": profile.AnswerCount, "comment_count": profile.CommentCount,
		"created_at": profile.CreatedAt,
	}
}

func kindFromPath(value string) (string, bool) {
	switch value {
	case "questions":
		return "question", true
	case "answers":
		return "answer", true
	case "articles":
		return "article", true
	case "comments":
		return "comment", true
	default:
		return "", false
	}
}

func pathID(r *http.Request) (int64, error) {
	value, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || value <= 0 {
		return 0, &ValidationError{Message: "用户编号不正确"}
	}
	return value, nil
}

func pagination(r *http.Request) (int, int, error) {
	page := 1
	pageSize := 20
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
		if err != nil || value < 1 || value > 50 {
			return 0, 0, &ValidationError{Message: "每页数量不正确"}
		}
		pageSize = value
	}
	return page, pageSize, nil
}

func (h *Handler) writeError(w http.ResponseWriter, r *http.Request, err error) {
	var validation *ValidationError
	switch {
	case errors.As(err, &validation):
		httpserver.Error(w, r, http.StatusBadRequest, "invalid_profile", validation.Message)
	case errors.Is(err, user.ErrNotFound):
		httpserver.Error(w, r, http.StatusNotFound, "user_not_found", "用户不存在")
	default:
		h.logger.Error("profile operation failed", "request_id", httpserver.RequestIDFromContext(r.Context()), "error", err)
		httpserver.Error(w, r, http.StatusInternalServerError, "profile_error", "资料操作未完成")
	}
}

func view(account user.Account, principal auth.Principal) map[string]any {
	return map[string]any{
		"id": account.ID, "username": account.Username, "email": account.Email,
		"avatar_key": account.AvatarKey, "cover_key": account.CoverKey,
		"headline": account.Headline, "bio": account.Bio, "blog": account.Blog,
		"company": account.Company, "location": account.Location,
		"roles": principal.Roles, "permissions": principal.Permissions,
	}
}
