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
