package media

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// FileStorage 把图片写到本地目录，key 只允许十六进制字符，杜绝路径穿越。
type FileStorage struct {
	root string
}

func NewFileStorage(root string) (*FileStorage, error) {
	if strings.TrimSpace(root) == "" {
		return nil, errors.New("upload dir must not be empty")
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, fmt.Errorf("create upload dir: %w", err)
	}
	return &FileStorage{root: root}, nil
}

func (s *FileStorage) Put(_ context.Context, key string, data []byte) error {
	path, err := s.path(key)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create image dir: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write image: %w", err)
	}
	return nil
}

func (s *FileStorage) Get(_ context.Context, key string) ([]byte, error) {
	path, err := s.path(key)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("read image: %w", err)
	}
	return data, nil
}

func (s *FileStorage) Delete(_ context.Context, key string) error {
	path, err := s.path(key)
	if err != nil {
		return err
	}
	if err := os.RemoveAll(filepath.Dir(path)); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("delete image: %w", err)
	}
	return nil
}

func (s *FileStorage) path(key string) (string, error) {
	// 统一按斜杠规范化，再拒绝绝对路径与上跳，最后转换为本地分隔符。
	clean := path.Clean(key)
	if clean != key || clean == "." || strings.HasPrefix(clean, "../") || path.IsAbs(clean) {
		return "", ErrInvalidImage
	}
	return filepath.Join(s.root, filepath.FromSlash(clean)), nil
}
