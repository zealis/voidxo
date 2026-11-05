package content

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	mysqlDriver "github.com/go-sql-driver/mysql"

	"github.com/voidxo/voidxo/internal/platform/outbox"
)

type MySQLRepository struct {
	db *sql.DB
}

func NewMySQLRepository(db *sql.DB) *MySQLRepository {
	return &MySQLRepository{db: db}
}

func (r *MySQLRepository) ListTopics(ctx context.Context, sort string, limit, offset int) ([]Topic, bool, error) {
	order, ok := topicSorts[sort]
	if !ok {
		order = topicSorts[defaultTopicSort]
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT topic_id, name, description, COALESCE(cover_key, ''), article_count, question_count, follower_count, created_at
		FROM vx_topic
		WHERE deleted_at IS NULL
		ORDER BY `+order+`
		LIMIT ? OFFSET ?`, limit+1, offset)
	if err != nil {
		return nil, false, fmt.Errorf("list topics: %w", err)
	}
	defer rows.Close()
	items, hasMore, err := scanTopics(rows, limit)
	return items, hasMore, err
}

func (r *MySQLRepository) FindTopic(ctx context.Context, id int64) (Topic, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT topic_id, name, description, COALESCE(cover_key, ''), article_count, question_count, follower_count, created_at
		FROM vx_topic WHERE topic_id = ? AND deleted_at IS NULL LIMIT 1`, id)
	var topic Topic
	err := row.Scan(&topic.ID, &topic.Name, &topic.Description, &topic.CoverKey, &topic.ArticleCount, &topic.QuestionCount, &topic.FollowerCount, &topic.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return Topic{}, ErrTopicNotFound
	}
	if err != nil {
		return Topic{}, fmt.Errorf("find topic: %w", err)
	}
	return topic, nil
}

func (r *MySQLRepository) CreateTopic(ctx context.Context, name, description, coverKey string) (Topic, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return Topic{}, fmt.Errorf("begin create topic: %w", err)
	}
	defer tx.Rollback()

	result, err := tx.ExecContext(ctx,
		"INSERT INTO vx_topic (name, description, cover_key) VALUES (?, ?, ?)",
		name, description, nullable(coverKey),
	)
	if err != nil {
		var mysqlErr *mysqlDriver.MySQLError
		if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
			return Topic{}, ErrTopicExists
		}
		return Topic{}, fmt.Errorf("create topic: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return Topic{}, fmt.Errorf("read topic id: %w", err)
	}
	if err := outbox.Enqueue(ctx, tx, outbox.EventTopicCreated, id, map[string]any{"topic_id": id}); err != nil {
		return Topic{}, err
	}
	if err := tx.Commit(); err != nil {
		return Topic{}, fmt.Errorf("commit create topic: %w", err)
	}
	return r.FindTopic(ctx, id)
}

func (r *MySQLRepository) CountTopics(ctx context.Context, ids []int64) (int, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	placeholders, args := placeholders(ids)
	var count int
	query := "SELECT COUNT(*) FROM vx_topic WHERE deleted_at IS NULL AND topic_id IN (" + placeholders + ")"
	if err := r.db.QueryRowContext(ctx, query, args...).Scan(&count); err != nil {
		return 0, fmt.Errorf("count topics: %w", err)
	}
	return count, nil
}

func (r *MySQLRepository) ListQuestions(ctx context.Context, topicID *int64, sort string, limit, offset int) ([]Question, bool, error) {
	query := `
		SELECT q.question_id, q.user_id, u.username, COALESCE(u.avatar_key, ''), q.title, q.comment_count, q.answer_count,
			q.follower_count, q.vote_count, q.last_answer_at, q.created_at, q.updated_at
		FROM vx_question q
		JOIN vx_user u ON u.user_id = q.user_id
		WHERE q.deleted_at IS NULL`
	args := make([]any, 0, 3)
	if topicID != nil {
		query += ` AND EXISTS (
			SELECT 1 FROM vx_topicable t
			WHERE t.topicable_type = 'question' AND t.topicable_id = q.question_id AND t.topic_id = ?
		)`
		args = append(args, *topicID)
	}
	order, ok := questionSorts[sort]
	if !ok {
		order = questionSorts[defaultQuestionSort]
	}
	query += " ORDER BY q." + order + " LIMIT ? OFFSET ?"
	args = append(args, limit+1, offset)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, false, fmt.Errorf("list questions: %w", err)
	}
	defer rows.Close()

	var items []Question
	for rows.Next() {
		question, err := scanQuestionSummary(rows)
		if err != nil {
			return nil, false, err
		}
		items = append(items, question)
	}
	if err := rows.Err(); err != nil {
		return nil, false, fmt.Errorf("iterate questions: %w", err)
	}
	hasMore := len(items) > limit
	if hasMore {
		items = items[:limit]
	}
	if err := r.attachTopics(ctx, items); err != nil {
		return nil, false, err
	}
	return items, hasMore, nil
}

func (r *MySQLRepository) FindQuestion(ctx context.Context, id int64) (Question, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT q.question_id, q.user_id, u.username, COALESCE(u.avatar_key, ''), q.title, q.content_markdown, q.content_rendered,
			q.comment_count, q.answer_count, q.follower_count, q.vote_count,
			q.last_answer_at, q.created_at, q.updated_at
		FROM vx_question q
		JOIN vx_user u ON u.user_id = q.user_id
		WHERE q.question_id = ? AND q.deleted_at IS NULL LIMIT 1`, id)
	question, err := scanQuestionDetail(row)
	if errors.Is(err, sql.ErrNoRows) || errors.Is(err, ErrNotFound) {
		return Question{}, ErrNotFound
	}
	if err != nil {
		return Question{}, err
	}
	topics, err := r.topicsFor(ctx, KindQuestion, []int64{question.ID})
	if err != nil {
		return Question{}, err
	}
	question.Topics = topics[question.ID]
	if question.Topics == nil {
		question.Topics = []TopicRef{}
	}
	return question, nil
}

func (r *MySQLRepository) CreateQuestion(ctx context.Context, userID int64, title, markdown, rendered string, topicIDs []int64) (Question, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return Question{}, fmt.Errorf("begin create question: %w", err)
	}
	defer tx.Rollback()

	result, err := tx.ExecContext(ctx,
		"INSERT INTO vx_question (user_id, title, content_markdown, content_rendered) VALUES (?, ?, ?, ?)",
		userID, title, markdown, rendered,
	)
	if err != nil {
		return Question{}, fmt.Errorf("insert question: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return Question{}, fmt.Errorf("read question id: %w", err)
	}
	if err := insertTopicables(ctx, tx, KindQuestion, id, topicIDs); err != nil {
		return Question{}, err
	}
	if err := incTopicQuestionCounts(ctx, tx, topicIDs); err != nil {
		return Question{}, err
	}
	if _, err := tx.ExecContext(ctx, "UPDATE vx_user SET question_count = question_count + 1 WHERE user_id = ?", userID); err != nil {
		return Question{}, fmt.Errorf("increment user question count: %w", err)
	}
	if err := outbox.Enqueue(ctx, tx, outbox.EventQuestionCreated, id, map[string]any{"question_id": id, "user_id": userID}); err != nil {
		return Question{}, err
	}
	if err := tx.Commit(); err != nil {
		return Question{}, fmt.Errorf("commit create question: %w", err)
	}
	return r.FindQuestion(ctx, id)
}

func (r *MySQLRepository) QuestionExists(ctx context.Context, id int64) (bool, error) {
	var exists int
	err := r.db.QueryRowContext(ctx, "SELECT 1 FROM vx_question WHERE question_id = ? AND deleted_at IS NULL LIMIT 1", id).Scan(&exists)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("check question: %w", err)
	}
	return true, nil
}

func (r *MySQLRepository) ListAnswers(ctx context.Context, questionID int64, limit, offset int) ([]Answer, bool, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT a.answer_id, a.question_id, a.user_id, u.username, COALESCE(u.avatar_key, ''), a.content_markdown, a.content_rendered,
			a.comment_count, a.vote_count, a.created_at, a.updated_at
		FROM vx_answer a
		JOIN vx_user u ON u.user_id = a.user_id
		WHERE a.question_id = ? AND a.deleted_at IS NULL
		ORDER BY a.created_at ASC, a.answer_id ASC
		LIMIT ? OFFSET ?`, questionID, limit+1, offset)
	if err != nil {
		return nil, false, fmt.Errorf("list answers: %w", err)
	}
	defer rows.Close()
	var items []Answer
	for rows.Next() {
		answer, err := scanAnswer(rows)
		if err != nil {
			return nil, false, err
		}
		items = append(items, answer)
	}
	if err := rows.Err(); err != nil {
		return nil, false, fmt.Errorf("iterate answers: %w", err)
	}
	hasMore := len(items) > limit
	if hasMore {
		items = items[:limit]
	}
	return items, hasMore, nil
}

func (r *MySQLRepository) CreateAnswer(ctx context.Context, questionID, userID int64, markdown, rendered string) (Answer, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return Answer{}, fmt.Errorf("begin create answer: %w", err)
	}
	defer tx.Rollback()

	var exists int64
	if err := tx.QueryRowContext(ctx,
		"SELECT question_id FROM vx_question WHERE question_id = ? AND deleted_at IS NULL FOR UPDATE",
		questionID,
	).Scan(&exists); errors.Is(err, sql.ErrNoRows) {
		return Answer{}, ErrNotFound
	} else if err != nil {
		return Answer{}, fmt.Errorf("lock question: %w", err)
	}
	result, err := tx.ExecContext(ctx,
		"INSERT INTO vx_answer (question_id, user_id, content_markdown, content_rendered) VALUES (?, ?, ?, ?)",
		questionID, userID, markdown, rendered,
	)
	if err != nil {
		return Answer{}, fmt.Errorf("insert answer: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return Answer{}, fmt.Errorf("read answer id: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE vx_question
		SET answer_count = answer_count + 1, last_answer_at = CURRENT_TIMESTAMP(6), updated_at = CURRENT_TIMESTAMP(6)
		WHERE question_id = ?`, questionID); err != nil {
		return Answer{}, fmt.Errorf("increment question answer count: %w", err)
	}
	if _, err := tx.ExecContext(ctx, "UPDATE vx_user SET answer_count = answer_count + 1 WHERE user_id = ?", userID); err != nil {
		return Answer{}, fmt.Errorf("increment user answer count: %w", err)
	}
	if err := outbox.Enqueue(ctx, tx, outbox.EventAnswerCreated, id, map[string]any{"answer_id": id, "question_id": questionID, "user_id": userID}); err != nil {
		return Answer{}, err
	}
	if err := tx.Commit(); err != nil {
		return Answer{}, fmt.Errorf("commit create answer: %w", err)
	}
	return r.FindAnswer(ctx, id)
}

func (r *MySQLRepository) FindAnswer(ctx context.Context, id int64) (Answer, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT a.answer_id, a.question_id, a.user_id, u.username, COALESCE(u.avatar_key, ''), a.content_markdown, a.content_rendered,
			a.comment_count, a.vote_count, a.created_at, a.updated_at
		FROM vx_answer a
		JOIN vx_user u ON u.user_id = a.user_id
		WHERE a.answer_id = ? AND a.deleted_at IS NULL LIMIT 1`, id)
	answer, err := scanAnswer(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Answer{}, ErrNotFound
	}
	if err != nil {
		return Answer{}, err
	}
	return answer, nil
}

func scanTopics(rows *sql.Rows, limit int) ([]Topic, bool, error) {
	var items []Topic
	for rows.Next() {
		var topic Topic
		if err := rows.Scan(&topic.ID, &topic.Name, &topic.Description, &topic.CoverKey, &topic.ArticleCount, &topic.QuestionCount, &topic.FollowerCount, &topic.CreatedAt); err != nil {
			return nil, false, fmt.Errorf("scan topic: %w", err)
		}
		items = append(items, topic)
	}
	if err := rows.Err(); err != nil {
		return nil, false, fmt.Errorf("iterate topics: %w", err)
	}
	hasMore := len(items) > limit
	if hasMore {
		items = items[:limit]
	}
	return items, hasMore, nil
}

func scanQuestionSummary(scanner interface{ Scan(...any) error }) (Question, error) {
	var question Question
	var lastAnswer sql.NullTime
	if err := scanner.Scan(
		&question.ID, &question.UserID, &question.Username, &question.AvatarKey, &question.Title,
		&question.CommentCount, &question.AnswerCount, &question.FollowerCount, &question.VoteCount,
		&lastAnswer, &question.CreatedAt, &question.UpdatedAt,
	); err != nil {
		return Question{}, fmt.Errorf("scan question: %w", err)
	}
	if lastAnswer.Valid {
		value := lastAnswer.Time
		question.LastAnswerAt = &value
	}
	return question, nil
}

func scanQuestionDetail(scanner interface{ Scan(...any) error }) (Question, error) {
	var question Question
	var lastAnswer sql.NullTime
	if err := scanner.Scan(
		&question.ID, &question.UserID, &question.Username, &question.AvatarKey, &question.Title,
		&question.ContentMarkdown, &question.ContentRendered,
		&question.CommentCount, &question.AnswerCount, &question.FollowerCount, &question.VoteCount,
		&lastAnswer, &question.CreatedAt, &question.UpdatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Question{}, ErrNotFound
		}
		return Question{}, fmt.Errorf("scan question detail: %w", err)
	}
	if lastAnswer.Valid {
		value := lastAnswer.Time
		question.LastAnswerAt = &value
	}
	return question, nil
}

func scanAnswer(scanner interface{ Scan(...any) error }) (Answer, error) {
	var answer Answer
	if err := scanner.Scan(
		&answer.ID, &answer.QuestionID, &answer.UserID, &answer.Username, &answer.AvatarKey,
		&answer.ContentMarkdown, &answer.ContentRendered,
		&answer.CommentCount, &answer.VoteCount, &answer.CreatedAt, &answer.UpdatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Answer{}, err
		}
		return Answer{}, fmt.Errorf("scan answer: %w", err)
	}
	return answer, nil
}

func (r *MySQLRepository) attachTopics(ctx context.Context, questions []Question) error {
	if len(questions) == 0 {
		return nil
	}
	ids := make([]int64, 0, len(questions))
	for _, question := range questions {
		ids = append(ids, question.ID)
	}
	topicsByQuestion, err := r.topicsFor(ctx, KindQuestion, ids)
	if err != nil {
		return err
	}
	for i := range questions {
		questions[i].Topics = topicsByQuestion[questions[i].ID]
		if questions[i].Topics == nil {
			questions[i].Topics = []TopicRef{}
		}
	}
	return nil
}

func (r *MySQLRepository) topicsFor(ctx context.Context, kind string, questionIDs []int64) (map[int64][]TopicRef, error) {
	if len(questionIDs) == 0 {
		return map[int64][]TopicRef{}, nil
	}
	placeholders, args := placeholders(questionIDs)
	rows, err := r.db.QueryContext(ctx, `
		SELECT t.topicable_id, tp.topic_id, tp.name
		FROM vx_topicable t
		JOIN vx_topic tp ON tp.topic_id = t.topic_id
		WHERE t.topicable_type = ? AND t.topicable_id IN (`+placeholders+`) AND tp.deleted_at IS NULL
		ORDER BY tp.name ASC`, append([]any{kind}, args...)...)
	if err != nil {
		return nil, fmt.Errorf("load question topics: %w", err)
	}
	defer rows.Close()
	result := make(map[int64][]TopicRef)
	for rows.Next() {
		var questionID int64
		var topic TopicRef
		if err := rows.Scan(&questionID, &topic.ID, &topic.Name); err != nil {
			return nil, fmt.Errorf("scan question topic: %w", err)
		}
		result[questionID] = append(result[questionID], topic)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate question topics: %w", err)
	}
	return result, nil
}

func insertTopicables(ctx context.Context, tx *sql.Tx, kind string, targetID int64, topicIDs []int64) error {
	values := make([]string, 0, len(topicIDs))
	args := make([]any, 0, len(topicIDs)*3)
	for _, topicID := range topicIDs {
		values = append(values, "(?, ?, ?)")
		args = append(args, topicID, targetID, kind)
	}
	if _, err := tx.ExecContext(ctx,
		"INSERT INTO vx_topicable (topic_id, topicable_id, topicable_type) VALUES "+strings.Join(values, ","),
		args...,
	); err != nil {
		return fmt.Errorf("insert question topics: %w", err)
	}
	return nil
}

func incTopicQuestionCounts(ctx context.Context, tx *sql.Tx, topicIDs []int64) error {
	return addTopicContentCounts(ctx, tx, KindQuestion, topicIDs, 1)
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

func nullable(value string) any {
	if strings.TrimSpace(value) == "" {
		return sql.NullString{}
	}
	return value
}

// ListSitemap 返回公开内容的编号与更新时间，按最近更新排序，供站点地图使用。
func (r *MySQLRepository) ListSitemap(ctx context.Context, limit int) ([]SitemapEntry, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT 'article' AS kind, article_id AS id, updated_at FROM vx_article WHERE deleted_at IS NULL
		UNION ALL
		SELECT 'question' AS kind, question_id AS id, updated_at FROM vx_question WHERE deleted_at IS NULL
		ORDER BY updated_at DESC
		LIMIT ?`, limit)
	if err != nil {
		return nil, fmt.Errorf("list sitemap: %w", err)
	}
	defer rows.Close()

	items := make([]SitemapEntry, 0, limit)
	for rows.Next() {
		var entry SitemapEntry
		if err := rows.Scan(&entry.Type, &entry.ID, &entry.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan sitemap entry: %w", err)
		}
		items = append(items, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate sitemap: %w", err)
	}
	return items, nil
}

// ListUserContent 返回指定用户的公开内容，按编号倒序，供个人主页分页展示。
func (r *MySQLRepository) ListUserContent(
	ctx context.Context, kind string, userID int64, limit, offset int,
) ([]UserContentItem, bool, error) {
	var query string
	switch kind {
	case KindQuestion:
		query = "SELECT 'question', question_id, title, LEFT(content_markdown, 160), '', 0," +
			" comment_count, vote_count, created_at FROM vx_question" +
			" WHERE user_id = ? AND deleted_at IS NULL ORDER BY question_id DESC LIMIT ? OFFSET ?"
	case KindArticle:
		query = "SELECT 'article', article_id, title, LEFT(content_markdown, 160), '', 0," +
			" comment_count, vote_count, created_at FROM vx_article" +
			" WHERE user_id = ? AND deleted_at IS NULL ORDER BY article_id DESC LIMIT ? OFFSET ?"
	case KindAnswer:
		query = "SELECT 'answer', answer_id, '', LEFT(content_markdown, 160), 'question', question_id," +
			" comment_count, vote_count, created_at FROM vx_answer" +
			" WHERE user_id = ? AND deleted_at IS NULL ORDER BY answer_id DESC LIMIT ? OFFSET ?"
	case KindComment:
		query = "SELECT 'comment', comment_id, '', LEFT(content, 160), commentable_type, commentable_id," +
			" 0, vote_count, created_at FROM vx_comment" +
			" WHERE user_id = ? AND deleted_at IS NULL ORDER BY comment_id DESC LIMIT ? OFFSET ?"
	default:
		return nil, false, fmt.Errorf("%w: 内容类型不支持", ErrNotFound)
	}

	rows, err := r.db.QueryContext(ctx, query, userID, limit+1, offset)
	if err != nil {
		return nil, false, fmt.Errorf("list user content: %w", err)
	}
	defer rows.Close()

	items := make([]UserContentItem, 0, limit)
	for rows.Next() {
		var item UserContentItem
		if err := rows.Scan(&item.Kind, &item.ID, &item.Title, &item.Excerpt, &item.TargetType,
			&item.TargetID, &item.CommentCount, &item.VoteCount, &item.CreatedAt); err != nil {
			return nil, false, fmt.Errorf("scan user content: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, false, fmt.Errorf("iterate user content: %w", err)
	}
	hasMore := len(items) > limit
	if hasMore {
		items = items[:limit]
	}
	return items, hasMore, nil
}
