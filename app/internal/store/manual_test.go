package store_test

import (
	"errors"
	"strconv"
	"testing"

	"github.com/laposa/testflow/app-go/internal/db/dbtest"
	"github.com/laposa/testflow/app-go/internal/store"
)

func TestManualRuns(t *testing.T) {
	pool := dbtest.New(t)
	s := store.New(pool)
	ctx := t.Context()

	ada, err := s.UpsertUser(ctx, store.GitHubUser{ID: 1, Login: "ada", Name: "Ada"})
	if err != nil {
		t.Fatal(err)
	}
	sessionID, err := s.CreateTestSession(ctx, store.NewTestSession{
		IssuerID: ada.ID, IssuerName: ada.Name, Name: "Release", Environment: "beta",
		Services: []store.NewService{
			{
				Type: "manual", Name: "web", Path: "tests/web", RepositoryID: 7, RepositoryName: "laposa/shop", CommitSHA: "abc",
				Suites: []store.NewSuite{{Name: "login", Path: "p", Tests: []store.NewTest{
					{Name: "a.manual.yaml", Path: "p/a.manual.yaml"},
					{Name: "b.manual.yaml", Path: "p/b.manual.yaml"},
					{Name: "c.manual.yaml", Path: "p/c.manual.yaml"},
				}}},
			},
			{
				Type: "manual", Name: "api", Path: "tests/api", RepositoryID: 7, RepositoryName: "laposa/shop", CommitSHA: "abc",
				Suites: []store.NewSuite{{Name: "orders", Path: "q", Tests: []store.NewTest{{Name: "x.manual.yaml", Path: "q/x.manual.yaml"}}}},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	ts, err := s.TestSession(ctx, sessionID)
	if err != nil {
		t.Fatal(err)
	}
	web, api := ts.Services[0], ts.Services[1]
	if web.Name != "web" {
		web, api = api, web
	}
	a, b, c := web.Suites[0].Tests[0], web.Suites[0].Tests[1], web.Suites[0].Tests[2]

	run, err := s.StartManualRun(ctx, web, ada)
	if err != nil {
		t.Fatal(err)
	}
	if run.Type != "manual" || run.Status != "in_progress" || run.StartedAt == nil || run.CommitSHA != "abc" || run.HasTotals {
		t.Errorf("started = %+v", run)
	}

	// Results are upserted and counted.
	for _, r := range []store.TestResult{
		{TestID: a.ID, Status: "fail", Comment: "button missing"},
		{TestID: b.ID, Status: "pass"},
		{TestID: a.ID, Status: "pass", Comment: "fixed"},
	} {
		if err := s.SaveManualResult(ctx, run.ID, r); err != nil {
			t.Fatal(err)
		}
	}
	d, err := s.RunDetail(ctx, sessionID, run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !d.HasTotals || d.Passed != 2 || d.Failed != 0 || d.Skipped != 0 || d.FinishedAt != nil ||
		d.Results[a.ID].Comment != "fixed" || len(d.Results) != 2 {
		t.Errorf("after saving = %+v %+v", d.Run, d.Results)
	}
	if err := s.SaveManualResult(ctx, run.ID, store.TestResult{TestID: api.Suites[0].Tests[0].ID, Status: "pass"}); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("test of another service: err = %v, want ErrNotFound", err)
	}
	if err := s.SaveManualResult(ctx, run.ID, store.TestResult{TestID: c.ID, Status: "error"}); err == nil {
		t.Error("invalid status saved")
	}

	// Starting again finishes the open run: c is skipped.
	again, err := s.StartManualRun(ctx, web, ada)
	if err != nil {
		t.Fatal(err)
	}
	if d, err = s.RunDetail(ctx, sessionID, run.ID); err != nil {
		t.Fatal(err)
	}
	if d.Status != "success" || d.FinishedAt == nil || d.Passed != 2 || d.Skipped != 1 || d.Results[c.ID].Status != "skipped" {
		t.Errorf("open run after a new start = %+v %+v", d.Run, d.Results)
	}
	if err := s.SaveManualResult(ctx, run.ID, store.TestResult{TestID: c.ID, Status: "pass"}); !errors.Is(err, store.ErrConflict) {
		t.Errorf("save on a finished run: err = %v, want ErrConflict", err)
	}

	// Finishing explicitly.
	if err := s.SaveManualResult(ctx, again.ID, store.TestResult{TestID: b.ID, Status: "fail", Comment: "broken"}); err != nil {
		t.Fatal(err)
	}
	if err := s.FinishManualRun(ctx, again.ID, ada.ID); err != nil {
		t.Fatal(err)
	}
	if d, err = s.RunDetail(ctx, sessionID, again.ID); err != nil || d.Status != "success" || d.FinishedAt == nil ||
		d.Passed != 0 || d.Failed != 1 || d.Skipped != 2 {
		t.Errorf("finished = %+v, %v", d.Run, err)
	}
	if err := s.FinishManualRun(ctx, again.ID, ada.ID); !errors.Is(err, store.ErrConflict) {
		t.Errorf("second finish: err = %v, want ErrConflict", err)
	}
	if err := s.FinishManualRun(ctx, again.ID+100, ada.ID); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("finish of a missing run: err = %v, want ErrNotFound", err)
	}
	auto, err := s.CreateRun(ctx, store.NewRun{ServiceID: web.ID, Type: store.RunAutomated, Status: store.RunInProgress})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.FinishManualRun(ctx, auto.ID, ada.ID); !errors.Is(err, store.ErrConflict) {
		t.Errorf("finish of an automated run: err = %v, want ErrConflict", err)
	}

	activity, err := s.SessionActivity(ctx, sessionID)
	if err != nil {
		t.Fatal(err)
	}
	var bodies []string
	for _, e := range activity {
		bodies = append(bodies, string(e.Type)+": "+e.Body)
		if e.Type != store.ActivitySessionCreated && (e.RunID == 0 || e.UserName != "Ada") {
			t.Errorf("entry without run or user: %+v", e)
		}
	}
	want := []string{
		"manual_run_finished: Finished run " + id(again.ID) + " for shop/web (manual) with 0 passed and 1 failed tests",
		"manual_run_started: Manual run " + id(again.ID) + " for shop/web (manual) started by Ada",
		"manual_run_finished: Finished run " + id(run.ID) + " for shop/web (manual) with 2 passed and 0 failed tests",
		"manual_run_started: Manual run " + id(run.ID) + " for shop/web (manual) started by Ada",
	}
	for i, w := range want {
		if i >= len(bodies) || bodies[i] != w {
			t.Errorf("activity[%d] = %q, want %q", i, at(bodies, i), w)
		}
	}
}

func id(n int64) string { return strconv.FormatInt(n, 10) }

func at(s []string, i int) string {
	if i < len(s) {
		return s[i]
	}
	return ""
}
