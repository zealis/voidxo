package content

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/voidxo/voidxo/internal/platform/markdown"
)

type contentRepoStub struct {
	topicCount      int
	createdMarkdown string
	createdRendered string
	createdTopicIDs []int64
	createdComment  string
	createdRootID   *int64
	metaOwner       int64
	metaComments    int
	metaAnswers     int
	metaMissing     bool
	metaContent     string
	lastMetaKind    string
	sitemap         []SitemapEntry
	sitemapCalls    int
	userContent     []UserContentItem
}

func (r *contentRepoStub) ListTopics(context.Context, string, int, int) ([]Topic, bool, error) {
	return []Topic{{ID: 1, Name: "技术"}}, false, nil
}
func (r *contentRepoStub) FindTopic(_ context.Context, id int64) (Topic, error) {
	if id != 1 {
		return Topic{}, ErrTopicNotFound
	}
	return Topic{ID: 1, Name: "技术"}, nil
}
func (r *contentRepoStub) CreateTopic(_ context.Context, name, description, coverKey string) (Topic, error) {
	return Topic{ID: 1, Name: name, Description: description, CoverKey: coverKey}, nil
}
func (r *contentRepoStub) CountTopics(context.Context, []int64) (int, error) {
	return r.topicCount, nil
}
func (r *contentRepoStub) ListQuestions(context.Context, *int64, string, int, int) ([]Question, bool, error) {
	return []Question{{ID: 1, Title: "标题"}}, false, nil
}
func (r *contentRepoStub) FindQuestion(_ context.Context, id int64) (Question, error) {
	if id != 1 {
		return Question{}, ErrNotFound
	}
	return Question{ID: 1, Title: "标题"}, nil
}
func (r *contentRepoStub) CreateQuestion(_ context.Context, _ int64, title, markdownSource, rendered string, topicIDs []int64) (Question, error) {
	r.createdMarkdown = markdownSource
	r.createdRendered = rendered
	r.createdTopicIDs = topicIDs
	return Question{ID: 1, Title: title, ContentRendered: rendered}, nil
}
func (r *contentRepoStub) QuestionExists(_ context.Context, id int64) (bool, error) {
	return id == 1, nil
}
func (r *contentRepoStub) ListAnswers(context.Context, int64, int, int) ([]Answer, bool, error) {
	return []Answer{{ID: 1, ContentRendered: "<p>回答</p>"}}, false, nil
}
func (r *contentRepoStub) CreateAnswer(_ context.Context, _, _ int64, _ string, rendered string) (Answer, error) {
	return Answer{ID: 1, ContentRendered: rendered}, nil
}

func TestCreateQuestionValidationAndRendering(t *testing.T) {
	repo := &contentRepoStub{topicCount: 1}
	service := NewService(repo, markdown.New())

	if _, err := service.CreateQuestion(context.Background(), 1, "短", "正文", []int64{1}); err == nil {
		t.Fatal("short title accepted")
	}
	if _, err := service.CreateQuestion(context.Background(), 1, "有效标题", "正文", nil); err == nil {
		t.Fatal("empty topics accepted")
	}
	repo.topicCount = 0
	if _, err := service.CreateQuestion(context.Background(), 1, "有效标题", "正文", []int64{1}); !errors.Is(err, ErrTopicNotFound) {
		t.Fatalf("missing topic returned %v", err)
	}

	repo.topicCount = 1
	question, err := service.CreateQuestion(context.Background(), 1, "有效标题", "# 标题\n\n<script>alert(1)</script>\n\n**正文**", []int64{1, 1})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(strings.ToLower(repo.createdRendered), "<script") {
		t.Fatal("script survived rendering")
	}
	if len(repo.createdTopicIDs) != 1 || repo.createdTopicIDs[0] != 1 {
		t.Fatalf("topic IDs were not normalized: %#v", repo.createdTopicIDs)
	}
	if question.Title != "有效标题" || !strings.Contains(question.ContentRendered, "<strong>正文</strong>") {
		t.Fatalf("unexpected question: %#v", question)
	}
}

func TestCreateAnswerAndPageDefaults(t *testing.T) {
	repo := &contentRepoStub{}
	service := NewService(repo, markdown.New())
	if _, err := service.CreateAnswer(context.Background(), 1, 1, "   "); err == nil {
		t.Fatal("empty answer accepted")
	}
	answer, err := service.CreateAnswer(context.Background(), 1, 1, "**回答**")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(answer.ContentRendered, "<strong>回答</strong>") {
		t.Fatalf("answer was not rendered: %#v", answer)
	}
	page, err := service.ListQuestions(context.Background(), nil, "", 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if page.Page != 1 || page.PageSize != defaultPageSize {
		t.Fatalf("page not normalized: %#v", page)
	}
}

func (r *contentRepoStub) ListSitemap(context.Context, int) ([]SitemapEntry, error) {
	r.sitemapCalls++
	return r.sitemap, nil
}

func (r *contentRepoStub) ListUserContent(_ context.Context, kind string, _ int64, _, _ int) ([]UserContentItem, bool, error) {
	return r.userContent, false, nil
}
