package web

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/laposa/testflow/app-go/internal/auth/authtest"
	"github.com/laposa/testflow/app-go/internal/discovery"
	"github.com/laposa/testflow/app-go/internal/store"
)

// fakeDiscovery returns res (or err) and knows only res as a snapshot.
type fakeDiscovery struct {
	mu        sync.Mutex
	res       *discovery.Result
	err       error
	refreshes int
}

func (f *fakeDiscovery) Tests(_ context.Context, refresh bool) (*discovery.Result, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if refresh {
		f.refreshes++
	}
	return f.res, f.err
}

func (f *fakeDiscovery) Snapshot(id string) (*discovery.Result, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.res != nil && f.res.ID == id {
		return f.res, true
	}
	return nil, false
}

// fakeTestSessions keeps created sessions in memory.
type fakeTestSessions struct {
	mu       sync.Mutex
	users    *authtest.Store
	created  []store.NewTestSession
	sessions map[int64]store.TestSession
	err      error
}

func newFakeTestSessions(users *authtest.Store) *fakeTestSessions {
	return &fakeTestSessions{users: users, sessions: map[int64]store.TestSession{}}
}

func (f *fakeTestSessions) CreateTestSession(_ context.Context, ns store.NewTestSession) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return 0, f.err
	}
	f.created = append(f.created, ns)
	id := int64(len(f.created))
	ts := store.TestSession{ID: id, IssuerID: ns.IssuerID, IssuerName: ns.IssuerName, Name: ns.Name,
		Environment: ns.Environment, CreatedAt: time.Now()}
	for _, svc := range ns.Services {
		s := store.Service{Type: svc.Type, Name: svc.Name, RepositoryName: svc.RepositoryName}
		for _, su := range svc.Suites {
			suite := store.Suite{Name: su.Name}
			for _, t := range su.Tests {
				suite.Tests = append(suite.Tests, store.Test{Name: t.Name, Path: t.Path})
			}
			s.Suites = append(s.Suites, suite)
		}
		ts.Services = append(ts.Services, s)
	}
	f.sessions[id] = ts
	return id, nil
}

func (f *fakeTestSessions) TestSession(_ context.Context, id int64) (store.TestSession, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	ts, ok := f.sessions[id]
	if !ok {
		return store.TestSession{}, store.ErrNotFound
	}
	return ts, nil
}

// sampleTests is one repository with an automated web service (with a
// workflow) and a mobile service without a workflow.
func sampleTests() *discovery.Result {
	key := func(p string) string { return discovery.TestKey(7, p) }
	return &discovery.Result{
		ID:        "snap1",
		FetchedAt: time.Date(2026, 11, 18, 9, 30, 0, 0, time.UTC),
		Repositories: []discovery.Repository{{
			ID: 7, Name: "laposa/shop-tests", CommitSHA: "abc123",
			Services: []discovery.Service{
				{Name: "web", Path: "tests/web", WorkflowID: 55, Suites: []discovery.Suite{{
					Name: "checkout", Path: "tests/web/tests/checkout",
					Tests: []discovery.Test{
						{Key: key("tests/web/tests/checkout/pay.cy.js"), Name: "pay.cy.js", Path: "tests/web/tests/checkout/pay.cy.js"},
						{Key: key("tests/web/tests/checkout/pay.manual.yaml"), Name: "pay.manual.yaml", Path: "tests/web/tests/checkout/pay.manual.yaml"},
					},
				}}},
				{Name: "mobile", Path: "tests/mobile", Suites: []discovery.Suite{{
					Name: "login", Path: "tests/mobile/tests/login",
					Tests: []discovery.Test{
						{Key: key("tests/mobile/tests/login/a.yaml"), Name: "a.yaml", Path: "tests/mobile/tests/login/a.yaml"},
						{Key: key("tests/mobile/tests/login/b.manual.yaml"), Name: "b.manual.yaml", Path: "tests/mobile/tests/login/b.manual.yaml"},
					},
				}}},
			},
		}},
		Problems: []discovery.Problem{{Repository: "laposa/broken", Message: "could not read tests/"}},
	}
}

func postForm(h http.Handler, target string, form url.Values, cookies ...*http.Cookie) *httptest.ResponseRecorder {
	r := sameOrigin(httptest.NewRequest(http.MethodPost, target, strings.NewReader(form.Encode())))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	for _, c := range cookies {
		r.AddCookie(c)
	}
	return do(h, r)
}

func TestSessionRoutesNeedLogin(t *testing.T) {
	e := newEnv(t, Options{})
	for _, target := range []string{"/sessions/new", "/sessions/1"} {
		if rec := get(e.h, target); rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/login" {
			t.Errorf("GET %s = %d %q, want redirect to /login", target, rec.Code, rec.Header().Get("Location"))
		}
	}
	if rec := postForm(e.h, "/sessions", url.Values{"name": {"x"}}); rec.Code != http.StatusSeeOther {
		t.Errorf("POST /sessions = %d, want redirect to /login", rec.Code)
	}
	if len(e.testSessions.created) != 0 {
		t.Error("session created without login")
	}
}

func TestNewSessionPage(t *testing.T) {
	e := newEnv(t, Options{})
	rec := get(e.h, "/sessions/new", e.signIn("Ada"))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{
		`<form method="post" action="/sessions" class="new-session">`,
		`<input type="hidden" name="discovery" value="snap1">`,
		`<option value="preprod" selected>Preprod</option>`,
		`<option value="prod">Prod</option>`,
		`<details class="list list-repository">`,
		`laposa/shop-tests`,
		`name="tests" value="7:tests/web/tests/checkout/pay.cy.js"`,
		// Manual tests need no workflow.
		`name="tests" value="7:tests/mobile/tests/login/b.manual.yaml"`,
		`This test has no run-tests-mobile.yaml workflow and cannot be selected.`,
		`laposa/broken: could not read tests/.`,
		`href="/sessions/new?refresh"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("page missing %q", want)
		}
	}
	if strings.Contains(body, `value="7:tests/mobile/tests/login/a.yaml"`) {
		t.Error("automated test without workflow is selectable")
	}
}

func TestNewSessionDiscoveryError(t *testing.T) {
	e := newEnv(t, Options{})
	e.discovery.res, e.discovery.err = nil, errors.New("github down")
	rec := get(e.h, "/sessions/new", e.signIn("Ada"))
	if rec.Code != http.StatusBadGateway {
		t.Errorf("status = %d, want 502", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, msgDiscoveryError) || strings.Contains(body, "github down") {
		t.Error("discovery error not shown with the fixed message")
	}
	if !strings.Contains(body, `<button type="submit" class="button filled" disabled>`) {
		t.Error("create button should be disabled without tests")
	}
}

func TestNewSessionRefresh(t *testing.T) {
	e := newEnv(t, Options{})
	rec := get(e.h, "/sessions/new?refresh", e.signIn("Ada"))
	if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/sessions/new" {
		t.Errorf("refresh = %d %q, want redirect to /sessions/new", rec.Code, rec.Header().Get("Location"))
	}
	if e.discovery.refreshes != 1 {
		t.Errorf("refreshes = %d, want 1", e.discovery.refreshes)
	}
}

func TestCreateSession(t *testing.T) {
	e := newEnv(t, Options{})
	cookie := e.signIn("Ada")
	rec := postForm(e.h, "/sessions", url.Values{
		"name":        {"  Release 1.2  "},
		"environment": {"beta"},
		"discovery":   {"snap1"},
		"tests": {
			"7:tests/mobile/tests/login/b.manual.yaml",
			"7:tests/web/tests/checkout/pay.manual.yaml",
			"7:tests/web/tests/checkout/pay.cy.js",
		},
	}, cookie)
	if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/sessions/1" {
		t.Fatalf("POST /sessions = %d %q, want redirect to /sessions/1\n%s", rec.Code, rec.Header().Get("Location"), rec.Body)
	}
	if len(e.testSessions.created) != 1 {
		t.Fatalf("created %d sessions", len(e.testSessions.created))
	}
	ns := e.testSessions.created[0]
	if ns.Name != "Release 1.2" || ns.Environment != "beta" || ns.IssuerName != "Ada" || ns.IssuerID == 0 {
		t.Errorf("session = %+v", ns)
	}
	// Automated first, then manual per repository+service, in tree order.
	type svc struct{ typ, name string }
	var got []svc
	for _, s := range ns.Services {
		got = append(got, svc{s.Type, s.Name})
	}
	want := []svc{{"automated", "web"}, {"manual", "web"}, {"manual", "mobile"}}
	if len(got) != len(want) {
		t.Fatalf("services = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("service %d = %v, want %v", i, got[i], want[i])
		}
	}
	web := ns.Services[0]
	if web.RepositoryID != 7 || web.RepositoryName != "laposa/shop-tests" || web.CommitSHA != "abc123" ||
		web.WorkflowID != 55 || web.Branch != "master" || web.Path != "tests/web" {
		t.Errorf("automated web service = %+v", web)
	}
	if len(web.Suites) != 1 || len(web.Suites[0].Tests) != 1 || web.Suites[0].Tests[0].Name != "pay.cy.js" {
		t.Errorf("automated web suites = %+v", web.Suites)
	}
	if ns.Services[2].WorkflowID != 0 {
		t.Error("mobile service has no workflow")
	}
}

func TestCreateSessionValidation(t *testing.T) {
	valid := func() url.Values {
		return url.Values{
			"name": {"Release"}, "environment": {"prod"}, "discovery": {"snap1"},
			"tests": {"7:tests/web/tests/checkout/pay.cy.js"},
		}
	}
	tests := []struct {
		name   string
		change func(url.Values)
		want   string
	}{
		{"missing name", func(v url.Values) { v.Set("name", "   ") }, msgNameRequired},
		{"long name", func(v url.Values) { v.Set("name", strings.Repeat("é", 256)) }, msgNameTooLong},
		{"unknown environment", func(v url.Values) { v.Set("environment", "staging") }, msgEnvironment},
		{"no tests", func(v url.Values) { v.Del("tests") }, msgNoTests},
		{"expired snapshot", func(v url.Values) { v.Set("discovery", "old") }, msgSnapshotGone},
		{"unknown test", func(v url.Values) { v.Add("tests", "7:tests/../../etc/passwd") }, msgTestsUnusable},
		{"test without workflow", func(v url.Values) { v.Set("tests", "7:tests/mobile/tests/login/a.yaml") }, msgTestsUnusable},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := newEnv(t, Options{})
			form := valid()
			tt.change(form)
			rec := postForm(e.h, "/sessions", form, e.signIn("Ada"))
			if rec.Code != http.StatusUnprocessableEntity {
				t.Errorf("status = %d, want 422", rec.Code)
			}
			body := rec.Body.String()
			if !strings.Contains(body, `class="error-list"`) || !strings.Contains(body, tt.want) {
				t.Errorf("page does not show %q", tt.want)
			}
			if len(e.testSessions.created) != 0 {
				t.Error("session created despite errors")
			}
		})
	}
}

func TestCreateSessionKeepsInput(t *testing.T) {
	e := newEnv(t, Options{})
	rec := postForm(e.h, "/sessions", url.Values{
		"name": {"<b>Release</b>"}, "environment": {"prod"}, "discovery": {"snap1"},
		"tests": {"7:tests/web/tests/checkout/pay.cy.js", "7:tests/../x"},
	}, e.signIn("Ada"))
	body := rec.Body.String()
	for _, want := range []string{
		`value="&lt;b&gt;Release&lt;/b&gt;"`,
		`<option value="prod" selected>Prod</option>`,
		`value="7:tests/web/tests/checkout/pay.cy.js" checked`,
		// Groups holding a selected test start open.
		`<details class="list list-repository" open>`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("re-rendered form missing %q", want)
		}
	}
	if strings.Contains(body, "<b>Release</b>") {
		t.Error("name not escaped")
	}
}

func TestCreateSessionStoreError(t *testing.T) {
	e := newEnv(t, Options{})
	e.testSessions.err = errors.New("db down")
	rec := postForm(e.h, "/sessions", url.Values{
		"name": {"Release"}, "environment": {"prod"}, "discovery": {"snap1"},
		"tests": {"7:tests/web/tests/checkout/pay.cy.js"},
	}, e.signIn("Ada"))
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", rec.Code)
	}
}

func TestShowSession(t *testing.T) {
	e := newEnv(t, Options{})
	cookie := e.signIn("Ada")
	e.testSessions.sessions[3] = store.TestSession{
		ID: 3, IssuerName: "Grace <Hopper>", Name: "Release <1>", Environment: "preprod",
		CreatedAt: time.Date(2026, 11, 18, 9, 30, 0, 0, time.UTC),
		Services: []store.Service{{
			Type: "automated", Name: "web", RepositoryName: "laposa/shop-tests",
			Suites: []store.Suite{{Name: "checkout", Tests: []store.Test{{Name: "pay.cy.js"}}}},
		}},
	}
	rec := get(e.h, "/sessions/3", cookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{
		"<title>Release &lt;1&gt; · TestFlow</title>",
		"<h2>Release &lt;1&gt;</h2>",
		"Environment: <span>Preprod</span>",
		"Created by Grace &lt;Hopper&gt; on",
		"<summary>Selected tests (1)</summary>",
		"<b>shop-tests/web (automated)</b>",
		"<b>checkout</b>",
		"<li>pay.cy.js</li>",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("session page missing %q", want)
		}
	}

	for _, target := range []string{"/sessions/4", "/sessions/abc", "/sessions/0", "/sessions/-1"} {
		if rec := get(e.h, target, cookie); rec.Code != http.StatusNotFound {
			t.Errorf("GET %s = %d, want 404", target, rec.Code)
		}
	}
}
