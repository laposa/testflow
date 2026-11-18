package web

import (
	"context"
	"errors"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/laposa/testflow/app-go/internal/auth"
	"github.com/laposa/testflow/app-go/internal/discovery"
	"github.com/laposa/testflow/app-go/internal/store"
	"github.com/laposa/testflow/app-go/internal/web/views"
)

// TestDiscovery finds the tests that can be selected (discovery.Discoverer).
type TestDiscovery interface {
	Tests(ctx context.Context, refresh bool) (*discovery.Result, error)
	Snapshot(id string) (*discovery.Result, bool)
}

// TestSessionStore creates and reads test sessions (store.Store).
type TestSessionStore interface {
	ListTestSessions(ctx context.Context) ([]store.SessionSummary, error)
	CreateTestSession(ctx context.Context, ns store.NewTestSession) (int64, error)
	TestSession(ctx context.Context, id int64) (store.TestSession, error)
}

// maxSessionName matches the legacy varchar(255) column.
const maxSessionName = 255

// Validation messages (spec §8.2; legacy: Actions/Session/CreateSession.php).
const (
	msgNameRequired   = "The session name is required."
	msgNameTooLong    = "The session name may not be longer than 255 characters."
	msgEnvironment    = "Please choose one of the listed environments."
	msgNoTests        = "Please select at least one test suite."
	msgSnapshotGone   = "The test list has expired. Please check your selection and create the session again."
	msgTestsUnusable  = "Some selected tests are not available any more. Please check your selection."
	msgDiscoveryError = "Could not load the tests from GitHub. Please try again."
	msgDiscoveryStale = "Could not load the tests from GitHub, so this is the last list that loaded and it may be outdated. Please try again."
)

// listSessions renders the session list (spec §8.3; legacy:
// SessionController::index).
func (s *server) listSessions(w http.ResponseWriter, r *http.Request) {
	list, err := s.TestSessions.ListTestSessions(r.Context())
	if err != nil {
		s.serverError(w, r, "list test sessions", err)
		return
	}
	s.render(w, r, http.StatusOK, views.SessionListPage(s.page(r, ""), list))
}

// newSession shows the test picker (legacy: SessionController::create and
// Livewire/Sessions/CreateSession). A successful ?refresh redirects back,
// so reloading the page does not repeat it. A failed one renders the page
// directly: redirecting would start another discovery straight away.
func (s *server) newSession(w http.ResponseWriter, r *http.Request) {
	refresh := r.URL.Query().Has("refresh")
	form := views.NewSessionForm{Environments: s.Environments, Environment: s.Environments[0]}
	status := s.loadTests(r, &form, refresh)
	if refresh && form.LoadError == "" && form.Warning == "" {
		http.Redirect(w, r, "/sessions/new", http.StatusSeeOther)
		return
	}
	s.render(w, r, status, views.NewSessionPage(s.page(r, "New session"), form))
}

// loadTests puts the discovery result into form. If discovery failed, it
// falls back to the last result with a warning that it may be outdated, or
// sets the error message if there is none. It returns the status to render
// with.
func (s *server) loadTests(r *http.Request, form *views.NewSessionForm, refresh bool) int {
	res, err := s.Discovery.Tests(r.Context(), refresh)
	if err != nil {
		s.Logger.ErrorContext(r.Context(), "test discovery failed", "err", err, "fallback", res != nil)
		if res == nil {
			form.LoadError = msgDiscoveryError
			return http.StatusBadGateway
		}
		form.Warning = msgDiscoveryStale
	}
	form.Tests = res
	return http.StatusOK
}

// createSession validates the form, stores the session with its services,
// suites and tests, and redirects to it (spec §8.2). Tests are looked up in
// the discovery snapshot the form was rendered from, so repository ids,
// paths and commit SHAs never come from the client.
func (s *server) createSession(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	keys := r.PostForm["tests"]
	form := views.NewSessionForm{
		Environments: s.Environments,
		Name:         strings.TrimSpace(r.PostForm.Get("name")),
		Environment:  r.PostForm.Get("environment"),
		Selected:     make(map[string]bool, len(keys)),
	}
	for _, k := range keys {
		form.Selected[k] = true
	}

	switch n := utf8.RuneCountInString(form.Name); {
	case n == 0:
		form.Errors = append(form.Errors, msgNameRequired)
	case n > maxSessionName:
		form.Errors = append(form.Errors, msgNameTooLong)
	}
	if !slices.Contains(s.Environments, form.Environment) {
		form.Errors = append(form.Errors, msgEnvironment)
	}

	var groups []discovery.Group
	snapshot, ok := s.Discovery.Snapshot(r.PostForm.Get("discovery"))
	switch {
	case len(keys) == 0:
		form.Errors = append(form.Errors, msgNoTests)
	case !ok:
		form.Errors = append(form.Errors, msgSnapshotGone)
	default:
		var err error
		if groups, err = snapshot.Select(keys); err != nil {
			s.Logger.WarnContext(ctx, "invalid test selection", "err", err)
			if errors.Is(err, discovery.ErrNoTests) {
				form.Errors = append(form.Errors, msgNoTests)
			} else {
				form.Errors = append(form.Errors, msgTestsUnusable)
			}
		}
	}

	if len(form.Errors) > 0 {
		status := http.StatusUnprocessableEntity
		if ok {
			form.Tests = snapshot
		} else if st := s.loadTests(r, &form, false); st != http.StatusOK {
			status = st
		}
		s.render(w, r, status, views.NewSessionPage(s.page(r, "New session"), form))
		return
	}

	u, _ := auth.UserFrom(ctx)
	ns := store.NewTestSession{
		IssuerID:    u.ID,
		IssuerName:  u.Name,
		Name:        form.Name,
		Environment: form.Environment,
		Services:    newServices(groups),
	}
	id, err := s.TestSessions.CreateTestSession(ctx, ns)
	if err != nil {
		s.serverError(w, r, "create test session", err)
		return
	}
	s.Logger.InfoContext(ctx, "test session created", "session.id", id, "services", len(ns.Services))
	http.Redirect(w, r, "/sessions/"+strconv.FormatInt(id, 10), http.StatusSeeOther)
}

// newServices maps selected test groups to the rows to store.
func newServices(groups []discovery.Group) []store.NewService {
	out := make([]store.NewService, 0, len(groups))
	for _, g := range groups {
		svc := store.NewService{
			Type:           g.Type,
			Name:           g.Service.Name,
			Path:           g.Service.Path,
			RepositoryID:   g.Repository.ID,
			RepositoryName: g.Repository.Name,
			WorkflowID:     g.Service.WorkflowID,
			CommitSHA:      g.Repository.CommitSHA,
			Branch:         discovery.Branch,
		}
		for _, su := range g.Service.Suites {
			suite := store.NewSuite{Name: su.Name, Path: su.Path}
			for _, t := range su.Tests {
				suite.Tests = append(suite.Tests, store.NewTest{Name: t.Name, Path: t.Path})
			}
			svc.Suites = append(svc.Suites, suite)
		}
		out = append(out, svc)
	}
	return out
}

// showSession renders the session detail page (legacy:
// SessionController::show). Runs, activity and reviews are added by later
// iterations.
func (s *server) showSession(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		http.NotFound(w, r)
		return
	}
	ts, err := s.TestSessions.TestSession(r.Context(), id)
	if errors.Is(err, store.ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		s.serverError(w, r, "load test session", err)
		return
	}
	s.render(w, r, http.StatusOK, views.SessionPage(s.page(r, ts.Name), ts))
}
