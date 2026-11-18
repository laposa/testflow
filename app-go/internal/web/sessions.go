package web

import (
	"context"
	"errors"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"
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

// ActivityStore reads a session's activity and adds comments (store.Store).
type ActivityStore interface {
	SessionActivity(ctx context.Context, sessionID int64) ([]store.Activity, error)
	AddComment(ctx context.Context, sessionID, userID int64, body string) error
}

// minComment is the legacy rule (required|min:5), counted in characters
// after trimming.
const minComment = 5

// maxSessionName matches the legacy varchar(255) column.
const maxSessionName = 255

// Validation messages (spec §8.2; legacy: Actions/Session/CreateSession.php).
const (
	msgNameRequired    = "The session name is required."
	msgNameTooLong     = "The session name may not be longer than 255 characters."
	msgEnvironment     = "Please choose one of the listed environments."
	msgNoTests         = "Please select at least one test suite."
	msgSnapshotGone    = "The test list has expired. Please check your selection and create the session again."
	msgTestsUnusable   = "Some selected tests are not available any more. Please check your selection."
	msgCommentTooShort = "The comment must be at least 5 characters."
	msgSessionGone     = "This session does not exist any more."
	// Not "try again": the comment may have been saved before the failure.
	msgActivityFailed = "Something went wrong. Please reload the page."
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
// SessionController::show). Runs and reviews are added by later iterations.
func (s *server) showSession(w http.ResponseWriter, r *http.Request) {
	id, ok := sessionID(w, r)
	if !ok {
		return
	}
	s.renderActivity(w, r, http.StatusOK, views.ActivityView{SessionID: id})
}

// addComment logs a comment (spec §8.9; legacy: Livewire/Comments/CreateComment).
// An htmx request gets the activity block back; a normal one is redirected
// to the session page (PRG). A validation error re-renders with 422.
func (s *server) addComment(w http.ResponseWriter, r *http.Request) {
	id, ok := sessionID(w, r)
	if !ok {
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	av := views.ActivityView{SessionID: id, Comment: strings.TrimSpace(r.PostForm.Get("body"))}
	if utf8.RuneCountInString(av.Comment) < minComment {
		av.Errors = []string{msgCommentTooShort}
		s.renderActivity(w, r, http.StatusUnprocessableEntity, av)
		return
	}

	ctx := r.Context()
	u, _ := auth.UserFrom(ctx)
	if err := s.Activity.AddComment(ctx, id, u.ID, av.Comment); err != nil {
		s.activityError(w, r, "add comment", err)
		return
	}
	s.Logger.InfoContext(ctx, "comment added", "session.id", id)
	if !isHTMX(r) {
		http.Redirect(w, r, "/sessions/"+strconv.FormatInt(id, 10)+"#"+views.ActivityID, http.StatusSeeOther)
		return
	}
	av.Comment = ""
	s.renderActivity(w, r, http.StatusOK, av)
}

// renderActivity loads the session and its activity into av and renders
// the session page, or only the activity block for htmx requests. A
// missing session is a 404.
func (s *server) renderActivity(w http.ResponseWriter, r *http.Request, status int, av views.ActivityView) {
	ctx := r.Context()
	ts, err := s.TestSessions.TestSession(ctx, av.SessionID)
	if err != nil {
		s.activityError(w, r, "load test session", err)
		return
	}
	if av.Entries, err = s.Activity.SessionActivity(ctx, av.SessionID); err != nil {
		s.activityError(w, r, "load session activity", err)
		return
	}
	av.Now = time.Now()
	// The same URL answers with the full page or the fragment.
	w.Header().Add("Vary", "HX-Request")
	if isHTMX(r) {
		s.render(w, r, status, views.Activity(av))
		return
	}
	s.render(w, r, status, views.SessionPage(s.page(r, ts.Name), ts, av))
}

// activityError answers a failed session page or comment request:
// ErrNotFound is a 404, anything else a logged 500. htmx 4 swaps error
// responses too, so an htmx request gets a short message in the comment
// form's error area instead of the plain error page replacing the whole
// activity block; the typed comment stays in the form.
func (s *server) activityError(w http.ResponseWriter, r *http.Request, what string, err error) {
	status, msg := http.StatusNotFound, msgSessionGone
	if !errors.Is(err, store.ErrNotFound) {
		status, msg = http.StatusInternalServerError, msgActivityFailed
		s.Logger.ErrorContext(r.Context(), what+" failed", "err", err)
	}
	if !isHTMX(r) {
		http.Error(w, http.StatusText(status), status)
		return
	}
	w.Header().Set("HX-Retarget", "#"+views.CommentErrorsID)
	w.Header().Set("HX-Reswap", "innerHTML")
	s.render(w, r, status, views.ErrorText(msg))
}

// sessionID parses the {id} path value. An invalid id is a 404, written to w.
func sessionID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		http.NotFound(w, r)
		return 0, false
	}
	return id, true
}

// isHTMX reports a request made by htmx, which gets a fragment instead of
// the full page (spec §10).
func isHTMX(r *http.Request) bool { return r.Header.Get("HX-Request") == "true" }
