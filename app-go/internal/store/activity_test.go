package store_test

import (
	"errors"
	"testing"

	"github.com/laposa/testflow/app-go/internal/db/dbtest"
	"github.com/laposa/testflow/app-go/internal/store"
)

func TestSessionActivityAndComments(t *testing.T) {
	pool := dbtest.New(t)
	s := store.New(pool)
	ctx := t.Context()

	ada, err := s.UpsertUser(ctx, store.GitHubUser{ID: 1, Login: "ada", Name: "Ada"})
	if err != nil {
		t.Fatal(err)
	}
	bob, err := s.UpsertUser(ctx, store.GitHubUser{ID: 2, Login: "bob", Name: "Bob"})
	if err != nil {
		t.Fatal(err)
	}
	newSession := func() int64 {
		id, err := s.CreateTestSession(ctx, store.NewTestSession{
			IssuerID: ada.ID, IssuerName: ada.Name, Name: "x", Environment: "beta",
			Services: []store.NewService{{Type: "manual", Name: "web", Path: "p", RepositoryName: "o/r", CommitSHA: "c"}},
		})
		if err != nil {
			t.Fatal(err)
		}
		return id
	}
	id, other := newSession(), newSession()

	if err := s.AddComment(ctx, id, bob.ID, "Looks good to me"); err != nil {
		t.Fatal(err)
	}
	if err := s.AddComment(ctx, id, ada.ID, "Thanks,\nmerging"); err != nil {
		t.Fatal(err)
	}
	// A system entry (user_id NULL), as the syncer will write.
	if _, err := pool.Exec(ctx, `INSERT INTO activity_log (session_id, type, body) VALUES ($1, 'run_status_changed', 'Run 1 changed')`, id); err != nil {
		t.Fatal(err)
	}

	got, err := s.SessionActivity(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 4 {
		t.Fatalf("activity = %+v", got)
	}
	if sys := got[0]; sys.Type != store.ActivityRunStatusChanged || sys.UserID != 0 || sys.UserName != "" {
		t.Errorf("system entry = %+v", sys)
	}
	if c := got[1]; c.Type != store.ActivityComment || c.UserID != ada.ID || c.UserName != "Ada" || c.Body != "Thanks,\nmerging" || c.CreatedAt.IsZero() {
		t.Errorf("newest comment = %+v", c)
	}
	if c := got[2]; c.UserName != "Bob" || c.Body != "Looks good to me" {
		t.Errorf("older comment = %+v", c)
	}
	if created := got[3]; created.Type != store.ActivitySessionCreated || created.Body != "Ada created the session." {
		t.Errorf("oldest entry = %+v, want session_created", created)
	}

	if other, err := s.SessionActivity(ctx, other); err != nil || len(other) != 1 {
		t.Errorf("other session = %+v, %v; want only its own session_created", other, err)
	}
	if err := s.AddComment(ctx, id+100, ada.ID, "Hello there"); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("comment on a missing session: err = %v, want ErrNotFound", err)
	}
}
