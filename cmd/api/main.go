package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/voidxo/voidxo/internal/app"
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
	"github.com/voidxo/voidxo/internal/modules/user"
	"github.com/voidxo/voidxo/internal/platform/cache"
	"github.com/voidxo/voidxo/internal/platform/config"
	"github.com/voidxo/voidxo/internal/platform/database"
	"github.com/voidxo/voidxo/internal/platform/markdown"
	"github.com/voidxo/voidxo/internal/platform/ratelimit"
	redisplatform "github.com/voidxo/voidxo/internal/platform/redis"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	cfg, err := config.Load()
	if err != nil {
		logger.Error("invalid configuration", "error", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	db, err := database.OpenMySQL(ctx, cfg.DatabaseDSN)
	if err != nil {
		logger.Error("database connection failed", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	redisClient, err := redisplatform.Open(ctx, cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB)
	if err != nil {
		logger.Error("redis connection failed", "error", err)
		os.Exit(1)
	}
	defer redisClient.Close()

	users := user.NewMySQLRepository(db)
	authService, err := auth.NewService(
		users,
		auth.NewPasswordHasher(auth.DefaultArgon2Params),
		auth.NewRedisSessionStore(redisClient),
		auth.NewMySQLSessionRepository(db),
		auth.NewRedisLoginLimiter(redisClient, 10, 15*time.Minute),
		cfg.SessionIdleTTL,
		cfg.SessionAbsoluteTTL,
		logger,
	)
	if err != nil {
		logger.Error("auth initialization failed", "error", err)
		os.Exit(1)
	}

	emailService := email.NewService(
		email.NewMySQLRepository(db, database.TablePrefix),
		users,
		ratelimit.NewRedisLimiter(redisClient, "emailcode", 1, time.Minute),
		cfg.EmailCodePepper,
		logger,
	)
	emailHandler := email.NewHandler(emailService, logger)
	authService.SetCodeVerifier(emailCodeVerifier{service: emailService})

	apiCache := cache.NewRedis(redisClient)
	optionRepository := option.NewMySQLRepository(db, database.TablePrefix)
	engagementService := engagement.NewService(engagement.NewMySQLRepository(db))
	contentService := content.NewService(content.NewMySQLRepository(db), markdown.New(), optionRepository)
	contentService.SetNotifier(notificationAdapter{service: engagementService})
	contentService.SetCache(apiCache)
	mediaStorage, err := media.NewFileStorage(cfg.UploadDir)
	if err != nil {
		logger.Error("media storage initialization failed", "error", err)
		os.Exit(1)
	}
	mediaService := media.NewService(media.NewMySQLRepository(db), mediaStorage, cfg.MediaBaseURL)
	mediaHandler := media.NewHandler(mediaService, users, logger)

	inboxService := inbox.NewService(inbox.NewMySQLRepository(db, database.TablePrefix), markdown.New(), logger)
	inboxService.SetLimiter(ratelimit.NewRedisLimiter(redisClient, "inbox", 30, time.Minute))
	inboxHandler := inbox.NewHandler(inboxService, logger)

	profileService := profile.NewService(users)
	profileService.SetContentLister(contentService)

	statsService := stats.NewService(stats.NewMySQLRepository(db, database.TablePrefix))
	statsService.SetCache(apiCache)
	statsHandler := stats.NewHandler(statsService, logger)

	adminService := admin.NewService(admin.NewMySQLRepository(db, database.TablePrefix))
	adminService.SetContentModerator(adminContentAdapter{service: contentService})
	adminService.SetMediaDeleter(adminMediaAdapter{service: mediaService})
	adminService.SetSessionRevoker(authService)
	adminHandler := admin.NewHandler(adminService, logger)

	searchService := search.NewService(
		search.NewMySQLRepository(db),
		ratelimit.NewRedisLimiter(redisClient, "search", 30, time.Minute),
	)

	server := &http.Server{
		Addr: cfg.HTTPAddr,
		Handler: app.NewRouter(cfg, logger, app.Dependencies{
			Database: db,
			RedisCheck: func(ctx context.Context) error {
				return redisClient.Ping(ctx).Err()
			},
			Options:    optionRepository,
			Auth:       authService,
			Profiles:   profileService,
			Content:    contentService,
			Engagement: engagementService,
			Search:     searchService,
			Cache:      apiCache,
			Media:      mediaHandler,
			Admin:      adminHandler,
			Inbox:      inboxHandler,
			Email:      emailHandler,
			Stats:      statsHandler,
		}),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			logger.Error("graceful shutdown failed", "error", err)
		}
	}()

	logger.Info("VOIDXO API listening", "address", cfg.HTTPAddr, "environment", cfg.Environment)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.Error("server stopped unexpectedly", "error", err)
		os.Exit(1)
	}
}

// notificationAdapter 把内容模块声明的事件转换成通知模块的输入，
// 两个模块本身互不依赖，装配关系只存在于这里。
type notificationAdapter struct {
	service *engagement.Service
}

func (a notificationAdapter) Notify(ctx context.Context, event content.NotificationEvent) error {
	return a.service.Notify(ctx, engagement.NotificationEvent{
		ReceiverID:     event.ReceiverID,
		SenderID:       event.SenderID,
		Type:           event.Type,
		ArticleID:      event.ArticleID,
		QuestionID:     event.QuestionID,
		AnswerID:       event.AnswerID,
		CommentID:      event.CommentID,
		ReplyID:        event.ReplyID,
		ContentDeleted: event.ContentDeleted,
	})
}

// adminContentAdapter 把内容模块的错误语义映射为后台模块的错误，
// 避免后台处理器直接依赖内容模块的内部错误值。
type adminContentAdapter struct {
	service *content.Service
}

func (a adminContentAdapter) DeleteQuestion(ctx context.Context, principal auth.Principal, id int64) error {
	return mapContentError(a.service.DeleteQuestion(ctx, principal, id))
}

func (a adminContentAdapter) DeleteAnswer(ctx context.Context, principal auth.Principal, id int64) error {
	return mapContentError(a.service.DeleteAnswer(ctx, principal, id))
}

func (a adminContentAdapter) DeleteArticle(ctx context.Context, principal auth.Principal, id int64) error {
	return mapContentError(a.service.DeleteArticle(ctx, principal, id))
}

func (a adminContentAdapter) DeleteComment(ctx context.Context, principal auth.Principal, id int64) error {
	return mapContentError(a.service.DeleteComment(ctx, principal, id))
}

func mapContentError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, content.ErrNotFound) {
		return admin.ErrNotFound
	}
	var permission *content.PermissionError
	if errors.As(err, &permission) {
		return admin.ErrForbidden
	}
	return err
}

// adminMediaAdapter 复用媒体模块的删除能力，isManager 由后台固定为 true。
type adminMediaAdapter struct {
	service *media.Service
}

func (a adminMediaAdapter) Delete(ctx context.Context, userID int64, isManager bool, key string) error {
	err := a.service.Delete(ctx, userID, isManager, key)
	switch {
	case err == nil:
		return nil
	case errors.Is(err, media.ErrNotFound):
		return admin.ErrNotFound
	case errors.Is(err, media.ErrForbidden):
		return admin.ErrForbidden
	default:
		return err
	}
}

// emailCodeVerifier 把邮箱模块的错误映射为认证模块的语义，
// 两个模块本身互不依赖，装配关系只存在于这里。
type emailCodeVerifier struct {
	service *email.Service
}

func (v emailCodeVerifier) Verify(ctx context.Context, address, purpose, code string) error {
	err := v.service.Verify(ctx, address, purpose, code)
	switch {
	case err == nil:
		return nil
	case errors.Is(err, email.ErrInvalidCode):
		return auth.ErrCodeInvalid
	case errors.Is(err, email.ErrCodeExpired):
		return auth.ErrCodeExpired
	case errors.Is(err, email.ErrTooManyAttempts):
		return auth.ErrCodeAttempts
	default:
		return err
	}
}
