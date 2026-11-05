package user

import (
	"context"
	"errors"
	"time"
)

var (
	ErrNotFound  = errors.New("user not found")
	ErrDuplicate = errors.New("user already exists")
)

type Account struct {
	ID           int64
	Username     string
	Email        string
	PasswordHash string
	Status       uint8
	AvatarKey    string
	CoverKey     string
	Headline     string
	Bio          string
	Blog         string
	Company      string
	Location     string
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DisabledAt   *time.Time
}

func (a Account) Active() bool {
	return a.Status == 1 && a.DisabledAt == nil
}

// PublicProfile 是公开主页展示的字段集合，不包含邮箱等隐私信息。
type PublicProfile struct {
	ID            int64     `json:"id"`
	Username      string    `json:"username"`
	AvatarKey     string    `json:"avatar_key"`
	CoverKey      string    `json:"cover_key"`
	Headline      string    `json:"headline"`
	Bio           string    `json:"bio"`
	Blog          string    `json:"blog"`
	Company       string    `json:"company"`
	Location      string    `json:"location"`
	FollowerCount int       `json:"follower_count"`
	FolloweeCount int       `json:"followee_count"`
	ArticleCount  int       `json:"article_count"`
	QuestionCount int       `json:"question_count"`
	AnswerCount   int       `json:"answer_count"`
	CommentCount  int       `json:"comment_count"`
	CreatedAt     time.Time `json:"created_at"`
}

type CreateParams struct {
	Username     string
	Email        string
	PasswordHash string
	CreateIP     string
	RoleCode     string
}

type UpdateProfileParams struct {
	Headline *string
	Bio      *string
	Blog     *string
	Company  *string
	Location *string
}

type Repository interface {
	FindByLogin(ctx context.Context, login string) (Account, error)
	FindByID(ctx context.Context, id int64) (Account, error)
	Create(ctx context.Context, params CreateParams) (Account, error)
	SetPassword(ctx context.Context, id int64, passwordHash string) error
	RecordLogin(ctx context.Context, id int64, ip string) error
	UpdateProfile(ctx context.Context, id int64, params UpdateProfileParams) (Account, error)
	Permissions(ctx context.Context, id int64) ([]string, error)
	Roles(ctx context.Context, id int64) ([]string, error)
	EmailExists(ctx context.Context, email string) (bool, error)
	FindPublic(ctx context.Context, id int64) (PublicProfile, error)
	ListPublic(ctx context.Context, sort string, limit, offset int) ([]PublicProfile, bool, error)
	FindByEmail(ctx context.Context, email string) (Account, error)
	SetAvatar(ctx context.Context, id int64, key string) error
	SetCover(ctx context.Context, id int64, key string) error
}
