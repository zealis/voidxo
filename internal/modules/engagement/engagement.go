package engagement

import (
	"context"
	"errors"
	"time"
)

var (
	ErrNotFound        = errors.New("engagement target not found")
	ErrDuplicateReport = errors.New("report already submitted")
	ErrSelfFollow      = errors.New("cannot follow yourself")
)

type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}

type PermissionError struct {
	Message string
}

func (e *PermissionError) Error() string {
	return e.Message
}

// 可投票与可关注的目标类型，与旧系统保持一致，用白名单避免表名拼接风险。
const (
	TypeQuestion = "question"
	TypeAnswer   = "answer"
	TypeArticle  = "article"
	TypeComment  = "comment"
	TypeUser     = "user"
	TypeTopic    = "topic"
)

type VoteSummary struct {
	VoteCount     int    `json:"vote_count"`
	VoteUpCount   int    `json:"vote_up_count"`
	VoteDownCount int    `json:"vote_down_count"`
	MyVote        string `json:"my_vote"`
}

type FollowSummary struct {
	Following     bool `json:"following"`
	FollowerCount int  `json:"follower_count"`
}

type Report struct {
	ID             int64     `json:"id"`
	ReportableID   int64     `json:"reportable_id"`
	ReportableType string    `json:"reportable_type"`
	UserID         int64     `json:"user_id"`
	Username       string    `json:"username"`
	Reason         string    `json:"reason"`
	Status         string    `json:"status"`
	CreatedAt      time.Time `json:"created_at"`
}

type Notification struct {
	ID             int64     `json:"id"`
	Type           string    `json:"type"`
	SenderID       int64     `json:"sender_id"`
	SenderName     string    `json:"sender_name"`
	ArticleID      int64     `json:"article_id,omitempty"`
	QuestionID     int64     `json:"question_id,omitempty"`
	AnswerID       int64     `json:"answer_id,omitempty"`
	CommentID      int64     `json:"comment_id,omitempty"`
	ReplyID        int64     `json:"reply_id,omitempty"`
	ContentDeleted string    `json:"content_deleted,omitempty"`
	Read           bool      `json:"read"`
	CreatedAt      time.Time `json:"created_at"`
}

// NotificationEvent 由内容模块在写操作成功后投递，避免内容模块直接依赖通知表。
type NotificationEvent struct {
	ReceiverID int64
	SenderID   int64
	Type       string
	ArticleID  int64
	QuestionID int64
	AnswerID   int64
	CommentID  int64
	ReplyID    int64
	// ContentDeleted 是删除通知中的内容备份，其他类型为空。
	ContentDeleted string
}

type Page[T any] struct {
	Items    []T  `json:"items"`
	Page     int  `json:"page"`
	PageSize int  `json:"page_size"`
	HasMore  bool `json:"has_more"`
}

type Repository interface {
	Vote(ctx context.Context, userID int64, kind string, targetID int64, voteType string) (VoteSummary, error)
	RemoveVote(ctx context.Context, userID int64, kind string, targetID int64) (VoteSummary, error)
	VoteState(ctx context.Context, userID int64, kind string, targetID int64) (VoteSummary, error)
	MyVote(ctx context.Context, userID int64, kind string, targetID int64) (string, error)

	Follow(ctx context.Context, userID int64, kind string, targetID int64) (FollowSummary, error)
	Unfollow(ctx context.Context, userID int64, kind string, targetID int64) (FollowSummary, error)
	FollowState(ctx context.Context, userID int64, kind string, targetID int64) (FollowSummary, error)
	IsFollowing(ctx context.Context, userID int64, kind string, targetID int64) (bool, error)
	FollowStates(ctx context.Context, userID int64, kind string, ids []int64) (map[int64]bool, error)

	CreateReport(ctx context.Context, userID int64, kind string, targetID int64, reason string) (Report, error)
	ListReports(ctx context.Context, status string, limit, offset int) ([]Report, bool, error)
	ResolveReport(ctx context.Context, id, handlerID int64, status string) (Report, error)

	CreateNotification(ctx context.Context, event NotificationEvent) error
	ListNotifications(ctx context.Context, receiverID int64, limit, offset int) ([]Notification, bool, error)
	UnreadNotificationCount(ctx context.Context, receiverID int64) (int, error)
	MarkNotificationsRead(ctx context.Context, receiverID int64, ids []int64) (int, error)
	DeleteNotifications(ctx context.Context, receiverID int64, ids []int64) (int, error)
}
