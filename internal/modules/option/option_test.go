package option

import (
	"context"
	"testing"
)

type repositoryStub struct {
	values map[string]string
	names  []string
}

func (r *repositoryStub) FindByNames(_ context.Context, names []string) (map[string]string, error) {
	r.names = names
	return r.values, nil
}

func TestPublicFiltersAndConvertsValues(t *testing.T) {
	repo := &repositoryStub{values: map[string]string{
		"site_name": "VOIDXO", "question_can_edit": "true", "question_can_edit_before": "300",
		"smtp_password": "must-not-leak",
	}}
	values, err := NewService(repo).Public(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if values["site_name"] != "VOIDXO" || values["question_can_edit"] != true || values["question_can_edit_before"] != 300 {
		t.Fatalf("unexpected options: %#v", values)
	}
	if _, exists := values["smtp_password"]; exists {
		t.Fatal("private option leaked")
	}
}

func TestPublicRejectsInvalidInteger(t *testing.T) {
	repo := &repositoryStub{values: map[string]string{"question_can_edit_before": "invalid"}}
	if _, err := NewService(repo).Public(context.Background()); err == nil {
		t.Fatal("expected invalid integer error")
	}
}
