package web

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"unicode/utf8"

	"github.com/laposa/testflow/app-go/internal/auth"
	"github.com/laposa/testflow/app-go/internal/github"
	"github.com/laposa/testflow/app-go/internal/store"
	"github.com/laposa/testflow/app-go/internal/testcase"
	"github.com/laposa/testflow/app-go/internal/web/views"
)

// ManualRuns saves results of manual runs and finishes them (store.Store).
type ManualRuns interface {
	SaveManualResult(ctx context.Context, runID int64, r store.TestResult) error
	FinishManualRun(ctx context.Context, runID, userID int64) (store.RunTotals, error)
}

// maxComment caps a tester's comment, in characters.
const maxComment = 10000

// Manual run messages (spec §8.7).
const (
	msgChooseResult   = "Choose pass or fail, or skip the test."
	msgCommentTooLong = "The comment is too long."
	msgRunFinished    = "This run is already finished."
	msgCaseFailed     = "Could not load the test case from GitHub. You can still record the result."
	msgCaseMissing    = "The test case is not in the repository at this commit. You can still record the result."
)

// ManualFinishedEvent is the HX-Trigger event of a manual run finished from
// the wizard; app.js closes the dialog on it.
const ManualFinishedEvent = "manual-run-finished"

// showManual answers GET /sessions/{id}/runs/{run}/manual: the wizard at
// ?test={n} (1-based), or at the first test without a result (spec §8.7;
// legacy: ManualTestRun::loadRun). htmx gets the wizard to put into the
// dialog, a normal request the wizard page.
func (s *server) showManual(w http.ResponseWriter, r *http.Request) {
	ts, d, svc, ok := s.loadManualRun(w, r)
	if !ok {
		return
	}
	v := views.ManualRunView{SessionID: ts.ID, Run: d, Service: svc, Tests: manualTests(svc)}
	v.Index = firstOpen(v.Tests, d)
	if n, err := strconv.Atoi(r.URL.Query().Get("test")); err == nil {
		v.Index = min(max(n-1, 0), len(v.Tests)-1)
	}
	s.renderManual(w, r, http.StatusOK, ts, v)
}

// saveManual stores the result of one test (legacy:
// ManualTestRun::updateResults) and shows the next. Saving the last test
// finishes the run, which skips the tests without a result.
func (s *server) saveManual(w http.ResponseWriter, r *http.Request) {
	ts, d, svc, ok := s.loadManualRun(w, r)
	if !ok {
		return
	}
	testID, ok := pathID(w, r, "test")
	if !ok {
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	v := views.ManualRunView{SessionID: ts.ID, Run: d, Service: svc, Tests: manualTests(svc), Index: -1}
	for i, t := range v.Tests {
		if t.ID == testID {
			v.Index = i
		}
	}
	if v.Index < 0 {
		http.NotFound(w, r)
		return
	}
	res := store.TestResult{TestID: testID, Status: r.PostForm.Get("result"), Comment: r.PostForm.Get("comment")}
	if r.PostForm.Has("skip") {
		res.Status = store.ResultSkipped
	}
	switch {
	case res.Status != store.ResultPass && res.Status != store.ResultFail && res.Status != store.ResultSkipped:
		v.Errors = []string{msgChooseResult}
	case utf8.RuneCountInString(res.Comment) > maxComment:
		v.Errors = []string{msgCommentTooLong}
	}
	if len(v.Errors) > 0 {
		v.Draft = &res
		s.renderManual(w, r, http.StatusUnprocessableEntity, ts, v)
		return
	}

	ctx := r.Context()
	err := s.Manual.SaveManualResult(ctx, d.ID, res)
	last := v.Index == len(v.Tests)-1
	if err == nil && last {
		u, _ := auth.UserFrom(ctx)
		_, err = s.Manual.FinishManualRun(ctx, d.ID, u.ID)
	}
	switch {
	case errors.Is(err, store.ErrConflict):
		v.Run.Status = store.RunSuccess // finished meanwhile
		v.Errors = []string{msgRunFinished}
		s.renderManual(w, r, http.StatusConflict, ts, v)
		return
	case err != nil:
		s.pageError(w, r, "save manual result", err)
		return
	}
	base := "/sessions/" + strconv.FormatInt(ts.ID, 10)
	if last {
		if !isHTMX(r) {
			http.Redirect(w, r, base, http.StatusSeeOther)
			return
		}
		w.Header().Set("HX-Trigger", ManualFinishedEvent)
		v.Run.Status = store.RunSuccess
		s.renderManual(w, r, http.StatusOK, ts, v)
		return
	}
	if !isHTMX(r) {
		http.Redirect(w, r, base+"/runs/"+strconv.FormatInt(d.ID, 10)+"/manual?test="+strconv.Itoa(v.Index+2), http.StatusSeeOther)
		return
	}
	if v.Run, err = s.Runs.RunDetail(ctx, ts.ID, d.ID); err != nil {
		s.pageError(w, r, "load run", err)
		return
	}
	v.Index++
	s.renderManual(w, r, http.StatusOK, ts, v)
}

// finishManual answers POST /sessions/{id}/runs/{run}/finish from the run
// list (legacy: ManualTestRun::markRunAsFinished): tests without a result
// are skipped. The run list comes back, or the session page without htmx.
func (s *server) finishManual(w http.ResponseWriter, r *http.Request) {
	ts, d, _, ok := s.loadManualRun(w, r)
	if !ok {
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	ctx := r.Context()
	u, _ := auth.UserFrom(ctx)
	var msgs []string
	_, err := s.Manual.FinishManualRun(ctx, d.ID, u.ID)
	switch {
	case errors.Is(err, store.ErrConflict):
		msgs = []string{"Run " + strconv.FormatInt(d.ID, 10) + " is already finished."}
	case err != nil:
		s.runsError(w, r, "finish manual run", err)
		return
	}
	if !isHTMX(r) && len(msgs) == 0 {
		http.Redirect(w, r, "/sessions/"+strconv.FormatInt(ts.ID, 10), http.StatusSeeOther)
		return
	}
	s.renderRuns(w, r, http.StatusOK, ts, parseRunState(r), msgs, isHTMX(r))
}

// loadManualRun is loadRun for a manual run; another run is a 404.
func (s *server) loadManualRun(w http.ResponseWriter, r *http.Request) (store.TestSession, store.RunDetail, store.Service, bool) {
	ts, d, svc, ok := s.loadRun(w, r)
	if ok && (d.Type != store.RunManual || len(manualTests(svc)) == 0) {
		http.NotFound(w, r)
		return store.TestSession{}, store.RunDetail{}, store.Service{}, false
	}
	return ts, d, svc, ok
}

// renderManual loads the current test case and renders the wizard, as a
// fragment for htmx or as a page.
func (s *server) renderManual(w http.ResponseWriter, r *http.Request, status int, ts store.TestSession, v views.ManualRunView) {
	if v.Run.Status == store.RunInProgress {
		s.loadTestCase(r.Context(), &v)
	}
	if isHTMX(r) {
		s.render(w, r, status, views.ManualRun(v))
		return
	}
	s.render(w, r, status, views.ManualRunPage(s.page(r, "Run #"+strconv.FormatInt(v.Run.ID, 10)), ts, v))
}

// loadTestCase reads the current test's file at the run's commit and
// parses it. Problems are shown in the wizard, which still records the
// result (spec §8.8).
func (s *server) loadTestCase(ctx context.Context, v *views.ManualRunView) {
	t := v.Tests[v.Index]
	ref := v.Run.CommitSHA
	if ref == "" {
		ref = v.Service.CommitSHA
	}
	b, err := s.TestFiles.File(ctx, v.Service.RepositoryName, t.Path, ref)
	switch {
	case errors.Is(err, github.ErrNotFound):
		v.LoadError = msgCaseMissing
		return
	case err != nil:
		s.Logger.WarnContext(ctx, "load test case failed", "run.id", v.Run.ID, "test.id", t.ID, "err", err)
		v.LoadError = msgCaseFailed
		return
	}
	tc, err := testcase.Parse(b)
	if err != nil {
		v.Raw, v.ParseError = string(b), err.Error()
		return
	}
	v.Case, v.Warnings = &tc, tc.Warnings(t.Name)
}

// manualTests are the tests of a manual service in wizard order.
func manualTests(svc store.Service) []store.Test {
	var tests []store.Test
	for _, suite := range svc.Suites {
		tests = append(tests, suite.Tests...)
	}
	return tests
}

// firstOpen is the index of the first test without a result, or 0.
func firstOpen(tests []store.Test, d store.RunDetail) int {
	for i, t := range tests {
		if _, ok := d.Results[t.ID]; !ok {
			return i
		}
	}
	return 0
}

// fileCache keeps test files read from GitHub. A file at a commit never
// changes, so entries do not expire; the cache starts over when full
// (legacy cached each test case for an hour).
type fileCache struct {
	TestFiles
	mu    sync.Mutex
	files map[string][]byte
}

const fileCacheSize = 256

func newFileCache(f TestFiles) *fileCache {
	return &fileCache{TestFiles: f, files: map[string][]byte{}}
}

func (c *fileCache) File(ctx context.Context, repo, path, ref string) ([]byte, error) {
	key := url.PathEscape(repo) + "/" + url.PathEscape(ref) + "/" + path
	c.mu.Lock()
	b, ok := c.files[key]
	c.mu.Unlock()
	if ok {
		return b, nil
	}
	b, err := c.TestFiles.File(ctx, repo, path, ref)
	if err != nil {
		return nil, err
	}
	c.mu.Lock()
	if len(c.files) >= fileCacheSize {
		clear(c.files)
	}
	c.files[key] = b
	c.mu.Unlock()
	return b, nil
}
