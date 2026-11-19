package store_test

import (
	"errors"
	"testing"
	"time"

	"github.com/laposa/testflow/app-go/internal/db/dbtest"
	"github.com/laposa/testflow/app-go/internal/store"
)

func TestRuns(t *testing.T) {
	pool := dbtest.New(t)
	s := store.New(pool)
	ctx := t.Context()

	ada, err := s.UpsertUser(ctx, store.GitHubUser{ID: 1, Login: "ada", Name: "Ada"})
	if err != nil {
		t.Fatal(err)
	}
	sessionID, err := s.CreateTestSession(ctx, store.NewTestSession{
		IssuerID: ada.ID, IssuerName: ada.Name, Name: "Release", Environment: "beta",
		Services: []store.NewService{{
			Type: "automated", Name: "web", Path: "tests/web", RepositoryID: 7, RepositoryName: "laposa/shop",
			WorkflowID: 55, CommitSHA: "abc",
			Suites: []store.NewSuite{{Name: "checkout", Path: "p", Tests: []store.NewTest{{Name: "pay.cy.js", Path: "p/pay.cy.js"}, {Name: "refund.cy.js", Path: "p/refund.cy.js"}}}},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	ts, err := s.TestSession(ctx, sessionID)
	if err != nil {
		t.Fatal(err)
	}
	svc := ts.Services[0]

	run, err := s.CreateRun(ctx, store.NewRun{ServiceID: svc.ID, Type: store.RunAutomated, Status: store.RunUnknown})
	if err != nil {
		t.Fatal(err)
	}
	if run.ID == 0 || run.Status != "unknown" || run.HasResults || run.CreatedAt.IsZero() || !run.Syncing(time.Now()) {
		t.Errorf("created run = %+v", run)
	}
	older, err := s.CreateRun(ctx, store.NewRun{ServiceID: svc.ID, Type: store.RunAutomated, Status: store.RunUnknown})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `UPDATE service_runs SET created_at = now() - interval '25 hours' WHERE id = $1`, older.ID); err != nil {
		t.Fatal(err)
	}

	// The syncer's work lists.
	pending, err := s.PendingRuns(ctx, time.Now().Add(-store.RunSyncWindow))
	if err != nil {
		t.Fatal(err)
	}
	if len(pending) != 1 || pending[0].ID != run.ID || pending[0].Service.RepositoryName != "laposa/shop" ||
		pending[0].Service.WorkflowID != 55 || pending[0].Service.SessionID != sessionID || pending[0].Service.Branch != "master" {
		t.Fatalf("pending = %+v", pending)
	}
	stale, err := s.StaleRuns(ctx, time.Now().Add(-store.RunSyncWindow))
	if err != nil || len(stale) != 1 || stale[0].ID != older.ID {
		t.Fatalf("stale = %+v, %v", stale, err)
	}
	tests, err := s.ServiceTests(ctx, svc.ID)
	if err != nil || len(tests) != 2 || tests[0].Name != "pay.cy.js" {
		t.Fatalf("service tests = %+v, %v", tests, err)
	}

	// A status update guarded by the old status.
	started := time.Date(2026, 11, 19, 10, 0, 0, 0, time.UTC)
	err = s.UpdateRun(ctx, store.RunUpdate{ID: run.ID, OldStatus: "unknown", Status: "in_progress",
		GitHubRunID: 900, CommitSHA: "sha1", StartedAt: &started},
		&store.NewActivity{SessionID: sessionID, Type: store.ActivityRunStatusChanged, Body: "changed",
			Metadata: map[string]any{store.MetaRunID: run.ID}})
	if err != nil {
		t.Fatal(err)
	}
	if taken, err := s.MatchedGitHubRuns(ctx, time.Now().Add(-time.Hour)); err != nil || len(taken) != 1 || !taken[900] {
		t.Errorf("matched github runs = %v, %v", taken, err)
	}
	if taken, err := s.MatchedGitHubRuns(ctx, time.Now().Add(time.Hour)); err != nil || len(taken) != 0 {
		t.Errorf("matched github runs from the future = %v, %v", taken, err)
	}
	err = s.UpdateRun(ctx, store.RunUpdate{ID: run.ID, OldStatus: "unknown", Status: "success"},
		&store.NewActivity{SessionID: sessionID, Type: store.ActivityRunStatusChanged, Body: "must not be logged"})
	if !errors.Is(err, store.ErrConflict) {
		t.Fatalf("stale old status: err = %v, want ErrConflict", err)
	}

	// Completion with results; zero fields keep the stored values.
	finished := started.Add(90 * time.Second)
	report := "<testsuites/>"
	err = s.UpdateRun(ctx, store.RunUpdate{ID: run.ID, OldStatus: "in_progress", Status: "failure",
		Completion: &store.RunCompletion{
			FinishedAt: &finished, ResultLog: &report, RunLog: "the log",
			Totals: &store.RunTotals{Passed: 1, Failed: 1, Duration: 4.5},
			Results: []store.TestResult{
				{TestID: tests[0].ID, Status: "pass", Duration: 1.5},
				{TestID: tests[1].ID, Status: "fail", Comment: "see video", FailureMessage: "boom", Duration: 3},
			},
		}},
		&store.NewActivity{SessionID: sessionID, Type: store.ActivityRunStatusChanged, Body: "done",
			Metadata: map[string]any{store.MetaRunID: run.ID, store.MetaMaestroURL: "https://console.mobile.dev/uploads/x"}})
	if err != nil {
		t.Fatal(err)
	}

	d, err := s.RunDetail(ctx, sessionID, run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if d.Status != "failure" || d.GitHubRunID != 900 || d.CommitSHA != "sha1" || !d.StartedAt.Equal(started) ||
		!d.FinishedAt.Equal(finished) || !d.HasResults || !d.HasRunLog || !d.HasTotals || d.Passed != 1 || d.Failed != 1 ||
		d.Skipped != 0 || d.Duration != 4.5 || d.ResultLog != report || d.RunLog != "the log" || d.Syncing(time.Now()) {
		t.Errorf("detail = %+v", d.Run)
	}
	if r := d.Results[tests[1].ID]; r != (store.TestResult{TestID: tests[1].ID, Status: "fail", Comment: "see video", FailureMessage: "boom", Duration: 3}) {
		t.Errorf("result = %+v", r)
	}
	if _, err := s.RunDetail(ctx, sessionID+1, run.ID); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("run of another session: err = %v, want ErrNotFound", err)
	}

	// A second completion replaces the results.
	err = s.UpdateRun(ctx, store.RunUpdate{ID: run.ID, OldStatus: "failure", Status: "failure",
		Completion: &store.RunCompletion{FinishedAt: &finished, RunLog: ""}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if d, err = s.RunDetail(ctx, sessionID, run.ID); err != nil || len(d.Results) != 0 || d.HasResults || d.HasTotals || !d.HasRunLog {
		t.Errorf("after second completion = %+v, %v", d, err)
	}

	list, err := s.SessionRuns(ctx, sessionID)
	if err != nil || len(list) != 2 || list[0].ID != older.ID || list[1].ID != run.ID {
		t.Errorf("session runs = %+v, %v", list, err)
	}

	activity, err := s.SessionActivity(ctx, sessionID)
	if err != nil {
		t.Fatal(err)
	}
	if len(activity) != 3 || activity[0].Body != "done" || activity[0].RunID != run.ID ||
		activity[0].MaestroURL != "https://console.mobile.dev/uploads/x" || activity[1].RunID != run.ID ||
		activity[1].MaestroURL != "" || activity[2].RunID != 0 {
		t.Errorf("activity = %+v", activity)
	}
}
