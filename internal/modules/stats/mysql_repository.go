package stats

import (
	"context"
	"database/sql"
	"fmt"
)

type MySQLRepository struct {
	db    *sql.DB
	table string
}

func NewMySQLRepository(db *sql.DB, tablePrefix string) *MySQLRepository {
	return &MySQLRepository{db: db, table: tablePrefix}
}

func (r *MySQLRepository) Counts(ctx context.Context) (Stats, error) {
	var result Stats
	// 只统计未删除内容与启用账号，口径与前台列表一致。
	err := r.db.QueryRowContext(ctx, `
		SELECT
			(SELECT COUNT(*) FROM `+r.t("user")+` WHERE status = 1 AND disabled_at IS NULL),
			(SELECT COUNT(*) FROM `+r.t("question")+` WHERE deleted_at IS NULL),
			(SELECT COUNT(*) FROM `+r.t("answer")+` WHERE deleted_at IS NULL),
			(SELECT COUNT(*) FROM `+r.t("article")+` WHERE deleted_at IS NULL),
			(SELECT COUNT(*) FROM `+r.t("comment")+` WHERE deleted_at IS NULL),
			(SELECT COUNT(*) FROM `+r.t("topic")+` WHERE deleted_at IS NULL)`).
		Scan(&result.Users, &result.Questions, &result.Answers, &result.Articles, &result.Comments, &result.Topics)
	if err != nil {
		return Stats{}, fmt.Errorf("load site stats: %w", err)
	}
	return result, nil
}

func (r *MySQLRepository) t(name string) string {
	return r.table + name
}
