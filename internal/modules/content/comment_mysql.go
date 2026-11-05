package content

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/voidxo/voidxo/internal/platform/outbox"
)

func (r *MySQLRepository) ListComments(ctx context.Context, kind string, targetID int64, limit, offset int) ([]Comment, bool, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT c.comment_id, c.commentable_id, c.commentable_type, c.root_id, c.user_id, u.username,
			COALESCE(u.avatar_key, ''),
			c.content, c.reply_count, c.vote_count, c.created_at, c.updated_at
		FROM vx_comment c
		JOIN vx_user u ON u.user_id = c.user_id
		WHERE c.commentable_type = ? AND c.commentable_id = ? AND c.deleted_at IS NULL
		ORDER BY c.created_at ASC, c.comment_id ASC
		LIMIT ? OFFSET ?`, kind, targetID, limit+1, offset)
	if err != nil {
		return nil, false, fmt.Errorf("list comments: %w", err)
	}
	defer rows.Close()
	var items []Comment
	for rows.Next() {
		comment, err := scanComment(rows)
		if err != nil {
			return nil, false, err
		}
		items = append(items, comment)
	}
	if err := rows.Err(); err != nil {
		return nil, false, fmt.Errorf("iterate comments: %w", err)
	}
	hasMore := len(items) > limit
	if hasMore {
		items = items[:limit]
	}
	return items, hasMore, nil
}

func (r *MySQLRepository) FindComment(ctx context.Context, id int64) (Comment, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT c.comment_id, c.commentable_id, c.commentable_type, c.root_id, c.user_id, u.username,
			COALESCE(u.avatar_key, ''),
			c.content, c.reply_count, c.vote_count, c.created_at, c.updated_at
		FROM vx_comment c
		JOIN vx_user u ON u.user_id = c.user_id
		WHERE c.comment_id = ? AND c.deleted_at IS NULL LIMIT 1`, id)
	comment, err := scanComment(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Comment{}, ErrNotFound
	}
	if err != nil {
		return Comment{}, err
	}
	return comment, nil
}

func (r *MySQLRepository) CreateComment(ctx context.Context, kind string, targetID, userID int64, content string, rootID *int64) (Comment, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return Comment{}, fmt.Errorf("begin create comment: %w", err)
	}
	defer tx.Rollback()

	var root any
	if rootID != nil {
		root = *rootID
	}
	result, err := tx.ExecContext(ctx, `
		INSERT INTO vx_comment (commentable_id, commentable_type, root_id, user_id, content)
		VALUES (?, ?, ?, ?, ?)`, targetID, kind, root, userID, content)
	if err != nil {
		return Comment{}, fmt.Errorf("insert comment: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return Comment{}, fmt.Errorf("read comment id: %w", err)
	}
	if err := bumpCommentCount(ctx, tx, kind, targetID, 1); err != nil {
		return Comment{}, err
	}
	if rootID != nil {
		if _, err := tx.ExecContext(ctx,
			"UPDATE vx_comment SET reply_count = reply_count + 1 WHERE comment_id = ? AND deleted_at IS NULL", *rootID); err != nil {
			return Comment{}, fmt.Errorf("increment reply count: %w", err)
		}
	}
	if _, err := tx.ExecContext(ctx, "UPDATE vx_user SET comment_count = comment_count + 1 WHERE user_id = ?", userID); err != nil {
		return Comment{}, fmt.Errorf("increment user comment count: %w", err)
	}
	if err := outbox.Enqueue(ctx, tx, outbox.EventCommentCreated, id, map[string]any{
		"comment_id": id, "commentable_type": kind, "commentable_id": targetID, "user_id": userID,
	}); err != nil {
		return Comment{}, err
	}
	if err := tx.Commit(); err != nil {
		return Comment{}, fmt.Errorf("commit create comment: %w", err)
	}
	return r.FindComment(ctx, id)
}

func (r *MySQLRepository) UpdateComment(ctx context.Context, id int64, content string) (Comment, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return Comment{}, fmt.Errorf("begin update comment: %w", err)
	}
	defer tx.Rollback()

	result, err := tx.ExecContext(ctx, `
		UPDATE vx_comment SET content = ?, updated_at = CURRENT_TIMESTAMP(6)
		WHERE comment_id = ? AND deleted_at IS NULL`, content, id)
	if err != nil {
		return Comment{}, fmt.Errorf("update comment: %w", err)
	}
	// 内容未变化时 RowsAffected 为 0，不能据此判断记录不存在。
	if rows, err := result.RowsAffected(); err != nil {
		return Comment{}, fmt.Errorf("update comment rows: %w", err)
	} else if rows == 0 {
		var exists int
		err := tx.QueryRowContext(ctx,
			"SELECT 1 FROM vx_comment WHERE comment_id = ? AND deleted_at IS NULL LIMIT 1", id).Scan(&exists)
		if errors.Is(err, sql.ErrNoRows) {
			return Comment{}, ErrNotFound
		}
		if err != nil {
			return Comment{}, fmt.Errorf("check comment: %w", err)
		}
	}
	if err := outbox.Enqueue(ctx, tx, outbox.EventCommentUpdated, id, map[string]any{"comment_id": id}); err != nil {
		return Comment{}, err
	}
	if err := tx.Commit(); err != nil {
		return Comment{}, fmt.Errorf("commit update comment: %w", err)
	}
	return r.FindComment(ctx, id)
}

func (r *MySQLRepository) DeleteComment(ctx context.Context, id int64) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin delete comment: %w", err)
	}
	defer tx.Rollback()

	var (
		kind     string
		targetID int64
		rootID   sql.NullInt64
		userID   int64
		replies  int
	)
	if err := tx.QueryRowContext(ctx, `
		SELECT commentable_type, commentable_id, root_id, user_id, reply_count
		FROM vx_comment WHERE comment_id = ? AND deleted_at IS NULL FOR UPDATE`, id,
	).Scan(&kind, &targetID, &rootID, &userID, &replies); errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	} else if err != nil {
		return fmt.Errorf("lock comment: %w", err)
	}

	removed := 1
	if !rootID.Valid {
		// 顶层评论连带回复一起删除，避免出现孤立回复。
		removed += replies
		if _, err := tx.ExecContext(ctx,
			"UPDATE vx_comment SET deleted_at = CURRENT_TIMESTAMP(6) WHERE root_id = ? AND deleted_at IS NULL", id); err != nil {
			return fmt.Errorf("soft delete replies: %w", err)
		}
	}
	if _, err := tx.ExecContext(ctx,
		"UPDATE vx_comment SET deleted_at = CURRENT_TIMESTAMP(6) WHERE comment_id = ?", id); err != nil {
		return fmt.Errorf("soft delete comment: %w", err)
	}
	if rootID.Valid {
		if _, err := tx.ExecContext(ctx,
			"UPDATE vx_comment SET reply_count = GREATEST(reply_count, 1) - 1 WHERE comment_id = ?", rootID.Int64); err != nil {
			return fmt.Errorf("decrement reply count: %w", err)
		}
	} else if err := bumpCommentCount(ctx, tx, kind, targetID, -removed); err != nil {
		return err
	}

	// 被删除评论的作者可能不同于当前操作者（管理员删除），逐作者回退计数。
	// 按作者聚合被删除的条数，同一作者的多条回复一次性回退。
	rows, err := tx.QueryContext(ctx,
		"SELECT user_id, COUNT(*) FROM vx_comment WHERE comment_id = ? OR root_id = ? GROUP BY user_id", id, id)
	if err != nil {
		return fmt.Errorf("load comment authors: %w", err)
	}
	type authorTotal struct {
		userID int64
		total  int
	}
	var authors []authorTotal
	for rows.Next() {
		var author authorTotal
		if err := rows.Scan(&author.userID, &author.total); err != nil {
			rows.Close()
			return fmt.Errorf("scan comment author: %w", err)
		}
		authors = append(authors, author)
	}
	rows.Close()
	for _, author := range authors {
		if _, err := tx.ExecContext(ctx,
			"UPDATE vx_user SET comment_count = GREATEST(comment_count, ?) - ? WHERE user_id = ?",
			author.total, author.total, author.userID); err != nil {
			return fmt.Errorf("decrement user comment count: %w", err)
		}
	}
	if _, err := tx.ExecContext(ctx,
		"DELETE FROM vx_vote WHERE votable_type = 'comment' AND votable_id = ?", id); err != nil {
		return fmt.Errorf("delete comment votes: %w", err)
	}
	if err := outbox.Enqueue(ctx, tx, outbox.EventCommentDeleted, id, map[string]any{
		"comment_id": id, "commentable_type": kind, "commentable_id": targetID,
	}); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit delete comment: %w", err)
	}
	return nil
}

func (r *MySQLRepository) commentExists(ctx context.Context, id int64) (bool, error) {
	var exists int
	err := r.db.QueryRowContext(ctx,
		"SELECT 1 FROM vx_comment WHERE comment_id = ? AND deleted_at IS NULL LIMIT 1", id).Scan(&exists)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("check comment: %w", err)
	}
	return true, nil
}

// bumpCommentCount 更新评论目标的评论计数，负值时不会低于 0。
func bumpCommentCount(ctx context.Context, tx *sql.Tx, kind string, targetID int64, delta int) error {
	var query string
	switch kind {
	case KindArticle:
		query = "UPDATE vx_article SET comment_count = "
	case KindQuestion:
		query = "UPDATE vx_question SET comment_count = "
	case KindAnswer:
		query = "UPDATE vx_answer SET comment_count = "
	default:
		return nil
	}
	if delta > 0 {
		query += "comment_count + ? WHERE " + targetColumn(kind) + " = ? AND deleted_at IS NULL"
		if _, err := tx.ExecContext(ctx, query, delta, targetID); err != nil {
			return fmt.Errorf("increment comment count: %w", err)
		}
		return nil
	}
	query += "GREATEST(comment_count, ?) - ? WHERE " + targetColumn(kind) + " = ?"
	if _, err := tx.ExecContext(ctx, query, -delta, -delta, targetID); err != nil {
		return fmt.Errorf("decrement comment count: %w", err)
	}
	return nil
}

func targetColumn(kind string) string {
	switch kind {
	case KindArticle:
		return "article_id"
	case KindQuestion:
		return "question_id"
	case KindAnswer:
		return "answer_id"
	}
	return "comment_id"
}

func scanComment(scanner interface{ Scan(...any) error }) (Comment, error) {
	var comment Comment
	var root sql.NullInt64
	if err := scanner.Scan(
		&comment.ID, &comment.CommentableID, &comment.CommentableType, &root, &comment.UserID, &comment.Username,
		&comment.AvatarKey, &comment.Content, &comment.ReplyCount, &comment.VoteCount, &comment.CreatedAt, &comment.UpdatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Comment{}, err
		}
		return Comment{}, fmt.Errorf("scan comment: %w", err)
	}
	if root.Valid {
		value := root.Int64
		comment.RootID = &value
	}
	comment.Content = strings.TrimSpace(comment.Content)
	return comment, nil
}

// softDeleteComments 软删目标下的所有评论，并按作者回退评论计数。
// where 由调用方用常量拼装，参数通过 args 传入，不接受外部字符串。
func softDeleteComments(ctx context.Context, tx *sql.Tx, where string, args ...any) (int, error) {
	rows, err := tx.QueryContext(ctx,
		"SELECT user_id, COUNT(*) FROM vx_comment WHERE deleted_at IS NULL AND ("+where+") GROUP BY user_id", args...)
	if err != nil {
		return 0, fmt.Errorf("load comment authors: %w", err)
	}
	type authorTotal struct {
		userID int64
		total  int
	}
	var authors []authorTotal
	for rows.Next() {
		var author authorTotal
		if err := rows.Scan(&author.userID, &author.total); err != nil {
			rows.Close()
			return 0, fmt.Errorf("scan comment author: %w", err)
		}
		authors = append(authors, author)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("iterate comment authors: %w", err)
	}
	for _, author := range authors {
		if _, err := tx.ExecContext(ctx,
			"UPDATE vx_user SET comment_count = GREATEST(comment_count, ?) - ? WHERE user_id = ?",
			author.total, author.total, author.userID); err != nil {
			return 0, fmt.Errorf("decrement comment count: %w", err)
		}
	}
	result, err := tx.ExecContext(ctx,
		"UPDATE vx_comment SET deleted_at = CURRENT_TIMESTAMP(6) WHERE deleted_at IS NULL AND ("+where+")", args...)
	if err != nil {
		return 0, fmt.Errorf("soft delete comments: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("soft delete comment rows: %w", err)
	}
	return int(affected), nil
}
