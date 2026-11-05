package engagement

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	mysqlDriver "github.com/go-sql-driver/mysql"
)

type MySQLRepository struct {
	db *sql.DB
}

func NewMySQLRepository(db *sql.DB) *MySQLRepository {
	return &MySQLRepository{db: db}
}

// targetTable 用白名单把类型映射到表名，任何 SQL 拼接都只使用这些常量。
func targetTable(kind string) (table, idColumn string, ok bool) {
	switch kind {
	case TypeQuestion:
		return "vx_question", "question_id", true
	case TypeAnswer:
		return "vx_answer", "answer_id", true
	case TypeArticle:
		return "vx_article", "article_id", true
	case TypeComment:
		return "vx_comment", "comment_id", true
	case TypeUser:
		return "vx_user", "user_id", true
	case TypeTopic:
		return "vx_topic", "topic_id", true
	}
	return "", "", false
}

func (r *MySQLRepository) Vote(ctx context.Context, userID int64, kind string, targetID int64, voteType string) (VoteSummary, error) {
	table, idColumn, ok := targetTable(kind)
	if !ok || kind == TypeUser || kind == TypeTopic {
		return VoteSummary{}, ErrNotFound
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return VoteSummary{}, fmt.Errorf("begin vote: %w", err)
	}
	defer tx.Rollback()

	var voteCount int
	if err := tx.QueryRowContext(ctx,
		"SELECT vote_count FROM "+table+" WHERE "+idColumn+" = ? AND deleted_at IS NULL FOR UPDATE", targetID,
	).Scan(&voteCount); errors.Is(err, sql.ErrNoRows) {
		return VoteSummary{}, ErrNotFound
	} else if err != nil {
		return VoteSummary{}, fmt.Errorf("lock vote target: %w", err)
	}

	var current string
	err = tx.QueryRowContext(ctx,
		"SELECT type FROM vx_vote WHERE user_id = ? AND votable_type = ? AND votable_id = ?",
		userID, kind, targetID).Scan(&current)
	if errors.Is(err, sql.ErrNoRows) {
		if _, err := tx.ExecContext(ctx,
			"INSERT INTO vx_vote (user_id, votable_type, votable_id, type) VALUES (?, ?, ?, ?)",
			userID, kind, targetID, voteType); err != nil {
			return VoteSummary{}, fmt.Errorf("insert vote: %w", err)
		}
		if err := adjustVoteCounters(ctx, tx, table, idColumn, targetID, voteType, 1); err != nil {
			return VoteSummary{}, err
		}
	} else if err != nil {
		return VoteSummary{}, fmt.Errorf("load vote: %w", err)
	} else if current != voteType {
		if _, err := tx.ExecContext(ctx, `
			UPDATE vx_vote SET type = ?, created_at = CURRENT_TIMESTAMP(6)
			WHERE user_id = ? AND votable_type = ? AND votable_id = ?`,
			voteType, userID, kind, targetID); err != nil {
			return VoteSummary{}, fmt.Errorf("update vote: %w", err)
		}
		// 先撤销旧票，再计入新票，两步在同一事务内完成。
		if err := adjustVoteCounters(ctx, tx, table, idColumn, targetID, current, -1); err != nil {
			return VoteSummary{}, err
		}
		if err := adjustVoteCounters(ctx, tx, table, idColumn, targetID, voteType, 1); err != nil {
			return VoteSummary{}, err
		}
	}
	if err := tx.Commit(); err != nil {
		return VoteSummary{}, fmt.Errorf("commit vote: %w", err)
	}
	return r.voteSummary(ctx, kind, targetID, userID, voteType)
}

func (r *MySQLRepository) RemoveVote(ctx context.Context, userID int64, kind string, targetID int64) (VoteSummary, error) {
	table, idColumn, ok := targetTable(kind)
	if !ok || kind == TypeUser || kind == TypeTopic {
		return VoteSummary{}, ErrNotFound
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return VoteSummary{}, fmt.Errorf("begin remove vote: %w", err)
	}
	defer tx.Rollback()

	var voteCount int
	if err := tx.QueryRowContext(ctx,
		"SELECT vote_count FROM "+table+" WHERE "+idColumn+" = ? AND deleted_at IS NULL FOR UPDATE", targetID,
	).Scan(&voteCount); errors.Is(err, sql.ErrNoRows) {
		return VoteSummary{}, ErrNotFound
	} else if err != nil {
		return VoteSummary{}, fmt.Errorf("lock vote target: %w", err)
	}
	var current string
	err = tx.QueryRowContext(ctx,
		"SELECT type FROM vx_vote WHERE user_id = ? AND votable_type = ? AND votable_id = ?",
		userID, kind, targetID).Scan(&current)
	if errors.Is(err, sql.ErrNoRows) {
		// 未投票时删除操作保持幂等。
		if err := tx.Commit(); err != nil {
			return VoteSummary{}, fmt.Errorf("commit remove vote: %w", err)
		}
		return r.voteSummary(ctx, kind, targetID, userID, "")
	}
	if err != nil {
		return VoteSummary{}, fmt.Errorf("load vote: %w", err)
	}
	if _, err := tx.ExecContext(ctx,
		"DELETE FROM vx_vote WHERE user_id = ? AND votable_type = ? AND votable_id = ?",
		userID, kind, targetID); err != nil {
		return VoteSummary{}, fmt.Errorf("delete vote: %w", err)
	}
	if err := adjustVoteCounters(ctx, tx, table, idColumn, targetID, current, -1); err != nil {
		return VoteSummary{}, err
	}
	if err := tx.Commit(); err != nil {
		return VoteSummary{}, fmt.Errorf("commit remove vote: %w", err)
	}
	return r.voteSummary(ctx, kind, targetID, userID, "")
}

func (r *MySQLRepository) VoteState(ctx context.Context, userID int64, kind string, targetID int64) (VoteSummary, error) {
	myVote, err := r.MyVote(ctx, userID, kind, targetID)
	if err != nil {
		return VoteSummary{}, err
	}
	return r.voteSummary(ctx, kind, targetID, userID, myVote)
}

func (r *MySQLRepository) MyVote(ctx context.Context, userID int64, kind string, targetID int64) (string, error) {
	if userID <= 0 {
		return "", nil
	}
	var voteType string
	err := r.db.QueryRowContext(ctx,
		"SELECT type FROM vx_vote WHERE user_id = ? AND votable_type = ? AND votable_id = ?",
		userID, kind, targetID).Scan(&voteType)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("load my vote: %w", err)
	}
	return voteType, nil
}

func (r *MySQLRepository) voteSummary(ctx context.Context, kind string, targetID, userID int64, myVote string) (VoteSummary, error) {
	table, idColumn, ok := targetTable(kind)
	if !ok {
		return VoteSummary{}, ErrNotFound
	}
	var summary VoteSummary
	if err := r.db.QueryRowContext(ctx,
		"SELECT vote_count, vote_up_count, vote_down_count FROM "+table+" WHERE "+idColumn+" = ?", targetID,
	).Scan(&summary.VoteCount, &summary.VoteUpCount, &summary.VoteDownCount); err != nil {
		return VoteSummary{}, fmt.Errorf("load vote summary: %w", err)
	}
	summary.MyVote = myVote
	return summary, nil
}

func adjustVoteCounters(ctx context.Context, tx *sql.Tx, table, idColumn string, targetID int64, voteType string, sign int) error {
	// sign 为 +1 表示新增/切换为新票，-1 表示撤销旧票。
	var voteDelta, upDelta, downDelta int
	if voteType == "up" {
		voteDelta, upDelta = sign, sign
	} else {
		voteDelta, downDelta = -sign, sign
	}
	query := "UPDATE " + table +
		" SET vote_count = vote_count + ?," +
		" vote_up_count = GREATEST(vote_up_count + ?, 0)," +
		" vote_down_count = GREATEST(vote_down_count + ?, 0)" +
		" WHERE " + idColumn + " = ?"
	if _, err := tx.ExecContext(ctx, query, voteDelta, upDelta, downDelta, targetID); err != nil {
		return fmt.Errorf("adjust vote counters: %w", err)
	}
	return nil
}

func (r *MySQLRepository) Follow(ctx context.Context, userID int64, kind string, targetID int64) (FollowSummary, error) {
	table, idColumn, ok := targetTable(kind)
	if !ok {
		return FollowSummary{}, ErrNotFound
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return FollowSummary{}, fmt.Errorf("begin follow: %w", err)
	}
	defer tx.Rollback()

	var followerCount int
	if err := tx.QueryRowContext(ctx,
		"SELECT follower_count FROM "+table+" WHERE "+idColumn+" = ? FOR UPDATE", targetID,
	).Scan(&followerCount); errors.Is(err, sql.ErrNoRows) {
		return FollowSummary{}, ErrNotFound
	} else if err != nil {
		return FollowSummary{}, fmt.Errorf("lock follow target: %w", err)
	}

	result, err := tx.ExecContext(ctx,
		"INSERT IGNORE INTO vx_follow (user_id, followable_type, followable_id) VALUES (?, ?, ?)",
		userID, kind, targetID)
	if err != nil {
		return FollowSummary{}, fmt.Errorf("insert follow: %w", err)
	}
	inserted, err := result.RowsAffected()
	if err != nil {
		return FollowSummary{}, fmt.Errorf("follow rows: %w", err)
	}
	if inserted > 0 {
		if _, err := tx.ExecContext(ctx,
			"UPDATE "+table+" SET follower_count = follower_count + 1 WHERE "+idColumn+" = ?", targetID); err != nil {
			return FollowSummary{}, fmt.Errorf("increment follower count: %w", err)
		}
		if err := adjustUserFollowCounters(ctx, tx, userID, kind, 1); err != nil {
			return FollowSummary{}, err
		}
	}
	if err := tx.Commit(); err != nil {
		return FollowSummary{}, fmt.Errorf("commit follow: %w", err)
	}
	return r.followSummary(ctx, kind, targetID, userID)
}

func (r *MySQLRepository) Unfollow(ctx context.Context, userID int64, kind string, targetID int64) (FollowSummary, error) {
	table, idColumn, ok := targetTable(kind)
	if !ok {
		return FollowSummary{}, ErrNotFound
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return FollowSummary{}, fmt.Errorf("begin unfollow: %w", err)
	}
	defer tx.Rollback()

	var followerCount int
	if err := tx.QueryRowContext(ctx,
		"SELECT follower_count FROM "+table+" WHERE "+idColumn+" = ? FOR UPDATE", targetID,
	).Scan(&followerCount); errors.Is(err, sql.ErrNoRows) {
		return FollowSummary{}, ErrNotFound
	} else if err != nil {
		return FollowSummary{}, fmt.Errorf("lock follow target: %w", err)
	}
	result, err := tx.ExecContext(ctx,
		"DELETE FROM vx_follow WHERE user_id = ? AND followable_type = ? AND followable_id = ?",
		userID, kind, targetID)
	if err != nil {
		return FollowSummary{}, fmt.Errorf("delete follow: %w", err)
	}
	removed, err := result.RowsAffected()
	if err != nil {
		return FollowSummary{}, fmt.Errorf("unfollow rows: %w", err)
	}
	if removed > 0 {
		if _, err := tx.ExecContext(ctx,
			"UPDATE "+table+" SET follower_count = GREATEST(follower_count, 1) - 1 WHERE "+idColumn+" = ?", targetID); err != nil {
			return FollowSummary{}, fmt.Errorf("decrement follower count: %w", err)
		}
		if err := adjustUserFollowCounters(ctx, tx, userID, kind, -1); err != nil {
			return FollowSummary{}, err
		}
	}
	if err := tx.Commit(); err != nil {
		return FollowSummary{}, fmt.Errorf("commit unfollow: %w", err)
	}
	return r.followSummary(ctx, kind, targetID, userID)
}

func (r *MySQLRepository) FollowState(ctx context.Context, userID int64, kind string, targetID int64) (FollowSummary, error) {
	return r.followSummary(ctx, kind, targetID, userID)
}

func (r *MySQLRepository) IsFollowing(ctx context.Context, userID int64, kind string, targetID int64) (bool, error) {
	if userID <= 0 {
		return false, nil
	}
	var exists int
	err := r.db.QueryRowContext(ctx,
		"SELECT 1 FROM vx_follow WHERE user_id = ? AND followable_type = ? AND followable_id = ? LIMIT 1",
		userID, kind, targetID).Scan(&exists)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("check follow: %w", err)
	}
	return true, nil
}

func (r *MySQLRepository) followSummary(ctx context.Context, kind string, targetID, userID int64) (FollowSummary, error) {
	table, idColumn, ok := targetTable(kind)
	if !ok {
		return FollowSummary{}, ErrNotFound
	}
	var summary FollowSummary
	if err := r.db.QueryRowContext(ctx,
		"SELECT follower_count FROM "+table+" WHERE "+idColumn+" = ?", targetID).Scan(&summary.FollowerCount); err != nil {
		return FollowSummary{}, fmt.Errorf("load follower count: %w", err)
	}
	following, err := r.IsFollowing(ctx, userID, kind, targetID)
	if err != nil {
		return FollowSummary{}, err
	}
	summary.Following = following
	return summary, nil
}

func adjustUserFollowCounters(ctx context.Context, tx *sql.Tx, userID int64, kind string, sign int) error {
	column := ""
	switch kind {
	case TypeUser:
		column = "followee_count"
	case TypeQuestion:
		column = "following_question_count"
	case TypeArticle:
		column = "following_article_count"
	case TypeTopic:
		column = "following_topic_count"
	}
	if column == "" {
		return nil
	}
	query := "UPDATE vx_user SET " + column + " = " + column + " + ? WHERE user_id = ?"
	if sign < 0 {
		query = "UPDATE vx_user SET " + column + " = GREATEST(" + column + ", 1) - 1 WHERE user_id = ?"
		if _, err := tx.ExecContext(ctx, query, userID); err != nil {
			return fmt.Errorf("decrement user follow counter: %w", err)
		}
		return nil
	}
	if _, err := tx.ExecContext(ctx, query, sign, userID); err != nil {
		return fmt.Errorf("increment user follow counter: %w", err)
	}
	return nil
}

func (r *MySQLRepository) CreateReport(ctx context.Context, userID int64, kind string, targetID int64, reason string) (Report, error) {
	table, idColumn, ok := targetTable(kind)
	if !ok {
		return Report{}, ErrNotFound
	}
	var exists int
	err := r.db.QueryRowContext(ctx,
		"SELECT 1 FROM "+table+" WHERE "+idColumn+" = ? LIMIT 1", targetID).Scan(&exists)
	if errors.Is(err, sql.ErrNoRows) {
		return Report{}, ErrNotFound
	}
	if err != nil {
		return Report{}, fmt.Errorf("check report target: %w", err)
	}
	result, err := r.db.ExecContext(ctx,
		"INSERT INTO vx_report (reportable_id, reportable_type, user_id, reason) VALUES (?, ?, ?, ?)",
		targetID, kind, userID, reason)
	if err != nil {
		var mysqlErr *mysqlDriver.MySQLError
		if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
			return Report{}, ErrDuplicateReport
		}
		return Report{}, fmt.Errorf("insert report: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return Report{}, fmt.Errorf("read report id: %w", err)
	}
	return r.findReport(ctx, id)
}

func (r *MySQLRepository) ListReports(ctx context.Context, status string, limit, offset int) ([]Report, bool, error) {
	query := `
		SELECT r.report_id, r.reportable_id, r.reportable_type, r.user_id, u.username,
			r.reason, r.status, r.created_at
		FROM vx_report r
		JOIN vx_user u ON u.user_id = r.user_id`
	args := make([]any, 0, 3)
	if status != "" {
		query += " WHERE r.status = ?"
		args = append(args, status)
	}
	query += " ORDER BY r.created_at DESC, r.report_id DESC LIMIT ? OFFSET ?"
	args = append(args, limit+1, offset)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, false, fmt.Errorf("list reports: %w", err)
	}
	defer rows.Close()
	var items []Report
	for rows.Next() {
		var report Report
		if err := rows.Scan(&report.ID, &report.ReportableID, &report.ReportableType, &report.UserID,
			&report.Username, &report.Reason, &report.Status, &report.CreatedAt); err != nil {
			return nil, false, fmt.Errorf("scan report: %w", err)
		}
		items = append(items, report)
	}
	if err := rows.Err(); err != nil {
		return nil, false, fmt.Errorf("iterate reports: %w", err)
	}
	hasMore := len(items) > limit
	if hasMore {
		items = items[:limit]
	}
	return items, hasMore, nil
}

func (r *MySQLRepository) ResolveReport(ctx context.Context, id, handlerID int64, status string) (Report, error) {
	result, err := r.db.ExecContext(ctx, `
		UPDATE vx_report SET status = ?, handled_by = ?, handled_at = CURRENT_TIMESTAMP(6)
		WHERE report_id = ?`, status, handlerID, id)
	if err != nil {
		return Report{}, fmt.Errorf("resolve report: %w", err)
	}
	if rows, err := result.RowsAffected(); err != nil {
		return Report{}, fmt.Errorf("resolve report rows: %w", err)
	} else if rows == 0 {
		return Report{}, ErrNotFound
	}
	return r.findReport(ctx, id)
}

func (r *MySQLRepository) findReport(ctx context.Context, id int64) (Report, error) {
	var report Report
	err := r.db.QueryRowContext(ctx, `
		SELECT r.report_id, r.reportable_id, r.reportable_type, r.user_id, u.username,
			r.reason, r.status, r.created_at
		FROM vx_report r
		JOIN vx_user u ON u.user_id = r.user_id
		WHERE r.report_id = ? LIMIT 1`, id).
		Scan(&report.ID, &report.ReportableID, &report.ReportableType, &report.UserID,
			&report.Username, &report.Reason, &report.Status, &report.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return Report{}, ErrNotFound
	}
	if err != nil {
		return Report{}, fmt.Errorf("find report: %w", err)
	}
	return report, nil
}

func (r *MySQLRepository) CreateNotification(ctx context.Context, event NotificationEvent) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin notification: %w", err)
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO vx_notification
			(receiver_id, sender_id, type, article_id, question_id, answer_id, comment_id, reply_id, content_deleted)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		event.ReceiverID, event.SenderID, event.Type, event.ArticleID,
		event.QuestionID, event.AnswerID, event.CommentID, event.ReplyID,
		nullableText(event.ContentDeleted)); err != nil {
		return fmt.Errorf("insert notification: %w", err)
	}
	if _, err := tx.ExecContext(ctx,
		"UPDATE vx_user SET notification_unread = notification_unread + 1 WHERE user_id = ?", event.ReceiverID); err != nil {
		return fmt.Errorf("increment notification unread: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit notification: %w", err)
	}
	return nil
}

func (r *MySQLRepository) ListNotifications(ctx context.Context, receiverID int64, limit, offset int) ([]Notification, bool, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT n.notification_id, n.type, n.sender_id, COALESCE(u.username, ''),
			n.article_id, n.question_id, n.answer_id, n.comment_id, n.reply_id,
			COALESCE(n.content_deleted, ''), n.read_at, n.created_at
		FROM vx_notification n
		LEFT JOIN vx_user u ON u.user_id = n.sender_id
		WHERE n.receiver_id = ?
		ORDER BY n.notification_id DESC
		LIMIT ? OFFSET ?`, receiverID, limit+1, offset)
	if err != nil {
		return nil, false, fmt.Errorf("list notifications: %w", err)
	}
	defer rows.Close()
	var items []Notification
	for rows.Next() {
		var (
			item   Notification
			readAt sql.NullTime
		)
		if err := rows.Scan(&item.ID, &item.Type, &item.SenderID, &item.SenderName,
			&item.ArticleID, &item.QuestionID, &item.AnswerID, &item.CommentID, &item.ReplyID,
			&item.ContentDeleted, &readAt, &item.CreatedAt); err != nil {
			return nil, false, fmt.Errorf("scan notification: %w", err)
		}
		item.Read = readAt.Valid
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, false, fmt.Errorf("iterate notifications: %w", err)
	}
	hasMore := len(items) > limit
	if hasMore {
		items = items[:limit]
	}
	return items, hasMore, nil
}

func (r *MySQLRepository) UnreadNotificationCount(ctx context.Context, receiverID int64) (int, error) {
	var count int
	if err := r.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM vx_notification WHERE receiver_id = ? AND read_at IS NULL", receiverID).Scan(&count); err != nil {
		return 0, fmt.Errorf("count unread notifications: %w", err)
	}
	return count, nil
}

func (r *MySQLRepository) MarkNotificationsRead(ctx context.Context, receiverID int64, ids []int64) (int, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	placeholders, args := placeholders(ids)
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("begin mark read: %w", err)
	}
	defer tx.Rollback()

	var unread int
	if err := tx.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM vx_notification
		WHERE receiver_id = ? AND read_at IS NULL AND notification_id IN (`+placeholders+`)`,
		append([]any{receiverID}, args...)...).Scan(&unread); err != nil {
		return 0, fmt.Errorf("count marking notifications: %w", err)
	}
	// receiver_id 条件必须保留，避免通过批量接口修改他人通知。
	result, err := tx.ExecContext(ctx, `
		UPDATE vx_notification SET read_at = CURRENT_TIMESTAMP(6)
		WHERE receiver_id = ? AND read_at IS NULL AND notification_id IN (`+placeholders+`)`,
		append([]any{receiverID}, args...)...)
	if err != nil {
		return 0, fmt.Errorf("mark notifications read: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("mark read rows: %w", err)
	}
	if unread > 0 {
		if _, err := tx.ExecContext(ctx, `
			UPDATE vx_user SET notification_unread = GREATEST(notification_unread, ?) - ? WHERE user_id = ?`,
			unread, unread, receiverID); err != nil {
			return 0, fmt.Errorf("decrement notification unread: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit mark read: %w", err)
	}
	return int(affected), nil
}

func (r *MySQLRepository) DeleteNotifications(ctx context.Context, receiverID int64, ids []int64) (int, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	placeholders, args := placeholders(ids)
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("begin delete notifications: %w", err)
	}
	defer tx.Rollback()

	var unread int
	if err := tx.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM vx_notification
		WHERE receiver_id = ? AND read_at IS NULL AND notification_id IN (`+placeholders+`)`,
		append([]any{receiverID}, args...)...).Scan(&unread); err != nil {
		return 0, fmt.Errorf("count deleting notifications: %w", err)
	}
	result, err := tx.ExecContext(ctx, `
		DELETE FROM vx_notification
		WHERE receiver_id = ? AND notification_id IN (`+placeholders+`)`,
		append([]any{receiverID}, args...)...)
	if err != nil {
		return 0, fmt.Errorf("delete notifications: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("delete notification rows: %w", err)
	}
	if unread > 0 {
		if _, err := tx.ExecContext(ctx, `
			UPDATE vx_user SET notification_unread = GREATEST(notification_unread, ?) - ? WHERE user_id = ?`,
			unread, unread, receiverID); err != nil {
			return 0, fmt.Errorf("decrement notification unread: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit delete notifications: %w", err)
	}
	return int(affected), nil
}

func nullableText(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}

func placeholders(ids []int64) (string, []any) {
	parts := make([]string, len(ids))
	args := make([]any, len(ids))
	for i, id := range ids {
		parts[i] = "?"
		args[i] = id
	}
	return strings.Join(parts, ","), args
}

// FollowStates 批量查询关注状态，供列表页一次取回，避免逐条请求。
func (r *MySQLRepository) FollowStates(ctx context.Context, userID int64, kind string, ids []int64) (map[int64]bool, error) {
	states := make(map[int64]bool, len(ids))
	if userID <= 0 || len(ids) == 0 {
		return states, nil
	}
	placeholders, args := placeholders(ids)
	args = append([]any{userID, kind}, args...)
	rows, err := r.db.QueryContext(ctx,
		"SELECT followable_id FROM vx_follow WHERE user_id = ? AND followable_type = ? AND followable_id IN ("+placeholders+")",
		args...)
	if err != nil {
		return nil, fmt.Errorf("list follow states: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan follow state: %w", err)
		}
		states[id] = true
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate follow states: %w", err)
	}
	return states, nil
}
