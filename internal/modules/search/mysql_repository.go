package search

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/voidxo/voidxo/internal/modules/content"
)

type MySQLRepository struct {
	db *sql.DB
}

func NewMySQLRepository(db *sql.DB) *MySQLRepository {
	return &MySQLRepository{db: db}
}

func (r *MySQLRepository) SearchQuestions(ctx context.Context, pattern string, limit, offset int) ([]content.Question, bool, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT q.question_id, q.user_id, u.username, COALESCE(u.avatar_key, ''), q.title, q.comment_count, q.answer_count,
			q.follower_count, q.vote_count, q.last_answer_at, q.created_at, q.updated_at
		FROM vx_question q
		JOIN vx_user u ON u.user_id = q.user_id
		WHERE q.deleted_at IS NULL
			AND (q.title LIKE ? ESCAPE '\\' OR q.content_markdown LIKE ? ESCAPE '\\')
		ORDER BY q.updated_at DESC, q.question_id DESC
		LIMIT ? OFFSET ?`, pattern, pattern, limit+1, offset)
	if err != nil {
		return nil, false, fmt.Errorf("search questions: %w", err)
	}
	defer rows.Close()

	var items []content.Question
	for rows.Next() {
		var question content.Question
		var lastAnswer sql.NullTime
		if err := rows.Scan(&question.ID, &question.UserID, &question.Username, &question.AvatarKey, &question.Title,
			&question.CommentCount, &question.AnswerCount, &question.FollowerCount, &question.VoteCount,
			&lastAnswer, &question.CreatedAt, &question.UpdatedAt); err != nil {
			return nil, false, fmt.Errorf("scan search question: %w", err)
		}
		if lastAnswer.Valid {
			value := lastAnswer.Time
			question.LastAnswerAt = &value
		}
		question.Topics = []content.TopicRef{}
		items = append(items, question)
	}
	if err := rows.Err(); err != nil {
		return nil, false, fmt.Errorf("iterate search questions: %w", err)
	}
	hasMore := len(items) > limit
	if hasMore {
		items = items[:limit]
	}
	return items, hasMore, nil
}

func (r *MySQLRepository) SearchArticles(ctx context.Context, pattern string, limit, offset int) ([]content.Article, bool, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT a.article_id, a.user_id, u.username, COALESCE(u.avatar_key, ''), a.title, a.comment_count, a.follower_count,
			a.vote_count, a.created_at, a.updated_at
		FROM vx_article a
		JOIN vx_user u ON u.user_id = a.user_id
		WHERE a.deleted_at IS NULL
			AND (a.title LIKE ? ESCAPE '\\' OR a.content_markdown LIKE ? ESCAPE '\\')
		ORDER BY a.created_at DESC, a.article_id DESC
		LIMIT ? OFFSET ?`, pattern, pattern, limit+1, offset)
	if err != nil {
		return nil, false, fmt.Errorf("search articles: %w", err)
	}
	defer rows.Close()

	var items []content.Article
	for rows.Next() {
		var article content.Article
		if err := rows.Scan(&article.ID, &article.UserID, &article.Username, &article.AvatarKey, &article.Title,
			&article.CommentCount, &article.FollowerCount, &article.VoteCount,
			&article.CreatedAt, &article.UpdatedAt); err != nil {
			return nil, false, fmt.Errorf("scan search article: %w", err)
		}
		article.Topics = []content.TopicRef{}
		items = append(items, article)
	}
	if err := rows.Err(); err != nil {
		return nil, false, fmt.Errorf("iterate search articles: %w", err)
	}
	hasMore := len(items) > limit
	if hasMore {
		items = items[:limit]
	}
	return items, hasMore, nil
}

func (r *MySQLRepository) SearchTopics(ctx context.Context, pattern string, limit, offset int) ([]content.Topic, bool, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT topic_id, name, description, COALESCE(cover_key, ''), article_count,
			question_count, follower_count, created_at
		FROM vx_topic
		WHERE deleted_at IS NULL
			AND (name LIKE ? ESCAPE '\\' OR description LIKE ? ESCAPE '\\')
		ORDER BY follower_count DESC, topic_id ASC
		LIMIT ? OFFSET ?`, pattern, pattern, limit+1, offset)
	if err != nil {
		return nil, false, fmt.Errorf("search topics: %w", err)
	}
	defer rows.Close()

	var items []content.Topic
	for rows.Next() {
		var topic content.Topic
		if err := rows.Scan(&topic.ID, &topic.Name, &topic.Description, &topic.CoverKey,
			&topic.ArticleCount, &topic.QuestionCount, &topic.FollowerCount, &topic.CreatedAt); err != nil {
			return nil, false, fmt.Errorf("scan search topic: %w", err)
		}
		items = append(items, topic)
	}
	if err := rows.Err(); err != nil {
		return nil, false, fmt.Errorf("iterate search topics: %w", err)
	}
	hasMore := len(items) > limit
	if hasMore {
		items = items[:limit]
	}
	return items, hasMore, nil
}
