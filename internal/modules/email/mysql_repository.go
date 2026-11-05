package email

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/voidxo/voidxo/internal/platform/outbox"
)

type MySQLRepository struct {
	db    *sql.DB
	table string
}

func NewMySQLRepository(db *sql.DB, tablePrefix string) *MySQLRepository {
	return &MySQLRepository{db: db, table: tablePrefix}
}

func (r *MySQLRepository) CreateCode(
	ctx context.Context,
	address, purpose, codeHash string,
	expiresAt time.Time,
	mail Mail,
) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin create email code: %w", err)
	}
	defer tx.Rollback()

	result, err := tx.ExecContext(ctx,
		"INSERT INTO "+r.t("email_code")+" (email, purpose, code_hash, expires_at) VALUES (?, ?, ?, ?)",
		address, purpose, codeHash, expiresAt)
	if err != nil {
		return fmt.Errorf("insert email code: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("read email code id: %w", err)
	}
	// 邮件与验证码在同一事务内投递，避免出现「码已入库但信没发」。
	if err := outbox.Enqueue(ctx, tx, outbox.EventEmailSend, id, map[string]any{
		"to": mail.To, "subject": mail.Subject, "text": mail.Text,
	}); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit create email code: %w", err)
	}
	return nil
}

func (r *MySQLRepository) LatestCode(ctx context.Context, address, purpose string) (Code, error) {
	var record Code
	err := r.db.QueryRowContext(ctx,
		"SELECT code_id, code_hash, attempts, expires_at FROM "+r.t("email_code")+
			" WHERE email = ? AND purpose = ? AND consumed_at IS NULL ORDER BY code_id DESC LIMIT 1",
		address, purpose).
		Scan(&record.ID, &record.CodeHash, &record.Attempts, &record.ExpiresAt)
	if errors.Is(err, sql.ErrNoRows) {
		return Code{}, ErrCodeExpired
	}
	if err != nil {
		return Code{}, fmt.Errorf("load email code: %w", err)
	}
	return record, nil
}

func (r *MySQLRepository) IncrementAttempts(ctx context.Context, id int64) error {
	if _, err := r.db.ExecContext(ctx,
		"UPDATE "+r.t("email_code")+" SET attempts = attempts + 1 WHERE code_id = ?", id); err != nil {
		return fmt.Errorf("increment email code attempts: %w", err)
	}
	return nil
}

func (r *MySQLRepository) Consume(ctx context.Context, id int64) error {
	result, err := r.db.ExecContext(ctx,
		"UPDATE "+r.t("email_code")+" SET consumed_at = CURRENT_TIMESTAMP(6)"+
			" WHERE code_id = ? AND consumed_at IS NULL", id)
	if err != nil {
		return fmt.Errorf("consume email code: %w", err)
	}
	if rows, err := result.RowsAffected(); err == nil && rows == 0 {
		// 并发校验时只有一个请求能消费成功，其余视为过期。
		return ErrCodeExpired
	}
	return nil
}

// PurgeExpired 清理一天前过期的验证码，避免表无限增长；调用方可以忽略错误。
func (r *MySQLRepository) PurgeExpired(ctx context.Context) error {
	if _, err := r.db.ExecContext(ctx,
		"DELETE FROM "+r.t("email_code")+" WHERE expires_at < CURRENT_TIMESTAMP(6) - INTERVAL 1 DAY"); err != nil {
		return fmt.Errorf("purge expired email codes: %w", err)
	}
	return nil
}

func (r *MySQLRepository) t(name string) string {
	return r.table + name
}
