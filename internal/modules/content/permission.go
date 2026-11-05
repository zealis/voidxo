package content

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/voidxo/voidxo/internal/modules/auth"
)

// OptionSource 读取站点配置。内容模块只关心与编辑、删除相关的开关，
// 因此不依赖整个 option 包的具体实现。
type OptionSource interface {
	FindByNames(ctx context.Context, names []string) (map[string]string, error)
}

// authorizeModification 复现旧系统的编辑/删除开关语义：
// 管理员始终放行；作者本人还需满足总开关、时间窗、以及“无评论/无回答”限制。
func (s *Service) authorizeModification(ctx context.Context, principal auth.Principal, kind, action string, meta ContentMeta) error {
	if principal.UserID <= 0 {
		return &PermissionError{Message: "请先登录"}
	}
	if principal.HasPermission("content.manage") || principal.HasPermission("content.moderate") {
		return nil
	}
	if meta.UserID != principal.UserID {
		return &PermissionError{Message: "只能修改自己发布的内容"}
	}

	names := []string{kind + "_can_" + action, kind + "_can_" + action + "_before"}
	switch kind {
	case KindQuestion:
		names = append(names, kind+"_can_"+action+"_only_no_answer", kind+"_can_"+action+"_only_no_comment")
	case KindAnswer, KindArticle:
		names = append(names, kind+"_can_"+action+"_only_no_comment")
	}
	values, err := s.options(ctx, names)
	if err != nil {
		return err
	}

	if !optionBool(values, names[0], true) {
		return &PermissionError{Message: "站点已关闭该内容的" + actionMessage(action) + "功能"}
	}
	if before := optionInt(values, names[1]); before > 0 && meta.CreatedAt.Add(time.Duration(before)*time.Second).Before(time.Now()) {
		return &PermissionError{Message: "已超过可" + actionMessage(action) + "的时间范围"}
	}
	if (kind == KindQuestion || kind == KindAnswer || kind == KindArticle) &&
		optionBool(values, kind+"_can_"+action+"_only_no_comment", false) && meta.CommentCount > 0 {
		return &PermissionError{Message: "内容已有评论，无法" + actionMessage(action)}
	}
	if kind == KindQuestion && optionBool(values, kind+"_can_"+action+"_only_no_answer", false) && meta.AnswerCount > 0 {
		return &PermissionError{Message: "问题已有回答，无法" + actionMessage(action)}
	}
	return nil
}

// options 在未注入配置源时返回空表，此时只保留作者校验，
// 便于不依赖数据库的单元测试。
func (s *Service) options(ctx context.Context, names []string) (map[string]string, error) {
	if s.optionSource == nil || len(names) == 0 {
		return map[string]string{}, nil
	}
	values, err := s.optionSource.FindByNames(ctx, names)
	if err != nil {
		return nil, err
	}
	if values == nil {
		values = map[string]string{}
	}
	return values, nil
}

func optionBool(values map[string]string, name string, fallback bool) bool {
	raw, ok := values[name]
	if !ok {
		return fallback
	}
	return raw == "true"
}

func optionInt(values map[string]string, name string) int {
	raw, ok := values[name]
	if !ok {
		return 0
	}
	parsed, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || parsed < 0 {
		return 0
	}
	return parsed
}

func actionMessage(action string) string {
	if action == "delete" {
		return "删除"
	}
	return "编辑"
}

// canManage 表示是否具备管理全部内容的权限，用于评论列表等场景的前端提示。
func canManage(principal auth.Principal) bool {
	return principal.HasPermission("content.manage") || principal.HasPermission("content.moderate")
}
