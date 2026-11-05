package inbox

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

type MySQLRepository struct {
	db    *sql.DB
	table string
}

func NewMySQLRepository(db *sql.DB, tablePrefix string) *MySQLRepository {
	return &MySQLRepository{db: db, table: tablePrefix}
}

func (r *MySQLRepository) FindRecipientByID(ctx context.Context, id int64) (Recipient, error) {
	return r.findRecipient(ctx, "user_id = ?", id)
}

func (r *MySQLRepository) FindRecipientByUsername(ctx context.Context, username string) (Recipient, error) {
	return r.findRecipient(ctx, "username = ?", username)
}

func (r *MySQLRepository) findRecipient(ctx context.Context, condition string, arg any) (Recipient, error) {
	var recipient Recipient
	var active bool
	err := r.db.QueryRowContext(ctx,
		"SELECT user_id, username, COALESCE(avatar_key, ''), (status = 1 AND disabled_at IS NULL)"+
			" FROM "+r.t("user")+" WHERE "+condition+" LIMIT 1", arg).
		Scan(&recipient.ID, &recipient.Username, &recipient.AvatarKey, &active)
	if errors.Is(err, sql.ErrNoRows) {
		return Recipient{}, ErrRecipientNotFound
	}
	if err != nil {
		return Recipient{}, fmt.Errorf("find inbox recipient: %w", err)
	}
	recipient.Active = active
	return recipient, nil
}

func (r *MySQLRepository) CreateMessage(ctx context.Context, senderID, receiverID int64, markdownSource, rendered string) (Message, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return Message{}, fmt.Errorf("begin create message: %w", err)
	}
	defer tx.Rollback()

	result, err := tx.ExecContext(ctx,
		"INSERT INTO "+r.t("inbox")+" (sender_id, receiver_id, content_markdown, content_rendered) VALUES (?, ?, ?, ?)",
		senderID, receiverID, markdownSource, rendered)
	if err != nil {
		return Message{}, fmt.Errorf("insert message: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return Message{}, fmt.Errorf("read message id: %w", err)
	}
	// 未读数是收件箱的展示指标，直接在写入事务里按实际未读数重算，避免自增累积误差。
	if err := refreshUnread(ctx, tx, r.t("inbox"), r.t("user"), receiverID); err != nil {
		return Message{}, err
	}
	message, err := scanMessageRow(tx.QueryRowContext(ctx,
		"SELECT i.inbox_id, i.sender_id, s.username, i.receiver_id, i.content_markdown,"+
			" i.content_rendered, i.read_at IS NOT NULL, i.created_at"+
			" FROM "+r.t("inbox")+" i JOIN "+r.t("user")+" s ON s.user_id = i.sender_id"+
			" WHERE i.inbox_id = ?", id))
	if err != nil {
		return Message{}, err
	}
	if err := tx.Commit(); err != nil {
		return Message{}, fmt.Errorf("commit create message: %w", err)
	}
	return message, nil
}

func (r *MySQLRepository) ListConversations(ctx context.Context, userID int64, limit, offset int) ([]Conversation, bool, error) {
	// 每个对话取双方之间最新的一条消息；对方编号用 IF 归一化后分组。
	query := `
		SELECT c.counterpart_id, u.username, COALESCE(u.avatar_key, ''),
			m.sender_id, LEFT(m.content_markdown, 80), m.created_at,
			(SELECT COUNT(*) FROM ` + r.t("inbox") + ` x
				WHERE x.receiver_id = ? AND x.sender_id = c.counterpart_id AND x.read_at IS NULL)
		FROM (
			SELECT IF(sender_id = ?, receiver_id, sender_id) AS counterpart_id, MAX(inbox_id) AS last_id
			FROM ` + r.t("inbox") + `
			WHERE sender_id = ? OR receiver_id = ?
			GROUP BY counterpart_id
		) c
		JOIN ` + r.t("inbox") + ` m ON m.inbox_id = c.last_id
		JOIN ` + r.t("user") + ` u ON u.user_id = c.counterpart_id
		ORDER BY c.last_id DESC
		LIMIT ? OFFSET ?`

	rows, err := r.db.QueryContext(ctx, query, userID, userID, userID, userID, limit+1, offset)
	if err != nil {
		return nil, false, fmt.Errorf("list conversations: %w", err)
	}
	defer rows.Close()

	items := make([]Conversation, 0, limit)
	for rows.Next() {
		var item Conversation
		if err := rows.Scan(&item.UserID, &item.Username, &item.AvatarKey,
			&item.LastSender, &item.LastExcerpt, &item.LastAt, &item.Unread); err != nil {
			return nil, false, fmt.Errorf("scan conversation: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, false, fmt.Errorf("iterate conversations: %w", err)
	}
	hasMore := len(items) > limit
	if hasMore {
		items = items[:limit]
	}
	return items, hasMore, nil
}

func (r *MySQLRepository) ListMessages(ctx context.Context, userID, counterpartID int64, limit, offset int) ([]Message, bool, error) {
	query := `
		SELECT i.inbox_id, i.sender_id, s.username, i.receiver_id, i.content_markdown,
			i.content_rendered, i.read_at IS NOT NULL, i.created_at
		FROM ` + r.t("inbox") + ` i
		JOIN ` + r.t("user") + ` s ON s.user_id = i.sender_id
		WHERE (i.sender_id = ? AND i.receiver_id = ?) OR (i.sender_id = ? AND i.receiver_id = ?)
		ORDER BY i.inbox_id DESC
		LIMIT ? OFFSET ?`
	rows, err := r.db.QueryContext(ctx, query, userID, counterpartID, counterpartID, userID, limit+1, offset)
	if err != nil {
		return nil, false, fmt.Errorf("list messages: %w", err)
	}
	defer rows.Close()

	items := make([]Message, 0, limit)
	for rows.Next() {
		message, err := scanMessage(rows)
		if err != nil {
			return nil, false, err
		}
		items = append(items, message)
	}
	if err := rows.Err(); err != nil {
		return nil, false, fmt.Errorf("iterate messages: %w", err)
	}
	hasMore := len(items) > limit
	if hasMore {
		items = items[:limit]
	}
	return items, hasMore, nil
}

func (r *MySQLRepository) MarkConversationRead(ctx context.Context, userID, counterpartID int64) (int, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("begin mark read: %w", err)
	}
	defer tx.Rollback()

	result, err := tx.ExecContext(ctx,
		"UPDATE "+r.t("inbox")+" SET read_at = CURRENT_TIMESTAMP(6)"+
			" WHERE receiver_id = ? AND sender_id = ? AND read_at IS NULL", userID, counterpartID)
	if err != nil {
		return 0, fmt.Errorf("mark conversation read: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("mark conversation read rows: %w", err)
	}
	if err := refreshUnread(ctx, tx, r.t("inbox"), r.t("user"), userID); err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit mark read: %w", err)
	}
	return int(affected), nil
}

func (r *MySQLRepository) UnreadCount(ctx context.Context, userID int64) (int, error) {
	var count int
	if err := r.db.QueryRowContext(ctx,
		"SELECT inbox_unread FROM "+r.t("user")+" WHERE user_id = ? LIMIT 1", userID).Scan(&count); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, nil
		}
		return 0, fmt.Errorf("load inbox unread: %w", err)
	}
	return count, nil
}

// refreshUnread 按实际未读条数重算收件人的未读数，避免并发下的加减漂移。
func refreshUnread(ctx context.Context, tx *sql.Tx, inboxTable, userTable string, userID int64) error {
	if _, err := tx.ExecContext(ctx,
		"UPDATE "+userTable+" SET inbox_unread ="+
			" (SELECT COUNT(*) FROM "+inboxTable+" WHERE receiver_id = ? AND read_at IS NULL)"+
			" WHERE user_id = ?", userID, userID); err != nil {
		return fmt.Errorf("refresh inbox unread: %w", err)
	}
	return nil
}

func scanMessageRow(scanner interface{ Scan(...any) error }) (Message, error) {
	message, err := scanMessage(scanner)
	if errors.Is(err, sql.ErrNoRows) {
		return Message{}, ErrRecipientNotFound
	}
	return message, err
}

func scanMessage(scanner interface{ Scan(...any) error }) (Message, error) {
	var message Message
	if err := scanner.Scan(&message.ID, &message.SenderID, &message.SenderName, &message.ReceiverID,
		&message.ContentMarkdown, &message.ContentRendered, &message.Read, &message.CreatedAt); err != nil {
		return Message{}, fmt.Errorf("scan message: %w", err)
	}
	return message, nil
}

func (r *MySQLRepository) t(name string) string {
	return r.table + name
}
