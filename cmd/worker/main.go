package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/voidxo/voidxo/internal/platform/cache"
	"github.com/voidxo/voidxo/internal/platform/config"
	"github.com/voidxo/voidxo/internal/platform/database"
	"github.com/voidxo/voidxo/internal/platform/mailer"
	"github.com/voidxo/voidxo/internal/platform/outbox"
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

	publicCache := cache.NewRedis(redisClient)
	store := outbox.NewMySQLStore(db)
	worker := outbox.NewWorker(store, logger)
	go purgeOutbox(ctx, store, logger)

	// 未配置 SMTP 时退化为日志发送器，保证本地开发能拿到验证码。
	var sender mailer.Sender
	if cfg.SMTPHost == "" {
		logger.Warn("smtp is not configured, using dev log sender")
		sender = mailer.NewLogSender(logger)
	} else {
		smtpSender, err := mailer.NewSMTPSender(mailer.SMTPConfig{
			Host: cfg.SMTPHost, Port: cfg.SMTPPort, Username: cfg.SMTPUsername,
			Password: cfg.SMTPPassword, From: cfg.SMTPFrom, FromName: cfg.SMTPFromName, Mode: cfg.SMTPMode,
		})
		if err != nil {
			logger.Error("smtp initialization failed", "error", err)
			os.Exit(1)
		}
		sender = smtpSender
	}

	worker.Handle(outbox.EventEmailSend, func(ctx context.Context, event outbox.Event) error {
		var payload struct {
			To      string `json:"to"`
			Subject string `json:"subject"`
			Text    string `json:"text"`
		}
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return fmt.Errorf("decode email payload: %w", err)
		}
		return sender.Send(ctx, mailer.Message{To: payload.To, Subject: payload.Subject, Text: payload.Text})
	})

	// 内容事件先失效数据缓存版本，再通知前端清空公开页 HTML 缓存。
	// 两步都是尽力而为：页面缓存清理失败只记录告警，不让事件重试。
	worker.Handle(outbox.EventOptionUpdated, func(ctx context.Context, event outbox.Event) error {
		if err := publicCache.Bump(ctx, "option"); err != nil {
			return err
		}
		logger.Info("option cache invalidated", "event_id", event.ID)
		revalidatePages(ctx, cfg, logger, "option.updated")
		return nil
	})

	worker.HandlePrefix("content.", func(ctx context.Context, event outbox.Event) error {
		if err := publicCache.Bump(ctx, "content"); err != nil {
			return err
		}
		logger.Info("content views invalidated", "event", event.Type, "event_id", event.ID)
		revalidatePages(ctx, cfg, logger, event.Type)
		return nil
	})

	go func() {
		<-ctx.Done()
		logger.Info("worker shutting down")
	}()

	if err := worker.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
		logger.Error("worker stopped unexpectedly", "error", err)
		os.Exit(1)
	}
}

// revalidateClient 超时短于事件处理预算，避免前端不可用时拖慢队列。
var revalidateClient = &http.Client{Timeout: 3 * time.Second}

// revalidatePages 通知 SvelteKit 清空公开页缓存。
// 未配置地址时直接跳过；失败只告警，因为缓存会在 TTL 后自然过期。
func revalidatePages(ctx context.Context, cfg config.Config, logger *slog.Logger, reason string) {
	if cfg.WebRevalidateURL == "" {
		return
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, cfg.WebRevalidateURL, strings.NewReader("{}"))
	if err != nil {
		logger.Warn("page revalidation request invalid", "error", err)
		return
	}
	request.Header.Set("content-type", "application/json")
	if cfg.WebRevalidateToken != "" {
		request.Header.Set("x-revalidate-token", cfg.WebRevalidateToken)
	}
	response, err := revalidateClient.Do(request)
	if err != nil {
		logger.Warn("page revalidation failed", "error", err, "reason", reason)
		return
	}
	defer response.Body.Close()
	_, _ = io.Copy(io.Discard, response.Body)
	if response.StatusCode >= 300 {
		logger.Warn("page revalidation rejected", "status", response.StatusCode, "reason", reason)
		return
	}
	logger.Info("page cache revalidated", "reason", reason)
}

// purgeOutbox 每天清理历史事件：完成事件保留 7 天便于排查，失败事件保留 30 天。
func purgeOutbox(ctx context.Context, store *outbox.MySQLStore, logger *slog.Logger) {
	purge := func() {
		removed, err := store.PurgeHistory(ctx, 7*24*time.Hour, 30*24*time.Hour)
		if err != nil {
			logger.Warn("purge outbox history failed", "error", err)
			return
		}
		if removed > 0 {
			logger.Info("outbox history purged", "removed", removed)
		}
	}
	purge()
	ticker := time.NewTicker(6 * time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			purge()
		}
	}
}
