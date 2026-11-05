package engagement

import (
	"context"
	"errors"
	"testing"
)

type repoStub struct {
	voteType      string
	followKind    string
	reportReason  string
	markedIDs     []int64
	markReceiver  int64
	deletedIDs    []int64
	notifiedEvent *NotificationEvent
}

func (r *repoStub) Vote(_ context.Context, _ int64, _ string, _ int64, voteType string) (VoteSummary, error) {
	r.voteType = voteType
	return VoteSummary{VoteCount: 1, VoteUpCount: 1, MyVote: voteType}, nil
}
func (r *repoStub) RemoveVote(context.Context, int64, string, int64) (VoteSummary, error) {
	return VoteSummary{}, nil
}
func (r *repoStub) VoteState(context.Context, int64, string, int64) (VoteSummary, error) {
	return VoteSummary{}, nil
}
func (r *repoStub) MyVote(context.Context, int64, string, int64) (string, error) { return "", nil }
func (r *repoStub) Follow(_ context.Context, _ int64, kind string, _ int64) (FollowSummary, error) {
	r.followKind = kind
	return FollowSummary{Following: true, FollowerCount: 1}, nil
}
func (r *repoStub) Unfollow(context.Context, int64, string, int64) (FollowSummary, error) {
	return FollowSummary{}, nil
}
func (r *repoStub) FollowState(context.Context, int64, string, int64) (FollowSummary, error) {
	return FollowSummary{}, nil
}
func (r *repoStub) IsFollowing(context.Context, int64, string, int64) (bool, error) {
	return false, nil
}
func (r *repoStub) CreateReport(_ context.Context, _ int64, _ string, _ int64, reason string) (Report, error) {
	r.reportReason = reason
	return Report{ID: 1, Reason: reason}, nil
}
func (r *repoStub) ListReports(context.Context, string, int, int) ([]Report, bool, error) {
	return nil, false, nil
}
func (r *repoStub) ResolveReport(_ context.Context, id, _ int64, status string) (Report, error) {
	return Report{ID: id, Status: status}, nil
}
func (r *repoStub) CreateNotification(_ context.Context, event NotificationEvent) error {
	r.notifiedEvent = &event
	return nil
}
func (r *repoStub) ListNotifications(context.Context, int64, int, int) ([]Notification, bool, error) {
	return nil, false, nil
}
func (r *repoStub) UnreadNotificationCount(context.Context, int64) (int, error) { return 3, nil }
func (r *repoStub) MarkNotificationsRead(_ context.Context, receiverID int64, ids []int64) (int, error) {
	r.markReceiver = receiverID
	r.markedIDs = ids
	return len(ids), nil
}
func (r *repoStub) DeleteNotifications(_ context.Context, receiverID int64, ids []int64) (int, error) {
	r.markReceiver = receiverID
	r.deletedIDs = ids
	return len(ids), nil
}

func TestVoteValidation(t *testing.T) {
	repo := &repoStub{}
	service := NewService(repo)

	if _, err := service.Vote(context.Background(), 0, TypeQuestion, 1, "up"); err == nil {
		t.Fatal("anonymous vote accepted")
	}
	if _, err := service.Vote(context.Background(), 1, "topic", 1, "up"); err == nil {
		t.Fatal("topic vote accepted")
	}
	if _, err := service.Vote(context.Background(), 1, TypeQuestion, 1, "sideways"); err == nil {
		t.Fatal("invalid vote type accepted")
	}
	if _, err := service.Vote(context.Background(), 1, TypeQuestion, 1, "down"); err != nil {
		t.Fatal(err)
	}
	if repo.voteType != "down" {
		t.Fatalf("vote type not forwarded: %s", repo.voteType)
	}
}

func TestFollowRejectsSelfAndUnknownType(t *testing.T) {
	service := NewService(&repoStub{})
	if _, err := service.Follow(context.Background(), 5, TypeUser, 5); !errors.Is(err, ErrSelfFollow) {
		t.Fatalf("self follow returned %v", err)
	}
	if _, err := service.Follow(context.Background(), 5, "comment", 1); err == nil {
		t.Fatal("comment follow accepted")
	}
	if _, err := service.Follow(context.Background(), 5, TypeUser, 6); err != nil {
		t.Fatal(err)
	}
}

func TestReportValidationAndDuplicate(t *testing.T) {
	repo := &repoStub{}
	service := NewService(repo)
	if _, err := service.CreateReport(context.Background(), 1, "topic", 1, "垃圾信息"); err == nil {
		t.Fatal("topic report accepted")
	}
	if _, err := service.CreateReport(context.Background(), 1, TypeQuestion, 1, "短"); err == nil {
		t.Fatal("too short reason accepted")
	}
	if _, err := service.CreateReport(context.Background(), 1, TypeQuestion, 1, "  垃圾信息  "); err != nil {
		t.Fatal(err)
	}
	if repo.reportReason != "垃圾信息" {
		t.Fatalf("reason not trimmed: %q", repo.reportReason)
	}
}

func TestBatchOperationsAlwaysCarryReceiver(t *testing.T) {
	repo := &repoStub{}
	service := NewService(repo)
	if _, err := service.MarkRead(context.Background(), 9, []int64{1, 0, 1, 2}); err != nil {
		t.Fatal(err)
	}
	if repo.markReceiver != 9 {
		t.Fatalf("receiver filter missing: %d", repo.markReceiver)
	}
	if len(repo.markedIDs) != 2 {
		t.Fatalf("IDs were not normalized: %#v", repo.markedIDs)
	}
	if _, err := service.MarkRead(context.Background(), 9, nil); err == nil {
		t.Fatal("empty batch accepted")
	}
	tooMany := make([]int64, maxBatchSize+1)
	for i := range tooMany {
		tooMany[i] = int64(i + 1)
	}
	if _, err := service.DeleteNotifications(context.Background(), 9, tooMany); err == nil {
		t.Fatal("oversized batch accepted")
	}
}

func TestNotificationSkipsSelfEvents(t *testing.T) {
	repo := &repoStub{}
	service := NewService(repo)
	if err := service.Notify(context.Background(), NotificationEvent{ReceiverID: 5, SenderID: 5, Type: "question_commented"}); err != nil {
		t.Fatal(err)
	}
	if repo.notifiedEvent != nil {
		t.Fatal("self notification was delivered")
	}
	if err := service.Notify(context.Background(), NotificationEvent{ReceiverID: 5, SenderID: 6, Type: "question_commented"}); err != nil {
		t.Fatal(err)
	}
	if repo.notifiedEvent == nil || repo.notifiedEvent.ReceiverID != 5 {
		t.Fatal("notification was not delivered")
	}
}

func TestEnsureItemsReturnsEmptySlice(t *testing.T) {
	if ensureItems[Report](nil) == nil {
		t.Fatal("nil list must serialize as []")
	}
}

func (r *repoStub) FollowStates(context.Context, int64, string, []int64) (map[int64]bool, error) {
	return map[int64]bool{}, nil
}
