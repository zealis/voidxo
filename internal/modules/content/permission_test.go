package content

import (
	"context"
	"testing"
	"time"

	"github.com/voidxo/voidxo/internal/modules/auth"
	"github.com/voidxo/voidxo/internal/platform/markdown"
)

// 以下实现只用于满足 Repository 接口，让服务层测试不依赖数据库。
func (r *contentRepoStub) UpdateQuestion(context.Context, int64, string, string, string, []int64) (Question, error) {
	return Question{ID: 1}, nil
}
func (r *contentRepoStub) DeleteQuestion(context.Context, int64) error { return nil }
func (r *contentRepoStub) FindAnswer(context.Context, int64) (Answer, error) {
	return Answer{ID: 1}, nil
}
func (r *contentRepoStub) UpdateAnswer(_ context.Context, id int64, _, rendered string) (Answer, error) {
	return Answer{ID: id, ContentRendered: rendered}, nil
}
func (r *contentRepoStub) DeleteAnswer(context.Context, int64) error { return nil }
func (r *contentRepoStub) ListArticles(context.Context, *int64, string, int, int) ([]Article, bool, error) {
	return []Article{{ID: 1, Title: "文章"}}, false, nil
}
func (r *contentRepoStub) FindArticle(_ context.Context, id int64) (Article, error) {
	if id != 1 {
		return Article{}, ErrNotFound
	}
	return Article{ID: 1, Title: "文章"}, nil
}
func (r *contentRepoStub) CreateArticle(_ context.Context, _ int64, title, markdownSource, rendered string, topicIDs []int64) (Article, error) {
	r.createdMarkdown = markdownSource
	r.createdRendered = rendered
	r.createdTopicIDs = topicIDs
	return Article{ID: 1, Title: title, ContentRendered: rendered}, nil
}
func (r *contentRepoStub) UpdateArticle(_ context.Context, id int64, title, _, rendered string, _ []int64) (Article, error) {
	return Article{ID: id, Title: title, ContentRendered: rendered}, nil
}
func (r *contentRepoStub) DeleteArticle(context.Context, int64) error { return nil }
func (r *contentRepoStub) ListComments(context.Context, string, int64, int, int) ([]Comment, bool, error) {
	return []Comment{{ID: 1, Content: "评论"}}, false, nil
}
func (r *contentRepoStub) FindComment(_ context.Context, id int64) (Comment, error) {
	if id != 1 {
		return Comment{}, ErrNotFound
	}
	return Comment{ID: 1, CommentableType: KindQuestion, CommentableID: 1}, nil
}
func (r *contentRepoStub) CreateComment(_ context.Context, kind string, targetID, userID int64, content string, rootID *int64) (Comment, error) {
	r.createdComment = content
	r.createdRootID = rootID
	return Comment{ID: 2, CommentableType: kind, CommentableID: targetID, UserID: userID, Content: content, RootID: rootID}, nil
}
func (r *contentRepoStub) UpdateComment(_ context.Context, id int64, content string) (Comment, error) {
	return Comment{ID: id, Content: content}, nil
}
func (r *contentRepoStub) DeleteComment(context.Context, int64) error { return nil }
func (r *contentRepoStub) CascadeAuthors(context.Context, string, int64) ([]AuthorContent, error) {
	return nil, nil
}
func (r *contentRepoStub) Meta(_ context.Context, kind string, id int64) (ContentMeta, error) {
	if r.metaMissing {
		return ContentMeta{}, ErrNotFound
	}
	meta := ContentMeta{ID: id, UserID: r.metaOwner, CommentCount: r.metaComments, AnswerCount: r.metaAnswers, Content: r.metaContent}
	r.lastMetaKind = kind
	return meta, nil
}

type optionStub struct {
	values map[string]string
	calls  int
}

func (o *optionStub) FindByNames(context.Context, []string) (map[string]string, error) {
	o.calls++
	return o.values, nil
}

func TestAuthorCanEditOwnArticleWithinWindow(t *testing.T) {
	options := &optionStub{values: map[string]string{
		"article_can_edit":                   "true",
		"article_can_edit_before":            "0",
		"article_can_edit_only_no_comment":   "true",
		"article_can_delete":                 "true",
		"article_can_delete_before":          "0",
		"article_can_delete_only_no_comment": "false",
	}}
	repo := &contentRepoStub{metaOwner: 7, topicCount: 1}
	service := NewService(repo, markdown.New(), options)
	principal := auth.Principal{UserID: 7}

	if _, err := service.UpdateArticle(context.Background(), principal, 1, "新标题", "正文", []int64{1}); err != nil {
		t.Fatalf("author edit rejected: %v", err)
	}
	if repo.lastMetaKind != KindArticle {
		t.Fatalf("unexpected meta kind: %s", repo.lastMetaKind)
	}
}

func TestArticleEditBlockedByCommentOrForeignAuthor(t *testing.T) {
	options := &optionStub{values: map[string]string{
		"article_can_edit":                 "true",
		"article_can_edit_only_no_comment": "true",
	}}
	repo := &contentRepoStub{metaOwner: 7, metaComments: 2, topicCount: 1}
	service := NewService(repo, markdown.New(), options)

	if _, err := service.UpdateArticle(context.Background(), auth.Principal{UserID: 7}, 1, "新标题", "正文", []int64{1}); err == nil {
		t.Fatal("edit with comments was accepted")
	} else if _, ok := err.(*PermissionError); !ok {
		t.Fatalf("expected permission error, got %v", err)
	}

	repo.metaComments = 0
	if _, err := service.UpdateArticle(context.Background(), auth.Principal{UserID: 8}, 1, "新标题", "正文", []int64{1}); err == nil {
		t.Fatal("foreign author edit was accepted")
	}

	// 管理员绕过站点开关。
	manager := auth.Principal{UserID: 9, Permissions: []string{"content.moderate"}}
	if _, err := service.UpdateArticle(context.Background(), manager, 1, "新标题", "正文", []int64{1}); err != nil {
		t.Fatalf("moderator edit rejected: %v", err)
	}
}

func TestArticleEditRespectsTimeWindow(t *testing.T) {
	options := &optionStub{values: map[string]string{
		"article_can_edit":        "true",
		"article_can_edit_before": "60",
	}}
	repo := &contentRepoStub{}
	service := NewService(repo, markdown.New(), options)
	if err := service.authorizeModification(context.Background(), auth.Principal{UserID: 7}, KindArticle, "edit",
		ContentMeta{UserID: 7, CreatedAt: timeAgo(120)}); err == nil {
		t.Fatal("expired edit window was accepted")
	}
	if err := service.authorizeModification(context.Background(), auth.Principal{UserID: 7}, KindArticle, "edit",
		ContentMeta{UserID: 7, CreatedAt: timeAgo(10)}); err != nil {
		t.Fatalf("fresh edit rejected: %v", err)
	}
}

func TestCommentReplyIsFlattenedToRoot(t *testing.T) {
	repo := &contentRepoStub{metaOwner: 1}
	service := NewService(repo, markdown.New(), nil)
	parentID := int64(1)
	comment, err := service.CreateComment(context.Background(), 5, KindQuestion, 1, "回复内容", &parentID)
	if err != nil {
		t.Fatal(err)
	}
	if comment.RootID == nil || *comment.RootID != 1 {
		t.Fatalf("root not preserved: %#v", comment)
	}
	if repo.createdComment != "回复内容" {
		t.Fatalf("content not passed through: %q", repo.createdComment)
	}
}

func TestCommentValidationAndUnknownTarget(t *testing.T) {
	repo := &contentRepoStub{metaOwner: 1}
	service := NewService(repo, markdown.New(), nil)
	if _, err := service.CreateComment(context.Background(), 5, KindQuestion, 1, "   ", nil); err == nil {
		t.Fatal("empty comment accepted")
	}
	if _, err := service.CreateComment(context.Background(), 5, "unknown", 1, "内容", nil); err == nil {
		t.Fatal("unknown target type accepted")
	}
	repo.metaMissing = true
	if _, err := service.CreateComment(context.Background(), 5, KindQuestion, 99, "内容", nil); err != ErrNotFound {
		t.Fatalf("missing target returned %v", err)
	}
}

func timeAgo(seconds int) time.Time {
	return time.Now().Add(-time.Duration(seconds) * time.Second)
}

func TestEnsureItemsReturnsEmptySlice(t *testing.T) {
	if ensureItems[Topic](nil) == nil {
		t.Fatal("nil list must serialize as []")
	}
	if len(ensureItems([]Topic{})) != 0 {
		t.Fatal("empty list changed")
	}
}

type notifierStub struct {
	events []NotificationEvent
}

func (n *notifierStub) Notify(_ context.Context, event NotificationEvent) error {
	n.events = append(n.events, event)
	return nil
}

func TestDeleteByManagerNotifiesAuthorWithBackup(t *testing.T) {
	repo := &contentRepoStub{metaOwner: 8, metaContent: "<p>被删正文</p>"}
	notifier := &notifierStub{}
	service := NewService(repo, markdown.New())
	service.SetNotifier(notifier)

	manager := auth.Principal{UserID: 1, Permissions: []string{"content.moderate"}}
	if err := service.DeleteArticle(context.Background(), manager, 5); err != nil {
		t.Fatalf("manager delete rejected: %v", err)
	}
	if len(notifier.events) != 1 {
		t.Fatalf("expected one notification, got %#v", notifier.events)
	}
	event := notifier.events[0]
	if event.ReceiverID != 8 || event.SenderID != 1 || event.Type != "article_deleted" || event.ArticleID != 5 {
		t.Fatalf("unexpected event: %#v", event)
	}
	if event.ContentDeleted != "<p>被删正文</p>" {
		t.Fatalf("content backup missing: %q", event.ContentDeleted)
	}
}

func TestSelfDeleteDoesNotNotify(t *testing.T) {
	repo := &contentRepoStub{metaOwner: 8}
	notifier := &notifierStub{}
	service := NewService(repo, markdown.New())
	service.SetNotifier(notifier)

	manager := auth.Principal{UserID: 8, Permissions: []string{"content.moderate"}}
	if err := service.DeleteArticle(context.Background(), manager, 5); err != nil {
		t.Fatal(err)
	}
	if len(notifier.events) != 0 {
		t.Fatalf("self delete produced notifications: %#v", notifier.events)
	}
}

func TestSitemapReturnsEmptySlice(t *testing.T) {
	repo := &contentRepoStub{}
	service := NewService(repo, markdown.New())
	items, err := service.Sitemap(context.Background(), 0)
	if err != nil {
		t.Fatalf("sitemap failed: %v", err)
	}
	if items == nil {
		t.Fatal("empty sitemap must serialize as []")
	}
	if repo.sitemapCalls != 1 {
		t.Fatalf("expected one query, got %d", repo.sitemapCalls)
	}
}
