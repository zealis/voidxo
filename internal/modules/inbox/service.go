package inbox

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"unicode/utf8"

	"github.com/voidxo/voidxo/internal/platform/markdown"
)

const (
	defaultPage     = 1
	defaultPageSize = 20
	maxPageSize     = 50
	// maxContentLength 与旧系统私信正文的展示宽度匹配，避免单条消息过长。
	maxContentLength = 1000
)

// Limiter 由 platform/ratelimit 实现，用于限制单个用户短时间内的发送频率。
type Limiter interface {
	Allow(ctx context.Context, subject string) (bool, error)
}

type Service struct {
	repo     Repository
	renderer *markdown.Renderer
	limiter  Limiter
	logger   *slog.Logger
}

func NewService(repo Repository, renderer *markdown.Renderer, logger *slog.Logger) *Service {
	return &Service{repo: repo, renderer: renderer, logger: logger}
}

func (s *Service) SetLimiter(limiter Limiter) {
	s.limiter = limiter
}

func (s *Service) Send(ctx context.Context, senderID, receiverID int64, source string) (Message, error) {
	if senderID <= 0 || receiverID <= 0 {
		return Message{}, ErrRecipientNotFound
	}
	if senderID == receiverID {
		return Message{}, &ValidationError{Message: "不能给自己发送私信"}
	}
	recipient, err := s.repo.FindRecipientByID(ctx, receiverID)
	if err != nil {
		return Message{}, err
	}
	if !recipient.Active {
		return Message{}, &ValidationError{Message: "该用户已停用，无法接收私信"}
	}
	source = strings.TrimSpace(source)
	if count := utf8.RuneCountInString(source); count < 1 || count > maxContentLength {
		return Message{}, &ValidationError{Message: fmt.Sprintf("私信内容需要 1 至 %d 个字", maxContentLength)}
	}
	rendered, plain, err := s.renderer.Render(source)
	if err != nil {
		return Message{}, fmt.Errorf("render inbox message: %w", err)
	}
	if strings.TrimSpace(plain) == "" {
		return Message{}, &ValidationError{Message: "私信内容不能为空"}
	}
	if s.limiter != nil {
		allowed, err := s.limiter.Allow(ctx, fmt.Sprintf("user:%d", senderID))
		if err != nil {
			// 限流不可用时放行私信，避免基础设施故障阻断核心功能；频率风险由后续审计覆盖。
			s.logger.Warn("inbox limiter unavailable", "error", err, "user_id", senderID)
		} else if !allowed {
			return Message{}, ErrRateLimited
		}
	}
	return s.repo.CreateMessage(ctx, senderID, receiverID, source, rendered)
}

func (s *Service) Conversations(ctx context.Context, userID int64, page, pageSize int) (Page[Conversation], error) {
	page, pageSize = normalizePage(page, pageSize)
	items, hasMore, err := s.repo.ListConversations(ctx, userID, pageSize, (page-1)*pageSize)
	if err != nil {
		return Page[Conversation]{}, err
	}
	items = ensureItems(items)
	return Page[Conversation]{Items: items, Page: page, PageSize: pageSize, HasMore: hasMore}, nil
}

func (s *Service) Messages(ctx context.Context, userID, counterpartID int64, page, pageSize int) (ConversationPage, error) {
	if userID <= 0 || counterpartID <= 0 {
		return ConversationPage{}, ErrRecipientNotFound
	}
	counterpart, err := s.repo.FindRecipientByID(ctx, counterpartID)
	if err != nil {
		return ConversationPage{}, err
	}
	if counterpartID == userID {
		return ConversationPage{}, &ValidationError{Message: "不能与自己对话"}
	}
	page, pageSize = normalizePage(page, pageSize)
	items, hasMore, err := s.repo.ListMessages(ctx, userID, counterpartID, pageSize, (page-1)*pageSize)
	if err != nil {
		return ConversationPage{}, err
	}
	items = ensureItems(items)
	return ConversationPage{
		Counterpart: counterpart,
		Page:        Page[Message]{Items: items, Page: page, PageSize: pageSize, HasMore: hasMore},
	}, nil
}

func (s *Service) MarkRead(ctx context.Context, userID, counterpartID int64) (int, int, error) {
	if userID <= 0 || counterpartID <= 0 {
		return 0, 0, ErrRecipientNotFound
	}
	updated, err := s.repo.MarkConversationRead(ctx, userID, counterpartID)
	if err != nil {
		return 0, 0, err
	}
	unread, err := s.repo.UnreadCount(ctx, userID)
	if err != nil {
		return 0, 0, err
	}
	return updated, unread, nil
}

func (s *Service) UnreadCount(ctx context.Context, userID int64) (int, error) {
	if userID <= 0 {
		return 0, nil
	}
	return s.repo.UnreadCount(ctx, userID)
}

func (s *Service) RecipientByUsername(ctx context.Context, username string) (Recipient, error) {
	username = strings.TrimSpace(username)
	if username == "" {
		return Recipient{}, &ValidationError{Message: "请填写用户名"}
	}
	recipient, err := s.repo.FindRecipientByUsername(ctx, username)
	if err != nil {
		return Recipient{}, err
	}
	if !recipient.Active {
		return Recipient{}, &ValidationError{Message: "该用户已停用，无法接收私信"}
	}
	return recipient, nil
}

// ensureItems 保证空列表序列化为 []，与前端分页解析保持一致。
func ensureItems[T any](items []T) []T {
	if items == nil {
		return []T{}
	}
	return items
}

func normalizePage(page, pageSize int) (int, int) {
	if page < 1 {
		page = defaultPage
	}
	if pageSize < 1 {
		pageSize = defaultPageSize
	}
	if pageSize > maxPageSize {
		pageSize = maxPageSize
	}
	return page, pageSize
}
