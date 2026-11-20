package web

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/laposa/testflow/app-go/internal/store"
)

// fakeManual records saved results in the runs fake's details.
type fakeManual struct {
	mu       sync.Mutex
	runs     *fakeRuns
	saved    []store.TestResult
	finished []int64
	err      error // returned by both methods
}

func (f *fakeManual) SaveManualResult(_ context.Context, runID int64, r store.TestResult) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return f.err
	}
	f.saved = append(f.saved, r)
	f.runs.mu.Lock()
	defer f.runs.mu.Unlock()
	d := f.runs.details[runID]
	d.Results[r.TestID] = r
	f.runs.details[runID] = d
	return nil
}

func (f *fakeManual) FinishManualRun(_ context.Context, runID, _ int64) (store.RunTotals, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return store.RunTotals{}, f.err
	}
	f.finished = append(f.finished, runID)
	return store.RunTotals{Passed: 1}, nil
}

const loginCase = `name: login
suite: auth
description: User can log in.
steps:
  - action: [ "Open the page" ]
    input:
      user: { description: "A user", value: "qa@example.com" }
      locale: en
    result: [ "It opens", "Form shown" ]
`

// manualEnv adds a manual run (20) in progress to runsEnv's manual
// service 12, which gets three tests (41-43). Test 41 has a result, and
// the files of 41 and 42 are on GitHub at the session's commit.
func manualEnv(t *testing.T) (*testEnv, *http.Cookie) {
	t.Helper()
	e, cookie := runsEnv(t)
	ts := e.testSessions.sessions[3]
	ts.Services[2].CommitSHA = "disc"
	ts.Services[2].Suites = []store.Suite{{ID: 30, Name: "auth", Tests: []store.Test{
		{ID: 41, Name: "login.manual.yaml", Path: "m/login.manual.yaml"},
		{ID: 42, Name: "logout.manual.yaml", Path: "m/logout.manual.yaml"},
		{ID: 43, Name: "reset.manual.yaml", Path: "m/reset.manual.yaml"},
	}}}
	e.testSessions.sessions[3] = ts
	started := time.Now().Add(-time.Hour)
	run := store.Run{ID: 20, ServiceID: 12, Type: "manual", Status: "in_progress", HasTotals: true, Passed: 1,
		CommitSHA: "disc", StartedAt: &started, CreatedAt: started}
	e.runs.runs[3] = append(e.runs.runs[3], run)
	e.runs.details[20] = store.RunDetail{Run: run, Results: map[int64]store.TestResult{
		41: {TestID: 41, Status: "pass", Comment: "fine"},
	}}
	e.runs.owner[20] = 3
	e.files.files = map[string]string{
		"m/login.manual.yaml@disc":  loginCase,
		"m/logout.manual.yaml@disc": "steps: [ unclosed",
	}
	return e, cookie
}

func TestManualRunList(t *testing.T) {
	e, cookie := manualEnv(t)
	body := get(e.h, "/sessions/3", cookie).Body.String()
	assertContains(t, body,
		`<span class="running">In progress</span>`,
		`<a href="/sessions/3/runs/20/manual" class="button filled" data-manual-run hx-get="/sessions/3/runs/20/manual" hx-target="#manual-run-content" hx-swap="innerHTML">Complete manual tests</a>`,
		`formaction="/sessions/3/runs/20/finish" formnovalidate hx-post="/sessions/3/runs/20/finish" hx-target="#runs" hx-swap="outerHTML" hx-confirm="Finish run 20? Tests without a result are marked as skipped.">Mark run as finished</button>`,
		`<dialog id="manual-run" class="manual-dialog" aria-label="Manual run">`,
		`<div id="manual-run-content"><div class="loading"><div class="loading-overlay relative"`,
	)
	// Manual runs are not synced: no polling.
	assertNotContains(t, body, "every 30s")

	// The run page links to the wizard page and marks no test "not found".
	body = get(e.h, "/sessions/3/runs/20", cookie).Body.String()
	assertContains(t, body, " and is still in progress.",
		`<a href="/sessions/3/runs/20/manual" class="button filled">Complete manual tests</a>`,
		`<span class="pass">1 passed</span>`, `<div class="comment"><i>fine</i></div>`)
	assertNotContains(t, body, "not found")
}

func TestManualWizard(t *testing.T) {
	e, cookie := manualEnv(t)

	// It opens at the first test without a result, whose file does not
	// parse: the raw file is shown with the error.
	rec := getHTMX(e, "/sessions/3/runs/20/manual", cookie)
	if rec.Code != http.StatusOK || strings.Contains(rec.Body.String(), "<html") {
		t.Fatalf("status = %d, body = %q", rec.Code, rec.Body.String())
	}
	assertContains(t, rec.Body.String(),
		`<input type="number" name="test" class="test-pagination-input" min="1" max="3" value="2"`,
		`<a class="prev" href="/sessions/3/runs/20/manual?test=1" hx-get="/sessions/3/runs/20/manual?test=1"`,
		`<span title="suite">auth</span> / <span title="test name">logout.manual.yaml</span>`,
		`The test case could not be read (test case: yaml:`,
		`<pre class="manual-test-raw">steps: [ unclosed</pre>`,
		`<form method="post" action="/sessions/3/runs/20/manual/42" hx-post="/sessions/3/runs/20/manual/42" hx-target="#manual-run-content" hx-swap="innerHTML" hx-sync="this:drop">`,
		`>Submit</button>`,
	)

	// Test 1 parses; its result is preselected with the comment.
	body := getHTMX(e, "/sessions/3/runs/20/manual?test=1", cookie).Body.String()
	assertContains(t, body,
		`<span title="suite">auth</span> / <span title="test name">login</span>`,
		`<span title="description">User can log in.</span>`,
		`<td>1</td><td>Open the page<div class="input"><div>user: A user: <button type="button" class="copy-to-clipboard" title="Copy to clipboard">qa@example.com</button></div><div>locale: en</div></div></td>`,
		`<ul class="bullet-list"><li>It opens</li><li>Form shown</li></ul>`,
		`<input type="radio" name="step-1" class="radio-pass" aria-label="Mark step 1 as passed">`,
		`>fine</textarea>`,
		`<input type="radio" name="result" value="pass" required class="result-pass" checked>`,
		`<span class="prev disabled" aria-hidden="true"></span>`,
	)
	// "login" matches login.manual.yaml: no warning.
	assertNotContains(t, body, "does not match")

	// The last test is missing on GitHub; out-of-range pages are clamped.
	body = getHTMX(e, "/sessions/3/runs/20/manual?test=99", cookie).Body.String()
	assertContains(t, body, msgCaseMissing, `value="3"`, `>Submit and finish</button>`, `<span class="next disabled"`)

	// Without htmx it is a page.
	body = get(e.h, "/sessions/3/runs/20/manual", cookie).Body.String()
	assertContains(t, body, "<!doctype html>", `<div id="manual-run-content"><div class="manual-wizard">`, "← Back to the session")

	// An automated run has no wizard; another session's run is not found.
	for _, target := range []string{"/sessions/3/runs/7/manual", "/sessions/4/runs/20/manual"} {
		if rec := get(e.h, target, cookie); rec.Code != http.StatusNotFound {
			t.Errorf("%s: status = %d, want 404", target, rec.Code)
		}
	}
}

func TestManualSave(t *testing.T) {
	e, cookie := manualEnv(t)

	// Without a choice the test is shown again with the message and the
	// comment kept.
	rec := postHTMX(e, "/sessions/3/runs/20/manual/42", url.Values{"comment": {"tried"}}, cookie)
	if rec.Code != http.StatusUnprocessableEntity || len(e.manual.saved) != 0 {
		t.Fatalf("no result: status = %d, saved = %v", rec.Code, e.manual.saved)
	}
	assertContains(t, rec.Body.String(), msgChooseResult, `>tried</textarea>`)

	// Skip wins over a chosen result and moves on to the next test.
	rec = postHTMX(e, "/sessions/3/runs/20/manual/42", url.Values{"result": {"pass"}, "skip": {"1"}, "comment": {"later"}}, cookie)
	if rec.Code != http.StatusOK || e.manual.saved[0] != (store.TestResult{TestID: 42, Status: "skipped", Comment: "later"}) {
		t.Fatalf("skip: status = %d, saved = %+v", rec.Code, e.manual.saved)
	}
	assertContains(t, rec.Body.String(), `value="3"`, `action="/sessions/3/runs/20/manual/43"`)

	// Without htmx: redirect to the next test.
	rec = postForm(e.h, "/sessions/3/runs/20/manual/41", url.Values{"result": {"fail"}}, cookie)
	if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/sessions/3/runs/20/manual?test=2" {
		t.Errorf("non-htmx save = %d %q", rec.Code, rec.Header().Get("Location"))
	}

	// The last test finishes the run and tells app.js to close the dialog.
	rec = postHTMX(e, "/sessions/3/runs/20/manual/43", url.Values{"result": {"pass"}}, cookie)
	if rec.Code != http.StatusOK || rec.Header().Get("HX-Trigger") != ManualFinishedEvent || len(e.manual.finished) != 1 {
		t.Errorf("last: status = %d, trigger = %q, finished = %v", rec.Code, rec.Header().Get("HX-Trigger"), e.manual.finished)
	}
	assertContains(t, rec.Body.String(), "Run #20 for shop/web (manual) is finished.", `href="/sessions/3/runs/20"`)
	rec = postForm(e.h, "/sessions/3/runs/20/manual/43", url.Values{"result": {"pass"}}, cookie)
	if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/sessions/3" {
		t.Errorf("non-htmx last = %d %q", rec.Code, rec.Header().Get("Location"))
	}

	// A run finished meanwhile.
	e.manual.err = store.ErrConflict
	rec = postHTMX(e, "/sessions/3/runs/20/manual/42", url.Values{"result": {"pass"}}, cookie)
	if rec.Code != http.StatusConflict {
		t.Errorf("finished run: status = %d", rec.Code)
	}
	assertContains(t, rec.Body.String(), msgRunFinished)

	// Other bad input.
	e.manual.err = nil
	long := url.Values{"result": {"pass"}, "comment": {strings.Repeat("x", maxComment+1)}}
	if rec := postHTMX(e, "/sessions/3/runs/20/manual/42", long, cookie); rec.Code != http.StatusUnprocessableEntity ||
		!strings.Contains(rec.Body.String(), msgCommentTooLong) {
		t.Errorf("long comment: status = %d", rec.Code)
	}
	if rec := postHTMX(e, "/sessions/3/runs/20/manual/31", url.Values{"result": {"pass"}}, cookie); rec.Code != http.StatusNotFound {
		t.Errorf("test of another service: status = %d", rec.Code)
	}
	e.manual.err = errors.New("db down")
	if rec := postHTMX(e, "/sessions/3/runs/20/manual/42", url.Values{"result": {"pass"}}, cookie); rec.Code != http.StatusInternalServerError {
		t.Errorf("store error: status = %d", rec.Code)
	}
}

func TestManualFinish(t *testing.T) {
	e, cookie := manualEnv(t)
	rec := postHTMX(e, "/sessions/3/runs/20/finish", url.Values{"services": {"10"}}, cookie)
	if rec.Code != http.StatusOK || len(e.manual.finished) != 1 || e.manual.finished[0] != 20 {
		t.Fatalf("finish = %d, finished = %v", rec.Code, e.manual.finished)
	}
	// The run list comes back with the selection kept.
	assertContains(t, rec.Body.String(), `<div id="runs"`, `value="10" class="run-selector" aria-label="Select shop/web (automated)" checked`)

	if rec := postForm(e.h, "/sessions/3/runs/20/finish", url.Values{}, cookie); rec.Code != http.StatusSeeOther {
		t.Errorf("non-htmx finish = %d", rec.Code)
	}
	e.manual.err = store.ErrConflict
	assertContains(t, postHTMX(e, "/sessions/3/runs/20/finish", url.Values{}, cookie).Body.String(), "Run 20 is already finished.")
	if rec := postHTMX(e, "/sessions/3/runs/7/finish", url.Values{}, cookie); rec.Code != http.StatusNotFound {
		t.Errorf("automated run: status = %d", rec.Code)
	}
}
