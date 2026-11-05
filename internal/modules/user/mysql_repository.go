package user

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

const accountColumns = "user_id, username, email, password_hash, status, avatar_key, cover_key, headline, bio, blog, company, location, created_at, updated_at, disabled_at"

func (r *MySQLRepository) FindByLogin(ctx context.Context, login string) (Account, error) {
	row := r.db.QueryRowContext(ctx,
		"SELECT "+accountColumns+" FROM vx_user WHERE username = ? OR email = ? LIMIT 1",
		login, login,
	)
	return scanAccount(row)
}

func (r *MySQLRepository) FindByID(ctx context.Context, id int64) (Account, error) {
	row := r.db.QueryRowContext(ctx,
		"SELECT "+accountColumns+" FROM vx_user WHERE user_id = ? LIMIT 1",
		id,
	)
	return scanAccount(row)
}

func (r *MySQLRepository) Create(ctx context.Context, params CreateParams) (Account, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return Account{}, fmt.Errorf("begin create user: %w", err)
	}
	defer tx.Rollback()

	result, err := tx.ExecContext(ctx,
		"INSERT INTO vx_user (username, email, password_hash, create_ip) VALUES (?, ?, ?, ?)",
		params.Username, params.Email, params.PasswordHash, params.CreateIP,
	)
	if err != nil {
		if isDuplicate(err) {
			return Account{}, ErrDuplicate
		}
		return Account{}, fmt.Errorf("insert user: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return Account{}, fmt.Errorf("read user id: %w", err)
	}
	if params.RoleCode != "" {
		roleResult, err := tx.ExecContext(ctx,
			"INSERT INTO vx_user_role (user_id, role_id) SELECT ?, role_id FROM vx_role WHERE code = ?",
			id, params.RoleCode,
		)
		if err != nil {
			return Account{}, fmt.Errorf("assign user role: %w", err)
		}
		if rows, err := roleResult.RowsAffected(); err != nil || rows != 1 {
			return Account{}, fmt.Errorf("role %q does not exist", params.RoleCode)
		}
	}
	if err := tx.Commit(); err != nil {
		return Account{}, fmt.Errorf("commit create user: %w", err)
	}
	return r.FindByID(ctx, id)
}

func (r *MySQLRepository) SetPassword(ctx context.Context, id int64, passwordHash string) error {
	result, err := r.db.ExecContext(ctx,
		"UPDATE vx_user SET password_hash = ?, password_changed_at = CURRENT_TIMESTAMP(6) WHERE user_id = ?",
		passwordHash, id,
	)
	if err != nil {
		return fmt.Errorf("update password: %w", err)
	}
	if rows, err := result.RowsAffected(); err != nil || rows == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *MySQLRepository) RecordLogin(ctx context.Context, id int64, ip string) error {
	if _, err := r.db.ExecContext(ctx,
		"UPDATE vx_user SET last_login_at = CURRENT_TIMESTAMP(6), last_login_ip = ? WHERE user_id = ?",
		nullIfEmpty(ip), id,
	); err != nil {
		return fmt.Errorf("record login: %w", err)
	}
	return nil
}

func (r *MySQLRepository) UpdateProfile(ctx context.Context, id int64, params UpdateProfileParams) (Account, error) {
	fields := make([]string, 0, 5)
	args := make([]any, 0, 6)
	appendField := func(column string, value *string) {
		if value != nil {
			fields = append(fields, column+" = ?")
			args = append(args, *value)
		}
	}
	appendField("headline", params.Headline)
	appendField("bio", params.Bio)
	appendField("blog", params.Blog)
	appendField("company", params.Company)
	appendField("location", params.Location)
	if len(fields) > 0 {
		args = append(args, id)
		query := "UPDATE vx_user SET " + strings.Join(fields, ", ") + " WHERE user_id = ?"
		if _, err := r.db.ExecContext(ctx, query, args...); err != nil {
			return Account{}, fmt.Errorf("update profile: %w", err)
		}
	}
	return r.FindByID(ctx, id)
}

func (r *MySQLRepository) Permissions(ctx context.Context, id int64) ([]string, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT DISTINCT p.code
		FROM vx_user_role ur
		JOIN vx_role_permission rp ON rp.role_id = ur.role_id
		JOIN vx_permission p ON p.permission_id = rp.permission_id
		WHERE ur.user_id = ?
		ORDER BY p.code`, id)
	if err != nil {
		return nil, fmt.Errorf("load permissions: %w", err)
	}
	defer rows.Close()
	// 无权限的用户也要返回空数组，避免序列化成 null 让前端判空失败。
	permissions := make([]string, 0, 8)
	for rows.Next() {
		var code string
		if err := rows.Scan(&code); err != nil {
			return nil, err
		}
		permissions = append(permissions, code)
	}
	return permissions, rows.Err()
}

func (r *MySQLRepository) Roles(ctx context.Context, id int64) ([]string, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT r.code
		FROM vx_user_role ur
		JOIN vx_role r ON r.role_id = ur.role_id
		WHERE ur.user_id = ?
		ORDER BY r.code`, id)
	if err != nil {
		return nil, fmt.Errorf("load roles: %w", err)
	}
	defer rows.Close()
	roles := make([]string, 0, 2)
	for rows.Next() {
		var code string
		if err := rows.Scan(&code); err != nil {
			return nil, err
		}
		roles = append(roles, code)
	}
	return roles, rows.Err()
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanAccount(row rowScanner) (Account, error) {
	var account Account
	var avatar, cover sql.NullString
	var disabled sql.NullTime
	err := row.Scan(
		&account.ID, &account.Username, &account.Email, &account.PasswordHash, &account.Status,
		&avatar, &cover, &account.Headline, &account.Bio, &account.Blog, &account.Company, &account.Location,
		&account.CreatedAt, &account.UpdatedAt, &disabled,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return Account{}, ErrNotFound
	}
	if err != nil {
		return Account{}, fmt.Errorf("scan user: %w", err)
	}
	account.AvatarKey = avatar.String
	account.CoverKey = cover.String
	if disabled.Valid {
		value := disabled.Time
		account.DisabledAt = &value
	}
	return account, nil
}

func isDuplicate(err error) bool {
	var mysqlErr *mysqlDriver.MySQLError
	return errors.As(err, &mysqlErr) && mysqlErr.Number == 1062
}

func nullIfEmpty(value string) any {
	if value == "" {
		return sql.NullString{}
	}
	return value
}

func (r *MySQLRepository) SetAvatar(ctx context.Context, id int64, key string) error {
	return r.setBrandKey(ctx, id, "avatar_key", key)
}

func (r *MySQLRepository) SetCover(ctx context.Context, id int64, key string) error {
	return r.setBrandKey(ctx, id, "cover_key", key)
}

func (r *MySQLRepository) setBrandKey(ctx context.Context, id int64, column, key string) error {
	// column 只由调用方用常量传入，不接受外部输入。
	result, err := r.db.ExecContext(ctx, "UPDATE vx_user SET "+column+" = ? WHERE user_id = ?", key, id)
	if err != nil {
		return fmt.Errorf("update %s: %w", column, err)
	}
	if rows, err := result.RowsAffected(); err != nil {
		return fmt.Errorf("update %s rows: %w", column, err)
	} else if rows == 0 {
		if _, err := r.FindByID(ctx, id); err != nil {
			return err
		}
	}
	return nil
}

func (r *MySQLRepository) EmailExists(ctx context.Context, email string) (bool, error) {
	var exists int
	err := r.db.QueryRowContext(ctx,
		"SELECT 1 FROM vx_user WHERE email = ? LIMIT 1", email).Scan(&exists)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("check email exists: %w", err)
	}
	return true, nil
}

func (r *MySQLRepository) FindByEmail(ctx context.Context, email string) (Account, error) {
	row := r.db.QueryRowContext(ctx, "SELECT "+accountColumns+" FROM vx_user WHERE email = ? LIMIT 1", email)
	return scanAccount(row)
}

// FindPublic 读取公开主页字段；停用账号按不存在处理。
func (r *MySQLRepository) FindPublic(ctx context.Context, id int64) (PublicProfile, error) {
	var profile PublicProfile
	err := r.db.QueryRowContext(ctx, `
		SELECT user_id, username, COALESCE(avatar_key, ''), COALESCE(cover_key, ''),
			headline, bio, blog, company, location,
			follower_count, followee_count, article_count, question_count, answer_count, comment_count,
			created_at
		FROM vx_user WHERE user_id = ? AND status = 1 AND disabled_at IS NULL LIMIT 1`, id).
		Scan(&profile.ID, &profile.Username, &profile.AvatarKey, &profile.CoverKey,
			&profile.Headline, &profile.Bio, &profile.Blog, &profile.Company, &profile.Location,
			&profile.FollowerCount, &profile.FolloweeCount, &profile.ArticleCount,
			&profile.QuestionCount, &profile.AnswerCount, &profile.CommentCount, &profile.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return PublicProfile{}, ErrNotFound
	}
	if err != nil {
		return PublicProfile{}, fmt.Errorf("load public profile: %w", err)
	}
	return profile, nil
}

// ListPublic 按推荐（关注者多）或最新注册列出启用账号，仅返回公开字段。
func (r *MySQLRepository) ListPublic(ctx context.Context, sort string, limit, offset int) ([]PublicProfile, bool, error) {
	order := "follower_count DESC, user_id DESC"
	if sort == "new" {
		order = "user_id DESC"
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT user_id, username, COALESCE(avatar_key, ''), COALESCE(cover_key, ''),
			headline, bio, blog, company, location,
			follower_count, followee_count, article_count, question_count, answer_count, comment_count,
			created_at
		FROM vx_user
		WHERE status = 1 AND disabled_at IS NULL
		ORDER BY `+order+`
		LIMIT ? OFFSET ?`, limit+1, offset)
	if err != nil {
		return nil, false, fmt.Errorf("list public users: %w", err)
	}
	defer rows.Close()

	items := make([]PublicProfile, 0, limit)
	for rows.Next() {
		var profile PublicProfile
		if err := rows.Scan(&profile.ID, &profile.Username, &profile.AvatarKey, &profile.CoverKey,
			&profile.Headline, &profile.Bio, &profile.Blog, &profile.Company, &profile.Location,
			&profile.FollowerCount, &profile.FolloweeCount, &profile.ArticleCount,
			&profile.QuestionCount, &profile.AnswerCount, &profile.CommentCount, &profile.CreatedAt); err != nil {
			return nil, false, fmt.Errorf("scan public user: %w", err)
		}
		items = append(items, profile)
	}
	if err := rows.Err(); err != nil {
		return nil, false, fmt.Errorf("iterate public users: %w", err)
	}
	hasMore := len(items) > limit
	if hasMore {
		items = items[:limit]
	}
	return items, hasMore, nil
}
