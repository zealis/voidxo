package content

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/voidxo/voidxo/internal/platform/outbox"
)

func (r *MySQLRepository) ListArticles(ctx context.Context, topicID *int64, sort string, limit, offset int) ([]Article, bool, error) {
	query := `
		SELECT a.article_id, a.user_id, u.username, COALESCE(u.avatar_key, ''), a.title, a.comment_count,
			a.follower_count, a.vote_count, a.created_at, a.updated_at
		FROM vx_article a
		JOIN vx_user u ON u.user_id = a.user_id
		WHERE a.deleted_at IS NULL`
	args := make([]any, 0, 3)
	if topicID != nil {
		query += ` AND EXISTS (
			SELECT 1 FROM vx_topicable t
			WHERE t.topicable_type = 'article' AND t.topicable_id = a.article_id AND t.topic_id = ?
		)`
		args = append(args, *topicID)
	}
	order, ok := articleSorts[sort]
	if !ok {
		order = articleSorts[defaultArticleSort]
	}
	query += " ORDER BY a." + order + " LIMIT ? OFFSET ?"
	args = append(args, limit+1, offset)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, false, fmt.Errorf("list articles: %w", err)
	}
	defer rows.Close()

	var items []Article
	for rows.Next() {
		var article Article
		if err := rows.Scan(&article.ID, &article.UserID, &article.Username, &article.AvatarKey, &article.Title,
			&article.CommentCount, &article.FollowerCount, &article.VoteCount,
			&article.CreatedAt, &article.UpdatedAt); err != nil {
			return nil, false, fmt.Errorf("scan article: %w", err)
		}
		items = append(items, article)
	}
	if err := rows.Err(); err != nil {
		return nil, false, fmt.Errorf("iterate articles: %w", err)
	}
	hasMore := len(items) > limit
	if hasMore {
		items = items[:limit]
	}
	if err := r.attachArticleTopics(ctx, items); err != nil {
		return nil, false, err
	}
	return items, hasMore, nil
}

func (r *MySQLRepository) FindArticle(ctx context.Context, id int64) (Article, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT a.article_id, a.user_id, u.username, COALESCE(u.avatar_key, ''), a.title, a.content_markdown, a.content_rendered,
			a.comment_count, a.follower_count, a.vote_count, a.created_at, a.updated_at
		FROM vx_article a
		JOIN vx_user u ON u.user_id = a.user_id
		WHERE a.article_id = ? AND a.deleted_at IS NULL LIMIT 1`, id)
	var article Article
	if err := row.Scan(&article.ID, &article.UserID, &article.Username, &article.AvatarKey, &article.Title,
		&article.ContentMarkdown, &article.ContentRendered,
		&article.CommentCount, &article.FollowerCount, &article.VoteCount,
		&article.CreatedAt, &article.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Article{}, ErrNotFound
		}
		return Article{}, fmt.Errorf("find article: %w", err)
	}
	topics, err := r.topicsFor(ctx, KindArticle, []int64{article.ID})
	if err != nil {
		return Article{}, err
	}
	article.Topics = topics[article.ID]
	if article.Topics == nil {
		article.Topics = []TopicRef{}
	}
	return article, nil
}

func (r *MySQLRepository) CreateArticle(ctx context.Context, userID int64, title, markdown, rendered string, topicIDs []int64) (Article, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return Article{}, fmt.Errorf("begin create article: %w", err)
	}
	defer tx.Rollback()

	result, err := tx.ExecContext(ctx,
		"INSERT INTO vx_article (user_id, title, content_markdown, content_rendered) VALUES (?, ?, ?, ?)",
		userID, title, markdown, rendered,
	)
	if err != nil {
		return Article{}, fmt.Errorf("insert article: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return Article{}, fmt.Errorf("read article id: %w", err)
	}
	if err := insertTopicables(ctx, tx, KindArticle, id, topicIDs); err != nil {
		return Article{}, err
	}
	if err := addTopicContentCounts(ctx, tx, KindArticle, topicIDs, 1); err != nil {
		return Article{}, err
	}
	if _, err := tx.ExecContext(ctx, "UPDATE vx_user SET article_count = article_count + 1 WHERE user_id = ?", userID); err != nil {
		return Article{}, fmt.Errorf("increment user article count: %w", err)
	}
	if err := outbox.Enqueue(ctx, tx, outbox.EventArticleCreated, id, map[string]any{"article_id": id, "user_id": userID}); err != nil {
		return Article{}, err
	}
	if err := tx.Commit(); err != nil {
		return Article{}, fmt.Errorf("commit create article: %w", err)
	}
	return r.FindArticle(ctx, id)
}

func (r *MySQLRepository) UpdateArticle(ctx context.Context, id int64, title, markdown, rendered string, topicIDs []int64) (Article, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return Article{}, fmt.Errorf("begin update article: %w", err)
	}
	defer tx.Rollback()

	result, err := tx.ExecContext(ctx, `
		UPDATE vx_article
		SET title = ?, content_markdown = ?, content_rendered = ?, updated_at = CURRENT_TIMESTAMP(6)
		WHERE article_id = ? AND deleted_at IS NULL`, title, markdown, rendered, id)
	if err != nil {
		return Article{}, fmt.Errorf("update article: %w", err)
	}
	if rows, err := result.RowsAffected(); err != nil {
		return Article{}, fmt.Errorf("update article rows: %w", err)
	} else if rows == 0 {
		if _, err := tx.ExecContext(ctx, "SELECT article_id FROM vx_article WHERE article_id = ?", id); err != nil {
			return Article{}, ErrNotFound
		}
	}
	if err := replaceTopicables(ctx, tx, KindArticle, id, topicIDs); err != nil {
		return Article{}, err
	}
	if err := outbox.Enqueue(ctx, tx, outbox.EventArticleUpdated, id, map[string]any{"article_id": id}); err != nil {
		return Article{}, err
	}
	if err := tx.Commit(); err != nil {
		return Article{}, fmt.Errorf("commit update article: %w", err)
	}
	return r.FindArticle(ctx, id)
}

func (r *MySQLRepository) DeleteArticle(ctx context.Context, id int64) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin delete article: %w", err)
	}
	defer tx.Rollback()

	var userID int64
	if err := tx.QueryRowContext(ctx,
		"SELECT user_id FROM vx_article WHERE article_id = ? AND deleted_at IS NULL FOR UPDATE", id,
	).Scan(&userID); errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	} else if err != nil {
		return fmt.Errorf("lock article: %w", err)
	}
	topicIDs, err := topicIDsFor(ctx, tx, KindArticle, id)
	if err != nil {
		return err
	}
	if _, err := softDeleteComments(ctx, tx, "commentable_type = 'article' AND commentable_id = ?", id); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx,
		"UPDATE vx_article SET deleted_at = CURRENT_TIMESTAMP(6), comment_count = 0 WHERE article_id = ?", id); err != nil {
		return fmt.Errorf("soft delete article: %w", err)
	}
	if _, err := tx.ExecContext(ctx, "DELETE FROM vx_vote WHERE votable_type = 'article' AND votable_id = ?", id); err != nil {
		return fmt.Errorf("delete article votes: %w", err)
	}
	if _, err := tx.ExecContext(ctx, "DELETE FROM vx_follow WHERE followable_type = 'article' AND followable_id = ?", id); err != nil {
		return fmt.Errorf("delete article follows: %w", err)
	}
	if len(topicIDs) > 0 {
		if err := addTopicContentCounts(ctx, tx, KindArticle, topicIDs, -1); err != nil {
			return err
		}
	}
	if _, err := tx.ExecContext(ctx, "UPDATE vx_user SET article_count = GREATEST(article_count, 1) - 1 WHERE user_id = ?", userID); err != nil {
		return fmt.Errorf("decrement user article count: %w", err)
	}
	if err := outbox.Enqueue(ctx, tx, outbox.EventArticleDeleted, id, map[string]any{"article_id": id, "user_id": userID}); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit delete article: %w", err)
	}
	return nil
}

func (r *MySQLRepository) UpdateQuestion(ctx context.Context, id int64, title, markdown, rendered string, topicIDs []int64) (Question, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return Question{}, fmt.Errorf("begin update question: %w", err)
	}
	defer tx.Rollback()

	result, err := tx.ExecContext(ctx, `
		UPDATE vx_question
		SET title = ?, content_markdown = ?, content_rendered = ?, updated_at = CURRENT_TIMESTAMP(6)
		WHERE question_id = ? AND deleted_at IS NULL`, title, markdown, rendered, id)
	if err != nil {
		return Question{}, fmt.Errorf("update question: %w", err)
	}
	if rows, err := result.RowsAffected(); err != nil {
		return Question{}, fmt.Errorf("update question rows: %w", err)
	} else if rows == 0 {
		if _, err := tx.ExecContext(ctx, "SELECT question_id FROM vx_question WHERE question_id = ?", id); err != nil {
			return Question{}, ErrNotFound
		}
	}
	if err := replaceTopicables(ctx, tx, KindQuestion, id, topicIDs); err != nil {
		return Question{}, err
	}
	if err := outbox.Enqueue(ctx, tx, outbox.EventQuestionUpdated, id, map[string]any{"question_id": id}); err != nil {
		return Question{}, err
	}
	if err := tx.Commit(); err != nil {
		return Question{}, fmt.Errorf("commit update question: %w", err)
	}
	return r.FindQuestion(ctx, id)
}

func (r *MySQLRepository) DeleteQuestion(ctx context.Context, id int64) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin delete question: %w", err)
	}
	defer tx.Rollback()

	var userID int64
	if err := tx.QueryRowContext(ctx,
		"SELECT user_id FROM vx_question WHERE question_id = ? AND deleted_at IS NULL FOR UPDATE", id,
	).Scan(&userID); errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	} else if err != nil {
		return fmt.Errorf("lock question: %w", err)
	}
	topicIDs, err := topicIDsFor(ctx, tx, KindQuestion, id)
	if err != nil {
		return err
	}
	// 作者回答计数按作者聚合回退，避免逐条更新。
	if _, err := tx.ExecContext(ctx, `
		UPDATE vx_user u
		JOIN (
			SELECT user_id, COUNT(*) AS total FROM vx_answer
			WHERE question_id = ? AND deleted_at IS NULL GROUP BY user_id
		) a ON a.user_id = u.user_id
		SET u.answer_count = GREATEST(u.answer_count, a.total) - a.total`, id); err != nil {
		return fmt.Errorf("decrement answer authors: %w", err)
	}
	// 评论随问题与回答一起删除，避免留下无法访问的孤儿评论。
	if _, err := softDeleteComments(ctx, tx, "commentable_type = 'question' AND commentable_id = ?", id); err != nil {
		return err
	}
	if _, err := softDeleteComments(ctx, tx,
		"commentable_type = 'answer' AND commentable_id IN (SELECT answer_id FROM vx_answer WHERE question_id = ?)", id); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, "UPDATE vx_answer SET deleted_at = CURRENT_TIMESTAMP(6) WHERE question_id = ? AND deleted_at IS NULL", id); err != nil {
		return fmt.Errorf("soft delete answers: %w", err)
	}
	if _, err := tx.ExecContext(ctx,
		"UPDATE vx_question SET deleted_at = CURRENT_TIMESTAMP(6), answer_count = 0, comment_count = 0 WHERE question_id = ?", id); err != nil {
		return fmt.Errorf("soft delete question: %w", err)
	}
	if _, err := tx.ExecContext(ctx, "DELETE FROM vx_vote WHERE votable_type = 'question' AND votable_id = ?", id); err != nil {
		return fmt.Errorf("delete question votes: %w", err)
	}
	if _, err := tx.ExecContext(ctx, "DELETE FROM vx_follow WHERE followable_type = 'question' AND followable_id = ?", id); err != nil {
		return fmt.Errorf("delete question follows: %w", err)
	}
	if len(topicIDs) > 0 {
		if err := addTopicContentCounts(ctx, tx, KindQuestion, topicIDs, -1); err != nil {
			return err
		}
	}
	if _, err := tx.ExecContext(ctx, "UPDATE vx_user SET question_count = GREATEST(question_count, 1) - 1 WHERE user_id = ?", userID); err != nil {
		return fmt.Errorf("decrement user question count: %w", err)
	}
	if err := outbox.Enqueue(ctx, tx, outbox.EventQuestionDeleted, id, map[string]any{"question_id": id, "user_id": userID}); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit delete question: %w", err)
	}
	return nil
}

func (r *MySQLRepository) UpdateAnswer(ctx context.Context, id int64, markdown, rendered string) (Answer, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return Answer{}, fmt.Errorf("begin update answer: %w", err)
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(ctx, `
		UPDATE vx_answer
		SET content_markdown = ?, content_rendered = ?, updated_at = CURRENT_TIMESTAMP(6)
		WHERE answer_id = ? AND deleted_at IS NULL`, markdown, rendered, id)
	if err != nil {
		return Answer{}, fmt.Errorf("update answer: %w", err)
	}
	if rows, err := result.RowsAffected(); err != nil {
		return Answer{}, fmt.Errorf("update answer rows: %w", err)
	} else if rows == 0 {
		if _, err := tx.ExecContext(ctx, "SELECT answer_id FROM vx_answer WHERE answer_id = ?", id); err != nil {
			return Answer{}, ErrNotFound
		}
	}
	if err := outbox.Enqueue(ctx, tx, outbox.EventAnswerUpdated, id, map[string]any{"answer_id": id}); err != nil {
		return Answer{}, err
	}
	if err := tx.Commit(); err != nil {
		return Answer{}, fmt.Errorf("commit update answer: %w", err)
	}
	return r.FindAnswer(ctx, id)
}

func (r *MySQLRepository) DeleteAnswer(ctx context.Context, id int64) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin delete answer: %w", err)
	}
	defer tx.Rollback()

	var userID, questionID int64
	if err := tx.QueryRowContext(ctx,
		"SELECT user_id, question_id FROM vx_answer WHERE answer_id = ? AND deleted_at IS NULL FOR UPDATE", id,
	).Scan(&userID, &questionID); errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	} else if err != nil {
		return fmt.Errorf("lock answer: %w", err)
	}
	if _, err := softDeleteComments(ctx, tx, "commentable_type = 'answer' AND commentable_id = ?", id); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx,
		"UPDATE vx_answer SET deleted_at = CURRENT_TIMESTAMP(6), comment_count = 0 WHERE answer_id = ?", id); err != nil {
		return fmt.Errorf("soft delete answer: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE vx_question
		SET answer_count = GREATEST(answer_count, 1) - 1, updated_at = CURRENT_TIMESTAMP(6)
		WHERE question_id = ?`, questionID); err != nil {
		return fmt.Errorf("decrement question answer count: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE vx_user SET answer_count = GREATEST(answer_count, 1) - 1 WHERE user_id = ?`, userID); err != nil {
		return fmt.Errorf("decrement user answer count: %w", err)
	}
	if _, err := tx.ExecContext(ctx, "DELETE FROM vx_vote WHERE votable_type = 'answer' AND votable_id = ?", id); err != nil {
		return fmt.Errorf("delete answer votes: %w", err)
	}
	if err := outbox.Enqueue(ctx, tx, outbox.EventAnswerDeleted, id, map[string]any{"answer_id": id, "user_id": userID}); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit delete answer: %w", err)
	}
	return nil
}

func (r *MySQLRepository) Meta(ctx context.Context, kind string, id int64) (ContentMeta, error) {
	var meta ContentMeta
	switch kind {
	case KindQuestion:
		err := r.db.QueryRowContext(ctx, `
			SELECT question_id, user_id, created_at, comment_count, answer_count, content_rendered
			FROM vx_question WHERE question_id = ? AND deleted_at IS NULL`, id).
			Scan(&meta.ID, &meta.UserID, &meta.CreatedAt, &meta.CommentCount, &meta.AnswerCount, &meta.Content)
		if errors.Is(err, sql.ErrNoRows) {
			return ContentMeta{}, ErrNotFound
		}
		if err != nil {
			return ContentMeta{}, fmt.Errorf("question meta: %w", err)
		}
	case KindAnswer:
		err := r.db.QueryRowContext(ctx, `
			SELECT answer_id, user_id, created_at, comment_count, content_rendered
			FROM vx_answer WHERE answer_id = ? AND deleted_at IS NULL`, id).
			Scan(&meta.ID, &meta.UserID, &meta.CreatedAt, &meta.CommentCount, &meta.Content)
		if errors.Is(err, sql.ErrNoRows) {
			return ContentMeta{}, ErrNotFound
		}
		if err != nil {
			return ContentMeta{}, fmt.Errorf("answer meta: %w", err)
		}
	case KindArticle:
		err := r.db.QueryRowContext(ctx, `
			SELECT article_id, user_id, created_at, comment_count, content_rendered
			FROM vx_article WHERE article_id = ? AND deleted_at IS NULL`, id).
			Scan(&meta.ID, &meta.UserID, &meta.CreatedAt, &meta.CommentCount, &meta.Content)
		if errors.Is(err, sql.ErrNoRows) {
			return ContentMeta{}, ErrNotFound
		}
		if err != nil {
			return ContentMeta{}, fmt.Errorf("article meta: %w", err)
		}
	case KindComment:
		err := r.db.QueryRowContext(ctx, `
			SELECT comment_id, user_id, created_at, content
			FROM vx_comment WHERE comment_id = ? AND deleted_at IS NULL`, id).
			Scan(&meta.ID, &meta.UserID, &meta.CreatedAt, &meta.Content)
		if errors.Is(err, sql.ErrNoRows) {
			return ContentMeta{}, ErrNotFound
		}
		if err != nil {
			return ContentMeta{}, fmt.Errorf("comment meta: %w", err)
		}
	default:
		return ContentMeta{}, ErrNotFound
	}
	return meta, nil
}

// CascadeAuthors 返回随目标一起被删除的作者内容，用于逐个投递删除通知。
func (r *MySQLRepository) CascadeAuthors(ctx context.Context, kind string, id int64) ([]AuthorContent, error) {
	var query string
	switch kind {
	case KindQuestion:
		query = "SELECT answer_id, user_id, content_rendered FROM vx_answer WHERE question_id = ? AND deleted_at IS NULL"
	case KindComment:
		query = "SELECT comment_id, user_id, content FROM vx_comment WHERE root_id = ? AND deleted_at IS NULL"
	default:
		return nil, nil
	}
	rows, err := r.db.QueryContext(ctx, query, id)
	if err != nil {
		return nil, fmt.Errorf("load cascade authors: %w", err)
	}
	defer rows.Close()
	var items []AuthorContent
	for rows.Next() {
		var item AuthorContent
		if err := rows.Scan(&item.ID, &item.UserID, &item.Content); err != nil {
			return nil, fmt.Errorf("scan cascade author: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate cascade authors: %w", err)
	}
	return items, nil
}

// replaceTopicables 在更新内容时同步话题关系与话题计数，保证两者在同一事务内一致。
func replaceTopicables(ctx context.Context, tx *sql.Tx, kind string, targetID int64, topicIDs []int64) error {
	rows, err := tx.QueryContext(ctx,
		"SELECT topic_id FROM vx_topicable WHERE topicable_type = ? AND topicable_id = ?", kind, targetID)
	if err != nil {
		return fmt.Errorf("load current topics: %w", err)
	}
	current := map[int64]struct{}{}
	for rows.Next() {
		var topicID int64
		if err := rows.Scan(&topicID); err != nil {
			rows.Close()
			return fmt.Errorf("scan current topic: %w", err)
		}
		current[topicID] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return fmt.Errorf("iterate current topics: %w", err)
	}
	rows.Close()

	next := make(map[int64]struct{}, len(topicIDs))
	for _, topicID := range topicIDs {
		next[topicID] = struct{}{}
	}
	var added, removed []int64
	for topicID := range next {
		if _, ok := current[topicID]; !ok {
			added = append(added, topicID)
		}
	}
	for topicID := range current {
		if _, ok := next[topicID]; !ok {
			removed = append(removed, topicID)
		}
	}
	if _, err := tx.ExecContext(ctx,
		"DELETE FROM vx_topicable WHERE topicable_type = ? AND topicable_id = ?", kind, targetID); err != nil {
		return fmt.Errorf("clear topic links: %w", err)
	}
	if err := insertTopicables(ctx, tx, kind, targetID, topicIDs); err != nil {
		return err
	}
	if len(added) > 0 {
		if err := addTopicContentCounts(ctx, tx, kind, added, 1); err != nil {
			return err
		}
	}
	if len(removed) > 0 {
		if err := addTopicContentCounts(ctx, tx, kind, removed, -1); err != nil {
			return err
		}
	}
	return nil
}

func topicIDsFor(ctx context.Context, tx *sql.Tx, kind string, targetID int64) ([]int64, error) {
	rows, err := tx.QueryContext(ctx,
		"SELECT topic_id FROM vx_topicable WHERE topicable_type = ? AND topicable_id = ?", kind, targetID)
	if err != nil {
		return nil, fmt.Errorf("load topic ids: %w", err)
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan topic id: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// addTopicContentCounts 按内容类型选择计数列。delta 允许为负，用 GREATEST 兜底避免负数。
func addTopicContentCounts(ctx context.Context, tx *sql.Tx, kind string, topicIDs []int64, delta int) error {
	if len(topicIDs) == 0 || delta == 0 {
		return nil
	}
	column := "question_count"
	if kind == KindArticle {
		column = "article_count"
	}
	placeholders, args := placeholders(topicIDs)
	var query string
	if delta > 0 {
		query = "UPDATE vx_topic SET " + column + " = " + column + " + ? WHERE topic_id IN (" + placeholders + ")"
	} else {
		query = "UPDATE vx_topic SET " + column + " = GREATEST(" + column + ", 1) - 1 WHERE topic_id IN (" + placeholders + ")"
	}
	queryArgs := make([]any, 0, len(args)+1)
	if delta > 0 {
		queryArgs = append(queryArgs, delta)
	}
	queryArgs = append(queryArgs, args...)
	if _, err := tx.ExecContext(ctx, query, queryArgs...); err != nil {
		return fmt.Errorf("update topic content counts: %w", err)
	}
	return nil
}

func (r *MySQLRepository) attachArticleTopics(ctx context.Context, articles []Article) error {
	if len(articles) == 0 {
		return nil
	}
	ids := make([]int64, 0, len(articles))
	for _, article := range articles {
		ids = append(ids, article.ID)
	}
	topicsByArticle, err := r.topicsFor(ctx, KindArticle, ids)
	if err != nil {
		return err
	}
	for i := range articles {
		articles[i].Topics = topicsByArticle[articles[i].ID]
		if articles[i].Topics == nil {
			articles[i].Topics = []TopicRef{}
		}
	}
	return nil
}
