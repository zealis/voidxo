package content

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/voidxo/voidxo/internal/platform/cache"
	"github.com/voidxo/voidxo/internal/platform/markdown"
)

const (
	defaultPageSize   = 20
	maxPageSize       = 50
	maxMarkdownSource = 200000
	maxRenderedText   = 100000
)

type Service struct {
	repo         Repository
	renderer     *markdown.Renderer
	optionSource OptionSource
	notifier     Notifier
	cache        cache.Cache
}

// cacheScope 是内容域缓存版本的作用域；Worker 消费内容事件后递增该版本。
const cacheScope = "content"

// SetCache 注入只读缓存；未注入时所有读取直接走数据库。
func (s *Service) SetCache(c cache.Cache) {
	s.cache = c
}

func NewService(repo Repository, renderer *markdown.Renderer, sources ...OptionSource) *Service {
	service := &Service{repo: repo, renderer: renderer}
	if len(sources) > 0 {
		service.optionSource = sources[0]
	}
	return service
}

func (s *Service) ListTopics(ctx context.Context, sort string, page, pageSize int) (Page[Topic], error) {
	page, pageSize = normalizePage(page, pageSize)
	sort = normalizeSort(sort, topicSorts, defaultTopicSort)
	key := fmt.Sprintf("topics:%s:%d:%d", sort, page, pageSize)
	if s.cache != nil {
		var cached Page[Topic]
		if ok, _ := s.cache.Get(ctx, cacheScope, key, &cached); ok {
			return cached, nil
		}
	}
	items, hasMore, err := s.repo.ListTopics(ctx, sort, pageSize, (page-1)*pageSize)
	if err != nil {
		return Page[Topic]{}, err
	}
	items = ensureItems(items)
	result := Page[Topic]{Items: items, Page: page, PageSize: pageSize, HasMore: hasMore}
	if s.cache != nil {
		_ = s.cache.Set(ctx, cacheScope, key, result, 60*time.Second)
	}
	return result, nil
}

func (s *Service) GetTopic(ctx context.Context, id int64) (Topic, error) {
	if id <= 0 {
		return Topic{}, ErrNotFound
	}
	return s.repo.FindTopic(ctx, id)
}

func (s *Service) CreateTopic(ctx context.Context, name, description, coverKey string) (Topic, error) {
	name = strings.TrimSpace(name)
	description = strings.TrimSpace(description)
	coverKey = strings.TrimSpace(coverKey)
	if count := utf8.RuneCountInString(name); count < 1 || count > 20 {
		return Topic{}, &ValidationError{Message: "话题名称需为 1 至 20 个字符"}
	}
	if count := utf8.RuneCountInString(description); count < 1 || count > 1000 {
		return Topic{}, &ValidationError{Message: "话题描述需为 1 至 1000 个字符"}
	}
	if utf8.RuneCountInString(coverKey) > 50 {
		return Topic{}, &ValidationError{Message: "话题封面标识过长"}
	}
	return s.repo.CreateTopic(ctx, name, description, coverKey)
}

// maxSitemapEntries 控制单次站点地图的最大条目数，避免一次查询拉全表。
const maxSitemapEntries = 5000

// Sitemap 返回公开内容的索引项。缓存与内容域同版本，
// 因此任何内容事件都会让站点地图缓存自然失效，无需额外失效逻辑。
func (s *Service) Sitemap(ctx context.Context, limit int) ([]SitemapEntry, error) {
	if limit <= 0 || limit > maxSitemapEntries {
		limit = maxSitemapEntries
	}
	key := fmt.Sprintf("sitemap:%d", limit)
	if s.cache != nil {
		var cached []SitemapEntry
		if ok, _ := s.cache.Get(ctx, cacheScope, key, &cached); ok {
			return cached, nil
		}
	}
	items, err := s.repo.ListSitemap(ctx, limit)
	if err != nil {
		return nil, err
	}
	items = ensureItems(items)
	if s.cache != nil {
		_ = s.cache.Set(ctx, cacheScope, key, items, 5*time.Minute)
	}
	return items, nil
}

// ListUserContent 列出用户在某一类内容下的公开条目。
func (s *Service) ListUserContent(ctx context.Context, kind string, userID int64, page, pageSize int) (Page[UserContentItem], error) {
	switch kind {
	case KindQuestion, KindAnswer, KindArticle, KindComment:
	default:
		return Page[UserContentItem]{}, &ValidationError{Message: "内容类型不支持"}
	}
	if userID <= 0 {
		return Page[UserContentItem]{}, ErrNotFound
	}
	page, pageSize = normalizePage(page, pageSize)
	items, hasMore, err := s.repo.ListUserContent(ctx, kind, userID, pageSize, (page-1)*pageSize)
	if err != nil {
		return Page[UserContentItem]{}, err
	}
	items = ensureItems(items)
	return Page[UserContentItem]{Items: items, Page: page, PageSize: pageSize, HasMore: hasMore}, nil
}

func (s *Service) ListQuestions(ctx context.Context, topicID *int64, sort string, page, pageSize int) (Page[Question], error) {
	if topicID != nil {
		if *topicID <= 0 {
			return Page[Question]{}, ErrTopicNotFound
		}
		if _, err := s.repo.FindTopic(ctx, *topicID); err != nil {
			return Page[Question]{}, err
		}
	}
	page, pageSize = normalizePage(page, pageSize)
	items, hasMore, err := s.repo.ListQuestions(ctx, topicID, normalizeSort(sort, questionSorts, defaultQuestionSort), pageSize, (page-1)*pageSize)
	if err != nil {
		return Page[Question]{}, err
	}
	items = ensureItems(items)
	return Page[Question]{Items: items, Page: page, PageSize: pageSize, HasMore: hasMore}, nil
}

func (s *Service) GetQuestion(ctx context.Context, id int64) (Question, error) {
	if id <= 0 {
		return Question{}, ErrNotFound
	}
	return s.repo.FindQuestion(ctx, id)
}

func (s *Service) CreateQuestion(ctx context.Context, userID int64, title, source string, topicIDs []int64) (Question, error) {
	if userID <= 0 {
		return Question{}, ErrNotFound
	}
	title = strings.TrimSpace(title)
	if count := utf8.RuneCountInString(title); count < 2 || count > 80 {
		return Question{}, &ValidationError{Message: "标题需为 2 至 80 个字符"}
	}
	topicIDs = normalizeIDs(topicIDs)
	if len(topicIDs) < 1 || len(topicIDs) > 10 {
		return Question{}, &ValidationError{Message: "请选择 1 至 10 个话题"}
	}
	count, err := s.repo.CountTopics(ctx, topicIDs)
	if err != nil {
		return Question{}, err
	}
	if count != len(topicIDs) {
		return Question{}, ErrTopicNotFound
	}
	rendered, plain, err := s.render(source)
	if err != nil {
		return Question{}, err
	}
	if plain == "" {
		return Question{}, &ValidationError{Message: "正文不能为空"}
	}
	return s.repo.CreateQuestion(ctx, userID, title, strings.TrimSpace(source), rendered, topicIDs)
}

func (s *Service) ListAnswers(ctx context.Context, questionID int64, page, pageSize int) (Page[Answer], error) {
	if questionID <= 0 {
		return Page[Answer]{}, ErrNotFound
	}
	exists, err := s.repo.QuestionExists(ctx, questionID)
	if err != nil {
		return Page[Answer]{}, err
	}
	if !exists {
		return Page[Answer]{}, ErrNotFound
	}
	page, pageSize = normalizePage(page, pageSize)
	items, hasMore, err := s.repo.ListAnswers(ctx, questionID, pageSize, (page-1)*pageSize)
	if err != nil {
		return Page[Answer]{}, err
	}
	items = ensureItems(items)
	return Page[Answer]{Items: items, Page: page, PageSize: pageSize, HasMore: hasMore}, nil
}

func (s *Service) CreateAnswer(ctx context.Context, userID, questionID int64, source string) (Answer, error) {
	if userID <= 0 || questionID <= 0 {
		return Answer{}, ErrNotFound
	}
	meta, err := s.repo.Meta(ctx, KindQuestion, questionID)
	if err != nil {
		return Answer{}, err
	}
	rendered, plain, err := s.render(source)
	if err != nil {
		return Answer{}, err
	}
	if plain == "" {
		return Answer{}, &ValidationError{Message: "回答不能为空"}
	}
	answer, err := s.repo.CreateAnswer(ctx, questionID, userID, strings.TrimSpace(source), rendered)
	if err != nil {
		return Answer{}, err
	}
	s.notify(ctx, NotificationEvent{
		ReceiverID: meta.UserID,
		SenderID:   userID,
		Type:       "question_answered",
		QuestionID: questionID,
		AnswerID:   answer.ID,
	})
	return answer, nil
}

func (s *Service) render(source string) (string, string, error) {
	source = strings.TrimSpace(source)
	if source == "" {
		return "", "", &ValidationError{Message: "正文不能为空"}
	}
	if utf8.RuneCountInString(source) > maxMarkdownSource {
		return "", "", &ValidationError{Message: "正文过长"}
	}
	rendered, plain, err := s.renderer.Render(source)
	if err != nil {
		return "", "", fmt.Errorf("render markdown: %w", err)
	}
	if utf8.RuneCountInString(plain) > maxRenderedText {
		return "", "", &ValidationError{Message: "正文不能超过 100000 个字符"}
	}
	return rendered, plain, nil
}

// 排序白名单：ORDER BY 片段只能来自这里，避免把外部输入拼进 SQL。
var (
	topicSorts = map[string]string{
		"recommended": "follower_count DESC, topic_id DESC",
		"new":         "topic_id DESC",
		"name":        "name ASC",
	}
	questionSorts = map[string]string{
		"recent":  "updated_at DESC, question_id DESC",
		"new":     "created_at DESC, question_id DESC",
		"popular": "vote_count DESC, question_id DESC",
	}
	articleSorts = map[string]string{
		"recent":  "updated_at DESC, article_id DESC",
		"new":     "created_at DESC, article_id DESC",
		"popular": "vote_count DESC, article_id DESC",
	}
	defaultTopicSort    = "recommended"
	defaultQuestionSort = "recent"
	defaultArticleSort  = "recent"
)

// normalizeSort 只接受白名单值，非法或空值回落到默认排序。
func normalizeSort(value string, allowed map[string]string, fallback string) string {
	if _, ok := allowed[value]; ok {
		return value
	}
	return fallback
}

// ensureItems 保证空列表序列化为 []，避免前端把 null 判定为响应异常。
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
