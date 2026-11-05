package markdown

import (
	"strings"
	"testing"
)

func TestRenderSanitizesActiveContent(t *testing.T) {
	renderer := New()
	rendered, plain, err := renderer.Render("# 标题\n\n<script>alert(1)</script>\n\n**正文** [链接](https://example.com)")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(strings.ToLower(rendered), "<script") || strings.Contains(rendered, "onerror=") {
		t.Fatalf("active content survived: %s", rendered)
	}
	if !strings.Contains(rendered, "<h1") || !strings.Contains(rendered, "<strong>正文</strong>") {
		t.Fatalf("safe markdown structure missing: %s", rendered)
	}
	if plain != "标题 正文 链接" {
		t.Fatalf("unexpected plain text: %q", plain)
	}
}
