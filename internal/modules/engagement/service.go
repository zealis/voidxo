package engagement

import (
	"context"
	"strings"
	"unicode/utf8"
)

const (
	defaultPageSize     = 20
	maxPageSize         = 50
	maxBatchSize        = 100
	maxReportReasonRune = 200
)

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func validVoteTarget(kind string) bool {
	switch kind {
	case TypeQuestion, TypeAnswer, TypeArticle, TypeComment:
		return true
	}
	return false
}

func validFollowTarget(kind string) bool {
	switch kind {
	case TypeUser, TypeQuestion, TypeArticle, TypeTopic:
		return true
	}
	return false
}

func validReportTarget(kind string) bool {
	switch kind {
	case TypeQuestion, TypeAnswer, TypeArticle, TypeComment, TypeUser:
		return true
	}
	return false
}

func (s *Service) Vote(ctx context.Context, userID int64, kind string, targetID int64, voteType string) (VoteSummary, error) {
	if userID <= 0 {
		return VoteSummary{}, &PermissionError{Message: "请先登录"}
	}
	if !validVoteTarget(kind) || targetID <= 0 {
		return VoteSummary{}, ErrNotFound
	}
	if voteType != "up" && voteType != "down" {
		return VoteSummary{}, &ValidationError{Message: "投票类型不正确"}
	}
	return s.repo.Vote(ctx, userID, kind, targetID, voteType)
}

func (s *Service) VoteState(ctx context.Context, userID int64, kind string, targetID int64) (VoteSummary, error) {
	if !validVoteTarget(kind) || targetID <= 0 {
		return VoteSummary{}, ErrNotFound
	}
	if userID <= 0 {
		return VoteSummary{}, &PermissionError{Message: "请先登录"}
	}
	return s.repo.VoteState(ctx, userID, kind, targetID)
}

func (s *Service) RemoveVote(ctx context.Context, userID int64, kind string, targetID int64) (VoteSummary, error) {
	if userID <= 0 {
		return VoteSummary{}, &PermissionError{Message: "请先登录"}
	}
	if !validVoteTarget(kind) || targetID <= 0 {
		return VoteSummary{}, ErrNotFound
	}
	return s.repo.RemoveVote(ctx, userID, kind, targetID)
}

func (s *Service) Follow(ctx context.Context, userID int64, kind string, targetID int64) (FollowSummary, error) {
	if userID <= 0 {
		return FollowSummary{}, &PermissionError{Message: "请先登录"}
	}
	if !validFollowTarget(kind) || targetID <= 0 {
		return FollowSummary{}, ErrNotFound
	}
	if kind == TypeUser && targetID == userID {
		return FollowSummary{}, ErrSelfFollow
	}
	return s.repo.Follow(ctx, userID, kind, targetID)
}

// FollowStates 批量返回关注状态，最多 50 个目标。
func (s *Service) FollowStates(ctx context.Context, userID int64, kind string, ids []int64) (map[int64]bool, error) {
	if !validFollowTarget(kind) || len(ids) == 0 || len(ids) > 50 {
		return nil, &ValidationError{Message: "关注目标不正确"}
	}
	for _, id := range ids {
		if id <= 0 {
			return nil, &ValidationError{Message: "关注目标不正确"}
		}
	}
	if userID <= 0 {
		return nil, &PermissionError{Message: "请先登录"}
	}
	return s.repo.FollowStates(ctx, userID, kind, ids)
}

func (s *Service) FollowState(ctx context.Context, userID int64, kind string, targetID int64) (FollowSummary, error) {
	if !validFollowTarget(kind) || targetID <= 0 {
		return FollowSummary{}, ErrNotFound
	}
	if userID <= 0 {
		return FollowSummary{}, &PermissionError{Message: "请先登录"}
	}
	return s.repo.FollowState(ctx, userID, kind, targetID)
}

func (s *Service) Unfollow(ctx context.Context, userID int64, kind string, targetID int64) (FollowSummary, error) {
	if userID <= 0 {
		return FollowSummary{}, &PermissionError{Message: "请先登录"}
	}
	if !validFollowTarget(kind) || targetID <= 0 {
		return FollowSummary{}, ErrNotFound
	}
	return s.repo.Unfollow(ctx, userID, kind, targetID)
}

func (s *Service) CreateReport(ctx context.Context, userID int64, kind string, targetID int64, reason string) (Report, error) {
	if userID <= 0 {
		return Report{}, &PermissionError{Message: "请先登录"}
	}
	if !validReportTarget(kind) || targetID <= 0 {
		return Report{}, ErrNotFound
	}
	reason = strings.TrimSpace(reason)
	if count := utf8.RuneCountInString(reason); count < 2 || count > maxReportReasonRune {
		return Report{}, &ValidationError{Message: "举报原因需为 2 至 200 个字符"}
	}
	return s.repo.CreateReport(ctx, userID, kind, targetID, reason)
}

func (s *Service) ListReports(ctx context.Context, status string, page, pageSize int) (Page[Report], error) {
	switch status {
	case "", "pending", "resolved", "dismissed":
	default:
		return Page[Report]{}, &ValidationError{Message: "举报状态不正确"}
	}
	page, pageSize = normalizePage(page, pageSize)
	items, hasMore, err := s.repo.ListReports(ctx, status, pageSize, (page-1)*pageSize)
	if err != nil {
		return Page[Report]{}, err
	}
	items = ensureItems(items)
	return Page[Report]{Items: items, Page: page, PageSize: pageSize, HasMore: hasMore}, nil
}

func (s *Service) ResolveReport(ctx context.Context, id, handlerID int64, status string) (Report, error) {
	if id <= 0 || handlerID <= 0 {
		return Report{}, ErrNotFound
	}
	if status != "resolved" && status != "dismissed" {
		return Report{}, &ValidationError{Message: "处理结果不正确"}
	}
	return s.repo.ResolveReport(ctx, id, handlerID, status)
}

func (s *Service) Notify(ctx context.Context, event NotificationEvent) error {
	// 自己触发的事件不通知自己。
	if event.ReceiverID <= 0 || event.ReceiverID == event.SenderID {
		return nil
	}
	return s.repo.CreateNotification(ctx, event)
}

func (s *Service) ListNotifications(ctx context.Context, receiverID int64, page, pageSize int) (Page[Notification], error) {
	page, pageSize = normalizePage(page, pageSize)
	items, hasMore, err := s.repo.ListNotifications(ctx, receiverID, pageSize, (page-1)*pageSize)
	if err != nil {
		return Page[Notification]{}, err
	}
	items = ensureItems(items)
	return Page[Notification]{Items: items, Page: page, PageSize: pageSize, HasMore: hasMore}, nil
}

func (s *Service) UnreadCount(ctx context.Context, receiverID int64) (int, error) {
	return s.repo.UnreadNotificationCount(ctx, receiverID)
}

func (s *Service) MarkRead(ctx context.Context, receiverID int64, ids []int64) (int, error) {
	ids = normalizeIDs(ids)
	if len(ids) == 0 {
		return 0, &ValidationError{Message: "请选择要标记的通知"}
	}
	if len(ids) > maxBatchSize {
		return 0, &ValidationError{Message: "单次最多处理 100 条通知"}
	}
	// repository 始终带上 receiver_id 条件，避免旧系统批量操作越权的问题。
	return s.repo.MarkNotificationsRead(ctx, receiverID, ids)
}

func (s *Service) DeleteNotifications(ctx context.Context, receiverID int64, ids []int64) (int, error) {
	ids = normalizeIDs(ids)
	if len(ids) == 0 {
		return 0, &ValidationError{Message: "请选择要删除的通知"}
	}
	if len(ids) > maxBatchSize {
		return 0, &ValidationError{Message: "单次最多处理 100 条通知"}
	}
	return s.repo.DeleteNotifications(ctx, receiverID, ids)
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
		page = 1
	}
	if pageSize < 1 {
		pageSize = defaultPageSize
	}
	if pageSize > maxPageSize {
		pageSize = maxPageSize
	}
	return page, pageSize
}

func normalizeIDs(ids []int64) []int64 {
	seen := make(map[int64]struct{}, len(ids))
	result := make([]int64, 0, len(ids))
	for _, id := range ids {
		if id <= 0 {
			continue
		}
		if _, ok := seen[id]; !ok {
			seen[id] = struct{}{}
			result = append(result, id)
		}
	}
	return result
}
