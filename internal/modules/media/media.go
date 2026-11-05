package media

import (
	"context"
	"errors"
	"time"
)

var (
	ErrNotFound     = errors.New("image not found")
	ErrInvalidImage = errors.New("invalid image")
	ErrTooLarge     = errors.New("image too large")
	ErrForbidden    = errors.New("image operation forbidden")
)

type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}

const (
	KindContent    = "content"
	KindAvatar     = "avatar"
	KindCover      = "cover"
	KindTopicCover = "topic_cover"
)

type Image struct {
	Key       string
	Filename  string
	MIME      string
	Size      int
	Width     int
	Height    int
	Kind      string
	UserID    int64
	CreatedAt time.Time
}

type Repository interface {
	Create(ctx context.Context, image Image) error
	FindByKey(ctx context.Context, key string) (Image, error)
	SoftDelete(ctx context.Context, key string) error
}

// Storage 抽象实际存储；本地开发用文件系统，生产可替换为 S3 兼容实现。
type Storage interface {
	Put(ctx context.Context, key string, data []byte) error
	Get(ctx context.Context, key string) ([]byte, error)
	Delete(ctx context.Context, key string) error
}

type UploadResult struct {
	Key    string            `json:"key"`
	URLs   map[string]string `json:"urls"`
	Width  int               `json:"width"`
	Height int               `json:"height"`
}
