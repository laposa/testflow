package web

import (
	"context"
	"errors"
	"net/http"
	netmail "net/mail"
	"strconv"
	"time"

	"github.com/laposa/testflow/app-go/internal/auth"
	"github.com/laposa/testflow/app-go/internal/mail"
	"github.com/laposa/testflow/app-go/internal/store"
	"github.com/laposa/testflow/app-go/internal/web/views"
)

// ReviewStore keeps review requests (store.Store).
type ReviewStore interface {
	PendingReviews(ctx context.Context, sessionID int64) ([]store.ReviewRequest, error)
	ReviewCandidates(ctx context.Context, sessionID int64) ([]store.User, error)
	RequestReview(ctx context.Context, sessionID int64, requester store.User, reviewerID int64) (store.ReviewRequest, error)
	CompleteReview(ctx context.Context, sessionID, reviewID int64, user store.User, approve bool) error
	RevokeReview(ctx context.Context, sessionID, reviewID, userID int64) error
}

// Review messages (spec §8.9).
const (
	msgSelectReviewer  = "Select a reviewer."
	msgAlreadyReviewer = "This user already has a pending review request on this session."
	msgReviewGone      = "This review request does not exist any more."
	msgReviewDone      = "This review request is no longer pending."
	msgNotReviewer     = "Only the reviewer can approve or reject this review."
	msgNotRequester    = "Only the user who requested this review can revoke it."
)

// requestReview asks a user to review the session, emails them and logs
// review_requested (spec §8.9; legacy: RequestReview::save).
func (s *server) requestReview(w http.ResponseWriter, r *http.Request) {
	ts, ok := s.loadReviewSession(w, r)
	if !ok {
		return
	}
	ctx := r.Context()
	u, _ := auth.UserFrom(ctx)
	reviewerID, err := strconv.ParseInt(r.PostForm.Get("reviewer_id"), 10, 64)
	if err != nil || reviewerID <= 0 {
		s.renderReviews(w, r, http.StatusUnprocessableEntity, ts, msgSelectReviewer)
		return
	}
	req, err := s.Reviews.RequestReview(ctx, ts.ID, u, reviewerID)
	switch {
	case errors.Is(err, store.ErrConflict):
		s.renderReviews(w, r, http.StatusUnprocessableEntity, ts, msgAlreadyReviewer)
		return
	case errors.Is(err, store.ErrNotFound): // the reviewer (the session was loaded)
		s.renderReviews(w, r, http.StatusUnprocessableEntity, ts, msgSelectReviewer)
		return
	case err != nil:
		s.reviewsError(w, r, "request review", err)
		return
	}
	s.Logger.InfoContext(ctx, "review requested", "session.id", ts.ID, "review.id", req.ID)
	s.sendReviewRequest(ctx, r, ts, req)
	s.reviewsDone(w, r, ts)
}

// sendReviewRequest emails the reviewer (spec §8.10; legacy:
// ReviewRequestNotification). A failure is logged only.
func (s *server) sendReviewRequest(ctx context.Context, r *http.Request, ts store.TestSession, req store.ReviewRequest) {
	if req.Reviewer.Email == "" {
		s.Logger.WarnContext(ctx, "review request email not sent: the reviewer has no email address",
			"session.id", ts.ID, "review.id", req.ID)
		return
	}
	// Not cancelled with the request; the mailer bounds the send
	// (mail.DefaultTimeout).
	err := s.Mailer.Send(context.WithoutCancel(ctx), mail.Message{
		To:      netmail.Address{Name: req.Reviewer.Name, Address: req.Reviewer.Email},
		Subject: "Review Request",
		Text: "Hi " + req.Reviewer.Name + ",\n\n" +
			req.Requester.Name + " has requested a review from you.\n\n" +
			"Review: " + s.absURL(r, "/sessions/"+strconv.FormatInt(ts.ID, 10)) + "\n",
	})
	if err != nil {
		s.Logger.ErrorContext(ctx, "review request email failed", "session.id", ts.ID, "review.id", req.ID, "err", err)
	}
}

// absURL is path on the public origin (BASE_URL), or on the request's host
// when none is configured.
func (s *server) absURL(r *http.Request, path string) string {
	if s.BaseURL != nil && s.BaseURL.Host != "" {
		return s.BaseURL.JoinPath(path).String()
	}
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	return scheme + "://" + r.Host + path
}

// approveReview, rejectReview and revokeReview answer the review buttons
// (legacy: ReviewForm::approve, ::reject, RevokeReviewForm::delete).
func (s *server) approveReview(w http.ResponseWriter, r *http.Request) { s.completeReview(w, r, true) }
func (s *server) rejectReview(w http.ResponseWriter, r *http.Request)  { s.completeReview(w, r, false) }

func (s *server) completeReview(w http.ResponseWriter, r *http.Request, approve bool) {
	ts, ok := s.loadReviewSession(w, r)
	if !ok {
		return
	}
	reviewID, ok := pathID(w, r, "review")
	if !ok {
		return
	}
	ctx := r.Context()
	u, _ := auth.UserFrom(ctx)
	err := s.Reviews.CompleteReview(ctx, ts.ID, reviewID, u, approve)
	if s.reviewFailed(w, r, ts, "complete review", err, msgNotReviewer) {
		return
	}
	s.Logger.InfoContext(ctx, "review completed", "session.id", ts.ID, "review.id", reviewID, "approved", approve)
	s.reviewsDone(w, r, ts)
}

func (s *server) revokeReview(w http.ResponseWriter, r *http.Request) {
	ts, ok := s.loadReviewSession(w, r)
	if !ok {
		return
	}
	reviewID, ok := pathID(w, r, "review")
	if !ok {
		return
	}
	ctx := r.Context()
	u, _ := auth.UserFrom(ctx)
	err := s.Reviews.RevokeReview(ctx, ts.ID, reviewID, u.ID)
	if s.reviewFailed(w, r, ts, "revoke review", err, msgNotRequester) {
		return
	}
	s.Logger.InfoContext(ctx, "review revoked", "session.id", ts.ID, "review.id", reviewID)
	s.reviewsDone(w, r, ts)
}

// reviewFailed answers a failed review action with the reviews block and
// a message, and reports whether err was one.
func (s *server) reviewFailed(w http.ResponseWriter, r *http.Request, ts store.TestSession, what string, err error, forbidden string) bool {
	switch {
	case err == nil:
		return false
	case errors.Is(err, store.ErrNotFound):
		s.renderReviews(w, r, http.StatusNotFound, ts, msgReviewGone)
	case errors.Is(err, store.ErrForbidden):
		s.renderReviews(w, r, http.StatusForbidden, ts, forbidden)
	case errors.Is(err, store.ErrConflict):
		s.renderReviews(w, r, http.StatusConflict, ts, msgReviewDone)
	default:
		s.reviewsError(w, r, what, err)
	}
	return true
}

// loadReviewSession parses the form and loads the session of a review
// action; a missing one is a 404.
func (s *server) loadReviewSession(w http.ResponseWriter, r *http.Request) (store.TestSession, bool) {
	id, ok := sessionID(w, r)
	if !ok {
		return store.TestSession{}, false
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return store.TestSession{}, false
	}
	ts, err := s.TestSessions.TestSession(r.Context(), id)
	if err != nil {
		s.reviewsError(w, r, "load test session", err)
		return store.TestSession{}, false
	}
	return ts, true
}

// reviewsDone answers a successful review action: back to the session
// without htmx (PRG), otherwise the reviews block and, out of band, the
// activity feed with the new entry.
func (s *server) reviewsDone(w http.ResponseWriter, r *http.Request, ts store.TestSession) {
	if !isHTMX(r) {
		http.Redirect(w, r, "/sessions/"+strconv.FormatInt(ts.ID, 10)+"#"+views.ReviewsID, http.StatusSeeOther)
		return
	}
	s.renderReviews(w, r, http.StatusOK, ts)
}

// renderReviews renders the reviews block with msgs for htmx (plus the
// activity feed out of band), or the session page.
func (s *server) renderReviews(w http.ResponseWriter, r *http.Request, status int, ts store.TestSession, msgs ...string) {
	ctx := r.Context()
	rv, err := s.reviewsView(ctx, ts.ID, msgs)
	if err != nil {
		s.reviewsError(w, r, "load reviews", err)
		return
	}
	av := views.ActivityView{SessionID: ts.ID, Now: time.Now()}
	if av.Entries, err = s.Activity.SessionActivity(ctx, ts.ID); err != nil {
		s.reviewsError(w, r, "load session activity", err)
		return
	}
	w.Header().Add("Vary", "HX-Request")
	if isHTMX(r) {
		s.render(w, r, status, views.ReviewsWithFeed(rv, av))
		return
	}
	rl, err := s.runList(ctx, ts, runState{})
	if err != nil {
		s.reviewsError(w, r, "load runs", err)
		return
	}
	s.render(w, r, status, views.SessionPage(s.page(r, ts.Name), ts, rl, rv, av))
}

// reviewsView loads the reviews block of a session for the signed-in user.
func (s *server) reviewsView(ctx context.Context, sessionID int64, msgs []string) (views.ReviewsView, error) {
	u, _ := auth.UserFrom(ctx)
	rv := views.ReviewsView{SessionID: sessionID, UserID: u.ID, Errors: msgs}
	var err error
	if rv.Pending, err = s.Reviews.PendingReviews(ctx, sessionID); err != nil {
		return rv, err
	}
	rv.Candidates, err = s.Reviews.ReviewCandidates(ctx, sessionID)
	return rv, err
}

// reviewsError is activityError for the reviews block.
func (s *server) reviewsError(w http.ResponseWriter, r *http.Request, what string, err error) {
	s.fragmentError(w, r, what, err, views.ReviewErrorsID, msgActivityFailed)
}
