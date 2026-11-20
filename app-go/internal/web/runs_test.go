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

	"github.com/laposa/testflow/app-go/internal/github"
	"github.com/laposa/testflow/app-go/internal/runs"
	"github.com/laposa/testflow/app-go/internal/store"
)

// fakeRuns keeps runs by session.
type fakeRuns struct {
	mu      sync.Mutex
	runs    map[int64][]store.Run     // by session, by service and newest first
	details map[int64]store.RunDetail // by run id
	owner   map[int64]int64           // run id → session id
	err     error
}

func (f *fakeRuns) SessionRuns(_ context.Context, sessionID int64) ([]store.Run, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.runs[sessionID], f.err
}

func (f *fakeRuns) RunDetail(_ context.Context, sessionID, runID int64) (store.RunDetail, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	d, ok := f.details[runID]
	if !ok || f.owner[runID] != sessionID {
		return store.RunDetail{}, store.ErrNotFound
	}
	return d, nil
}

type fakeDispatcher struct {
	mu    sync.Mutex
	calls [][]int64
	res   runs.Result
	err   error
}

func (f *fakeDispatcher) Dispatch(_ context.Context, _ store.TestSession, _ store.User, ids []int64) (runs.Result, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, ids)
	return f.res, f.err
}

type fakeSyncer struct {
	mu       sync.Mutex
	sessions []int64
	err      error
}

func (f *fakeSyncer) SyncSession(_ context.Context, id int64) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.sessions = append(f.sessions, id)
	return f.err
}

// fakeFiles serves files by "path@ref".
type fakeFiles struct {
	files map[string]string
	err   error
}

func (f *fakeFiles) File(_ context.Context, _ string, path, ref string) ([]byte, error) {
	if f.err != nil {
		return nil, f.err
	}
	b, ok := f.files[path+"@"+ref]
	if !ok {
		return nil, github.ErrNotFound
	}
	return []byte(b), nil
}

// runsEnv has session 3 with an automated web service (id 10) with three
// runs, an automated api service (11) without runs and a manual one (12).
func runsEnv(t *testing.T) (*testEnv, *http.Cookie) {
	t.Helper()
	e := newEnv(t, Options{})
	cookie := e.signIn("Ada")
	now := time.Now()
	web := store.Service{ID: 10, SessionID: 3, Type: "automated", Name: "web", RepositoryName: "laposa/shop",
		CommitSHA: "disc", WorkflowID: 55,
		Suites: []store.Suite{{ID: 20, Name: "checkout", Tests: []store.Test{
			{ID: 31, Name: "pay.cy.js", Path: "tests/web/tests/checkout/pay.cy.js"},
			{ID: 32, Name: "refund.cy.js", Path: "tests/web/tests/checkout/refund.cy.js"},
			{ID: 33, Name: "missing.cy.js", Path: "tests/web/tests/checkout/missing.cy.js"},
		}}}}
	api := store.Service{ID: 11, SessionID: 3, Type: "automated", Name: "api", RepositoryName: "laposa/shop"}
	manual := store.Service{ID: 12, SessionID: 3, Type: "manual", Name: "web", RepositoryName: "laposa/shop"}
	e.testSessions.sessions[3] = store.TestSession{ID: 3, IssuerName: "Ada", Name: "Release", Environment: "beta",
		CreatedAt: now.Add(-3 * time.Hour), Services: []store.Service{web, api, manual}}

	started, finished := now.Add(-2*time.Hour), now.Add(-2*time.Hour+125*time.Second)
	latest := store.Run{ID: 7, ServiceID: 10, Type: "automated", Status: "failure", HasResults: true, HasRunLog: true,
		HasTotals: true, Passed: 2, Failed: 1, CommitSHA: "run-sha", CreatedAt: started.Add(-time.Minute), StartedAt: &started, FinishedAt: &finished}
	older := store.Run{ID: 5, ServiceID: 10, Type: "automated", Status: "cancelled", HasRunLog: true,
		CreatedAt: now.Add(-150 * time.Minute), FinishedAt: &finished}
	oldest := store.Run{ID: 4, ServiceID: 10, Type: "automated", Status: "stale", CreatedAt: now.Add(-48 * time.Hour)}
	e.runs.runs[3] = []store.Run{latest, older, oldest}
	e.runs.details[7] = store.RunDetail{Run: latest,
		ResultLog: "<testsuites><testsuite name=\"x\"/></testsuites>",
		RunLog:    "\x1b[32mgreen\x1b[0m line <b>",
		Results: map[int64]store.TestResult{
			31: {TestID: 31, Status: "pass", Duration: 1.25},
			32: {TestID: 32, Status: "fail", Comment: "see <video>", Duration: 3},
		}}
	e.runs.owner[7] = 3
	return e, cookie
}

func getHTMX(e *testEnv, target string, cookie *http.Cookie) *httptest.ResponseRecorder {
	r := httptest.NewRequest(http.MethodGet, target, nil)
	r.Header.Set("HX-Request", "true")
	r.AddCookie(cookie)
	return do(e.h, r)
}

func assertContains(t *testing.T, body string, want ...string) {
	t.Helper()
	for _, w := range want {
		if !strings.Contains(body, w) {
			t.Errorf("missing %q", w)
		}
	}
}

func assertNotContains(t *testing.T, body string, unwanted ...string) {
	t.Helper()
	for _, u := range unwanted {
		if strings.Contains(body, u) {
			t.Errorf("unexpected %q", u)
		}
	}
}

func TestSessionPageRunList(t *testing.T) {
	e, cookie := runsEnv(t)
	rec := get(e.h, "/sessions/3", cookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	body := rec.Body.String()
	assertContains(t, body,
		`<form id="runs-form" method="post" action="/sessions/3/runs" hx-post="/sessions/3/runs" hx-target="#runs" hx-swap="outerHTML" hx-sync="this:drop">`,
		`<td>shop/web (automated)</td>`,
		`<a href="/sessions/3/runs/7" title="Show logs">7</a>`,
		`title="Picked by runner at `, `>2 minutes 5 seconds</td>`,
		`<span class="pass">2 passed</span> and <span class="fail">1 failed</span>`,
		`<td>shop/api (automated)</td><td>No runs</td>`,
		`<label class="checkbox"><input type="checkbox" name="services" value="12" class="run-selector" aria-label="Select shop/web (manual)"><span class="checkmark"></span></label>`,
		`name="toggle" value="10" formmethod="get" formaction="/sessions/3" formnovalidate hx-get="/sessions/3/runs" hx-include="#runs-form"`,
		`aria-expanded="false"`,
		`>Run All Tests</button>`,
		`formaction="/sessions/3/runs/sync" formnovalidate hx-post="/sessions/3/runs/sync" hx-target="#runs" hx-swap="outerHTML" hx-indicator="#runs-loading"`,
		`<div id="runs-loading" class="loading-overlay relative htmx-indicator" role="status" aria-label="Checking GitHub for the runs"><svg`,
	)
	// Nothing syncs, so no polling, only the refresh event; older runs are
	// collapsed.
	assertContains(t, body, `<div id="runs" class="runs" hx-get="/sessions/3/runs" hx-trigger="runs-refresh from:body" hx-include="#runs-form" hx-swap="outerHTML">`)
	assertNotContains(t, body, `every 30s`, `/sessions/3/runs/5`)
}

func TestRunListPollsWhileSyncing(t *testing.T) {
	e, cookie := runsEnv(t)
	e.runs.runs[3] = append([]store.Run{{ID: 8, ServiceID: 11, Type: "automated", Status: "unknown", CreatedAt: time.Now()}},
		e.runs.runs[3]...)
	body := get(e.h, "/sessions/3", cookie).Body.String()
	assertContains(t, body,
		`<div id="runs" class="runs" hx-get="/sessions/3/runs" hx-trigger="every 30s, runs-refresh from:body" hx-include="#runs-form" hx-swap="outerHTML">`,
		`<span class="running">Queued</span>`)

	// Picked up by a runner: running.
	e.runs.runs[3][0].Status = "in_progress"
	body = get(e.h, "/sessions/3", cookie).Body.String()
	assertContains(t, body, `<span class="running">Running</span>`)
	assertNotContains(t, body, `>Queued<`)
}

func TestRunListFragmentKeepsState(t *testing.T) {
	e, cookie := runsEnv(t)
	// The poll: the selection and expanded services come back as they were.
	rec := getHTMX(e, "/sessions/3/runs?services=11&expand=10", cookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	body := rec.Body.String()
	if strings.HasPrefix(body, "<!doctype") || !strings.HasPrefix(body, `<div id="runs"`) {
		t.Errorf("want only the run list fragment, got %.80q", body)
	}
	assertContains(t, body,
		`value="11" class="run-selector" aria-label="Select shop/api (automated)" checked`,
		`<input type="hidden" name="expand" value="10">`,
		`<tr class="more-runs-list">`, `/sessions/3/runs/5`, `/sessions/3/runs/4`,
		`<span class="run-status run-status--cancelled">cancelled</span>`,
		`<span class="run-status run-status--stale">stale</span>`,
		`aria-expanded="true"`,
		`>Run Selected Tests (1)</button>`)
	if got := rec.Header().Values("Vary"); len(got) == 0 {
		t.Error("missing Vary: HX-Request")
	}

	// The toggle collapses it again; without htmx the page carries the state.
	body = get(e.h, "/sessions/3?expand=10&toggle=10", cookie).Body.String()
	assertContains(t, body, `<!doctype html>`)
	assertNotContains(t, body, `name="expand"`, `/sessions/3/runs/5`)

	// A non-htmx GET of the fragment URL gets the page.
	if body := get(e.h, "/sessions/3/runs", cookie).Body.String(); !strings.HasPrefix(body, "<!doctype html>") {
		t.Errorf("non-htmx /runs = %.80q", body)
	}
}

func TestStartRuns(t *testing.T) {
	e, cookie := runsEnv(t)
	rec := postForm(e.h, "/sessions/3/runs", url.Values{"services": {"10", "999", "x"}}, cookie)
	if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/sessions/3" {
		t.Fatalf("status = %d, location = %q", rec.Code, rec.Header().Get("Location"))
	}
	// Unknown ids are dropped; no selection means all services.
	if len(e.dispatcher.calls) != 1 || len(e.dispatcher.calls[0]) != 1 || e.dispatcher.calls[0][0] != 10 {
		t.Errorf("dispatch calls = %v", e.dispatcher.calls)
	}
	postForm(e.h, "/sessions/3/runs", url.Values{}, cookie)
	if e.dispatcher.calls[1] != nil {
		t.Errorf("run all: ids = %v, want nil", e.dispatcher.calls[1])
	}

	// htmx gets the list back, with the selection cleared.
	rec = postHTMX(e, "/sessions/3/runs", url.Values{"services": {"11"}, "expand": {"10"}}, cookie)
	if rec.Code != http.StatusOK || !strings.HasPrefix(rec.Body.String(), `<div id="runs"`) {
		t.Fatalf("htmx: status = %d, body = %.80q", rec.Code, rec.Body.String())
	}
	assertContains(t, rec.Body.String(), `>Run All Tests</button>`, `name="expand" value="10"`)
	assertNotContains(t, rec.Body.String(), "checked")
}

func TestStartRunsFailures(t *testing.T) {
	e, cookie := runsEnv(t)
	ts := e.testSessions.sessions[3]
	e.dispatcher.res = runs.Result{Failed: []runs.Failure{
		{Service: ts.Services[0], Err: &github.APIError{StatusCode: 422, Message: "Workflow does not have 'workflow_dispatch' trigger"}},
		{Service: ts.Services[1], Err: runs.ErrNoWorkflow},
		{Service: ts.Services[1], Err: errors.New("dial tcp: timeout")},
	}}
	rec := postHTMX(e, "/sessions/3/runs", url.Values{}, cookie)
	if rec.Code != http.StatusBadGateway {
		t.Errorf("status = %d, want 502", rec.Code)
	}
	assertContains(t, rec.Body.String(),
		`<div id="runs-errors" class="field-error" role="alert">`,
		"Could not start shop/web (automated): GitHub answered “Workflow does not have &#39;workflow_dispatch&#39; trigger”.",
		"Could not start shop/api (automated): it has no run-tests-api.yaml workflow.",
		"Could not start shop/api (automated): GitHub could not be reached.")

	// Without htmx the page is rendered with the messages, not redirected.
	rec = postForm(e.h, "/sessions/3/runs", url.Values{}, cookie)
	if rec.Code != http.StatusBadGateway || !strings.Contains(rec.Body.String(), "<!doctype html>") {
		t.Errorf("non-htmx: status = %d", rec.Code)
	}

	// A manual run started while GitHub refused the rest: no 502.
	e.dispatcher.res.Manual = []runs.Started{{Run: store.Run{ID: 9}}}
	if rec = postHTMX(e, "/sessions/3/runs", url.Values{}, cookie); rec.Code != http.StatusOK {
		t.Errorf("with a manual run: status = %d", rec.Code)
	}

	// A storage failure is a 500 into the error area.
	e.dispatcher.err = errors.New("db down")
	rec = postHTMX(e, "/sessions/3/runs", url.Values{}, cookie)
	if rec.Code != http.StatusInternalServerError || rec.Header().Get("HX-Retarget") != "#runs-errors" ||
		!strings.Contains(rec.Body.String(), msgRunsFailed) {
		t.Errorf("store error: status = %d, retarget = %q, body = %q", rec.Code, rec.Header().Get("HX-Retarget"), rec.Body.String())
	}
}

func TestRunsMissingSession(t *testing.T) {
	e, cookie := runsEnv(t)
	rec := postHTMX(e, "/sessions/99/runs", url.Values{}, cookie)
	if rec.Code != http.StatusNotFound || rec.Header().Get("HX-Retarget") != "#runs-errors" ||
		!strings.Contains(rec.Body.String(), msgSessionGone) {
		t.Errorf("status = %d, retarget = %q", rec.Code, rec.Header().Get("HX-Retarget"))
	}
	if rec := get(e.h, "/sessions/99/runs/7", cookie); rec.Code != http.StatusNotFound {
		t.Errorf("run page of a missing session: %d", rec.Code)
	}
	if len(e.dispatcher.calls) != 0 {
		t.Error("dispatched for a missing session")
	}
}

func TestSyncRuns(t *testing.T) {
	e, cookie := runsEnv(t)
	rec := postForm(e.h, "/sessions/3/runs/sync", url.Values{}, cookie)
	if rec.Code != http.StatusSeeOther || len(e.syncer.sessions) != 1 || e.syncer.sessions[0] != 3 {
		t.Fatalf("status = %d, synced = %v", rec.Code, e.syncer.sessions)
	}
	rec = postHTMX(e, "/sessions/3/runs/sync", url.Values{"services": {"10"}}, cookie)
	assertContains(t, rec.Body.String(), `value="10" class="run-selector" aria-label="Select shop/web (automated)" checked`)

	e.syncer.err = context.DeadlineExceeded
	rec = postHTMX(e, "/sessions/3/runs/sync", url.Values{}, cookie)
	if rec.Code != http.StatusOK {
		t.Errorf("slow sync: status = %d", rec.Code)
	}
	assertContains(t, rec.Body.String(), msgSyncSlow)
	e.syncer.err = errors.New("boom")
	assertContains(t, postHTMX(e, "/sessions/3/runs/sync", url.Values{}, cookie).Body.String(), msgSyncFailed)
}

func TestRunPage(t *testing.T) {
	e, cookie := runsEnv(t)
	rec := get(e.h, "/sessions/3/runs/7", cookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	assertContains(t, rec.Body.String(),
		"<title>Run #7 · TestFlow</title>",
		"Run #7 for shop/web (automated).",
		"and ran for 2 minutes 5 seconds.",
		`<span class="run-status run-status--failure">failure</span>`,
		`<span class="pass">2 passed</span>, <span class="fail">1 failed</span> and <span class="skip">0 skipped</span>`,
		`<a href="/sessions/3/runs/7/tests/31" target="_blank" rel="noopener" data-test-file title="Show file contents">pay.cy.js</a>`,
		`<dialog id="test-dialog" class="test-dialog" aria-labelledby="test-dialog-title">`,
		`<form method="dialog"><button type="submit" class="close" aria-label="Close">×</button></form>`,
		`<div class="loading"><div class="loading-overlay relative" role="status" aria-label="Loading the test file"><svg`,
		`<span class="test-status pass" title="Execution time 1.2s" aria-label="pass"></span>`,
		`<span class="test-status fail" title="Execution time 3.0s" aria-label="fail"></span>`,
		`<div class="comment"><i>see &lt;video&gt;</i></div>`,
		`missing.cy.js</a> <span class="test-status not-found">not found</span>`,
		`<summary>Results in JUnit format</summary><pre>&lt;testsuites&gt;`,
		// ANSI colours are stripped and the log is escaped.
		`<summary>Detailed run log</summary><pre>green line &lt;b&gt;</pre>`,
	)
	for _, target := range []string{"/sessions/3/runs/8", "/sessions/3/runs/x", "/sessions/4/runs/7"} {
		if rec := get(e.h, target, cookie); rec.Code != http.StatusNotFound {
			t.Errorf("%s: status = %d, want 404", target, rec.Code)
		}
	}
}

// A report that did not parse has no counts: the list and the run page
// show the run's status instead of "None", and no test is "not found".
func TestUnparsableReport(t *testing.T) {
	e, cookie := runsEnv(t)
	latest := e.runs.runs[3][0]
	latest.HasTotals, latest.Passed, latest.Failed = false, 0, 0
	e.runs.runs[3][0] = latest
	d := e.runs.details[7]
	d.Run, d.Results = latest, nil
	e.runs.details[7] = d

	body := get(e.h, "/sessions/3", cookie).Body.String()
	assertContains(t, body, `<td><span class="run-status run-status--failure">failure</span></td>`)
	assertNotContains(t, body, `class="no-runs">None<`)

	body = get(e.h, "/sessions/3/runs/7", cookie).Body.String()
	assertContains(t, body, `<summary>Results in JUnit format</summary>`)
	assertNotContains(t, body, "not found", `class="pass"`)
}

func TestTestFile(t *testing.T) {
	e, cookie := runsEnv(t)
	e.files.files = map[string]string{
		"tests/web/tests/checkout/pay.cy.js@run-sha": "describe('pay', () => {})",
		"tests/web/tests/checkout/pay.cy.js@disc":    "the discovery version",
	}
	rec := get(e.h, "/sessions/3/runs/7/tests/31", cookie)
	if rec.Code != http.StatusOK || rec.Body.String() != "describe('pay', () => {})" ||
		rec.Header().Get("Content-Type") != "text/plain; charset=utf-8" || rec.Header().Get("Cache-Control") != "private, max-age=3600" {
		t.Errorf("file: status = %d, type = %q, cache = %q, body = %q", rec.Code, rec.Header().Get("Content-Type"),
			rec.Header().Get("Cache-Control"), rec.Body.String())
	}

	// Before GitHub reports the run's commit, the discovery commit is used.
	d := e.runs.details[7]
	d.CommitSHA = ""
	e.runs.details[7] = d
	// The same URL serves the run's commit later, so it is not cached.
	if rec := get(e.h, "/sessions/3/runs/7/tests/31", cookie); rec.Body.String() != "the discovery version" ||
		rec.Header().Get("Cache-Control") != "no-cache" {
		t.Errorf("fallback ref: cache = %q, body = %q", rec.Header().Get("Cache-Control"), rec.Body.String())
	}

	for target, want := range map[string]int{
		"/sessions/3/runs/7/tests/32": http.StatusNotFound, // not on GitHub
		"/sessions/3/runs/7/tests/99": http.StatusNotFound, // not a test of the run's service
		"/sessions/3/runs/7/tests/x":  http.StatusNotFound,
	} {
		if rec := get(e.h, target, cookie); rec.Code != want {
			t.Errorf("%s: status = %d, want %d", target, rec.Code, want)
		}
	}
	// Test 33 was not read yet: files read once are cached.
	e.files.err = errors.New("timeout")
	if rec := get(e.h, "/sessions/3/runs/7/tests/33", cookie); rec.Code != http.StatusBadGateway {
		t.Errorf("GitHub error: status = %d, want 502", rec.Code)
	}
}

func TestActivityRunLinks(t *testing.T) {
	e, cookie := runsEnv(t)
	e.activity.entries[3] = []store.Activity{{
		ID: 1, Type: store.ActivityRunStatusChanged, CreatedAt: time.Now(), RunID: 7,
		Body:       "Run 7 for shop/mobile (automated) has changed status from in_progress to success.",
		MaestroURL: "https://console.mobile.dev/uploads/abc?x=1&y=2",
	}}
	assertContains(t, get(e.h, "/sessions/3", cookie).Body.String(),
		`has changed status from in_progress to success. <a href="/sessions/3/runs/7">Show run</a>`,
		`<div>See <a href="https://console.mobile.dev/uploads/abc?x=1&amp;y=2" target="_blank" rel="noopener noreferrer">Maestro Cloud Console</a> recorded videos for more details.</div>`)
}
