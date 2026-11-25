package store_test

import (
	"errors"
	"testing"

	"github.com/laposa/testflow/app-go/internal/db/dbtest"
	"github.com/laposa/testflow/app-go/internal/store"
)

func TestReviews(t *testing.T) {
	pool := dbtest.New(t)
	s := store.New(pool)
	ctx := t.Context()

	user := func(id int64, name, email string) store.User {
		u, err := s.UpsertUser(ctx, store.GitHubUser{ID: id, Login: name, Name: name, Email: email})
		if err != nil {
			t.Fatal(err)
		}
		return u
	}
	ada, bob, cy := user(1, "Ada", "ada@example.com"), user(2, "bob", ""), user(3, "Cy", "cy@example.com")
	sessionID, err := s.CreateTestSession(ctx, store.NewTestSession{
		IssuerID: ada.ID, IssuerName: ada.Name, Name: "Release", Environment: "beta",
		Services: []store.NewService{{Type: "manual", Name: "web", Path: "p", RepositoryID: 7, RepositoryName: "laposa/shop",
			CommitSHA: "abc", Suites: []store.NewSuite{{Name: "s", Path: "p", Tests: []store.NewTest{{Name: "a.manual.yaml", Path: "p/a"}}}}}},
	})
	if err != nil {
		t.Fatal(err)
	}

	cands, err := s.ReviewCandidates(ctx, sessionID)
	if err != nil || len(cands) != 3 || cands[0].Name != "Ada" || cands[1].Name != "bob" {
		t.Fatalf("candidates = %+v, %v", cands, err)
	}

	r, err := s.RequestReview(ctx, sessionID, ada, bob.ID)
	if err != nil {
		t.Fatal(err)
	}
	if r.ID == 0 || r.Reviewer.Name != "bob" || r.Reviewer.Email != "" || r.Status != "pending" {
		t.Errorf("request = %+v", r)
	}
	if _, err := s.RequestReview(ctx, sessionID, cy, bob.ID); !errors.Is(err, store.ErrConflict) {
		t.Errorf("second pending request for bob: err = %v, want ErrConflict", err)
	}
	if _, err := s.RequestReview(ctx, sessionID, ada, 999); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("unknown reviewer: err = %v, want ErrNotFound", err)
	}
	if _, err := s.RequestReview(ctx, sessionID+1, ada, cy.ID); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("unknown session: err = %v, want ErrNotFound", err)
	}
	r2, err := s.RequestReview(ctx, sessionID, bob, cy.ID)
	if err != nil || r2.Reviewer.Email != "cy@example.com" {
		t.Fatalf("request from bob = %+v, %v", r2, err)
	}

	pending, err := s.PendingReviews(ctx, sessionID)
	if err != nil || len(pending) != 2 || pending[0].ID != r.ID || pending[0].Requester.Name != "Ada" || pending[1].Reviewer.Name != "Cy" {
		t.Fatalf("pending = %+v, %v", pending, err)
	}
	if cands, _ = s.ReviewCandidates(ctx, sessionID); len(cands) != 1 || cands[0].ID != ada.ID {
		t.Errorf("candidates with two pending = %+v", cands)
	}

	// Only the reviewer completes; only pending requests; only the session's.
	if err := s.CompleteReview(ctx, sessionID, r.ID, ada, true); !errors.Is(err, store.ErrForbidden) {
		t.Errorf("approve by the requester: err = %v, want ErrForbidden", err)
	}
	if err := s.CompleteReview(ctx, sessionID+1, r.ID, bob, true); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("approve via another session: err = %v, want ErrNotFound", err)
	}
	if err := s.CompleteReview(ctx, sessionID, r.ID, bob, true); err != nil {
		t.Fatal(err)
	}
	if err := s.CompleteReview(ctx, sessionID, r.ID, bob, false); !errors.Is(err, store.ErrConflict) {
		t.Errorf("reject after approve: err = %v, want ErrConflict", err)
	}
	if err := s.RevokeReview(ctx, sessionID, r.ID, ada.ID); !errors.Is(err, store.ErrConflict) {
		t.Errorf("revoke after approve: err = %v, want ErrConflict", err)
	}
	// bob can be asked again once his request is done.
	if _, err := s.RequestReview(ctx, sessionID, cy, bob.ID); err != nil {
		t.Errorf("new request after approval: %v", err)
	}

	// Only the requester revokes; nothing is logged.
	if err := s.RevokeReview(ctx, sessionID, r2.ID, cy.ID); !errors.Is(err, store.ErrForbidden) {
		t.Errorf("revoke by the reviewer: err = %v, want ErrForbidden", err)
	}
	if err := s.RevokeReview(ctx, sessionID, r2.ID, bob.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.CompleteReview(ctx, sessionID, r2.ID, cy, true); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("approve a revoked request: err = %v, want ErrNotFound", err)
	}

	activity, err := s.SessionActivity(ctx, sessionID)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, e := range activity {
		got = append(got, string(e.Type)+": "+e.Body+" ["+e.UserName+"]")
	}
	want := []string{
		"review_requested: Cy requested a review from bob [Cy]",
		"review_approved: bob approved the review [bob]",
		"review_requested: bob requested a review from Cy [bob]",
		"review_requested: Ada requested a review from bob [Ada]",
		"session_created: Ada created the session. [Ada]",
	}
	if len(got) != len(want) {
		t.Fatalf("activity = %q", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("activity[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}
