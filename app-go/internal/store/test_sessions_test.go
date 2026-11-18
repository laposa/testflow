package store_test

import (
	"errors"
	"testing"

	"github.com/laposa/testflow/app-go/internal/db/dbtest"
	"github.com/laposa/testflow/app-go/internal/store"
)

func TestServiceDisplayName(t *testing.T) {
	svc := store.Service{RepositoryName: "laposa/shop-tests", Name: "web", Type: "manual"}
	if got := svc.DisplayName(); got != "shop-tests/web (manual)" {
		t.Errorf("DisplayName = %q", got)
	}
}

func TestCreateAndReadTestSession(t *testing.T) {
	pool := dbtest.New(t)
	s := store.New(pool)
	ctx := t.Context()

	u, err := s.UpsertUser(ctx, store.GitHubUser{ID: 1, Login: "ada", Name: "Ada"})
	if err != nil {
		t.Fatal(err)
	}
	id, err := s.CreateTestSession(ctx, store.NewTestSession{
		IssuerID: u.ID, IssuerName: u.Name, Name: "Release 1.2", Environment: "beta",
		Services: []store.NewService{
			{
				Type: "automated", Name: "web", Path: "tests/web", RepositoryID: 1 << 40,
				RepositoryName: "laposa/shop-tests", WorkflowID: 1 << 35, CommitSHA: "abc",
				Suites: []store.NewSuite{
					{Name: "checkout", Path: "tests/web/tests/checkout", Tests: []store.NewTest{
						{Name: "z.cy.js", Path: "tests/web/tests/checkout/z.cy.js"},
						{Name: "a.cy.js", Path: "tests/web/tests/checkout/a.cy.js"},
					}},
					{Name: "account", Path: "tests/web/tests/account", Tests: []store.NewTest{
						{Name: "login.cy.js", Path: "tests/web/tests/account/login.cy.js"},
					}},
				},
			},
			{
				Type: "manual", Name: "mobile", Path: "tests/mobile", RepositoryID: 2,
				RepositoryName: "laposa/app", CommitSHA: "def", Branch: "master",
				Suites: []store.NewSuite{{Name: "login", Path: "tests/mobile/tests/login", Tests: []store.NewTest{
					{Name: "a.manual.yaml", Path: "tests/mobile/tests/login/a.manual.yaml"},
				}}},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	ts, err := s.TestSession(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if ts.ID != id || ts.IssuerID != u.ID || ts.IssuerName != "Ada" || ts.Name != "Release 1.2" || ts.Environment != "beta" || ts.CreatedAt.IsZero() {
		t.Errorf("session = %+v", ts)
	}
	if len(ts.Services) != 2 {
		t.Fatalf("services = %+v", ts.Services)
	}
	web, mobile := ts.Services[0], ts.Services[1]
	if web.Type != "automated" || web.RepositoryID != 1<<40 || web.WorkflowID != 1<<35 || web.Branch != "master" || web.CommitSHA != "abc" || web.SessionID != id {
		t.Errorf("web = %+v", web)
	}
	if mobile.WorkflowID != 0 {
		t.Errorf("mobile workflow = %d, want 0 (NULL)", mobile.WorkflowID)
	}
	if len(web.Suites) != 2 || web.Suites[0].Name != "checkout" || web.Suites[1].Name != "account" {
		t.Fatalf("web suites = %+v", web.Suites)
	}
	if tests := web.Suites[0].Tests; len(tests) != 2 || tests[0].Name != "z.cy.js" || tests[1].Name != "a.cy.js" || tests[0].ID == 0 {
		t.Errorf("checkout tests keep the given order: %+v", tests)
	}
	if len(mobile.Suites) != 1 || len(mobile.Suites[0].Tests) != 1 {
		t.Errorf("mobile suites = %+v", mobile.Suites)
	}

	var (
		typ, body string
		userID    *int64
		meta      string
	)
	if err := pool.QueryRow(ctx, `SELECT type, body, user_id, metadata::text FROM activity_log WHERE session_id = $1`, id).
		Scan(&typ, &body, &userID, &meta); err != nil {
		t.Fatal(err)
	}
	if typ != "session_created" || body != "Ada created the session." || userID == nil || *userID != u.ID || meta != "{}" {
		t.Errorf("activity = %q %q %v %q", typ, body, userID, meta)
	}

	if _, err := s.TestSession(ctx, id+100); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("missing session: err = %v, want ErrNotFound", err)
	}
}

func TestCreateTestSessionIsAtomic(t *testing.T) {
	pool := dbtest.New(t)
	s := store.New(pool)
	ctx := t.Context()
	u, err := s.UpsertUser(ctx, store.GitHubUser{ID: 1, Login: "ada"})
	if err != nil {
		t.Fatal(err)
	}
	// The second service violates the type check, so nothing is stored.
	_, err = s.CreateTestSession(ctx, store.NewTestSession{
		IssuerID: u.ID, IssuerName: "ada", Name: "x", Environment: "prod",
		Services: []store.NewService{
			{Type: "automated", Name: "web", Path: "p", RepositoryName: "o/r", CommitSHA: "c"},
			{Type: "bogus", Name: "web", Path: "p", RepositoryName: "o/r", CommitSHA: "c"},
		},
	})
	if err == nil {
		t.Fatal("want error")
	}
	var n int
	if err := pool.QueryRow(ctx, `SELECT (SELECT count(*) FROM test_sessions) + (SELECT count(*) FROM session_services) + (SELECT count(*) FROM activity_log)`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Errorf("%d rows left after a failed create", n)
	}

	if _, err := s.CreateTestSession(ctx, store.NewTestSession{IssuerID: u.ID, Name: "x", Environment: "prod"}); err == nil {
		t.Error("session without services accepted")
	}
}

func TestListTestSessions(t *testing.T) {
	pool := dbtest.New(t)
	s := store.New(pool)
	ctx := t.Context()

	u, err := s.UpsertUser(ctx, store.GitHubUser{ID: 1, Login: "ada", Name: "Ada"})
	if err != nil {
		t.Fatal(err)
	}
	if list, err := s.ListTestSessions(ctx); err != nil || len(list) != 0 {
		t.Fatalf("empty list = %v, %v", list, err)
	}

	svc := func(name string) store.NewService {
		return store.NewService{Type: "automated", Name: name, Path: "tests/" + name, RepositoryName: "o/r", CommitSHA: "c",
			Suites: []store.NewSuite{{Name: "s", Path: "p", Tests: []store.NewTest{{Name: "t", Path: "p/t"}}}}}
	}
	create := func(name string, services ...store.NewService) int64 {
		id, err := s.CreateTestSession(ctx, store.NewTestSession{IssuerID: u.ID, IssuerName: u.Name,
			Name: name, Environment: "beta", Services: services})
		if err != nil {
			t.Fatal(err)
		}
		return id
	}
	serviceID := func(session int64, name string) int64 {
		var id int64
		if err := pool.QueryRow(ctx, `SELECT id FROM session_services WHERE session_id = $1 AND name = $2`, session, name).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	run := func(service int64, status string, counts ...int) {
		var p, f, sk *int
		if len(counts) == 3 {
			p, f, sk = &counts[0], &counts[1], &counts[2]
		}
		if _, err := pool.Exec(ctx, `INSERT INTO service_runs (service_id, type, status, passed, failed, skipped)
			VALUES ($1, 'automated', $2, $3, $4, $5)`, service, status, p, f, sk); err != nil {
			t.Fatal(err)
		}
	}

	noRuns := create("no runs", svc("web"))

	finished := create("finished", svc("web"), svc("api"))
	web, api := serviceID(finished, "web"), serviceID(finished, "api")
	run(web, "failure", 1, 9, 9) // superseded by the next run of web
	run(web, "success", 3, 1, 0)
	run(api, "failure", 0, 2, 4)

	running := create("running", svc("web"), svc("api"), svc("mobile"))
	run(serviceID(running, "web"), "success", 5, 0, 1)
	run(serviceID(running, "api"), "queued")
	run(serviceID(running, "mobile"), "in_progress")
	// A success without results counts as finished (legacy parity).
	emptySuccess := create("empty success", svc("web"))
	run(serviceID(emptySuccess, "web"), "success", 0, 0, 0)

	list, err := s.ListTestSessions(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 4 {
		t.Fatalf("list = %+v", list)
	}
	byID := map[int64]store.SessionSummary{}
	for i, ss := range list {
		byID[ss.ID] = ss
		if i > 0 && list[i-1].ID < ss.ID {
			t.Errorf("not ordered newest first: %+v", list)
		}
	}

	if got := byID[noRuns]; got.Runs != 0 || got.Running() || got.IssuerName != "Ada" || got.Environment != "beta" || got.CreatedAt.IsZero() {
		t.Errorf("no runs = %+v", got)
	}
	if got := byID[finished]; got.Runs != 2 || got.Running() || got.Passed != 3 || got.Failed != 3 || got.Skipped != 4 {
		t.Errorf("finished = %+v, want only the latest run per service added up", got)
	}
	if got := byID[running]; got.Runs != 3 || got.RunningStatus != "in_progress" || got.Passed != 5 || got.Skipped != 1 {
		t.Errorf("running = %+v, want the newest unfinished status", got)
	}
	if got := byID[emptySuccess]; got.Runs != 1 || got.Running() || got.Passed+got.Failed+got.Skipped != 0 {
		t.Errorf("empty success = %+v", got)
	}
}
