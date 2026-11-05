package admin

import (
	"context"
	"errors"
	"testing"

	"github.com/voidxo/voidxo/internal/modules/auth"
)

type repoStub struct {
	users          []User
	hasMore        bool
	findResult     User
	findErr        error
	updateCalled   bool
	lastUpdate     UserUpdate
	activeAdmins   int
	contentItems   []ContentItem
	topics         []TopicItem
	createdTopicID int64
	optionValues   map[string]string
	images         []ImageItem
}

func (r *repoStub) Stats(context.Context) (Stats, error) { return Stats{Users: 3}, nil }

func (r *repoStub) ListUsers(context.Context, UserFilter) ([]User, bool, error) {
	return r.users, r.hasMore, nil
}

func (r *repoStub) FindUser(context.Context, int64) (User, error) {
	return r.findResult, r.findErr
}

func (r *repoStub) UpdateUser(_ context.Context, _ int64, update UserUpdate) error {
	r.updateCalled = true
	r.lastUpdate = update
	return nil
}

func (r *repoStub) CountActiveAdmins(context.Context, int64) (int, error) { return r.activeAdmins, nil }

func (r *repoStub) ListContent(context.Context, string, ContentFilter) ([]ContentItem, bool, error) {
	return r.contentItems, r.hasMore, nil
}

func (r *repoStub) ListTopics(context.Context, string, bool, int, int) ([]TopicItem, bool, error) {
	return r.topics, false, nil
}

func (r *repoStub) FindTopic(_ context.Context, id int64) (TopicItem, error) {
	return TopicItem{ID: id}, nil
}

func (r *repoStub) CreateTopic(context.Context, string, string, string) (int64, error) {
	return r.createdTopicID, nil
}

func (r *repoStub) UpdateTopic(_ context.Context, id int64, name, _, _ string) (TopicItem, error) {
	return TopicItem{ID: id, Name: name}, nil
}

func (r *repoStub) DeleteTopic(context.Context, int64) error { return nil }

func (r *repoStub) ListOptions(context.Context) ([]OptionItem, error) { return []OptionItem{}, nil }

func (r *repoStub) UpdateOptions(_ context.Context, values map[string]string) error {
	r.optionValues = values
	return nil
}

func (r *repoStub) ListImages(context.Context, ContentFilter) ([]ImageItem, bool, error) {
	return r.images, false, nil
}

type sessionStub struct {
	revoked []int64
	reason  string
}

func (s *sessionStub) RevokeUserSessions(_ context.Context, userID int64, reason string) error {
	s.revoked = append(s.revoked, userID)
	s.reason = reason
	return nil
}

func TestUpdateUserBlocksSelfRoleChange(t *testing.T) {
	repo := &repoStub{}
	service := NewService(repo)
	role := "user"
	_, err := service.UpdateUser(context.Background(), auth.Principal{UserID: 1}, 1, UserUpdate{Role: &role})
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected forbidden, got %v", err)
	}
	if repo.updateCalled {
		t.Fatal("self update must not reach repository")
	}
}

func TestUpdateUserBlocksDemotingLastAdmin(t *testing.T) {
	repo := &repoStub{
		findResult:   User{ID: 2, Roles: []string{"admin"}},
		activeAdmins: 0,
	}
	service := NewService(repo)
	role := "moderator"
	_, err := service.UpdateUser(context.Background(), auth.Principal{UserID: 1}, 2, UserUpdate{Role: &role})
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected forbidden, got %v", err)
	}
	if repo.updateCalled {
		t.Fatal("last admin must not be demoted")
	}
}

func TestUpdateUserRejectsUnknownRole(t *testing.T) {
	repo := &repoStub{findResult: User{ID: 2, Roles: []string{"user"}}, activeAdmins: 1}
	service := NewService(repo)
	role := "superuser"
	if _, err := service.UpdateUser(context.Background(), auth.Principal{UserID: 1}, 2, UserUpdate{Role: &role}); !errors.Is(err, ErrValidation) {
		t.Fatalf("expected validation error, got %v", err)
	}
}

func TestUpdateUserRevokesSessionsOnRoleChange(t *testing.T) {
	repo := &repoStub{findResult: User{ID: 2, Roles: []string{"user"}}, activeAdmins: 1}
	sessions := &sessionStub{}
	service := NewService(repo)
	service.SetSessionRevoker(sessions)

	role := "moderator"
	if _, err := service.UpdateUser(context.Background(), auth.Principal{UserID: 1}, 2, UserUpdate{Role: &role}); err != nil {
		t.Fatalf("update failed: %v", err)
	}
	if len(sessions.revoked) != 1 || sessions.revoked[0] != 2 || sessions.reason != "role_changed" {
		t.Fatalf("unexpected revoke: %+v", sessions)
	}
}

func TestUpdateUserRejectsUnknownStatus(t *testing.T) {
	repo := &repoStub{findResult: User{ID: 2, Roles: []string{"user"}}}
	service := NewService(repo)
	status := uint8(7)
	if _, err := service.UpdateUser(context.Background(), auth.Principal{UserID: 1}, 2, UserUpdate{Status: &status}); !errors.Is(err, ErrValidation) {
		t.Fatalf("expected validation error, got %v", err)
	}
}

func TestListContentRejectsUnknownKind(t *testing.T) {
	service := NewService(&repoStub{})
	if _, err := service.ListContent(context.Background(), "topic", ContentFilter{}); !errors.Is(err, ErrValidation) {
		t.Fatalf("expected validation error, got %v", err)
	}
}

func TestListContentDefaultsToLive(t *testing.T) {
	repo := &repoStub{}
	service := NewService(repo)
	page, err := service.ListContent(context.Background(), KindQuestion, ContentFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if page.Items == nil {
		t.Fatal("empty list must serialize as []")
	}
	if page.Page != 1 || page.PageSize != defaultPageSize {
		t.Fatalf("page not normalized: %+v", page)
	}
}

func TestCreateTopicValidatesName(t *testing.T) {
	service := NewService(&repoStub{createdTopicID: 9})
	if _, err := service.CreateTopic(context.Background(), "  ", "", ""); !errors.Is(err, ErrValidation) {
		t.Fatalf("expected validation error, got %v", err)
	}
	if _, err := service.CreateTopic(context.Background(), "123456789012345678901", "", ""); !errors.Is(err, ErrValidation) {
		t.Fatalf("expected length validation, got %v", err)
	}
	if _, err := service.CreateTopic(context.Background(), "有效话题", "描述", ""); err != nil {
		t.Fatalf("valid topic rejected: %v", err)
	}
}

func TestUpdateOptionsRejectsUnknownName(t *testing.T) {
	service := NewService(&repoStub{})
	if err := service.UpdateOptions(context.Background(), map[string]string{"unknown_option": "1"}); !errors.Is(err, ErrValidation) {
		t.Fatalf("expected validation error, got %v", err)
	}
}

func TestUpdateOptionsRejectsInvalidBool(t *testing.T) {
	service := NewService(&repoStub{})
	if err := service.UpdateOptions(context.Background(), map[string]string{"article_can_edit": "yes"}); !errors.Is(err, ErrValidation) {
		t.Fatalf("expected validation error, got %v", err)
	}
}

func TestUpdateOptionsAcceptsKnownValues(t *testing.T) {
	repo := &repoStub{}
	service := NewService(repo)
	values := map[string]string{"article_can_edit": "true", "article_can_edit_before": "3600"}
	if err := service.UpdateOptions(context.Background(), values); err != nil {
		t.Fatalf("valid options rejected: %v", err)
	}
	if repo.optionValues["article_can_edit"] != "true" {
		t.Fatalf("values not persisted: %+v", repo.optionValues)
	}
}
