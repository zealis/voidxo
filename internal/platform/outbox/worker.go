package outbox

import (
	"context"
	"log/slog"
	"strings"
	"time"
)

type Handler func(ctx context.Context, event Event) error

const (
	defaultBatch        = 20
	defaultPollInterval = 1 * time.Second
	maxAttempts         = 5
)

type Worker struct {
	store        Store
	logger       *slog.Logger
	handlers     map[string]Handler
	prefixes     []prefixHandler
	batch        int
	pollInterval time.Duration
}

type prefixHandler struct {
	prefix  string
	handler Handler
}

func NewWorker(store Store, logger *slog.Logger) *Worker {
	return &Worker{
		store:        store,
		logger:       logger,
		handlers:     make(map[string]Handler),
		batch:        defaultBatch,
		pollInterval: defaultPollInterval,
	}
}

func (w *Worker) Handle(eventType string, handler Handler) {
	w.handlers[eventType] = handler
}

// HandlePrefix 注册兜底处理器，例如统一处理所有 content.* 事件。
func (w *Worker) HandlePrefix(prefix string, handler Handler) {
	w.prefixes = append(w.prefixes, prefixHandler{prefix: prefix, handler: handler})
}

func (w *Worker) Run(ctx context.Context) error {
	w.logger.Info("outbox worker started", "batch", w.batch, "poll_interval", w.pollInterval.String())
	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()
	for {
		if _, err := w.ProcessOnce(ctx); err != nil {
			w.logger.Error("outbox batch failed", "error", err)
		}
		select {
		case <-ctx.Done():
			w.logger.Info("outbox worker stopped")
			return nil
		case <-ticker.C:
		}
	}
}

// ProcessOnce 领取并处理一批事件，返回处理条数。
func (w *Worker) ProcessOnce(ctx context.Context) (int, error) {
	events, err := w.store.Claim(ctx, w.batch)
	if err != nil {
		return 0, err
	}
	for _, event := range events {
		handler := w.handlerFor(event.Type)
		if handler == nil {
			// 未知事件直接完成，避免阻塞后续事件；新类型上线前会被记录。
			w.logger.Warn("no handler for outbox event", "type", event.Type, "id", event.ID)
			if err := w.store.Complete(ctx, event.ID); err != nil {
				return 0, err
			}
			continue
		}
		if err := handler(ctx, event); err != nil {
			w.logger.Warn("outbox event failed", "type", event.Type, "id", event.ID, "attempt", event.Attempts, "error", err)
			retryAfter := backoff(event.Attempts)
			if failErr := w.store.Fail(ctx, event.ID, err.Error(), retryAfter); failErr != nil {
				return 0, failErr
			}
			continue
		}
		if err := w.store.Complete(ctx, event.ID); err != nil {
			return 0, err
		}
	}
	return len(events), nil
}

func (w *Worker) handlerFor(eventType string) Handler {
	if handler, ok := w.handlers[eventType]; ok {
		return handler
	}
	for _, entry := range w.prefixes {
		if strings.HasPrefix(eventType, entry.prefix) {
			return entry.handler
		}
	}
	return nil
}

// backoff 返回 2s、4s、8s、16s 的退避；超过上限返回 0 表示标记失败。
func backoff(attempts int) time.Duration {
	if attempts >= maxAttempts {
		return 0
	}
	delay := time.Duration(1<<attempts) * time.Second
	if delay > time.Minute {
		delay = time.Minute
	}
	return delay
}
