package web

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/laposa/testflow/app-go/internal/auth"
	"github.com/laposa/testflow/app-go/internal/discovery"
	"github.com/laposa/testflow/app-go/internal/github"
	"github.com/laposa/testflow/app-go/internal/runs"
	"github.com/laposa/testflow/app-go/internal/store"
	"github.com/laposa/testflow/app-go/internal/web/views"
)

// RunStore reads runs (store.Store).
type RunStore interface {
	SessionRuns(ctx context.Context, sessionID int64) ([]store.Run, error)
	RunDetail(ctx context.Context, sessionID, runID int64) (store.RunDetail, error)
}

// RunDispatcher starts automated runs (runs.Dispatcher).
type RunDispatcher interface {
	Dispatch(ctx context.Context, ts store.TestSession, user store.User, serviceIDs []int64) (runs.Result, error)
}

// RunSyncer syncs a session's runs with GitHub now (runs.Syncer).
type RunSyncer interface {
	SyncSession(ctx context.Context, sessionID int64) error
}

// TestFiles reads a file of a tested repository at a commit (github.Client).
type TestFiles interface {
	File(ctx context.Context, repo, path, ref string) ([]byte, error)
}

// syncTimeout bounds "refresh now", below the server's write timeout. The
// background syncer finishes whatever it cuts short.
const syncTimeout = 45 * time.Second

// Run list messages (spec §8.4, §8.5).
const (
	msgManualNotYet = "Manual runs are not available yet."
	msgSyncSlow     = "GitHub is slow to answer. The runs keep updating in the background."
	msgSyncFailed   = "Could not refresh the runs from GitHub. They keep updating in the background."
	msgRunsFailed   = "Something went wrong. Please reload the page."
	msgFileFailed   = "Could not load the file from GitHub."
)

// runState is what the run list keeps across swaps and reloads: the
// selected services, and the services whose older runs are shown. The
// form sends it with every request; a toggle button adds toggle={service}.
type runState struct {
	selected, expanded map[int64]bool
}

func parseRunState(r *http.Request) runState {
	st := runState{selected: idSet(r.Form["services"]), expanded: idSet(r.Form["expand"])}
	for id := range idSet(r.Form["toggle"]) {
		st.expanded[id] = !st.expanded[id]
	}
	return st
}

func idSet(values []string) map[int64]bool {
	set := make(map[int64]bool, len(values))
	for _, v := range values {
		if id, err := strconv.ParseInt(v, 10, 64); err == nil && id > 0 {
			set[id] = true
		}
	}
	return set
}

// runList loads the run list of ts.
func (s *server) runList(ctx context.Context, ts store.TestSession, st runState) (views.RunListView, error) {
	list, err := s.Runs.SessionRuns(ctx, ts.ID)
	if err != nil {
		return views.RunListView{}, err
	}
	return views.RunListView{
		SessionID: ts.ID, Services: ts.Services, Runs: list,
		Selected: st.selected, Expanded: st.expanded, Now: time.Now(),
	}, nil
}

// showRuns answers GET /sessions/{id}/runs: the run list for htmx (the
// poll and the older-runs toggle, spec §8.5), the session page otherwise.
// It reads the database only and never calls GitHub.
func (s *server) showRuns(w http.ResponseWriter, r *http.Request) { s.showRunList(w, r, isHTMX(r)) }

// showSession renders the session detail page (legacy:
// SessionController::show). The query can carry the run list's state, as
// sent by its buttons without JS.
func (s *server) showSession(w http.ResponseWriter, r *http.Request) { s.showRunList(w, r, false) }

func (s *server) showRunList(w http.ResponseWriter, r *http.Request, fragment bool) {
	id, ok := sessionID(w, r)
	if !ok {
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	ts, err := s.TestSessions.TestSession(r.Context(), id)
	if err != nil {
		s.runsError(w, r, "load test session", err)
		return
	}
	s.renderRuns(w, r, http.StatusOK, ts, parseRunState(r), nil, fragment)
}

// startRuns dispatches the selected automated services, or all of them
// (spec §8.4; legacy: ListRuns::createRun). Services GitHub refuses are
// listed above the run list; the others are still started. A normal
// request without problems is redirected back to the session (PRG).
func (s *server) startRuns(w http.ResponseWriter, r *http.Request) {
	id, ok := sessionID(w, r)
	if !ok {
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	ctx := r.Context()
	ts, err := s.TestSessions.TestSession(ctx, id)
	if err != nil {
		s.runsError(w, r, "load test session", err)
		return
	}
	st := parseRunState(r)
	var ids []int64
	for _, svc := range ts.Services {
		if st.selected[svc.ID] {
			ids = append(ids, svc.ID)
		}
	}
	u, _ := auth.UserFrom(ctx)
	res, err := s.Dispatcher.Dispatch(ctx, ts, u, ids)
	if err != nil {
		s.runsError(w, r, "start runs", err)
		return
	}
	var msgs []string
	for _, f := range res.Failed {
		msgs = append(msgs, "Could not start "+f.Service.DisplayName()+": "+dispatchReason(f)+".")
	}
	if res.Manual > 0 && len(res.Started) == 0 && len(res.Failed) == 0 {
		msgs = append(msgs, msgManualNotYet)
	}
	status := http.StatusOK
	if len(res.Failed) > 0 && len(res.Started) == 0 {
		status = http.StatusBadGateway
	}
	if !isHTMX(r) && len(msgs) == 0 {
		http.Redirect(w, r, "/sessions/"+strconv.FormatInt(id, 10), http.StatusSeeOther)
		return
	}
	st.selected = nil // legacy clears the selection once runs start
	s.renderRuns(w, r, status, ts, st, msgs, isHTMX(r))
}

// dispatchReason says why a service could not be started.
func dispatchReason(f runs.Failure) string {
	if errors.Is(f.Err, runs.ErrNoWorkflow) {
		return "it has no " + discovery.WorkflowFile(f.Service.Name) + " workflow"
	}
	if apiErr, ok := errors.AsType[*github.APIError](f.Err); ok && apiErr.Message != "" {
		return "GitHub answered “" + apiErr.Message + "”"
	}
	return "GitHub could not be reached"
}

// syncRuns syncs the session's runs with GitHub now ("refresh now", spec
// §8.5). The sync is not cancelled when the request goes away, but is
// bounded by syncTimeout.
func (s *server) syncRuns(w http.ResponseWriter, r *http.Request) {
	id, ok := sessionID(w, r)
	if !ok {
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	ctx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), syncTimeout)
	defer cancel()
	ts, err := s.TestSessions.TestSession(ctx, id)
	if err != nil {
		s.runsError(w, r, "load test session", err)
		return
	}
	var msgs []string
	if err := s.Syncer.SyncSession(ctx, id); err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			msgs = []string{msgSyncSlow}
		} else {
			s.Logger.WarnContext(ctx, "refresh runs failed", "session.id", id, "err", err)
			msgs = []string{msgSyncFailed}
		}
	}
	if !isHTMX(r) && len(msgs) == 0 {
		http.Redirect(w, r, "/sessions/"+strconv.FormatInt(id, 10), http.StatusSeeOther)
		return
	}
	s.renderRuns(w, r, http.StatusOK, ts, parseRunState(r), msgs, isHTMX(r))
}

// renderRuns renders the run list as a fragment, or the session page.
func (s *server) renderRuns(w http.ResponseWriter, r *http.Request, status int, ts store.TestSession, st runState, msgs []string, fragment bool) {
	ctx := r.Context()
	rl, err := s.runList(ctx, ts, st)
	if err != nil {
		s.runsError(w, r, "load runs", err)
		return
	}
	rl.Messages = msgs
	w.Header().Add("Vary", "HX-Request")
	if fragment {
		s.render(w, r, status, views.RunList(rl))
		return
	}
	av := views.ActivityView{SessionID: ts.ID, Now: rl.Now}
	if av.Entries, err = s.Activity.SessionActivity(ctx, ts.ID); err != nil {
		s.runsError(w, r, "load session activity", err)
		return
	}
	s.render(w, r, status, views.SessionPage(s.page(r, ts.Name), ts, rl, av))
}

// runsError is activityError for the run list: htmx requests get the
// message in the run list's error area.
func (s *server) runsError(w http.ResponseWriter, r *http.Request, what string, err error) {
	s.fragmentError(w, r, what, err, views.RunsErrorsID, msgRunsFailed)
}

// showRun renders a run with its results and logs (legacy:
// SessionRunController::show).
func (s *server) showRun(w http.ResponseWriter, r *http.Request) {
	ts, d, svc, ok := s.loadRun(w, r)
	if !ok {
		return
	}
	s.render(w, r, http.StatusOK, views.RunPage(s.page(r, "Run #"+strconv.FormatInt(d.ID, 10)), ts, svc, d, time.Now()))
}

// loadRun loads the session and run named by the path, writing a 404 or
// 500 if it cannot.
func (s *server) loadRun(w http.ResponseWriter, r *http.Request) (store.TestSession, store.RunDetail, store.Service, bool) {
	id, ok := sessionID(w, r)
	if !ok {
		return store.TestSession{}, store.RunDetail{}, store.Service{}, false
	}
	runID, ok := pathID(w, r, "run")
	if !ok {
		return store.TestSession{}, store.RunDetail{}, store.Service{}, false
	}
	ctx := r.Context()
	ts, err := s.TestSessions.TestSession(ctx, id)
	if err == nil {
		var d store.RunDetail
		if d, err = s.Runs.RunDetail(ctx, id, runID); err == nil {
			for _, svc := range ts.Services {
				if svc.ID == d.ServiceID {
					return ts, d, svc, true
				}
			}
			err = store.ErrNotFound
		}
	}
	s.pageError(w, r, "load run", err)
	return store.TestSession{}, store.RunDetail{}, store.Service{}, false
}

// testFile serves a test file of the run's service at the run's commit as
// plain text (legacy: SessionRunController::showTestFile). Before GitHub
// reported a commit, the session's discovery commit is used.
func (s *server) testFile(w http.ResponseWriter, r *http.Request) {
	ts, d, svc, ok := s.loadRun(w, r)
	if !ok {
		return
	}
	testID, ok := pathID(w, r, "test")
	if !ok {
		return
	}
	var test *store.Test
	for _, suite := range svc.Suites {
		for i := range suite.Tests {
			if suite.Tests[i].ID == testID {
				test = &suite.Tests[i]
			}
		}
	}
	if test == nil {
		http.NotFound(w, r)
		return
	}
	ref := d.CommitSHA
	if ref == "" {
		ref = svc.CommitSHA
	}
	b, err := s.TestFiles.File(r.Context(), svc.RepositoryName, test.Path, ref)
	switch {
	case errors.Is(err, github.ErrNotFound):
		http.NotFound(w, r)
		return
	case err != nil:
		s.Logger.WarnContext(r.Context(), "load test file failed", "session.id", ts.ID, "test.id", testID, "err", err)
		http.Error(w, msgFileFailed, http.StatusBadGateway)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	// The file at a commit never changes.
	w.Header().Set("Cache-Control", "private, max-age=3600")
	_, _ = w.Write(b)
}

// pageError answers a failed full-page request: ErrNotFound is a 404,
// anything else a logged 500.
func (s *server) pageError(w http.ResponseWriter, r *http.Request, what string, err error) {
	if errors.Is(err, store.ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	s.serverError(w, r, what, err)
}

// pathID parses a positive id path value. An invalid id is a 404.
func pathID(w http.ResponseWriter, r *http.Request, name string) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue(name), 10, 64)
	if err != nil || id <= 0 {
		http.NotFound(w, r)
		return 0, false
	}
	return id, true
}
