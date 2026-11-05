package profile

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/voidxo/voidxo/internal/modules/user"
)

type profileRepoStub struct {
	account      user.Account
	updateCalled bool
	lastParams   user.UpdateProfileParams
}

func (r *profileRepoStub) FindByLogin(context.Context, string) (user.Account, error) {
	return r.account, nil
}
func (r *profileRepoStub) FindByID(context.Context, int64) (user.Account, error) {
	return r.account, nil
}
func (r *profileRepoStub) Create(context.Context, user.CreateParams) (user.Account, error) {
	return user.Account{}, nil
}
func (r *profileRepoStub) SetPassword(context.Context, int64, string) error { return nil }
func (r *profileRepoStub) RecordLogin(context.Context, int64, string) error { return nil }
func (r *profileRepoStub) UpdateProfile(_ context.Context, _ int64, params user.UpdateProfileParams) (user.Account, error) {
	r.updateCalled = true
	r.lastParams = params
	if params.Headline != nil {
		r.account.Headline = *params.Headline
	}
	return r.account, nil
}
func (r *profileRepoStub) Permissions(context.Context, int64) ([]string, error) { return nil, nil }
func (r *profileRepoStub) Roles(context.Context, int64) ([]string, error)       { return nil, nil }

func TestUpdateTrimsAndValidatesProfile(t *testing.T) {
	repo := &profileRepoStub{account: user.Account{ID: 1, Username: "alice"}}
	service := NewService(repo)
	headline := "  一句话介绍  "
	account, err := service.Update(context.Background(), 1, user.UpdateProfileParams{Headline: &headline})
	if err != nil {
		t.Fatal(err)
	}
	if !repo.updateCalled || account.Headline != "一句话介绍" {
		t.Fatalf("profile update did not normalize: %#v", account)
	}

	tooLong := strings.Repeat("字", 161)
	_, err = service.Update(context.Background(), 1, user.UpdateProfileParams{Bio: &tooLong})
	var validation *ValidationError
	if err == nil || !errors.As(err, &validation) {
		t.Fatalf("long bio was accepted: %v", err)
	}
}

func (r *profileRepoStub) SetAvatar(context.Context, int64, string) error    { return nil }
func (r *profileRepoStub) EmailExists(context.Context, string) (bool, error) { return false, nil }
func (r *profileRepoStub) FindByEmail(context.Context, string) (user.Account, error) {
	return user.Account{}, user.ErrNotFound
}
func (r *profileRepoStub) SetCover(context.Context, int64, string) error { return nil }

func (r *profileRepoStub) FindPublic(context.Context, int64) (user.PublicProfile, error) {
	return user.PublicProfile{}, user.ErrNotFound
}

func TestPublicRejectsUnknownUser(t *testing.T) {
	service := NewService(&profileRepoStub{})
	if _, err := service.Public(context.Background(), 0); err == nil {
		t.Fatal("zero id must be rejected")
	}
}

func TestUserContentRequiresLister(t *testing.T) {
	service := NewService(&profileRepoStub{})
	if _, err := service.UserContent(context.Background(), "question", 1, 1, 20); err == nil {
		t.Fatal("missing content lister must fail loudly")
	}
}

func (r *profileRepoStub) ListPublic(context.Context, string, int, int) ([]user.PublicProfile, bool, error) {
	return nil, false, nil
}
