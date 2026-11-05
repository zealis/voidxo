package outbox

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// 事件类型保持稳定字符串，消费者按前缀/精确匹配分发。
const (
	EventQuestionCreated = "content.question.created"
	EventQuestionUpdated = "content.question.updated"
	EventQuestionDeleted = "content.question.deleted"
	EventAnswerCreated   = "content.answer.created"
	EventAnswerUpdated   = "content.answer.updated"
	EventAnswerDeleted   = "content.answer.deleted"
	EventArticleCreated  = "content.article.created"
	EventArticleUpdated  = "content.article.updated"
	EventArticleDeleted  = "content.article.deleted"
	EventCommentCreated  = "content.comment.created"
	EventCommentUpdated  = "content.comment.updated"
	EventCommentDeleted  = "content.comment.deleted"
	EventTopicCreated    = "content.topic.created"
	EventTopicUpdated    = "content.topic.updated"
	EventTopicDeleted    = "content.topic.deleted"

	// 邮件投递：payload 携带收件人、主题与正文。
	EventEmailSend = "email.send"

	// 后台配置与权限变更：Worker 据此失效对应缓存作用域。
	EventOptionUpdated   = "option.updated"
	EventUserRoleChanged = "user.role.changed"
)

var ErrNotFound = errors.New("outbox event not found")

type Event struct {
	ID       int64
	Type     string
	Key      string
	Payload  []byte
	Attempts int
}

// Enqueue 在业务事务内写入事件，保证业务数据与事件同生共死。
func Enqueue(ctx context.Context, tx *sql.Tx, eventType string, entityID int64, payload any) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("encode outbox payload: %w", err)
	}
	key := fmt.Sprintf("%s:%d:%d", eventType, entityID, time.Now().UnixNano())
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO vx_outbox (event_type, event_key, payload)
		VALUES (?, ?, ?)`, eventType, key, raw); err != nil {
		return fmt.Errorf("insert outbox event: %w", err)
	}
	return nil
}

type Store interface {
	Claim(ctx context.Context, limit int) ([]Event, error)
	Complete(ctx context.Context, id int64) error
	Fail(ctx context.Context, id int64, reason string, retryAfter time.Duration) error
}

type MySQLStore struct {
	db *sql.DB
}

func NewMySQLStore(db *sql.DB) *MySQLStore {
	return &MySQLStore{db: db}
}

// Claim 使用 SKIP LOCKED 领取待处理事件，支持多个 Worker 并发消费而不互相阻塞。
func (s *MySQLStore) Claim(ctx context.Context, limit int) ([]Event, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin claim outbox: %w", err)
	}
	defer tx.Rollback()

	rows, err := tx.QueryContext(ctx, `
		SELECT outbox_id, event_type, event_key, payload, attempts
		FROM vx_outbox
		WHERE status = 'pending' AND available_at <= CURRENT_TIMESTAMP(6)
		ORDER BY outbox_id
		LIMIT ?
		FOR UPDATE SKIP LOCKED`, limit)
	if err != nil {
		return nil, fmt.Errorf("select outbox: %w", err)
	}
	var events []Event
	for rows.Next() {
		var event Event
		if err := rows.Scan(&event.ID, &event.Type, &event.Key, &event.Payload, &event.Attempts); err != nil {
			rows.Close()
			return nil, fmt.Errorf("scan outbox: %w", err)
		}
		events = append(events, event)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate outbox: %w", err)
	}
	if len(events) == 0 {
		return nil, nil
	}
	placeholders, args := idsPlaceholders(events)
	if _, err := tx.ExecContext(ctx,
		"UPDATE vx_outbox SET status = 'processing', attempts = attempts + 1 WHERE outbox_id IN ("+placeholders+")",
		args...); err != nil {
		return nil, fmt.Errorf("mark outbox processing: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit claim outbox: %w", err)
	}
	for i := range events {
		events[i].Attempts++
	}
	return events, nil
}

func (s *MySQLStore) Complete(ctx context.Context, id int64) error {
	// 处理完成后清空载荷：邮件正文含验证码，不应长期留在事件表里。
	if _, err := s.db.ExecContext(ctx,
		"UPDATE vx_outbox SET status = 'done', processed_at = CURRENT_TIMESTAMP(6),"+
			" last_error = '', payload = JSON_OBJECT() WHERE outbox_id = ?", id); err != nil {
		return fmt.Errorf("complete outbox event: %w", err)
	}
	return nil
}

// PurgeHistory 删除已完成/已失败的历史事件，避免事件表无限增长。
func (s *MySQLStore) PurgeHistory(ctx context.Context, doneBefore, failedBefore time.Duration) (int64, error) {
	result, err := s.db.ExecContext(ctx,
		"DELETE FROM vx_outbox WHERE (status = 'done' AND processed_at < CURRENT_TIMESTAMP(6) - INTERVAL ? SECOND)"+
			" OR (status = 'failed' AND processed_at < CURRENT_TIMESTAMP(6) - INTERVAL ? SECOND)",
		int64(doneBefore.Seconds()), int64(failedBefore.Seconds()))
	if err != nil {
		return 0, fmt.Errorf("purge outbox history: %w", err)
	}
	removed, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("purge outbox rows: %w", err)
	}
	return removed, nil
}

// Fail 采用指数退避重试；超过 5 次后标记 failed，交由人工排查。
func (s *MySQLStore) Fail(ctx context.Context, id int64, reason string, retryAfter time.Duration) error {
	if len(reason) > 500 {
		reason = reason[:500]
	}
	status := "pending"
	if retryAfter <= 0 {
		status = "failed"
	}
	if _, err := s.db.ExecContext(ctx, `
		UPDATE vx_outbox
		SET status = ?, available_at = CURRENT_TIMESTAMP(6) + INTERVAL ? MICROSECOND, last_error = ?
		WHERE outbox_id = ?`,
		status, retryAfter.Microseconds(), reason, id); err != nil {
		return fmt.Errorf("fail outbox event: %w", err)
	}
	return nil
}

func idsPlaceholders(events []Event) (string, []any) {
	parts := make([]string, len(events))
	args := make([]any, len(events))
	for i, event := range events {
		parts[i] = "?"
		args[i] = event.ID
	}
	return strings.Join(parts, ","), args
}
