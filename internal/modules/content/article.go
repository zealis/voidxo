package content

import (
	"context"
	"strings"
	"unicode/utf8"

	"github.com/voidxo/voidxo/internal/modules/auth"
)

func (s *Service) UpdateQuestion(ctx context.Context, principal auth.Principal, id int64, title, source string, topicIDs []int64) (Question, error) {
	meta, err := s.repo.Meta(ctx, KindQuestion, id)
	if err != nil {
		return Question{}, err
	}
	if err := s.authorizeModification(ctx, principal, KindQuestion, "edit", meta); err != nil {
		return Question{}, err
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
	return s.repo.UpdateQuestion(ctx, id, title, strings.TrimSpace(source), rendered, topicIDs)
}

func (s *Service) DeleteQuestion(ctx context.Context, principal auth.Principal, id int64) error {
	meta, err := s.repo.Meta(ctx, KindQuestion, id)
	if err != nil {
		return err
	}
	if err := s.authorizeModification(ctx, principal, KindQuestion, "delete", meta); err != nil {
		return err
	}
	return s.repo.DeleteQuestion(ctx, id)
}

func (s *Service) UpdateAnswer(ctx context.Context, principal auth.Principal, id int64, source string) (Answer, error) {
	meta, err := s.repo.Meta(ctx, KindAnswer, id)
	if err != nil {
		return Answer{}, err
	}
	if err := s.authorizeModification(ctx, principal, KindAnswer, "edit", meta); err != nil {
		return Answer{}, err
	}
	rendered, plain, err := s.render(source)
	if err != nil {
		return Answer{}, err
	}
	if plain == "" {
		return Answer{}, &ValidationError{Message: "回答不能为空"}
	}
	return s.repo.UpdateAnswer(ctx, id, strings.TrimSpace(source), rendered)
}

func (s *Service) DeleteAnswer(ctx context.Context, principal auth.Principal, id int64) error {
	meta, err := s.repo.Meta(ctx, KindAnswer, id)
	if err != nil {
		return err
	}
	if err := s.authorizeModification(ctx, principal, KindAnswer, "delete", meta); err != nil {
		return err
	}
	return s.repo.DeleteAnswer(ctx, id)
}

func (s *Service) ListArticles(ctx context.Context, topicID *int64, sort string, page, pageSize int) (Page[Article], error) {
	if topicID != nil {
		if *topicID <= 0 {
			return Page[Article]{}, ErrTopicNotFound
		}
		if _, err := s.repo.FindTopic(ctx, *topicID); err != nil {
			return Page[Article]{}, err
		}
	}
	page, pageSize = normalizePage(page, pageSize)
	items, hasMore, err := s.repo.ListArticles(ctx, topicID, normalizeSort(sort, articleSorts, defaultArticleSort), pageSize, (page-1)*pageSize)
	if err != nil {
		return Page[Article]{}, err
	}
	items = ensureItems(items)
	return Page[Article]{Items: items, Page: page, PageSize: pageSize, HasMore: hasMore}, nil
}

func (s *Service) GetArticle(ctx context.Context, id int64) (Article, error) {
	if id <= 0 {
		return Article{}, ErrNotFound
	}
	return s.repo.FindArticle(ctx, id)
}

func (s *Service) CreateArticle(ctx context.Context, userID int64, title, source string, topicIDs []int64) (Article, error) {
	if userID <= 0 {
		return Article{}, ErrNotFound
	}
	title = strings.TrimSpace(title)
	if count := utf8.RuneCountInString(title); count < 2 || count > 80 {
		return Article{}, &ValidationError{Message: "标题需为 2 至 80 个字符"}
	}
	topicIDs = normalizeIDs(topicIDs)
	if len(topicIDs) < 1 || len(topicIDs) > 10 {
		return Article{}, &ValidationError{Message: "请选择 1 至 10 个话题"}
	}
	count, err := s.repo.CountTopics(ctx, topicIDs)
	if err != nil {
		return Article{}, err
	}
	if count != len(topicIDs) {
		return Article{}, ErrTopicNotFound
	}
	rendered, plain, err := s.render(source)
	if err != nil {
		return Article{}, err
	}
	if plain == "" {
		return Article{}, &ValidationError{Message: "正文不能为空"}
	}
	return s.repo.CreateArticle(ctx, userID, title, strings.TrimSpace(source), rendered, topicIDs)
}

func (s *Service) UpdateArticle(ctx context.Context, principal auth.Principal, id int64, title, source string, topicIDs []int64) (Article, error) {
	meta, err := s.repo.Meta(ctx, KindArticle, id)
	if err != nil {
		return Article{}, err
	}
	if err := s.authorizeModification(ctx, principal, KindArticle, "edit", meta); err != nil {
		return Article{}, err
	}
	title = strings.TrimSpace(title)
	if count := utf8.RuneCountInString(title); count < 2 || count > 80 {
		return Article{}, &ValidationError{Message: "标题需为 2 至 80 个字符"}
	}
	topicIDs = normalizeIDs(topicIDs)
	if len(topicIDs) < 1 || len(topicIDs) > 10 {
		return Article{}, &ValidationError{Message: "请选择 1 至 10 个话题"}
	}
	count, err := s.repo.CountTopics(ctx, topicIDs)
	if err != nil {
		return Article{}, err
	}
	if count != len(topicIDs) {
		return Article{}, ErrTopicNotFound
	}
	rendered, plain, err := s.render(source)
	if err != nil {
		return Article{}, err
	}
	if plain == "" {
		return Article{}, &ValidationError{Message: "正文不能为空"}
	}
	return s.repo.UpdateArticle(ctx, id, title, strings.TrimSpace(source), rendered, topicIDs)
}

func (s *Service) DeleteArticle(ctx context.Context, principal auth.Principal, id int64) error {
	meta, err := s.repo.Meta(ctx, KindArticle, id)
	if err != nil {
		return err
	}
	if err := s.authorizeModification(ctx, principal, KindArticle, "delete", meta); err != nil {
		return err
	}
	if err := s.repo.DeleteArticle(ctx, id); err != nil {
		return err
	}
	s.notify(ctx, NotificationEvent{
		ReceiverID:     meta.UserID,
		SenderID:       principal.UserID,
		Type:           "article_deleted",
		ArticleID:      id,
		ContentDeleted: meta.Content,
	})
	return nil
}
