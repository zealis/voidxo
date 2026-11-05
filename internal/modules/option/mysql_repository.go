package option

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

type MySQLRepository struct {
	db    *sql.DB
	table string
}

func NewMySQLRepository(db *sql.DB, tablePrefix string) *MySQLRepository {
	return &MySQLRepository{db: db, table: tablePrefix + "option"}
}

func (r *MySQLRepository) FindByNames(ctx context.Context, names []string) (map[string]string, error) {
	result := make(map[string]string, len(names))
	if len(names) == 0 {
		return result, nil
	}
	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(names)), ",")
	args := make([]any, len(names))
	for i, name := range names {
		args[i] = name
	}
	query := fmt.Sprintf("SELECT name, value FROM `%s` WHERE name IN (%s) ORDER BY name", r.table, placeholders)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var name, value string
		if err := rows.Scan(&name, &value); err != nil {
			return nil, err
		}
		result[name] = value
	}
	return result, rows.Err()
}
