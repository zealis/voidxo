package app

import (
	"context"
	"log/slog"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/voidxo/voidxo/internal/modules/admin"
	"github.com/voidxo/voidxo/internal/modules/auth"
	"github.com/voidxo/voidxo/internal/modules/content"
	"github.com/voidxo/voidxo/internal/modules/email"
	"github.com/voidxo/voidxo/internal/modules/engagement"
	"github.com/voidxo/voidxo/internal/modules/inbox"
	"github.com/voidxo/voidxo/internal/modules/media"
	"github.com/voidxo/voidxo/internal/modules/option"
	"github.com/voidxo/voidxo/internal/modules/profile"
	"github.com/voidxo/voidxo/internal/modules/search"
	"github.com/voidxo/voidxo/internal/modules/stats"
	"github.com/voidxo/voidxo/internal/platform/cache"
	"github.com/voidxo/voidxo/internal/platform/clientip"
	"github.com/voidxo/voidxo/internal/platform/config"
	"github.com/voidxo/voidxo/internal/platform/httpserver"
)

type Pinger interface {
	PingContext(context.Context) error
}

type Dependencies struct {
	Database   Pinger
	RedisCheck func(context.Context) error
	Options    option.Repository
	Auth       *auth.Service
	Profiles   *profile.Service
	Content    *content.Service
	Engagement *engagement.Service
	Search     *search.Service
	Cache      cache.Cache
	Media      *media.Handler
	Admin      *admin.Handler
	Inbox      *inbox.Handler
	Email      *email.Handler
	Stats      *stats.Handler
}

func NewRouter(cfg config.Config, logger *slog.Logger, deps Dependencies) http.Handler {
	mux := http.NewServeMux()
	optionService := option.NewService(deps.Options)
	if deps.Cache != nil {
		optionService.SetCache(deps.Cache)
	}
	optionHandler := option.NewHandler(optionService, logger)

	mux.HandleFunc("/health/live", onlyGET(func(w http.ResponseWriter, r *http.Request) {
		httpserver.JSON(w, http.StatusOK, map[string]any{"status": "ok", "service": "voidxo-api"})
	}))
	mux.HandleFunc("/health/ready", onlyGET(func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if deps.Database == nil || deps.Database.PingContext(ctx) != nil ||
			(deps.RedisCheck != nil && deps.RedisCheck(ctx) != nil) {
			httpserver.Error(w, r, http.StatusServiceUnavailable, "not_ready", "服务尚未就绪")
			return
		}
		httpserver.JSON(w, http.StatusOK, map[string]any{"status": "ready"})
	}))
	mux.HandleFunc("/api/v1/options", onlyGET(optionHandler.GetPublic))
	if deps.Content != nil {
		// 站点地图只包含公开内容，不需要登录。
		mux.HandleFunc("/api/v1/sitemap", onlyGET(content.NewHandler(deps.Content, logger).GetSitemap))
	}
	if deps.Stats != nil {
		mux.HandleFunc("/api/v1/stats", onlyGET(deps.Stats.Public))
	}

	if deps.Auth != nil {
		authHandler := auth.NewHandler(deps.Auth, cfg.CookieSecure, logger)
		mux.HandleFunc("/api/v1/auth/csrf", noStore(onlyGET(authHandler.GetCSRF)))
		mux.HandleFunc("/api/v1/auth/login", noStore(onlyPOST(authHandler.RequireCSRF(authHandler.Login))))
		mux.HandleFunc("/api/v1/auth/me", noStore(onlyGET(authHandler.RequireAuth(authHandler.Me))))
		mux.HandleFunc("/api/v1/auth/logout", noStore(onlyPOST(authHandler.RequireAuth(authHandler.RequireCSRF(authHandler.Logout)))))
		mux.HandleFunc("/api/v1/auth/password", noStore(onlyPOST(authHandler.RequireAuth(authHandler.RequireCSRF(authHandler.ChangePassword)))))
		mux.HandleFunc("/api/v1/auth/register", noStore(onlyPOST(authHandler.RequireCSRF(authHandler.Register))))
		mux.HandleFunc("/api/v1/auth/password/reset", noStore(onlyPOST(authHandler.RequireCSRF(authHandler.ResetPassword))))
		if deps.Email != nil {
			// 发送验证码是公开操作，靠 CSRF 与模块内限流约束。
			mux.HandleFunc("/api/v1/email/code", noStore(onlyPOST(authHandler.RequireCSRF(deps.Email.SendCode))))
		}
		if deps.Profiles != nil {
			profileHandler := profile.NewHandler(deps.Profiles, logger)
			mux.HandleFunc("/api/v1/user", noStore(onlyMethods(map[string]http.HandlerFunc{
				http.MethodGet:   authHandler.RequireAuth(profileHandler.GetMe),
				http.MethodPatch: authHandler.RequireAuth(authHandler.RequireCSRF(profileHandler.UpdateMe)),
			})))
			// 公开用户主页与内容列表，无需登录。
			mux.HandleFunc("/api/v1/users", noStore(onlyGET(profileHandler.ListPublic)))
			mux.HandleFunc("/api/v1/users/{id}", noStore(onlyGET(profileHandler.GetPublic)))
			mux.HandleFunc("/api/v1/users/{id}/{kind}", noStore(onlyGET(profileHandler.ListUserContent)))
		}
		if deps.Content != nil {
			contentHandler := content.NewHandler(deps.Content, logger)
			mux.HandleFunc("/api/v1/topics", onlyMethods(map[string]http.HandlerFunc{
				http.MethodGet:  contentHandler.ListTopics,
				http.MethodPost: authHandler.RequirePermission("content.manage", authHandler.RequireCSRF(contentHandler.CreateTopic)),
			}))
			mux.HandleFunc("/api/v1/topics/{id}", onlyGET(contentHandler.GetTopic))
			mux.HandleFunc("/api/v1/questions", onlyMethods(map[string]http.HandlerFunc{
				http.MethodGet:  contentHandler.ListQuestions,
				http.MethodPost: authHandler.RequireAuth(authHandler.RequireCSRF(contentHandler.CreateQuestion)),
			}))
			mux.HandleFunc("/api/v1/questions/{id}", onlyMethods(map[string]http.HandlerFunc{
				http.MethodGet:    contentHandler.GetQuestion,
				http.MethodPatch:  authHandler.RequireAuth(authHandler.RequireCSRF(contentHandler.UpdateQuestion)),
				http.MethodDelete: authHandler.RequireAuth(authHandler.RequireCSRF(contentHandler.DeleteQuestion)),
			}))
			mux.HandleFunc("/api/v1/questions/{id}/answers", onlyMethods(map[string]http.HandlerFunc{
				http.MethodGet:  contentHandler.ListAnswers,
				http.MethodPost: authHandler.RequireAuth(authHandler.RequireCSRF(contentHandler.CreateAnswer)),
			}))
			mux.HandleFunc("/api/v1/answers/{answer_id}", onlyMethods(map[string]http.HandlerFunc{
				http.MethodPatch:  authHandler.RequireAuth(authHandler.RequireCSRF(contentHandler.UpdateAnswer)),
				http.MethodDelete: authHandler.RequireAuth(authHandler.RequireCSRF(contentHandler.DeleteAnswer)),
			}))
			mux.HandleFunc("/api/v1/articles", onlyMethods(map[string]http.HandlerFunc{
				http.MethodGet:  contentHandler.ListArticles,
				http.MethodPost: authHandler.RequireAuth(authHandler.RequireCSRF(contentHandler.CreateArticle)),
			}))
			mux.HandleFunc("/api/v1/articles/{id}", onlyMethods(map[string]http.HandlerFunc{
				http.MethodGet:    contentHandler.GetArticle,
				http.MethodPatch:  authHandler.RequireAuth(authHandler.RequireCSRF(contentHandler.UpdateArticle)),
				http.MethodDelete: authHandler.RequireAuth(authHandler.RequireCSRF(contentHandler.DeleteArticle)),
			}))
			mux.HandleFunc("/api/v1/comments", noStore(onlyMethods(map[string]http.HandlerFunc{
				http.MethodGet:  contentHandler.ListComments,
				http.MethodPost: authHandler.RequireAuth(authHandler.RequireCSRF(contentHandler.CreateComment)),
			})))
			mux.HandleFunc("/api/v1/comments/{id}", noStore(onlyMethods(map[string]http.HandlerFunc{
				http.MethodPatch:  authHandler.RequireAuth(authHandler.RequireCSRF(contentHandler.UpdateComment)),
				http.MethodDelete: authHandler.RequireAuth(authHandler.RequireCSRF(contentHandler.DeleteComment)),
			})))
		}
		if deps.Media != nil {
			mux.HandleFunc("/api/v1/images", noStore(onlyPOST(authHandler.RequireAuth(authHandler.RequireCSRF(deps.Media.UploadContentImage)))))
			mux.HandleFunc("/api/v1/images/{key}", noStore(onlyMethod(http.MethodDelete, authHandler.RequireAuth(authHandler.RequireCSRF(deps.Media.Delete)))))
			mux.HandleFunc("/api/v1/user/avatar", noStore(onlyPOST(authHandler.RequireAuth(authHandler.RequireCSRF(deps.Media.UploadAvatar)))))
			mux.HandleFunc("/api/v1/user/cover", noStore(onlyPOST(authHandler.RequireAuth(authHandler.RequireCSRF(deps.Media.UploadCover)))))
		}
		if deps.Search != nil {
			searchHandler := search.NewHandler(deps.Search, logger)
			mux.HandleFunc("/api/v1/search", noStore(onlyGET(authHandler.RequireAuth(searchHandler.Search))))
		}
		if deps.Engagement != nil {
			engagementHandler := engagement.NewHandler(deps.Engagement, logger)
			mux.HandleFunc("/api/v1/votes", noStore(onlyMethods(map[string]http.HandlerFunc{
				http.MethodGet:    authHandler.RequireAuth(engagementHandler.GetVote),
				http.MethodPost:   authHandler.RequireAuth(authHandler.RequireCSRF(engagementHandler.CreateVote)),
				http.MethodDelete: authHandler.RequireAuth(authHandler.RequireCSRF(engagementHandler.DeleteVote)),
			})))
			mux.HandleFunc("/api/v1/follows", noStore(onlyMethods(map[string]http.HandlerFunc{
				http.MethodGet:    authHandler.RequireAuth(engagementHandler.GetFollow),
				http.MethodPost:   authHandler.RequireAuth(authHandler.RequireCSRF(engagementHandler.Follow)),
				http.MethodDelete: authHandler.RequireAuth(authHandler.RequireCSRF(engagementHandler.Unfollow)),
			})))
			mux.HandleFunc("/api/v1/reports", noStore(onlyMethods(map[string]http.HandlerFunc{
				http.MethodGet:  authHandler.RequirePermission("report.manage", engagementHandler.ListReports),
				http.MethodPost: authHandler.RequireAuth(authHandler.RequireCSRF(engagementHandler.CreateReport)),
			})))
			mux.HandleFunc("/api/v1/reports/{id}", noStore(onlyMethod(http.MethodPatch,
				authHandler.RequirePermission("report.manage", authHandler.RequireCSRF(engagementHandler.ResolveReport)))))
			mux.HandleFunc("/api/v1/notifications", noStore(onlyGET(authHandler.RequireAuth(engagementHandler.ListNotifications))))
			mux.HandleFunc("/api/v1/notifications/unread_count", noStore(onlyGET(authHandler.RequireAuth(engagementHandler.UnreadCount))))
			mux.HandleFunc("/api/v1/notifications/read", noStore(onlyPOST(authHandler.RequireAuth(authHandler.RequireCSRF(engagementHandler.MarkNotificationsRead)))))
			mux.HandleFunc("/api/v1/notifications/delete", noStore(onlyPOST(authHandler.RequireAuth(authHandler.RequireCSRF(engagementHandler.DeleteNotifications)))))
		}
		if deps.Inbox != nil {
			inboxHandler := deps.Inbox
			mux.HandleFunc("/api/v1/inbox", noStore(onlyMethods(map[string]http.HandlerFunc{
				http.MethodGet:  authHandler.RequireAuth(inboxHandler.ListConversations),
				http.MethodPost: authHandler.RequireAuth(authHandler.RequireCSRF(inboxHandler.Send)),
			})))
			mux.HandleFunc("/api/v1/inbox/unread_count", noStore(onlyGET(authHandler.RequireAuth(inboxHandler.UnreadCount))))
			mux.HandleFunc("/api/v1/inbox/recipient", noStore(onlyGET(authHandler.RequireAuth(inboxHandler.Recipient))))
			mux.HandleFunc("/api/v1/inbox/{id}", noStore(onlyGET(authHandler.RequireAuth(inboxHandler.ListMessages))))
			mux.HandleFunc("/api/v1/inbox/{id}/read", noStore(onlyPOST(
				authHandler.RequireAuth(authHandler.RequireCSRF(inboxHandler.MarkRead)))))
		}
		if deps.Admin != nil {
			adminHandler := deps.Admin
			adminOnly := func(handler http.HandlerFunc) http.HandlerFunc {
				return authHandler.RequirePermission("admin.access", handler)
			}
			adminWrite := func(handler http.HandlerFunc) http.HandlerFunc {
				return authHandler.RequireCSRF(handler)
			}
			mux.HandleFunc("/api/v1/admin/stats", noStore(onlyGET(adminOnly(adminHandler.Stats))))

			mux.HandleFunc("/api/v1/admin/users", noStore(onlyGET(
				authHandler.RequirePermission("user.manage", adminHandler.ListUsers))))
			mux.HandleFunc("/api/v1/admin/users/{id}", noStore(onlyMethod(http.MethodPatch,
				authHandler.RequirePermission("user.manage", adminWrite(adminHandler.UpdateUser)))))

			mux.HandleFunc("/api/v1/admin/content/{kind}", noStore(onlyGET(
				authHandler.RequirePermission("content.moderate", adminHandler.ListContent))))
			mux.HandleFunc("/api/v1/admin/content/{kind}/{id}", noStore(onlyMethod(http.MethodDelete,
				authHandler.RequirePermission("content.moderate", adminWrite(adminHandler.DeleteContent)))))

			mux.HandleFunc("/api/v1/admin/topics", noStore(onlyMethods(map[string]http.HandlerFunc{
				http.MethodGet:  authHandler.RequirePermission("content.manage", adminHandler.ListTopics),
				http.MethodPost: authHandler.RequirePermission("content.manage", adminWrite(adminHandler.CreateTopic)),
			})))
			mux.HandleFunc("/api/v1/admin/topics/{id}", noStore(onlyMethods(map[string]http.HandlerFunc{
				http.MethodPatch:  authHandler.RequirePermission("content.manage", adminWrite(adminHandler.UpdateTopic)),
				http.MethodDelete: authHandler.RequirePermission("content.manage", adminWrite(adminHandler.DeleteTopic)),
			})))

			mux.HandleFunc("/api/v1/admin/options", noStore(onlyMethods(map[string]http.HandlerFunc{
				http.MethodGet:   authHandler.RequirePermission("option.manage", adminHandler.ListOptions),
				http.MethodPatch: authHandler.RequirePermission("option.manage", adminWrite(adminHandler.UpdateOptions)),
			})))

			mux.HandleFunc("/api/v1/admin/images", noStore(onlyGET(
				authHandler.RequirePermission("file.manage", adminHandler.ListImages))))
			mux.HandleFunc("/api/v1/admin/images/{key}", noStore(onlyMethod(http.MethodDelete,
				authHandler.RequirePermission("file.manage", adminWrite(adminHandler.DeleteImage)))))
		}
	}

	if deps.Media != nil {
		// 媒体文件不依赖登录态，可长缓存；键为随机值，内容不可变。
		mux.HandleFunc("/media/{key}/{variant}", onlyGET(deps.Media.Serve))
	}

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		httpserver.Error(w, r, http.StatusNotFound, "not_found", "接口不存在")
	})

	resolver, err := clientip.New(cfg.TrustedProxies)
	if err != nil {
		logger.Error("invalid trusted proxy configuration", "error", err)
		resolver, _ = clientip.New(nil)
	}

	return httpserver.Chain(mux,
		httpserver.RequestID,
		httpserver.Recover(logger),
		withClientIP(resolver),
		httpserver.AccessLog(logger),
		httpserver.SecurityHeaders,
		httpserver.CORS(cfg.AllowedOrigins),
	)
}

func withClientIP(resolver *clientip.Resolver) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r.WithContext(httpserver.WithClientIP(r.Context(), resolver.FromRequest(r))))
		})
	}
}

func onlyGET(next http.HandlerFunc) http.HandlerFunc {
	return onlyMethod(http.MethodGet, next)
}

func noStore(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Pragma", "no-cache")
		next(w, r)
	}
}

func onlyPOST(next http.HandlerFunc) http.HandlerFunc {
	return onlyMethod(http.MethodPost, next)
}

func onlyMethod(method string, next http.HandlerFunc) http.HandlerFunc {
	return onlyMethods(map[string]http.HandlerFunc{method: next})
}

func onlyMethods(handlers map[string]http.HandlerFunc) http.HandlerFunc {
	allow := make([]string, 0, len(handlers))
	for method := range handlers {
		allow = append(allow, method)
	}
	sort.Strings(allow)
	allowHeader := strings.Join(allow, ", ")
	return func(w http.ResponseWriter, r *http.Request) {
		next, ok := handlers[r.Method]
		if !ok {
			w.Header().Set("Allow", allowHeader)
			httpserver.Error(w, r, http.StatusMethodNotAllowed, "method_not_allowed", "请求方法不被支持")
			return
		}
		next(w, r)
	}
}
