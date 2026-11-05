package admin

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/voidxo/voidxo/internal/modules/auth"
	"github.com/voidxo/voidxo/internal/modules/option"
)

const (
	defaultPage     = 1
	defaultPageSize = 20
	maxPageSize     = 100
)

// ContentModerator 由内容模块实现，后台只借用其删除能力（含计数与通知）。
type ContentModerator interface {
	DeleteQuestion(ctx context.Context, principal auth.Principal, id int64) error
	DeleteAnswer(ctx context.Context, principal auth.Principal, id int64) error
	DeleteArticle(ctx context.Context, principal auth.Principal, id int64) error
	DeleteComment(ctx context.Context, principal auth.Principal, id int64) error
}

// MediaDeleter 由媒体模块实现，isManager 为 true 时跳过归属校验。
type MediaDeleter interface {
	Delete(ctx context.Context, userID int64, isManager bool, key string) error
}

// SessionRevoker 由认证模块实现，用于在角色或状态变更后立即失效旧会话。
type SessionRevoker interface {
	RevokeUserSessions(ctx context.Context, userID int64, reason string) error
}

type Service struct {
	repo     Repository
	content  ContentModerator
	media    MediaDeleter
	sessions SessionRevoker
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) SetContentModerator(moderator ContentModerator) {
	s.content = moderator
}

func (s *Service) SetMediaDeleter(deleter MediaDeleter) {
	s.media = deleter
}

func (s *Service) SetSessionRevoker(revoker SessionRevoker) {
	s.sessions = revoker
}

func (s *Service) Stats(ctx context.Context) (Stats, error) {
	return s.repo.Stats(ctx)
}

func (s *Service) ListUsers(ctx context.Context, filter UserFilter) (Page[User], error) {
	filter.Page, filter.PageSize = normalizePage(filter.Page, filter.PageSize)
	filter.Query = strings.TrimSpace(filter.Query)
	if filter.Role != "" {
		if _, ok := validRoles[filter.Role]; !ok {
			return Page[User]{}, fmt.Errorf("%w: 角色不存在", ErrValidation)
		}
	}
	switch filter.Status {
	case "", "active", "disabled":
	default:
		return Page[User]{}, fmt.Errorf("%w: 状态筛选不支持", ErrValidation)
	}
	items, hasMore, err := s.repo.ListUsers(ctx, filter)
	if err != nil {
		return Page[User]{}, err
	}
	if items == nil {
		items = []User{}
	}
	for i := range items {
		if items[i].Roles == nil {
			items[i].Roles = []string{}
		}
	}
	return Page[User]{Items: items, Page: filter.Page, PageSize: filter.PageSize, HasMore: hasMore}, nil
}

func (s *Service) UpdateUser(ctx context.Context, operator auth.Principal, id int64, update UserUpdate) (User, error) {
	if id <= 0 {
		return User{}, ErrNotFound
	}
	// 自我保护：管理员不能改自己的角色或状态，避免把自己锁在后台之外。
	if id == operator.UserID && (update.Role != nil || update.Status != nil) {
		return User{}, fmt.Errorf("%w: 不能修改自己的角色或状态", ErrForbidden)
	}
	if update.Role != nil {
		if _, ok := validRoles[*update.Role]; !ok {
			return User{}, fmt.Errorf("%w: 角色不存在", ErrValidation)
		}
	}
	if update.Status != nil && *update.Status != StatusActive && *update.Status != StatusDisabled {
		return User{}, fmt.Errorf("%w: 状态取值不支持", ErrValidation)
	}

	target, err := s.repo.FindUser(ctx, id)
	if err != nil {
		return User{}, err
	}
	demoting := update.Role != nil && *update.Role != "admin" && contains(target.Roles, "admin")
	disabling := update.Status != nil && *update.Status != StatusActive && contains(target.Roles, "admin")
	if demoting || disabling {
		// 最后一个管理员被降级或停用会让后台永久失去入口，必须拦住。
		remaining, err := s.repo.CountActiveAdmins(ctx, id)
		if err != nil {
			return User{}, err
		}
		if remaining == 0 {
			return User{}, fmt.Errorf("%w: 至少需要保留一名管理员", ErrForbidden)
		}
	}

	if err := s.repo.UpdateUser(ctx, id, update); err != nil {
		return User{}, err
	}
	if s.sessions != nil && (update.Role != nil || update.Status != nil) {
		reason := "role_changed"
		if update.Status != nil {
			reason = "status_changed"
		}
		// 会话里缓存了角色与权限，变更后必须立刻失效，否则旧权限还能继续用。
		if err := s.sessions.RevokeUserSessions(ctx, id, reason); err != nil {
			return User{}, err
		}
	}
	return s.repo.FindUser(ctx, id)
}

func (s *Service) ListContent(ctx context.Context, kind string, filter ContentFilter) (Page[ContentItem], error) {
	if !validKind(kind) {
		return Page[ContentItem]{}, fmt.Errorf("%w: 内容类型不支持", ErrValidation)
	}
	filter.Page, filter.PageSize = normalizePage(filter.Page, filter.PageSize)
	filter.Query = strings.TrimSpace(filter.Query)
	switch filter.Deleted {
	case "":
		filter.Deleted = "live"
	case "live", "deleted", "all":
	default:
		return Page[ContentItem]{}, fmt.Errorf("%w: 删除状态筛选不支持", ErrValidation)
	}
	items, hasMore, err := s.repo.ListContent(ctx, kind, filter)
	if err != nil {
		return Page[ContentItem]{}, err
	}
	if items == nil {
		items = []ContentItem{}
	}
	return Page[ContentItem]{Items: items, Page: filter.Page, PageSize: filter.PageSize, HasMore: hasMore}, nil
}

func (s *Service) DeleteContent(ctx context.Context, principal auth.Principal, kind string, id int64) error {
	if !validKind(kind) || id <= 0 {
		return ErrNotFound
	}
	if s.content == nil {
		return fmt.Errorf("%w: 内容删除未装配", ErrForbidden)
	}
	switch kind {
	case KindQuestion:
		return s.content.DeleteQuestion(ctx, principal, id)
	case KindAnswer:
		return s.content.DeleteAnswer(ctx, principal, id)
	case KindArticle:
		return s.content.DeleteArticle(ctx, principal, id)
	default:
		return s.content.DeleteComment(ctx, principal, id)
	}
}

func (s *Service) ListTopics(ctx context.Context, query string, includeDeleted bool, page, pageSize int) (Page[TopicItem], error) {
	page, pageSize = normalizePage(page, pageSize)
	items, hasMore, err := s.repo.ListTopics(ctx, strings.TrimSpace(query), includeDeleted, page, pageSize)
	if err != nil {
		return Page[TopicItem]{}, err
	}
	if items == nil {
		items = []TopicItem{}
	}
	return Page[TopicItem]{Items: items, Page: page, PageSize: pageSize, HasMore: hasMore}, nil
}

func (s *Service) CreateTopic(ctx context.Context, name, description, coverKey string) (TopicItem, error) {
	name, description, err := validateTopic(name, description)
	if err != nil {
		return TopicItem{}, err
	}
	id, err := s.repo.CreateTopic(ctx, name, description, strings.TrimSpace(coverKey))
	if err != nil {
		return TopicItem{}, err
	}
	return s.repo.FindTopic(ctx, id)
}

func (s *Service) UpdateTopic(ctx context.Context, id int64, name, description, coverKey string) (TopicItem, error) {
	if id <= 0 {
		return TopicItem{}, ErrNotFound
	}
	name, description, err := validateTopic(name, description)
	if err != nil {
		return TopicItem{}, err
	}
	return s.repo.UpdateTopic(ctx, id, name, description, strings.TrimSpace(coverKey))
}

func (s *Service) DeleteTopic(ctx context.Context, id int64) error {
	if id <= 0 {
		return ErrNotFound
	}
	return s.repo.DeleteTopic(ctx, id)
}

func (s *Service) ListOptions(ctx context.Context) ([]OptionItem, error) {
	items, err := s.repo.ListOptions(ctx)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []OptionItem{}
	}
	return items, nil
}

func (s *Service) UpdateOptions(ctx context.Context, values map[string]string) error {
	if len(values) == 0 {
		return fmt.Errorf("%w: 没有需要保存的配置", ErrValidation)
	}
	for name, value := range values {
		if err := option.Validate(name, value); err != nil {
			return fmt.Errorf("%w: %s", ErrValidation, err.Error())
		}
	}
	return s.repo.UpdateOptions(ctx, values)
}

func (s *Service) ListImages(ctx context.Context, filter ContentFilter) (Page[ImageItem], error) {
	filter.Page, filter.PageSize = normalizePage(filter.Page, filter.PageSize)
	filter.Query = strings.TrimSpace(filter.Query)
	switch filter.Deleted {
	case "":
		filter.Deleted = "live"
	case "live", "deleted", "all":
	default:
		return Page[ImageItem]{}, fmt.Errorf("%w: 删除状态筛选不支持", ErrValidation)
	}
	items, hasMore, err := s.repo.ListImages(ctx, filter)
	if err != nil {
		return Page[ImageItem]{}, err
	}
	if items == nil {
		items = []ImageItem{}
	}
	return Page[ImageItem]{Items: items, Page: filter.Page, PageSize: filter.PageSize, HasMore: hasMore}, nil
}

func (s *Service) DeleteImage(ctx context.Context, principal auth.Principal, key string) error {
	if s.media == nil {
		return fmt.Errorf("%w: 图片删除未装配", ErrForbidden)
	}
	return s.media.Delete(ctx, principal.UserID, true, strings.TrimSpace(key))
}

func validateTopic(name, description string) (string, string, error) {
	name = strings.TrimSpace(name)
	description = strings.TrimSpace(description)
	if count := utf8.RuneCountInString(name); count < 1 || count > 20 {
		return "", "", fmt.Errorf("%w: 话题名称需要 1 至 20 个字", ErrValidation)
	}
	if utf8.RuneCountInString(description) > 1000 {
		return "", "", fmt.Errorf("%w: 话题描述不能超过 1000 个字", ErrValidation)
	}
	return name, description, nil
}

func validKind(kind string) bool {
	switch kind {
	case KindQuestion, KindAnswer, KindArticle, KindComment:
		return true
	}
	return false
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
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
