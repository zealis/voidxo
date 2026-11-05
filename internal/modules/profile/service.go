package profile

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/voidxo/voidxo/internal/modules/content"
	"github.com/voidxo/voidxo/internal/modules/user"
)

type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}

// ContentLister 由内容模块实现，用于公开主页的内容列表。
type ContentLister interface {
	ListUserContent(ctx context.Context, kind string, userID int64, page, pageSize int) (content.Page[content.UserContentItem], error)
}

type Service struct {
	users   user.Repository
	content ContentLister
}

func NewService(users user.Repository) *Service {
	return &Service{users: users}
}

func (s *Service) SetContentLister(lister ContentLister) {
	s.content = lister
}

// PublicPage 是公开用户列表的分页结构。
type PublicPage struct {
	Items    []user.PublicProfile `json:"items"`
	Page     int                  `json:"page"`
	PageSize int                  `json:"page_size"`
	HasMore  bool                 `json:"has_more"`
}

// ListPublic 列出启用账号，供「人脉」页使用。
func (s *Service) ListPublic(ctx context.Context, sort string, page, pageSize int) (PublicPage, error) {
	if page < 1 {
		page = 1
	}
	switch {
	case pageSize < 1:
		pageSize = 20
	case pageSize > 50:
		pageSize = 50
	}
	if sort != "new" {
		sort = "recommended"
	}
	items, hasMore, err := s.users.ListPublic(ctx, sort, pageSize, (page-1)*pageSize)
	if err != nil {
		return PublicPage{}, err
	}
	if items == nil {
		items = []user.PublicProfile{}
	}
	return PublicPage{Items: items, Page: page, PageSize: pageSize, HasMore: hasMore}, nil
}

// Public 返回公开主页资料；不包含邮箱等隐私字段。
func (s *Service) Public(ctx context.Context, userID int64) (user.PublicProfile, error) {
	if userID <= 0 {
		return user.PublicProfile{}, user.ErrNotFound
	}
	return s.users.FindPublic(ctx, userID)
}

// UserContent 列出用户在指定类型下的公开内容。
func (s *Service) UserContent(ctx context.Context, kind string, userID int64, page, pageSize int) (content.Page[content.UserContentItem], error) {
	if s.content == nil {
		return content.Page[content.UserContentItem]{}, fmt.Errorf("content lister is not configured")
	}
	if _, err := s.users.FindPublic(ctx, userID); err != nil {
		return content.Page[content.UserContentItem]{}, err
	}
	return s.content.ListUserContent(ctx, kind, userID, page, pageSize)
}

func (s *Service) Get(ctx context.Context, userID int64) (user.Account, error) {
	return s.users.FindByID(ctx, userID)
}

func (s *Service) Update(ctx context.Context, userID int64, params user.UpdateProfileParams) (user.Account, error) {
	if err := normalize(&params.Headline, 40, "一句话介绍"); err != nil {
		return user.Account{}, err
	}
	if err := normalize(&params.Bio, 160, "个人简介"); err != nil {
		return user.Account{}, err
	}
	if err := normalize(&params.Blog, 255, "个人主页"); err != nil {
		return user.Account{}, err
	}
	if err := normalize(&params.Company, 255, "公司名称"); err != nil {
		return user.Account{}, err
	}
	if err := normalize(&params.Location, 255, "地址"); err != nil {
		return user.Account{}, err
	}
	return s.users.UpdateProfile(ctx, userID, params)
}

func normalize(value **string, max int, label string) error {
	if *value == nil {
		return nil
	}
	normalized := strings.TrimSpace(**value)
	if utf8.RuneCountInString(normalized) > max {
		return &ValidationError{Message: fmt.Sprintf("%s不能超过 %d 个字符", label, max)}
	}
	*value = &normalized
	return nil
}
