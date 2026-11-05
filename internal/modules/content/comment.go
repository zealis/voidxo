package content

import (
	"context"
	"strings"
	"unicode/utf8"

	"github.com/voidxo/voidxo/internal/modules/auth"
)

const maxCommentLength = 1000

// CommentTargets 列出允许评论的目标类型，与旧系统的四种目标保持一致。
func ValidCommentTarget(kind string) bool {
	switch kind {
	case KindArticle, KindQuestion, KindAnswer, KindComment:
		return true
	}
	return false
}

func (s *Service) ListComments(ctx context.Context, kind string, targetID int64, page, pageSize int) (Page[Comment], error) {
	if !ValidCommentTarget(kind) || targetID <= 0 {
		return Page[Comment]{}, ErrNotFound
	}
	if _, err := s.repo.Meta(ctx, kind, targetID); err != nil {
		return Page[Comment]{}, err
	}
	page, pageSize = normalizePage(page, pageSize)
	items, hasMore, err := s.repo.ListComments(ctx, kind, targetID, pageSize, (page-1)*pageSize)
	if err != nil {
		return Page[Comment]{}, err
	}
	items = ensureItems(items)
	return Page[Comment]{Items: items, Page: page, PageSize: pageSize, HasMore: hasMore}, nil
}

func (s *Service) CreateComment(ctx context.Context, userID int64, kind string, targetID int64, content string, rootID *int64) (Comment, error) {
	if userID <= 0 {
		return Comment{}, ErrNotFound
	}
	if !ValidCommentTarget(kind) || targetID <= 0 {
		return Comment{}, ErrNotFound
	}
	content = strings.TrimSpace(content)
	if count := utf8.RuneCountInString(content); count < 1 || count > maxCommentLength {
		return Comment{}, &ValidationError{Message: "评论需为 1 至 1000 个字符"}
	}
	target, err := s.repo.Meta(ctx, kind, targetID)
	if err != nil {
		return Comment{}, err
	}
	var parentAuthor int64
	if rootID != nil {
		// 只允许两级结构：对评论的回复统一挂到同一顶层评论下，
		// 避免旧系统无限嵌套导致列表查询退化为递归。
		if *rootID <= 0 {
			return Comment{}, &ValidationError{Message: "回复目标不正确"}
		}
		parent, err := s.repo.FindComment(ctx, *rootID)
		if err != nil {
			return Comment{}, err
		}
		if parent.CommentableType != kind || parent.CommentableID != targetID {
			return Comment{}, &ValidationError{Message: "回复目标不属于当前内容"}
		}
		if parent.RootID != nil {
			rootID = parent.RootID
		}
		parentAuthor = parent.UserID
	}
	comment, err := s.repo.CreateComment(ctx, kind, targetID, userID, content, rootID)
	if err != nil {
		return Comment{}, err
	}
	if rootID != nil {
		s.notify(ctx, NotificationEvent{
			ReceiverID: parentAuthor,
			SenderID:   userID,
			Type:       "comment_replied",
			ArticleID:  targetIDFor(kind, KindArticle, targetID),
			QuestionID: targetIDFor(kind, KindQuestion, targetID),
			AnswerID:   targetIDFor(kind, KindAnswer, targetID),
			CommentID:  comment.ID,
			ReplyID:    *rootID,
		})
	} else if kind != KindComment {
		s.notify(ctx, NotificationEvent{
			ReceiverID: target.UserID,
			SenderID:   userID,
			Type:       kind + "_commented",
			ArticleID:  targetIDFor(kind, KindArticle, targetID),
			QuestionID: targetIDFor(kind, KindQuestion, targetID),
			AnswerID:   targetIDFor(kind, KindAnswer, targetID),
			CommentID:  comment.ID,
		})
	}
	return comment, nil
}

// targetIDFor 只在与内容类型匹配时填入编号，其余保持 0，避免跨类型误关联。
func targetIDFor(kind, want string, id int64) int64 {
	if kind == want {
		return id
	}
	return 0
}

func (s *Service) UpdateComment(ctx context.Context, principal auth.Principal, id int64, content string) (Comment, error) {
	meta, err := s.repo.Meta(ctx, KindComment, id)
	if err != nil {
		return Comment{}, err
	}
	if err := s.authorizeModification(ctx, principal, KindComment, "edit", meta); err != nil {
		return Comment{}, err
	}
	content = strings.TrimSpace(content)
	if count := utf8.RuneCountInString(content); count < 1 || count > maxCommentLength {
		return Comment{}, &ValidationError{Message: "评论需为 1 至 1000 个字符"}
	}
	return s.repo.UpdateComment(ctx, id, content)
}

func (s *Service) DeleteComment(ctx context.Context, principal auth.Principal, id int64) error {
	meta, err := s.repo.Meta(ctx, KindComment, id)
	if err != nil {
		return err
	}
	if err := s.authorizeModification(ctx, principal, KindComment, "delete", meta); err != nil {
		return err
	}
	cascade, err := s.repo.CascadeAuthors(ctx, KindComment, id)
	if err != nil {
		return err
	}
	target := s.replyTarget(ctx, id)
	if err := s.repo.DeleteComment(ctx, id); err != nil {
		return err
	}
	s.notify(ctx, NotificationEvent{
		ReceiverID:     meta.UserID,
		SenderID:       principal.UserID,
		Type:           "comment_deleted",
		ArticleID:      targetIDFor(target.CommentableType, KindArticle, target.CommentableID),
		QuestionID:     targetIDFor(target.CommentableType, KindQuestion, target.CommentableID),
		AnswerID:       targetIDFor(target.CommentableType, KindAnswer, target.CommentableID),
		CommentID:      id,
		ContentDeleted: meta.Content,
	})
	for _, reply := range cascade {
		s.notify(ctx, NotificationEvent{
			ReceiverID:     reply.UserID,
			SenderID:       principal.UserID,
			Type:           "comment_deleted",
			CommentID:      reply.ID,
			ReplyID:        id,
			ContentDeleted: reply.Content,
		})
	}
	return nil
}

// replyTarget 在删除评论前读取其所属目标，用于把通知关联到正确的内容。
func (s *Service) replyTarget(ctx context.Context, id int64) Comment {
	comment, err := s.repo.FindComment(ctx, id)
	if err != nil {
		return Comment{}
	}
	return comment
}
