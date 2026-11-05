package admin

import (
	"strings"
	"testing"
)

// 各内容类型的主表别名不同：question/article 用 t，answer 用 a，comment 用 c。
// 曾经把删除过滤硬编码成 t.deleted_at，导致后台「回答」分区查询报
// Unknown column 't.deleted_at'，页面显示「列表加载失败」。
func TestContentQueryDeletedFilterFollowsMainTableAlias(t *testing.T) {
	repo := NewMySQLRepository(nil, "vx_")

	cases := []struct {
		kind     string
		wantLive string
		wantGone string
	}{
		{KindQuestion, "t.deleted_at IS NULL", "t.deleted_at IS NOT NULL"},
		{KindArticle, "t.deleted_at IS NULL", "t.deleted_at IS NOT NULL"},
		{KindAnswer, "a.deleted_at IS NULL", "a.deleted_at IS NOT NULL"},
		{KindComment, "c.deleted_at IS NULL", "c.deleted_at IS NOT NULL"},
	}

	for _, tc := range cases {
		live, _ := repo.contentQuery(tc.kind, ContentFilter{Deleted: "live", Page: 1, PageSize: 20})
		if live == "" {
			t.Fatalf("%s: 未生成查询", tc.kind)
		}
		if !strings.Contains(live, tc.wantLive) {
			t.Fatalf("%s: live 过滤应为 %q，实际 %s", tc.kind, tc.wantLive, live)
		}

		gone, _ := repo.contentQuery(tc.kind, ContentFilter{Deleted: "deleted", Page: 1, PageSize: 20})
		if !strings.Contains(gone, tc.wantGone) {
			t.Fatalf("%s: deleted 过滤应为 %q，实际 %s", tc.kind, tc.wantGone, gone)
		}

		all, _ := repo.contentQuery(tc.kind, ContentFilter{Deleted: "all", Page: 1, PageSize: 20})
		// SELECT 列表里本来就有 deleted_at IS NOT NULL（用于返回删除标记），
		// 这里只关心 WHERE 子句是否被加上删除条件。
		if strings.Contains(all, " WHERE ") {
			t.Fatalf("%s: deleted=all 且无搜索词时不应有 WHERE，实际 %s", tc.kind, all)
		}
	}
}

func TestContentQueryAnswerNeverUsesQuestionAlias(t *testing.T) {
	repo := NewMySQLRepository(nil, "vx_")
	query, _ := repo.contentQuery(KindAnswer, ContentFilter{Deleted: "live", Page: 1, PageSize: 20})
	if strings.Contains(query, "t.deleted_at") {
		t.Fatalf("answer 查询不应出现 t.deleted_at（没有 t 别名）: %s", query)
	}
}
