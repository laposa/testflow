package web

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/laposa/testflow/app-go/internal/mail"
	"github.com/laposa/testflow/app-go/internal/store"
)

// fakeReviews keeps review requests in memory with the store's rules.
type fakeReviews struct {
	mu       sync.Mutex
	users    map[int64]store.User
	activity *fakeActivity
	requests []store.ReviewRequest
	err      error // returned by every method
}

func (f *fakeReviews) PendingReviews(_ context.Context, sessionID int64) ([]store.ReviewRequest, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []store.ReviewRequest
	for _, r := range f.requests {
		if r.SessionID == sessionID && r.Status == store.ReviewPending {
			out = append(out, r)
		}
	}
	return out, f.err
}

func (f *fakeReviews) ReviewCandidates(_ context.Context, sessionID int64) ([]store.User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []store.User
	for _, u := range f.users {
		if !f.pendingFor(sessionID, u.ID) {
			out = append(out, u)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, f.err
}

func (f *fakeReviews) pendingFor(sessionID, userID int64) bool {
	for _, r := range f.requests {
		if r.SessionID == sessionID && r.Reviewer.ID == userID && r.Status == store.ReviewPending {
			return true
		}
	}
	return false
}

func (f *fakeReviews) RequestReview(_ context.Context, sessionID int64, requester store.User, reviewerID int64) (store.ReviewRequest, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return store.ReviewRequest{}, f.err
	}
	reviewer, ok := f.users[reviewerID]
	if !ok {
		return store.ReviewRequest{}, store.ErrNotFound
	}
	if f.pendingFor(sessionID, reviewerID) {
		return store.ReviewRequest{}, store.ErrConflict
	}
	r := store.ReviewRequest{ID: int64(len(f.requests) + 1), SessionID: sessionID, Requester: requester,
		Reviewer: reviewer, Status: store.ReviewPending}
	f.requests = append(f.requests, r)
	f.log(sessionID, requester.Name, store.ActivityReviewRequested, store.ReviewRequestedBody(requester.Name, reviewer.Name))
	return r, nil
}

func (f *fakeReviews) CompleteReview(_ context.Context, sessionID, reviewID int64, user store.User, approve bool) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	r, err := f.find(sessionID, reviewID)
	switch {
	case err != nil:
		return err
	case r.Reviewer.ID != user.ID:
		return store.ErrForbidden
	case r.Status != store.ReviewPending:
		return store.ErrConflict
	}
	r.Status = store.ReviewRejected
	typ := store.ActivityReviewRejected
	if approve {
		r.Status, typ = store.ReviewApproved, store.ActivityReviewApproved
	}
	f.log(sessionID, user.Name, typ, store.ReviewCompletedBody(user.Name, approve))
	return nil
}

func (f *fakeReviews) RevokeReview(_ context.Context, sessionID, reviewID, userID int64) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	r, err := f.find(sessionID, reviewID)
	switch {
	case err != nil:
		return err
	case r.Requester.ID != userID:
		return store.ErrForbidden
	case r.Status != store.ReviewPending:
		return store.ErrConflict
	}
	r.Status = "revoked"
	return nil
}

func (f *fakeReviews) find(sessionID, reviewID int64) (*store.ReviewRequest, error) {
	if f.err != nil {
		return nil, f.err
	}
	for i := range f.requests {
		if r := &f.requests[i]; r.ID == reviewID && r.SessionID == sessionID && r.Status != "revoked" {
			return r, nil
		}
	}
	return nil, store.ErrNotFound
}

func (f *fakeReviews) log(sessionID int64, user string, typ store.ActivityType, body string) {
	f.activity.mu.Lock()
	defer f.activity.mu.Unlock()
	a := store.Activity{ID: int64(len(f.activity.entries[sessionID]) + 200), UserName: user, Type: typ, Body: body, CreatedAt: time.Now()}
	f.activity.entries[sessionID] = append([]store.Activity{a}, f.activity.entries[sessionID]...)
}

// fakeMailer records sent messages.
type fakeMailer struct {
	mu   sync.Mutex
	sent []mail.Message
	err  error
}

func (f *fakeMailer) Send(_ context.Context, m mail.Message) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.sent = append(f.sent, m)
	return f.err
}

// reviewsEnv is runsEnv with users Ada (signed in, 1), Bob (2, with an
// email) and Cy (3, without one).
func reviewsEnv(t *testing.T) (*testEnv, *http.Cookie) {
	t.Helper()
	e, cookie := runsEnv(t) // signs in Ada as user 1
	e.signIn("Bob")
	e.signIn("Cy")
	bob := e.store.Users[2]
	bob.Email = "bob@example.com"
	e.store.Users[2] = bob
	return e, cookie
}

func TestRequestReview(t *testing.T) {
	e, cookie := reviewsEnv(t)

	body := get(e.h, "/sessions/3", cookie).Body.String()
	assertContains(t, body,
		`<section id="reviews" class="reviews"><div id="review-errors" class="field-error" role="alert"></div>`,
		`<form method="post" action="/sessions/3/reviews" class="request-review" hx-post="/sessions/3/reviews" hx-target="#reviews" hx-swap="outerHTML" hx-sync="this:drop">`,
		`<option value="">Select a reviewer</option><option value="1">Ada</option><option value="2">Bob</option><option value="3">Cy</option>`,
		`<div id="activity-feed" class="activity-feed">`,
	)

	// No reviewer, or one who does not exist.
	for _, v := range []string{"", "x", "99"} {
		rec := postHTMX(e, "/sessions/3/reviews", url.Values{"reviewer_id": {v}}, cookie)
		if rec.Code != http.StatusUnprocessableEntity || !strings.Contains(rec.Body.String(), msgSelectReviewer) {
			t.Errorf("reviewer %q: status = %d", v, rec.Code)
		}
	}

	// Bob is asked: the block comes back with his request, the feed out of
	// band with the entry, and he gets the email.
	rec := postHTMX(e, "/sessions/3/reviews", url.Values{"reviewer_id": {"2"}}, cookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("request: status = %d, body = %s", rec.Code, rec.Body.String())
	}
	assertContains(t, rec.Body.String(),
		`<div class="review-form"><p>Review pending for Bob</p><div class="review-form-actions"><form method="post" action="/sessions/3/reviews/1/revoke"`,
		`<div id="activity-feed" class="activity-feed" hx-swap-oob="true">`,
		"Ada requested a review from Bob",
	)
	assertNotContains(t, rec.Body.String(), `<option value="2">Bob</option>`, "<html")
	if len(e.mailer.sent) != 1 {
		t.Fatalf("sent = %+v", e.mailer.sent)
	}
	m := e.mailer.sent[0]
	if m.To.Address != "bob@example.com" || m.To.Name != "Bob" || m.Subject != "Review Request" ||
		!strings.Contains(m.Text, "Hi Bob,") || !strings.Contains(m.Text, "Ada has requested a review from you.") ||
		!strings.Contains(m.Text, "http://example.com/sessions/3") {
		t.Errorf("message = %+v", m)
	}

	// A second pending request for Bob is refused.
	rec = postHTMX(e, "/sessions/3/reviews", url.Values{"reviewer_id": {"2"}}, cookie)
	if rec.Code != http.StatusUnprocessableEntity || !strings.Contains(rec.Body.String(), msgAlreadyReviewer) {
		t.Errorf("duplicate: status = %d", rec.Code)
	}

	// Cy has no email: no message. A failing mailer does not fail the
	// request (without htmx: back to the session).
	e.mailer.err = errors.New("smtp down")
	rec = postForm(e.h, "/sessions/3/reviews", url.Values{"reviewer_id": {"3"}}, cookie)
	if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/sessions/3#reviews" || len(e.mailer.sent) != 1 {
		t.Errorf("Cy: status = %d %q, sent = %d", rec.Code, rec.Header().Get("Location"), len(e.mailer.sent))
	}
	rec = postForm(e.h, "/sessions/3/reviews", url.Values{"reviewer_id": {"1"}}, cookie)
	if rec.Code != http.StatusSeeOther || len(e.mailer.sent) != 1 {
		t.Errorf("Ada without email: status = %d, sent = %d", rec.Code, len(e.mailer.sent))
	}
	ada := e.store.Users[1]
	ada.Email = "ada@example.com"
	e.store.Users[1] = ada
	e.reviews.requests = e.reviews.requests[:2] // drop Ada's own request
	rec = postForm(e.h, "/sessions/3/reviews", url.Values{"reviewer_id": {"1"}}, cookie)
	if rec.Code != http.StatusSeeOther || len(e.mailer.sent) != 2 || !strings.Contains(e.logs.String(), "review request email failed") {
		t.Errorf("failing mailer: status = %d, sent = %d", rec.Code, len(e.mailer.sent))
	}

	// A missing session.
	rec = postHTMX(e, "/sessions/4/reviews", url.Values{"reviewer_id": {"2"}}, cookie)
	if rec.Code != http.StatusNotFound || rec.Header().Get("HX-Retarget") != "#review-errors" {
		t.Errorf("missing session: status = %d, retarget = %q", rec.Code, rec.Header().Get("HX-Retarget"))
	}
}

func TestCompleteReview(t *testing.T) {
	e, ada := reviewsEnv(t)
	bob := &http.Cookie{Name: ada.Name, Value: e.store.Add(2, time.Now().Add(time.Hour))}
	e.reviews.requests = []store.ReviewRequest{
		{ID: 1, SessionID: 3, Requester: e.store.Users[2], Reviewer: e.store.Users[1], Status: store.ReviewPending},
		{ID: 2, SessionID: 3, Requester: e.store.Users[1], Reviewer: e.store.Users[3], Status: store.ReviewPending},
	}

	// Ada reviews request 1 and revokes request 2.
	body := get(e.h, "/sessions/3", ada).Body.String()
	assertContains(t, body,
		`<div class="review-form-title"><h3>Review Required</h3><p>Bob requested a review</p></div>`,
		`<form method="post" action="/sessions/3/reviews/1/approve" hx-post="/sessions/3/reviews/1/approve" hx-target="#reviews" hx-swap="outerHTML" hx-sync="this:drop"><button type="submit" class="button filled">Approve</button></form>`,
		`<button type="submit" class="button danger">Reject</button>`,
		`<p>Review pending for Cy</p><div class="review-form-actions"><form method="post" action="/sessions/3/reviews/2/revoke"`,
	)
	// Bob can revoke his request to Ada, and sees Ada's request to Cy
	// without buttons.
	body = get(e.h, "/sessions/3", bob).Body.String()
	assertContains(t, body,
		`<p>Review pending for Ada</p><div class="review-form-actions"><form method="post" action="/sessions/3/reviews/1/revoke"`,
		`<div class="review-form"><p>Review pending for Cy</p></div>`)
	assertNotContains(t, body, "/reviews/1/approve", "/reviews/2/revoke")

	// Session 1 exists but has none of these requests.
	other := e.testSessions.sessions[3]
	other.ID = 1
	e.testSessions.sessions[1] = other

	for _, tt := range []struct {
		name, target string
		cookie       *http.Cookie
		status       int
		msg          string
	}{
		{"Bob approves a request to Ada", "/sessions/3/reviews/1/approve", bob, http.StatusForbidden, msgNotReviewer},
		{"Bob revokes Ada's request", "/sessions/3/reviews/2/revoke", bob, http.StatusForbidden, msgNotRequester},
		{"another session", "/sessions/1/reviews/1/approve", ada, http.StatusNotFound, msgReviewGone},
		{"unknown request", "/sessions/3/reviews/9/reject", ada, http.StatusNotFound, msgReviewGone},
	} {
		rec := postHTMX(e, tt.target, url.Values{}, tt.cookie)
		if rec.Code != tt.status || !strings.Contains(rec.Body.String(), tt.msg) {
			t.Errorf("%s: status = %d, body = %s", tt.name, rec.Code, rec.Body.String())
		}
	}

	rec := postHTMX(e, "/sessions/3/reviews/1/approve", url.Values{}, ada)
	if rec.Code != http.StatusOK {
		t.Fatalf("approve: status = %d", rec.Code)
	}
	assertContains(t, rec.Body.String(), "Ada approved the review", `hx-swap-oob="true"`)
	assertNotContains(t, rec.Body.String(), "Review Required")
	rec = postHTMX(e, "/sessions/3/reviews/1/reject", url.Values{}, ada)
	if rec.Code != http.StatusConflict || !strings.Contains(rec.Body.String(), msgReviewDone) {
		t.Errorf("reject after approve: status = %d", rec.Code)
	}

	rec = postForm(e.h, "/sessions/3/reviews/2/revoke", url.Values{}, ada)
	if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/sessions/3#reviews" {
		t.Errorf("revoke: status = %d %q", rec.Code, rec.Header().Get("Location"))
	}
	// Errors without htmx render the session page with the message.
	rec = postForm(e.h, "/sessions/3/reviews/2/revoke", url.Values{}, ada)
	if rec.Code != http.StatusNotFound || !strings.Contains(rec.Body.String(), "<!doctype html>") ||
		!strings.Contains(rec.Body.String(), msgReviewGone) {
		t.Errorf("revoke again: status = %d", rec.Code)
	}

	e.reviews.err = errors.New("db down")
	rec = postHTMX(e, "/sessions/3/reviews/1/approve", url.Values{}, ada)
	if rec.Code != http.StatusInternalServerError || rec.Header().Get("HX-Retarget") != "#review-errors" {
		t.Errorf("store error: status = %d, retarget = %q", rec.Code, rec.Header().Get("HX-Retarget"))
	}
}
