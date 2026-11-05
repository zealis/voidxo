// Package admin 提供管理后台所需的只读统计、用户与内容治理、话题与配置维护能力。
// 它不重复实现内容写入逻辑：删除内容复用内容模块，删除图片复用媒体模块。
package admin

import (
	"context"
	"errors"
	"time"
)

var (
	ErrNotFound   = errors.New("admin resource not found")
	ErrValidation = errors.New("invalid admin input")
	ErrForbidden  = errors.New("admin operation forbidden")
)

// 内容类型常量与内容模块保持一致，避免前端出现两套命名。
const (
	KindQuestion = "question"
	KindAnswer   = "answer"
	KindArticle  = "article"
	KindComment  = "comment"
)

// 用户状态取值与 vx_user.status 对应。
const (
	StatusActive   uint8 = 1
	StatusDisabled uint8 = 0
)

// 角色编码与 vx_role.code 对应。
var validRoles = map[string]struct{}{
	"admin":     {},
	"moderator": {},
	"user":      {},
}

type Stats struct {
	Users          int `json:"users"`
	DisabledUsers  int `json:"disabled_users"`
	Questions      int `json:"questions"`
	Answers        int `json:"answers"`
	Articles       int `json:"articles"`
	Comments       int `json:"comments"`
	Topics         int `json:"topics"`
	Images         int `json:"images"`
	PendingReports int `json:"pending_reports"`
	NewUsers7d     int `json:"new_users_7d"`
	NewQuestions7d int `json:"new_questions_7d"`
	NewArticles7d  int `json:"new_articles_7d"`
	NewComments7d  int `json:"new_comments_7d"`
}

// Page 与内容模块的分页结构保持一致，前端可复用同一套解析。
type Page[T any] struct {
	Items    []T  `json:"items"`
	Page     int  `json:"page"`
	PageSize int  `json:"page_size"`
	HasMore  bool `json:"has_more"`
}

type User struct {
	ID            int64      `json:"id"`
	Username      string     `json:"username"`
	Email         string     `json:"email"`
	Status        uint8      `json:"status"`
	Headline      string     `json:"headline"`
	Roles         []string   `json:"roles"`
	ArticleCount  int        `json:"article_count"`
	QuestionCount int        `json:"question_count"`
	AnswerCount   int        `json:"answer_count"`
	CommentCount  int        `json:"comment_count"`
	CreatedAt     time.Time  `json:"created_at"`
	LastLoginAt   *time.Time `json:"last_login_at"`
	DisabledAt    *time.Time `json:"disabled_at"`
}

type UserFilter struct {
	Query    string
	Role     string
	Status   string
	Page     int
	PageSize int
}

// UserUpdate 使用指针区分“未提交”和“显式设置”。
type UserUpdate struct {
	Status *uint8
	Role   *string
}

type ContentItem struct {
	ID           int64     `json:"id"`
	Title        string    `json:"title"`
	Excerpt      string    `json:"excerpt"`
	AuthorID     int64     `json:"user_id"`
	Author       string    `json:"username"`
	CommentCount int       `json:"comment_count"`
	VoteCount    int       `json:"vote_count"`
	CreatedAt    time.Time `json:"created_at"`
	Deleted      bool      `json:"deleted"`
	// Context 说明回答所属问题、评论所属内容，列表页直接展示。
	Context string `json:"context"`
	// ParentID/ParentType 指向所属内容，便于后台跳转到前台页面核对。
	ParentID   int64  `json:"parent_id"`
	ParentType string `json:"parent_type,omitempty"`
}

type ContentFilter struct {
	Query string
	// Deleted 取值 live / deleted / all，默认 live。
	Deleted  string
	Page     int
	PageSize int
}

type TopicItem struct {
	ID            int64     `json:"id"`
	Name          string    `json:"name"`
	Description   string    `json:"description"`
	CoverKey      string    `json:"cover_key"`
	ArticleCount  int       `json:"article_count"`
	QuestionCount int       `json:"question_count"`
	FollowerCount int       `json:"follower_count"`
	CreatedAt     time.Time `json:"created_at"`
	Deleted       bool      `json:"deleted"`
}

type OptionItem struct {
	Name  string `json:"name"`
	Value string `json:"value"`
	// Public 表示该配置会出现在公开接口中，后台可据此分组。
	Public bool   `json:"public"`
	Type   string `json:"type"`
}

type ImageItem struct {
	Key       string    `json:"key"`
	Filename  string    `json:"filename"`
	Mime      string    `json:"mime"`
	Size      int       `json:"size"`
	Width     int       `json:"width"`
	Height    int       `json:"height"`
	Kind      string    `json:"kind"`
	UserID    int64     `json:"user_id"`
	Username  string    `json:"username"`
	CreatedAt time.Time `json:"created_at"`
	Deleted   bool      `json:"deleted"`
}

type Repository interface {
	Stats(ctx context.Context) (Stats, error)

	ListUsers(ctx context.Context, filter UserFilter) ([]User, bool, error)
	FindUser(ctx context.Context, id int64) (User, error)
	UpdateUser(ctx context.Context, id int64, update UserUpdate) error
	CountActiveAdmins(ctx context.Context, excludeUserID int64) (int, error)

	ListContent(ctx context.Context, kind string, filter ContentFilter) ([]ContentItem, bool, error)

	ListTopics(ctx context.Context, query string, includeDeleted bool, page, pageSize int) ([]TopicItem, bool, error)
	FindTopic(ctx context.Context, id int64) (TopicItem, error)
	CreateTopic(ctx context.Context, name, description, coverKey string) (int64, error)
	UpdateTopic(ctx context.Context, id int64, name, description, coverKey string) (TopicItem, error)
	DeleteTopic(ctx context.Context, id int64) error

	ListOptions(ctx context.Context) ([]OptionItem, error)
	UpdateOptions(ctx context.Context, values map[string]string) error

	ListImages(ctx context.Context, filter ContentFilter) ([]ImageItem, bool, error)
}
