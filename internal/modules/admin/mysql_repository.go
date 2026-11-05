package admin

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	mysqlDriver "github.com/go-sql-driver/mysql"

	"github.com/voidxo/voidxo/internal/modules/option"
	"github.com/voidxo/voidxo/internal/platform/outbox"
)

type MySQLRepository struct {
	db    *sql.DB
	table string
	// 缓存自 option 包的定义表，避免每次请求重复构造。
	optionDefinitions map[string]option.Definition
}

func NewMySQLRepository(db *sql.DB, tablePrefix string) *MySQLRepository {
	return &MySQLRepository{
		db:                db,
		table:             tablePrefix,
		optionDefinitions: option.Definitions(),
	}
}

func (r *MySQLRepository) Stats(ctx context.Context) (Stats, error) {
	var stats Stats
	err := r.db.QueryRowContext(ctx, `
		SELECT
			(SELECT COUNT(*) FROM `+r.t("user")+`),
			(SELECT COUNT(*) FROM `+r.t("user")+` WHERE status <> 1 OR disabled_at IS NOT NULL),
			(SELECT COUNT(*) FROM `+r.t("question")+` WHERE deleted_at IS NULL),
			(SELECT COUNT(*) FROM `+r.t("answer")+` WHERE deleted_at IS NULL),
			(SELECT COUNT(*) FROM `+r.t("article")+` WHERE deleted_at IS NULL),
			(SELECT COUNT(*) FROM `+r.t("comment")+` WHERE deleted_at IS NULL),
			(SELECT COUNT(*) FROM `+r.t("topic")+` WHERE deleted_at IS NULL),
			(SELECT COUNT(*) FROM `+r.t("image")+` WHERE deleted_at IS NULL),
			(SELECT COUNT(*) FROM `+r.t("report")+` WHERE status = 'pending'),
			(SELECT COUNT(*) FROM `+r.t("user")+` WHERE created_at >= CURRENT_TIMESTAMP(6) - INTERVAL 7 DAY),
			(SELECT COUNT(*) FROM `+r.t("question")+` WHERE deleted_at IS NULL AND created_at >= CURRENT_TIMESTAMP(6) - INTERVAL 7 DAY),
			(SELECT COUNT(*) FROM `+r.t("article")+` WHERE deleted_at IS NULL AND created_at >= CURRENT_TIMESTAMP(6) - INTERVAL 7 DAY),
			(SELECT COUNT(*) FROM `+r.t("comment")+` WHERE deleted_at IS NULL AND created_at >= CURRENT_TIMESTAMP(6) - INTERVAL 7 DAY)`).
		Scan(&stats.Users, &stats.DisabledUsers, &stats.Questions, &stats.Answers, &stats.Articles,
			&stats.Comments, &stats.Topics, &stats.Images, &stats.PendingReports,
			&stats.NewUsers7d, &stats.NewQuestions7d, &stats.NewArticles7d, &stats.NewComments7d)
	if err != nil {
		return Stats{}, fmt.Errorf("load admin stats: %w", err)
	}
	return stats, nil
}

const userSelect = `
	SELECT u.user_id, u.username, u.email, u.status, u.headline,
		u.article_count, u.question_count, u.answer_count, u.comment_count,
		u.created_at, u.last_login_at, u.disabled_at,
		COALESCE((SELECT GROUP_CONCAT(r.code ORDER BY r.code)
			FROM %s ur JOIN %s r ON r.role_id = ur.role_id
			WHERE ur.user_id = u.user_id), '')
	FROM %s u`

func (r *MySQLRepository) ListUsers(ctx context.Context, filter UserFilter) ([]User, bool, error) {
	where := []string{"1 = 1"}
	args := []any{}
	if filter.Query != "" {
		like := escapeLike(filter.Query)
		where = append(where, "(u.username LIKE ? ESCAPE '\\\\' OR u.email LIKE ? ESCAPE '\\\\')")
		args = append(args, like, like)
	}
	if filter.Role != "" {
		where = append(where, `EXISTS (SELECT 1 FROM `+r.t("user_role")+` ur JOIN `+r.t("role")+` r ON r.role_id = ur.role_id
			WHERE ur.user_id = u.user_id AND r.code = ?)`)
		args = append(args, filter.Role)
	}
	switch filter.Status {
	case "active":
		where = append(where, "u.status = 1 AND u.disabled_at IS NULL")
	case "disabled":
		where = append(where, "(u.status <> 1 OR u.disabled_at IS NOT NULL)")
	}

	query := fmt.Sprintf(userSelect,
		r.t("user_role"), r.t("role"), r.t("user")) +
		" WHERE " + strings.Join(where, " AND ") +
		" ORDER BY u.user_id DESC LIMIT ? OFFSET ?"
	args = append(args, filter.PageSize+1, (filter.Page-1)*filter.PageSize)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, false, fmt.Errorf("list admin users: %w", err)
	}
	defer rows.Close()

	items := make([]User, 0, filter.PageSize)
	for rows.Next() {
		user, err := scanUser(rows)
		if err != nil {
			return nil, false, err
		}
		items = append(items, user)
	}
	if err := rows.Err(); err != nil {
		return nil, false, fmt.Errorf("iterate admin users: %w", err)
	}
	hasMore := len(items) > filter.PageSize
	if hasMore {
		items = items[:filter.PageSize]
	}
	return items, hasMore, nil
}

func (r *MySQLRepository) FindUser(ctx context.Context, id int64) (User, error) {
	query := fmt.Sprintf(userSelect, r.t("user_role"), r.t("role"), r.t("user")) + " WHERE u.user_id = ?"
	user, err := scanUser(r.db.QueryRowContext(ctx, query, id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return User{}, ErrNotFound
		}
		return User{}, err
	}
	return user, nil
}

func (r *MySQLRepository) UpdateUser(ctx context.Context, id int64, update UserUpdate) error {
	sets := []string{}
	args := []any{}
	if update.Status != nil {
		if *update.Status == StatusActive {
			sets = append(sets, "status = 1", "disabled_at = NULL")
		} else {
			sets = append(sets, "status = 0", "disabled_at = CURRENT_TIMESTAMP(6)")
		}
	}
	if update.Role != nil {
		role := *update.Role
		sets = append(sets, "updated_at = CURRENT_TIMESTAMP(6)")
		if err := r.replaceRole(ctx, id, role); err != nil {
			return err
		}
	}
	if len(sets) == 0 {
		if update.Role == nil {
			return nil
		}
	}
	if len(sets) > 0 {
		query := "UPDATE " + r.t("user") + " SET " + strings.Join(sets, ", ") + " WHERE user_id = ?"
		args = append(args, id)
		result, err := r.db.ExecContext(ctx, query, args...)
		if err != nil {
			return fmt.Errorf("update admin user: %w", err)
		}
		if rows, err := result.RowsAffected(); err == nil && rows == 0 {
			if _, findErr := r.FindUser(ctx, id); findErr != nil {
				return findErr
			}
		}
	}
	return nil
}

// replaceRole 把用户角色替换为单一角色；后台只支持在三个内置角色间切换。
func (r *MySQLRepository) replaceRole(ctx context.Context, userID int64, role string) error {
	var roleID int64
	if err := r.db.QueryRowContext(ctx,
		"SELECT role_id FROM "+r.t("role")+" WHERE code = ? LIMIT 1", role).Scan(&roleID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("%w: unknown role", ErrValidation)
		}
		return fmt.Errorf("load role: %w", err)
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin replace role: %w", err)
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, "DELETE FROM "+r.t("user_role")+" WHERE user_id = ?", userID); err != nil {
		return fmt.Errorf("clear user roles: %w", err)
	}
	if _, err := tx.ExecContext(ctx,
		"INSERT INTO "+r.t("user_role")+" (user_id, role_id) VALUES (?, ?)", userID, roleID); err != nil {
		return fmt.Errorf("assign user role: %w", err)
	}
	if err := outbox.Enqueue(ctx, tx, outbox.EventUserRoleChanged, userID, map[string]any{
		"user_id": userID, "role": role,
	}); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit replace role: %w", err)
	}
	return nil
}

func (r *MySQLRepository) CountActiveAdmins(ctx context.Context, excludeUserID int64) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(DISTINCT u.user_id) FROM `+r.t("user")+` u
		JOIN `+r.t("user_role")+` ur ON ur.user_id = u.user_id
		JOIN `+r.t("role")+` r ON r.role_id = ur.role_id
		WHERE r.code = 'admin' AND u.status = 1 AND u.disabled_at IS NULL AND u.user_id <> ?`, excludeUserID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count active admins: %w", err)
	}
	return count, nil
}

func (r *MySQLRepository) ListContent(ctx context.Context, kind string, filter ContentFilter) ([]ContentItem, bool, error) {
	query, args := r.contentQuery(kind, filter)
	if query == "" {
		return nil, false, fmt.Errorf("%w: 内容类型不支持", ErrValidation)
	}
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, false, fmt.Errorf("list admin %s: %w", kind, err)
	}
	defer rows.Close()

	items := make([]ContentItem, 0, filter.PageSize)
	for rows.Next() {
		item, err := scanContent(kind, rows)
		if err != nil {
			return nil, false, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, false, fmt.Errorf("iterate admin %s: %w", kind, err)
	}
	hasMore := len(items) > filter.PageSize
	if hasMore {
		items = items[:filter.PageSize]
	}
	return items, hasMore, nil
}

func (r *MySQLRepository) contentQuery(kind string, filter ContentFilter) (string, []any) {
	limit := filter.PageSize + 1
	offset := (filter.Page - 1) * filter.PageSize

	where := []string{}
	args := []any{}
	if filter.Query != "" {
		like := escapeLike(filter.Query)
		switch kind {
		case KindQuestion, KindArticle:
			where = append(where, "title LIKE ? ESCAPE '\\\\'")
			args = append(args, like)
		case KindAnswer:
			where = append(where, "(a.content_markdown LIKE ? ESCAPE '\\\\' OR q.title LIKE ? ESCAPE '\\\\')")
			args = append(args, like, like)
		case KindComment:
			where = append(where, "c.content LIKE ? ESCAPE '\\\\'")
			args = append(args, like)
		}
	}
	// 各内容类型的主表别名不同（question/article 用 t，answer 用 a，comment 用 c），
	// 删除状态过滤必须跟随主表别名，否则 answer 会生成 t.deleted_at 而报 Unknown column。
	deletedColumn := "t.deleted_at"
	switch kind {
	case KindAnswer:
		deletedColumn = "a.deleted_at"
	case KindComment:
		deletedColumn = "c.deleted_at"
	}
	switch filter.Deleted {
	case "deleted":
		where = append(where, deletedColumn+" IS NOT NULL")
	case "all":
	default:
		where = append(where, deletedColumn+" IS NULL")
	}
	clause := ""
	if len(where) > 0 {
		clause = " WHERE " + strings.Join(where, " AND ")
	}

	var query string
	switch kind {
	case KindQuestion:
		query = `SELECT t.question_id, t.title, LEFT(t.content_markdown, 200), t.user_id, u.username,
				t.comment_count, t.vote_count, t.created_at, t.deleted_at IS NOT NULL, '', 0, ''
			FROM ` + r.t("question") + ` t JOIN ` + r.t("user") + ` u ON u.user_id = t.user_id` +
			clause + " ORDER BY t.question_id DESC LIMIT ? OFFSET ?"
	case KindArticle:
		query = `SELECT t.article_id, t.title, LEFT(t.content_markdown, 200), t.user_id, u.username,
				t.comment_count, t.vote_count, t.created_at, t.deleted_at IS NOT NULL, '', 0, ''
			FROM ` + r.t("article") + ` t JOIN ` + r.t("user") + ` u ON u.user_id = t.user_id` +
			clause + " ORDER BY t.article_id DESC LIMIT ? OFFSET ?"
	case KindAnswer:
		query = `SELECT a.answer_id, '', LEFT(a.content_markdown, 200), a.user_id, u.username,
				a.comment_count, a.vote_count, a.created_at, a.deleted_at IS NOT NULL, q.title, q.question_id, 'question' 
			FROM ` + r.t("answer") + ` a
			JOIN ` + r.t("user") + ` u ON u.user_id = a.user_id
			JOIN ` + r.t("question") + ` q ON q.question_id = a.question_id` +
			clause + " ORDER BY a.answer_id DESC LIMIT ? OFFSET ?"
	case KindComment:
		query = `SELECT c.comment_id, '', LEFT(c.content, 200), c.user_id, u.username,
				0, c.vote_count, c.created_at, c.deleted_at IS NOT NULL,
				CONCAT(c.commentable_type, '#', c.commentable_id), c.commentable_id, c.commentable_type
			FROM ` + r.t("comment") + ` c JOIN ` + r.t("user") + ` u ON u.user_id = c.user_id` +
			clause + " ORDER BY c.comment_id DESC LIMIT ? OFFSET ?"
	default:
		return "", nil
	}
	args = append(args, limit, offset)
	return query, args
}

func (r *MySQLRepository) ListTopics(ctx context.Context, query string, includeDeleted bool, page, pageSize int) ([]TopicItem, bool, error) {
	where := []string{}
	args := []any{}
	if !includeDeleted {
		where = append(where, "deleted_at IS NULL")
	}
	if query != "" {
		like := escapeLike(query)
		where = append(where, "(name LIKE ? ESCAPE '\\\\' OR description LIKE ? ESCAPE '\\\\')")
		args = append(args, like, like)
	}
	clause := ""
	if len(where) > 0 {
		clause = " WHERE " + strings.Join(where, " AND ")
	}
	sqlText := "SELECT topic_id, name, description, COALESCE(cover_key, ''), article_count, question_count," +
		" follower_count, created_at, deleted_at IS NOT NULL FROM " + r.t("topic") + clause +
		" ORDER BY topic_id DESC LIMIT ? OFFSET ?"
	args = append(args, pageSize+1, (page-1)*pageSize)

	rows, err := r.db.QueryContext(ctx, sqlText, args...)
	if err != nil {
		return nil, false, fmt.Errorf("list admin topics: %w", err)
	}
	defer rows.Close()

	items := make([]TopicItem, 0, pageSize)
	for rows.Next() {
		var topic TopicItem
		if err := rows.Scan(&topic.ID, &topic.Name, &topic.Description, &topic.CoverKey,
			&topic.ArticleCount, &topic.QuestionCount, &topic.FollowerCount, &topic.CreatedAt, &topic.Deleted); err != nil {
			return nil, false, fmt.Errorf("scan admin topic: %w", err)
		}
		items = append(items, topic)
	}
	if err := rows.Err(); err != nil {
		return nil, false, fmt.Errorf("iterate admin topics: %w", err)
	}
	hasMore := len(items) > pageSize
	if hasMore {
		items = items[:pageSize]
	}
	return items, hasMore, nil
}

func (r *MySQLRepository) FindTopic(ctx context.Context, id int64) (TopicItem, error) {
	var topic TopicItem
	err := r.db.QueryRowContext(ctx, "SELECT topic_id, name, description, COALESCE(cover_key, ''),"+
		" article_count, question_count, follower_count, created_at, deleted_at IS NOT NULL"+
		" FROM "+r.t("topic")+" WHERE topic_id = ?", id).
		Scan(&topic.ID, &topic.Name, &topic.Description, &topic.CoverKey,
			&topic.ArticleCount, &topic.QuestionCount, &topic.FollowerCount, &topic.CreatedAt, &topic.Deleted)
	if errors.Is(err, sql.ErrNoRows) {
		return TopicItem{}, ErrNotFound
	}
	if err != nil {
		return TopicItem{}, fmt.Errorf("find admin topic: %w", err)
	}
	return topic, nil
}

func (r *MySQLRepository) CreateTopic(ctx context.Context, name, description, coverKey string) (int64, error) {
	result, err := r.db.ExecContext(ctx,
		"INSERT INTO "+r.t("topic")+" (name, description, cover_key) VALUES (?, ?, ?)",
		name, description, nullable(coverKey))
	if err != nil {
		var mysqlErr *mysqlDriver.MySQLError
		if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
			return 0, fmt.Errorf("%w: 话题名称已存在", ErrValidation)
		}
		return 0, fmt.Errorf("create admin topic: %w", err)
	}
	return result.LastInsertId()
}

func (r *MySQLRepository) UpdateTopic(ctx context.Context, id int64, name, description, coverKey string) (TopicItem, error) {
	result, err := r.db.ExecContext(ctx,
		"UPDATE "+r.t("topic")+" SET name = ?, description = ?, cover_key = ? WHERE topic_id = ? AND deleted_at IS NULL",
		name, description, nullable(coverKey), id)
	if err != nil {
		var mysqlErr *mysqlDriver.MySQLError
		if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
			return TopicItem{}, fmt.Errorf("%w: 话题名称已存在", ErrValidation)
		}
		return TopicItem{}, fmt.Errorf("update admin topic: %w", err)
	}
	if rows, err := result.RowsAffected(); err == nil && rows == 0 {
		if _, findErr := r.FindTopic(ctx, id); findErr != nil {
			return TopicItem{}, findErr
		}
	}
	return r.FindTopic(ctx, id)
}

func (r *MySQLRepository) DeleteTopic(ctx context.Context, id int64) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin delete topic: %w", err)
	}
	defer tx.Rollback()

	result, err := tx.ExecContext(ctx,
		"UPDATE "+r.t("topic")+" SET deleted_at = CURRENT_TIMESTAMP(6) WHERE topic_id = ? AND deleted_at IS NULL", id)
	if err != nil {
		return fmt.Errorf("soft delete topic: %w", err)
	}
	if rows, err := result.RowsAffected(); err == nil && rows == 0 {
		return ErrNotFound
	}
	// 话题删除后关系表留痕即可，查询侧已按话题 deleted_at 过滤。
	if err := outbox.Enqueue(ctx, tx, outbox.EventTopicDeleted, id, map[string]any{"topic_id": id}); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit delete topic: %w", err)
	}
	return nil
}

func (r *MySQLRepository) ListOptions(ctx context.Context) ([]OptionItem, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT name, value FROM "+r.t("option")+" ORDER BY name")
	if err != nil {
		return nil, fmt.Errorf("list options: %w", err)
	}
	defer rows.Close()

	items := make([]OptionItem, 0, len(r.optionDefinitions))
	for rows.Next() {
		var name, value string
		if err := rows.Scan(&name, &value); err != nil {
			return nil, fmt.Errorf("scan option: %w", err)
		}
		item := OptionItem{Name: name, Value: value, Type: "string"}
		if definition, ok := r.optionDefinitions[name]; ok {
			item.Public = definition.Public
			item.Type = definition.Type
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate options: %w", err)
	}
	return items, nil
}

func (r *MySQLRepository) UpdateOptions(ctx context.Context, values map[string]string) error {
	if len(values) == 0 {
		return nil
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin update options: %w", err)
	}
	defer tx.Rollback()

	names := make([]string, 0, len(values))
	for name, value := range values {
		names = append(names, name)
		if _, err := tx.ExecContext(ctx,
			"INSERT INTO "+r.t("option")+" (name, value) VALUES (?, ?)"+
				" ON DUPLICATE KEY UPDATE value = VALUES(value)", name, value); err != nil {
			return fmt.Errorf("update option %s: %w", name, err)
		}
	}
	if err := outbox.Enqueue(ctx, tx, outbox.EventOptionUpdated, 0, map[string]any{"names": names}); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit update options: %w", err)
	}
	return nil
}

func (r *MySQLRepository) ListImages(ctx context.Context, filter ContentFilter) ([]ImageItem, bool, error) {
	where := []string{}
	args := []any{}
	if filter.Query != "" {
		like := escapeLike(filter.Query)
		where = append(where, "(i.filename LIKE ? ESCAPE '\\\\' OR u.username LIKE ? ESCAPE '\\\\')")
		args = append(args, like, like)
	}
	switch filter.Deleted {
	case "deleted":
		where = append(where, "i.deleted_at IS NOT NULL")
	case "all":
	default:
		where = append(where, "i.deleted_at IS NULL")
	}
	clause := ""
	if len(where) > 0 {
		clause = " WHERE " + strings.Join(where, " AND ")
	}
	query := "SELECT i.storage_key, i.filename, i.mime, i.size, i.width, i.height, i.kind," +
		" i.user_id, u.username, i.created_at, i.deleted_at IS NOT NULL" +
		" FROM " + r.t("image") + " i JOIN " + r.t("user") + " u ON u.user_id = i.user_id" +
		clause + " ORDER BY i.image_id DESC LIMIT ? OFFSET ?"
	args = append(args, filter.PageSize+1, (filter.Page-1)*filter.PageSize)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, false, fmt.Errorf("list admin images: %w", err)
	}
	defer rows.Close()

	items := make([]ImageItem, 0, filter.PageSize)
	for rows.Next() {
		var item ImageItem
		if err := rows.Scan(&item.Key, &item.Filename, &item.Mime, &item.Size, &item.Width, &item.Height,
			&item.Kind, &item.UserID, &item.Username, &item.CreatedAt, &item.Deleted); err != nil {
			return nil, false, fmt.Errorf("scan admin image: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, false, fmt.Errorf("iterate admin images: %w", err)
	}
	hasMore := len(items) > filter.PageSize
	if hasMore {
		items = items[:filter.PageSize]
	}
	return items, hasMore, nil
}

func (r *MySQLRepository) t(name string) string {
	return r.table + name
}

func scanUser(scanner interface{ Scan(...any) error }) (User, error) {
	var user User
	var roles string
	if err := scanner.Scan(&user.ID, &user.Username, &user.Email, &user.Status, &user.Headline,
		&user.ArticleCount, &user.QuestionCount, &user.AnswerCount, &user.CommentCount,
		&user.CreatedAt, &user.LastLoginAt, &user.DisabledAt, &roles); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return User{}, err
		}
		return User{}, fmt.Errorf("scan admin user: %w", err)
	}
	user.Roles = splitRoles(roles)
	return user, nil
}

func splitRoles(value string) []string {
	if strings.TrimSpace(value) == "" {
		return []string{}
	}
	parts := strings.Split(value, ",")
	roles := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			roles = append(roles, trimmed)
		}
	}
	return roles
}

func scanContent(kind string, scanner interface{ Scan(...any) error }) (ContentItem, error) {
	var item ContentItem
	if err := scanner.Scan(&item.ID, &item.Title, &item.Excerpt, &item.AuthorID, &item.Author,
		&item.CommentCount, &item.VoteCount, &item.CreatedAt, &item.Deleted, &item.Context,
		&item.ParentID, &item.ParentType); err != nil {
		return ContentItem{}, fmt.Errorf("scan admin %s: %w", kind, err)
	}
	if kind == KindAnswer && item.Context != "" {
		item.Context = "问题：" + item.Context
	}
	return item, nil
}

// escapeLike 转义 LIKE 通配符，避免用户输入被当作模式匹配。
func escapeLike(value string) string {
	replacer := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return "%" + replacer.Replace(value) + "%"
}

func nullable(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}
