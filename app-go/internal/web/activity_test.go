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

	"github.com/laposa/testflow/app-go/internal/store"
)

// fakeActivity keeps activity in memory. AddComment knows the sessions of
// the fake test session store.
type fakeActivity struct {
	mu       sync.Mutex
	sessions *fakeTestSessions
	entries  map[int64][]store.Activity // newest first
	added    []store.Activity
	userIDs  []int64 // who added each comment
	err      error
}

func (f *fakeActivity) SessionActivity(_ context.Context, sessionID int64) ([]store.Activity, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.entries[sessionID], f.err
}

func (f *fakeActivity) AddComment(_ context.Context, sessionID, userID int64, body string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return f.err
	}
	if _, ok := f.sessions.sessions[sessionID]; !ok {
		return store.ErrNotFound
	}
	f.userIDs = append(f.userIDs, userID)
	a := store.Activity{ID: int64(len(f.added) + 100), UserName: "Ada",
		Type: store.ActivityComment, Body: body, CreatedAt: time.Now()}
	f.added = append(f.added, a)
	f.entries[sessionID] = append([]store.Activity{a}, f.entries[sessionID]...)
	return nil
}

// activityEnv has session 3 with two entries.
func activityEnv(t *testing.T) (*testEnv, *http.Cookie) {
	e := newEnv(t, Options{})
	cookie := e.signIn("Ada")
	e.testSessions.sessions[3] = store.TestSession{ID: 3, IssuerName: "Ada", Name: "Release", Environment: "beta",
		CreatedAt: time.Now().Add(-3 * time.Hour)}
	e.activity.entries[3] = []store.Activity{
		{ID: 2, UserName: "Bob <B>", Type: store.ActivityComment, Body: "Line one\n<script>x</script>",
			CreatedAt: time.Now().Add(-5 * time.Minute)},
		{ID: 1, UserName: "Ada", Type: store.ActivitySessionCreated, Body: "Ada created the session.",
			CreatedAt: time.Now().Add(-3 * time.Hour)},
	}
	return e, cookie
}

func postHTMX(e *testEnv, target string, form url.Values, cookie *http.Cookie) *httptest.ResponseRecorder {
	r := sameOrigin(httptest.NewRequest(http.MethodPost, target, strings.NewReader(form.Encode())))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.Header.Set("HX-Request", "true")
	r.AddCookie(cookie)
	return do(e.h, r)
}

func TestSessionPageActivity(t *testing.T) {
	e, cookie := activityEnv(t)
	rec := get(e.h, "/sessions/3", cookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if got := rec.Header().Get("Vary"); got != "HX-Request" {
		t.Errorf("Vary = %q: the page and the htmx fragment share the URL", got)
	}
	body := rec.Body.String()
	for _, want := range []string{
		`<section id="activity" class="activity">`,
		`<form method="post" action="/sessions/3/comments" class="create-comment" hx-post="/sessions/3/comments" hx-target="#activity" hx-swap="outerHTML" hx-sync="this:drop">`,
		`<label for="comment-body">Add a comment</label>`,
		`<div id="comment-errors" class="field-error" role="alert"></div>`,
		`<div class="activity-feed-item activity-feed--comment">`,
		`<strong>Bob &lt;B&gt;</strong> commented`,
		"Line one\n&lt;script&gt;x&lt;/script&gt;",
		`>5 minutes ago</time>`,
		`<div class="activity-feed-item activity-feed--session_created">`,
		`>3 hours ago</time>`,
		`<svg `,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("session page missing %q", want)
		}
	}
	// The form is above the feed, which is newest first.
	if f, c, s := strings.Index(body, "create-comment"), strings.Index(body, "activity-feed--comment"),
		strings.Index(body, "activity-feed--session_created"); !(f < c && c < s) {
		t.Error("want the form, then the newest entry, then the oldest")
	}
}

func TestAddComment(t *testing.T) {
	e, cookie := activityEnv(t)
	rec := postForm(e.h, "/sessions/3/comments", url.Values{"body": {"  Looks good  "}}, cookie)
	if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/sessions/3#activity" {
		t.Fatalf("POST = %d %q, want redirect to the session", rec.Code, rec.Header().Get("Location"))
	}
	if len(e.activity.added) != 1 || e.activity.added[0].Body != "Looks good" || e.activity.userIDs[0] != 1 {
		t.Errorf("added = %+v, want the trimmed body by the signed-in user", e.activity.added)
	}

	// Five characters, not five bytes.
	if rec := postForm(e.h, "/sessions/3/comments", url.Values{"body": {"žluťá"}}, cookie); rec.Code != http.StatusSeeOther {
		t.Errorf("5-letter non-ASCII comment = %d, want 303", rec.Code)
	}
}

func TestAddCommentHTMX(t *testing.T) {
	e, cookie := activityEnv(t)
	rec := postHTMX(e, "/sessions/3/comments", url.Values{"body": {"Ship it!"}}, cookie)
	body := rec.Body.String()
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if !strings.HasPrefix(body, `<section id="activity" class="activity">`) || strings.Contains(body, "<html") {
		t.Errorf("want only the activity block, got %.200s", body)
	}
	if !strings.Contains(body, "Ship it!") || !strings.Contains(body, `rows="10" required></textarea>`) {
		t.Error("want the new comment in the feed and an empty form")
	}
}

func TestAddCommentValidation(t *testing.T) {
	e, cookie := activityEnv(t)
	for _, comment := range []string{"", "   ", " abcd "} {
		rec := postForm(e.h, "/sessions/3/comments", url.Values{"body": {comment}}, cookie)
		if rec.Code != http.StatusUnprocessableEntity {
			t.Errorf("%q = %d, want 422", comment, rec.Code)
		}
		body := rec.Body.String()
		if !strings.Contains(body, msgCommentTooShort) || !strings.Contains(body, "<title>Release · TestFlow</title>") {
			t.Errorf("%q: want the full page with the error", comment)
		}
	}
	rec := postForm(e.h, "/sessions/3/comments", url.Values{"body": {"<b>"}}, cookie)
	if !strings.Contains(rec.Body.String(), `aria-describedby="comment-errors">&lt;b&gt;</textarea>`) {
		t.Error("the typed comment should be kept, escaped")
	}

	rec = postHTMX(e, "/sessions/3/comments", url.Values{"body": {"abc"}}, cookie)
	if body := rec.Body.String(); rec.Code != http.StatusUnprocessableEntity ||
		!strings.HasPrefix(body, `<section id="activity"`) || !strings.Contains(body, msgCommentTooShort) {
		t.Errorf("htmx validation = %d, want 422 with the activity block", rec.Code)
	}
	if len(e.activity.added) != 0 {
		t.Errorf("invalid comments stored: %+v", e.activity.added)
	}
}

func TestAddCommentErrors(t *testing.T) {
	e, cookie := activityEnv(t)
	for _, target := range []string{"/sessions/4/comments", "/sessions/x/comments"} {
		if rec := postForm(e.h, target, url.Values{"body": {"Hello there"}}, cookie); rec.Code != http.StatusNotFound {
			t.Errorf("POST %s = %d, want 404", target, rec.Code)
		}
	}
	if rec := postForm(e.h, "/sessions/4/comments", url.Values{"body": {"abc"}}, cookie); rec.Code != http.StatusNotFound {
		t.Errorf("invalid comment on a missing session = %d, want 404", rec.Code)
	}
	if rec := postForm(e.h, "/sessions/3/comments", url.Values{"body": {"Hello there"}}); rec.Code != http.StatusSeeOther ||
		rec.Header().Get("Location") != "/login" {
		t.Errorf("signed out = %d, want redirect to /login", rec.Code)
	}

	e.activity.err = errors.New("db down")
	if rec := postForm(e.h, "/sessions/3/comments", url.Values{"body": {"Hello there"}}, cookie); rec.Code != http.StatusInternalServerError {
		t.Errorf("store error on add = %d, want 500", rec.Code)
	}
	if rec := get(e.h, "/sessions/3", cookie); rec.Code != http.StatusInternalServerError {
		t.Errorf("store error on read = %d, want 500", rec.Code)
	}
}

func TestAddCommentHTMXErrors(t *testing.T) {
	e, cookie := activityEnv(t)
	check := func(name string, rec *httptest.ResponseRecorder, status int, msg string) {
		t.Helper()
		if rec.Code != status {
			t.Errorf("%s: status = %d, want %d", name, rec.Code, status)
		}
		if rec.Header().Get("HX-Retarget") != "#comment-errors" || rec.Header().Get("HX-Reswap") != "innerHTML" {
			t.Errorf("%s: want the message retargeted into the form's error area, headers %v", name, rec.Header())
		}
		if got := rec.Body.String(); got != "<p>"+msg+"</p>" {
			t.Errorf("%s: body = %q", name, got)
		}
	}
	check("missing session", postHTMX(e, "/sessions/4/comments", url.Values{"body": {"Hello there"}}, cookie),
		http.StatusNotFound, msgSessionGone)
	e.activity.err = errors.New("db down")
	check("store error", postHTMX(e, "/sessions/3/comments", url.Values{"body": {"Hello there"}}, cookie),
		http.StatusInternalServerError, msgActivityFailed)
}
