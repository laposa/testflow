package web

import (
	"errors"
	"net/http"
	"strings"
	"testing"
)

func TestExecutiveReport(t *testing.T) {
	e, cookie := runsEnv(t)
	rec := get(e.h, "/sessions/3/report/executive", cookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	body := rec.Body.String()
	assertContains(t, body,
		"<title>Executive report · Release · TestFlow</title>",
		`<button type="button" class="button filled" data-print hidden>Print</button>`,
		`Environment: <span>Beta</span>`,
		`<table class="runs-list report-runs">`,
		// The latest run of each service; services without runs say so.
		`<td>shop/web (automated)</td><td><a href="/sessions/3/runs/7" title="Show logs">7</a></td>`,
		`<span class="pass">2 passed</span> and <span class="fail">1 failed</span>`,
		`<td>shop/api (automated)</td><td>No runs</td>`,
	)
	// No form, older runs, polling or per-run details.
	assertNotContains(t, body, `name="services"`, `/sessions/3/runs/5"`, "hx-trigger", "Run #7 for", "Run All Tests")

	// The session page links to both reports.
	assertContains(t, get(e.h, "/sessions/3", cookie).Body.String(),
		`<a href="/sessions/3/report/executive" class="button">View Executive Report</a>`,
		`<a href="/sessions/3/report/full" class="button">View Full Report</a>`)
}

func TestFullReport(t *testing.T) {
	e, cookie := manualEnv(t)
	rec := get(e.h, "/sessions/3/report/full", cookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	body := rec.Body.String()
	assertContains(t, body,
		"<title>Full report · Release · TestFlow</title>",
		`<table class="runs-list report-runs">`,
		// Run 7 with its results: icon, time written out, comment.
		`<section class="page-break report-run"><h3>Run #7 for shop/web (automated)</h3>`,
		"and ran for 2 minutes 5 seconds.",
		`<span class="pass">2 passed</span>, <span class="fail">1 failed</span> and <span class="skip">0 skipped</span>`,
		`<li class="test-file">pay.cy.js <span class="test-status pass" title="Execution time 1.2s" aria-label="pass"></span><span class="test-time"> 1.2s</span></li>`,
		`<span class="test-time"> 3.0s</span><div class="comment"><i>see &lt;video&gt;</i></div></li>`,
		`<li class="test-file">missing.cy.js <span class="test-status not-found">not found</span></li>`,
		// A run without counts shows its status; a stale one never reported.
		`<h3>Run #4 for shop/web (automated)</h3>`, " and never reported back.",
		`<span class="run-status run-status--stale">stale</span>`,
		// The manual run in progress: answered test with its comment, the
		// others without a mark.
		`<h3>Run #20 for shop/web (manual)</h3>`, " and is still in progress.",
		`<li class="test-file">login.manual.yaml <span class="test-status pass" aria-label="pass"></span><div class="comment"><i>fine</i></div></li>`,
		`<li class="test-file">logout.manual.yaml </li>`,
	)
	// The open manual run shows its counts so far in the summary table.
	assertContains(t, body, `<span class="running">In progress</span></td><td><span class="pass">1 passed</span></td>`)
	// Comments sit inside their test's list item, never directly in a list.
	assertNotContains(t, body, `</li><div class="comment">`)

	// Oldest run first.
	if i4, i7, i20 := strings.Index(body, "Run #4 for"), strings.Index(body, "Run #7 for"), strings.Index(body, "Run #20 for"); !(i4 < i7 && i7 < i20) {
		t.Errorf("run order: #4 at %d, #7 at %d, #20 at %d", i4, i7, i20)
	}

	for _, target := range []string{"/sessions/4/report/full", "/sessions/x/report/executive"} {
		if rec := get(e.h, target, cookie); rec.Code != http.StatusNotFound {
			t.Errorf("%s: status = %d, want 404", target, rec.Code)
		}
	}
	e.runs.err = errors.New("db down")
	if rec := get(e.h, "/sessions/3/report/full", cookie); rec.Code != http.StatusInternalServerError {
		t.Errorf("store error: status = %d", rec.Code)
	}
}
