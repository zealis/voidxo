// Package inbox 实现用户之间的私信：会话列表、会话消息、发送与已读标记。
// 数据模型沿用旧系统的扁平消息表，按收件双方聚合出会话，不额外引入会话表。
package inbox

import (
	"context"
	"errors"
	"time"
)

var (
	ErrRecipientNotFound = errors.New("recipient not found")
	ErrRateLimited       = errors.New("inbox rate limited")
)

// ValidationError 表示私信入参不合法，处理器映射为 400。
type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}

type Recipient struct {
	ID        int64  `json:"id"`
	Username  string `json:"username"`
	AvatarKey string `json:"avatar_key"`
	Active    bool   `json:"-"`
}

type Message struct {
	ID              int64     `json:"id"`
	SenderID        int64     `json:"sender_id"`
	SenderName      string    `json:"sender_name"`
	ReceiverID      int64     `json:"receiver_id"`
	ContentMarkdown string    `json:"content_markdown"`
	ContentRendered string    `json:"content_rendered"`
	Read            bool      `json:"read"`
	CreatedAt       time.Time `json:"created_at"`
}

type Conversation struct {
	UserID      int64     `json:"user_id"`
	Username    string    `json:"username"`
	AvatarKey   string    `json:"avatar_key"`
	LastExcerpt string    `json:"last_excerpt"`
	LastSender  int64     `json:"last_sender_id"`
	LastAt      time.Time `json:"last_at"`
	Unread      int       `json:"unread"`
}

// Page 与其他模块的分页结构保持一致。
type Page[T any] struct {
	Items    []T  `json:"items"`
	Page     int  `json:"page"`
	PageSize int  `json:"page_size"`
	HasMore  bool `json:"has_more"`
}

// ConversationPage 在分页之外带上会话对象，便于前端渲染标题。
type ConversationPage struct {
	Counterpart Recipient `json:"counterpart"`
	Page[Message]
}

type Repository interface {
	FindRecipientByID(ctx context.Context, id int64) (Recipient, error)
	FindRecipientByUsername(ctx context.Context, username string) (Recipient, error)
	CreateMessage(ctx context.Context, senderID, receiverID int64, markdown, rendered string) (Message, error)
	ListConversations(ctx context.Context, userID int64, limit, offset int) ([]Conversation, bool, error)
	ListMessages(ctx context.Context, userID, counterpartID int64, limit, offset int) ([]Message, bool, error)
	MarkConversationRead(ctx context.Context, userID, counterpartID int64) (int, error)
	UnreadCount(ctx context.Context, userID int64) (int, error)
}
