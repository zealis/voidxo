package auth

import (
	"context"
	"database/sql"
	"fmt"
)

type MySQLSessionRepository struct {
	db *sql.DB
}

func NewMySQLSessionRepository(db *sql.DB) *MySQLSessionRepository {
	return &MySQLSessionRepository{db: db}
}

func (r *MySQLSessionRepository) Create(ctx context.Context, record SessionRecord) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO vx_user_sessions
			(user_id, token_hash, device_id, ip_address, user_agent, created_at, last_active_at, expires_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		record.UserID, record.TokenHash, record.DeviceID, nullIfEmpty(record.IPAddress), record.UserAgent,
		record.CreatedAt, record.CreatedAt, record.ExpiresAt,
	)
	if err != nil {
		return fmt.Errorf("create session record: %w", err)
	}
	return nil
}

func (r *MySQLSessionRepository) ActiveHashes(ctx context.Context, userID int64) ([]string, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT token_hash FROM vx_user_sessions
		WHERE user_id = ? AND revoked_at IS NULL AND expires_at > CURRENT_TIMESTAMP(6)`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("list active session hashes: %w", err)
	}
	defer rows.Close()
	var hashes []string
	for rows.Next() {
		var hash string
		if err := rows.Scan(&hash); err != nil {
			return nil, fmt.Errorf("scan active session hash: %w", err)
		}
		hashes = append(hashes, hash)
	}
	return hashes, rows.Err()
}

func (r *MySQLSessionRepository) Revoke(ctx context.Context, tokenHash, reason string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE vx_user_sessions
		SET revoked_at = CURRENT_TIMESTAMP(6), revoked_reason = ?
		WHERE token_hash = ? AND revoked_at IS NULL`,
		reason, tokenHash,
	)
	if err != nil {
		return fmt.Errorf("revoke session: %w", err)
	}
	return nil
}

func (r *MySQLSessionRepository) RevokeAll(ctx context.Context, userID int64, reason string) ([]string, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin revoke user sessions: %w", err)
	}
	defer tx.Rollback()

	rows, err := tx.QueryContext(ctx, `
		SELECT token_hash FROM vx_user_sessions
		WHERE user_id = ? AND revoked_at IS NULL AND expires_at > CURRENT_TIMESTAMP(6)`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("list active sessions: %w", err)
	}
	var hashes []string
	for rows.Next() {
		var hash string
		if err := rows.Scan(&hash); err != nil {
			rows.Close()
			return nil, fmt.Errorf("scan active session: %w", err)
		}
		hashes = append(hashes, hash)
	}
	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("close active session rows: %w", err)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate active sessions: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE vx_user_sessions
		SET revoked_at = CURRENT_TIMESTAMP(6), revoked_reason = ?
		WHERE user_id = ? AND revoked_at IS NULL`,
		reason, userID,
	); err != nil {
		return nil, fmt.Errorf("revoke user sessions: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit revoke user sessions: %w", err)
	}
	return hashes, nil
}

func nullIfEmpty(value string) any {
	if value == "" {
		return sql.NullString{}
	}
	return value
}
