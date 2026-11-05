package content

import (
	"context"
	"errors"
	"time"
)

var (
	ErrNotFound      = errors.New("content not found")
	ErrTopicNotFound = errors.New("topic not found")
	ErrTopicExists   = errors.New("topic already exists")
)

// PermissionError 表示已登录但无权执行该操作，与未登录的 401 区分开。
type PermissionError struct {
	Message string
}

func (e *PermissionError) Error() string {
	return e.Message
}

type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}

type TopicRef struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type Topic struct {
	ID            int64     `json:"id"`
	Name          string    `json:"name"`
	Description   string    `json:"description"`
	CoverKey      string    `json:"cover_key"`
	ArticleCount  int       `json:"article_count"`
	QuestionCount int       `json:"question_count"`
	FollowerCount int       `json:"follower_count"`
	CreatedAt     time.Time `json:"created_at"`
}

type Question struct {
	ID              int64      `json:"id"`
	UserID          int64      `json:"user_id"`
	Username        string     `json:"username"`
	AvatarKey       string     `json:"avatar_key"`
	Title           string     `json:"title"`
	ContentMarkdown string     `json:"content_markdown,omitempty"`
	ContentRendered string     `json:"content_rendered,omitempty"`
	CommentCount    int        `json:"comment_count"`
	AnswerCount     int        `json:"answer_count"`
	FollowerCount   int        `json:"follower_count"`
	VoteCount       int        `json:"vote_count"`
	LastAnswerAt    *time.Time `json:"last_answer_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	Topics          []TopicRef `json:"topics"`
}

type Answer struct {
	ID              int64     `json:"id"`
	QuestionID      int64     `json:"question_id"`
	UserID          int64     `json:"user_id"`
	Username        string    `json:"username"`
	AvatarKey       string    `json:"avatar_key"`
	ContentMarkdown string    `json:"content_markdown,omitempty"`
	ContentRendered string    `json:"content_rendered"`
	CommentCount    int       `json:"comment_count"`
	VoteCount       int       `json:"vote_count"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type Page[T any] struct {
	Items    []T  `json:"items"`
	Page     int  `json:"page"`
	PageSize int  `json:"page_size"`
	HasMore  bool `json:"has_more"`
}

// Article 是文章列表与详情共用的结构。
// ContentRendered 仅在详情接口返回，列表接口留空以控制响应体积。
type Article struct {
	ID              int64      `json:"id"`
	UserID          int64      `json:"user_id"`
	Username        string     `json:"username"`
	AvatarKey       string     `json:"avatar_key"`
	Title           string     `json:"title"`
	ContentMarkdown string     `json:"content_markdown,omitempty"`
	ContentRendered string     `json:"content_rendered,omitempty"`
	CommentCount    int        `json:"comment_count"`
	FollowerCount   int        `json:"follower_count"`
	VoteCount       int        `json:"vote_count"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	Topics          []TopicRef `json:"topics"`
}

// Comment 的正文是纯文本，入库前不做 Markdown 渲染，输岀时由前端转义。
type Comment struct {
	ID              int64     `json:"id"`
	CommentableID   int64     `json:"commentable_id"`
	CommentableType string    `json:"commentable_type"`
	RootID          *int64    `json:"root_id,omitempty"`
	UserID          int64     `json:"user_id"`
	Username        string    `json:"username"`
	AvatarKey       string    `json:"avatar_key"`
	Content         string    `json:"content"`
	ReplyCount      int       `json:"reply_count"`
	VoteCount       int       `json:"vote_count"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// ContentMeta 汇总权限判定所需的元信息，避免为每类内容重复查询。
type ContentMeta struct {
	ID           int64
	UserID       int64
	CreatedAt    time.Time
	CommentCount int
	AnswerCount  int
	// Content 是内容的渲染结果，仅在删除时用于通知备份。
	Content string
}

// UserContentItem 是公开主页列出的单条内容，统一四类内容的展示字段。
type UserContentItem struct {
	Kind         string    `json:"kind"`
	ID           int64     `json:"id"`
	Title        string    `json:"title"`
	Excerpt      string    `json:"excerpt"`
	TargetType   string    `json:"target_type,omitempty"`
	TargetID     int64     `json:"target_id,omitempty"`
	CommentCount int       `json:"comment_count"`
	VoteCount    int       `json:"vote_count"`
	CreatedAt    time.Time `json:"created_at"`
}

// SitemapEntry 是公开可索引内容的索引项，供 sitemap.xml 使用。
type SitemapEntry struct {
	Type      string    `json:"type"`
	ID        int64     `json:"id"`
	UpdatedAt time.Time `json:"updated_at"`
}

// AuthorContent 描述被级联删除内容的作者与正文。
type AuthorContent struct {
	ID      int64
	UserID  int64
	Content string
}

const (
	KindQuestion = "question"
	KindAnswer   = "answer"
	KindArticle  = "article"
	KindComment  = "comment"
)

func ValidContentKind(kind string) bool {
	switch kind {
	case KindQuestion, KindAnswer, KindArticle, KindComment:
		return true
	}
	return false
}

type Repository interface {
	ListTopics(ctx context.Context, sort string, limit, offset int) ([]Topic, bool, error)
	FindTopic(ctx context.Context, id int64) (Topic, error)
	CreateTopic(ctx context.Context, name, description, coverKey string) (Topic, error)
	CountTopics(ctx context.Context, ids []int64) (int, error)

	ListQuestions(ctx context.Context, topicID *int64, sort string, limit, offset int) ([]Question, bool, error)
	FindQuestion(ctx context.Context, id int64) (Question, error)
	CreateQuestion(ctx context.Context, userID int64, title, markdown, rendered string, topicIDs []int64) (Question, error)
	UpdateQuestion(ctx context.Context, id int64, title, markdown, rendered string, topicIDs []int64) (Question, error)
	DeleteQuestion(ctx context.Context, id int64) error
	QuestionExists(ctx context.Context, id int64) (bool, error)

	ListAnswers(ctx context.Context, questionID int64, limit, offset int) ([]Answer, bool, error)
	FindAnswer(ctx context.Context, id int64) (Answer, error)
	CreateAnswer(ctx context.Context, questionID, userID int64, markdown, rendered string) (Answer, error)
	UpdateAnswer(ctx context.Context, id int64, markdown, rendered string) (Answer, error)
	DeleteAnswer(ctx context.Context, id int64) error

	ListArticles(ctx context.Context, topicID *int64, sort string, limit, offset int) ([]Article, bool, error)
	FindArticle(ctx context.Context, id int64) (Article, error)
	CreateArticle(ctx context.Context, userID int64, title, markdown, rendered string, topicIDs []int64) (Article, error)
	UpdateArticle(ctx context.Context, id int64, title, markdown, rendered string, topicIDs []int64) (Article, error)
	DeleteArticle(ctx context.Context, id int64) error

	ListComments(ctx context.Context, kind string, targetID int64, limit, offset int) ([]Comment, bool, error)
	FindComment(ctx context.Context, id int64) (Comment, error)
	CreateComment(ctx context.Context, kind string, targetID, userID int64, content string, rootID *int64) (Comment, error)
	UpdateComment(ctx context.Context, id int64, content string) (Comment, error)
	DeleteComment(ctx context.Context, id int64) error

	Meta(ctx context.Context, kind string, id int64) (ContentMeta, error)
	CascadeAuthors(ctx context.Context, kind string, id int64) ([]AuthorContent, error)
	ListSitemap(ctx context.Context, limit int) ([]SitemapEntry, error)
	ListUserContent(ctx context.Context, kind string, userID int64, limit, offset int) ([]UserContentItem, bool, error)
}
