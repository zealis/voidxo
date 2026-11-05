package media

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

type MySQLRepository struct {
	db *sql.DB
}

func NewMySQLRepository(db *sql.DB) *MySQLRepository {
	return &MySQLRepository{db: db}
}

func (r *MySQLRepository) Create(ctx context.Context, image Image) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO vx_image (storage_key, filename, mime, size, width, height, kind, user_id)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		image.Key, image.Filename, image.MIME, image.Size, image.Width, image.Height, image.Kind, image.UserID)
	if err != nil {
		return fmt.Errorf("insert image: %w", err)
	}
	return nil
}

func (r *MySQLRepository) FindByKey(ctx context.Context, key string) (Image, error) {
	var image Image
	err := r.db.QueryRowContext(ctx, `
		SELECT storage_key, filename, mime, size, width, height, kind, user_id, created_at
		FROM vx_image WHERE storage_key = ? AND deleted_at IS NULL LIMIT 1`, key).
		Scan(&image.Key, &image.Filename, &image.MIME, &image.Size, &image.Width,
			&image.Height, &image.Kind, &image.UserID, &image.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return Image{}, ErrNotFound
	}
	if err != nil {
		return Image{}, fmt.Errorf("find image: %w", err)
	}
	return image, nil
}

func (r *MySQLRepository) SoftDelete(ctx context.Context, key string) error {
	result, err := r.db.ExecContext(ctx,
		"UPDATE vx_image SET deleted_at = CURRENT_TIMESTAMP(6) WHERE storage_key = ? AND deleted_at IS NULL", key)
	if err != nil {
		return fmt.Errorf("soft delete image: %w", err)
	}
	if rows, err := result.RowsAffected(); err != nil {
		return fmt.Errorf("delete image rows: %w", err)
	} else if rows == 0 {
		return ErrNotFound
	}
	return nil
}
