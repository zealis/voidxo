package content

import (
	"context"
	"log/slog"
)

// NotificationEvent 描述一次内容事件对通知模块的最小输入。
// 内容模块只声明需要什么，不依赖通知模块的具体实现。
type NotificationEvent struct {
	ReceiverID int64
	SenderID   int64
	Type       string
	ArticleID  int64
	QuestionID int64
	AnswerID   int64
	CommentID  int64
	ReplyID    int64
	// ContentDeleted 保存被删除内容的渲染结果，供作者在通知中查看备份。
	ContentDeleted string
}

type Notifier interface {
	Notify(ctx context.Context, event NotificationEvent) error
}

// SetNotifier 允许在装配阶段注入通知实现；未注入时内容写入仍然成功。
func (s *Service) SetNotifier(notifier Notifier) {
	s.notifier = notifier
}

func (s *Service) notify(ctx context.Context, event NotificationEvent) {
	if s.notifier == nil || event.ReceiverID <= 0 || event.ReceiverID == event.SenderID {
		return
	}
	if err := s.notifier.Notify(ctx, event); err != nil {
		// 通知失败不能回滚已经提交的内容写入，只记录日志。
		slog.Default().Warn("failed to deliver content notification", "type", event.Type, "error", err)
	}
}
