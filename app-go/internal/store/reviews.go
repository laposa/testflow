package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// ErrForbidden is an action the user may not take, e.g. approving a review
// requested from someone else.
var ErrForbidden = errors.New("forbidden")

// Review request statuses (spec §6.10).
const (
	ReviewPending  = "pending"
	ReviewApproved = "approved"
	ReviewRejected = "rejected"
)

// ReviewRequest is a request for one user to review a session.
type ReviewRequest struct {
	ID        int64
	SessionID int64
	Requester User // ID and Name
	Reviewer  User // ID, Name and Email
	Status    string
	CreatedAt time.Time
}

// ReviewRequestedBody is the review_requested activity text (spec §8.9).
func ReviewRequestedBody(requester, reviewer string) string {
	return requester + " requested a review from " + reviewer
}

// ReviewCompletedBody is the review_approved or review_rejected activity
// text (spec §8.9).
func ReviewCompletedBody(user string, approved bool) string {
	if approved {
		return user + " approved the review"
	}
	return user + " rejected the review"
}

// PendingReviews returns a session's pending review requests, oldest first
// (legacy: SessionController::show).
func (s *Store) PendingReviews(ctx context.Context, sessionID int64) ([]ReviewRequest, error) {
	rows, err := s.db.Query(ctx, `
		SELECT rr.id, rr.session_id, rq.id, rq.name, rv.id, rv.name, COALESCE(rv.email, ''), rr.status, rr.created_at
		FROM review_requests rr
		JOIN users rq ON rq.id = rr.requester_id
		JOIN users rv ON rv.id = rr.reviewer_id
		WHERE rr.session_id = $1 AND rr.status = 'pending'
		ORDER BY rr.id`, sessionID)
	if err != nil {
		return nil, fmt.Errorf("pending reviews: %w", err)
	}
	list, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (ReviewRequest, error) {
		var r ReviewRequest
		err := row.Scan(&r.ID, &r.SessionID, &r.Requester.ID, &r.Requester.Name, &r.Reviewer.ID, &r.Reviewer.Name,
			&r.Reviewer.Email, &r.Status, &r.CreatedAt)
		return r, err
	})
	if err != nil {
		return nil, fmt.Errorf("pending reviews: %w", err)
	}
	return list, nil
}

// ReviewCandidates returns the users who can be asked to review a session:
// everyone without a pending request on it, by name (legacy:
// RequestReview::mount).
func (s *Store) ReviewCandidates(ctx context.Context, sessionID int64) ([]User, error) {
	rows, err := s.db.Query(ctx, `
		SELECT u.id, u.name FROM users u
		WHERE NOT EXISTS (SELECT 1 FROM review_requests rr
		                  WHERE rr.session_id = $1 AND rr.reviewer_id = u.id AND rr.status = 'pending')
		ORDER BY lower(u.name), u.id`, sessionID)
	if err != nil {
		return nil, fmt.Errorf("review candidates: %w", err)
	}
	list, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (User, error) {
		var u User
		err := row.Scan(&u.ID, &u.Name)
		return u, err
	})
	if err != nil {
		return nil, fmt.Errorf("review candidates: %w", err)
	}
	return list, nil
}

// RequestReview asks reviewerID to review a session and logs
// review_requested, in one transaction (spec §8.9). It returns the request
// with the reviewer's email for the notification. It is ErrNotFound if the
// session or the reviewer does not exist, and ErrConflict if the reviewer
// already has a pending request on the session.
func (s *Store) RequestReview(ctx context.Context, sessionID int64, requester User, reviewerID int64) (ReviewRequest, error) {
	r := ReviewRequest{SessionID: sessionID, Requester: requester, Status: ReviewPending}
	err := s.inTx(ctx, func(tx pgx.Tx) error {
		if err := tx.QueryRow(ctx, `SELECT id, name, COALESCE(email, '') FROM users WHERE id = $1`, reviewerID).
			Scan(&r.Reviewer.ID, &r.Reviewer.Name, &r.Reviewer.Email); err != nil {
			return notFound(err)
		}
		err := tx.QueryRow(ctx, `
			INSERT INTO review_requests (session_id, requester_id, reviewer_id) VALUES ($1, $2, $3)
			RETURNING id, created_at`, sessionID, requester.ID, reviewerID).Scan(&r.ID, &r.CreatedAt)
		if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok {
			switch pgErr.Code {
			case "23505": // review_requests_one_pending_idx
				return ErrConflict
			case "23503": // the session is gone
				return ErrNotFound
			}
		}
		if err != nil {
			return err
		}
		return insertActivity(ctx, tx, NewActivity{
			SessionID: sessionID, UserID: requester.ID, Type: ActivityReviewRequested,
			Body: ReviewRequestedBody(requester.Name, r.Reviewer.Name),
		})
	})
	if err != nil {
		return ReviewRequest{}, fmt.Errorf("request review: %w", err)
	}
	return r, nil
}

// CompleteReview approves or rejects a pending review request of a session
// as user, its reviewer, and logs review_approved or review_rejected, in
// one transaction (spec §8.9; legacy: ReviewForm). It is ErrNotFound if the
// request is not the session's, ErrForbidden if user is not its reviewer,
// and ErrConflict if it is no longer pending.
func (s *Store) CompleteReview(ctx context.Context, sessionID, reviewID int64, user User, approve bool) error {
	status, typ := ReviewRejected, ActivityReviewRejected
	if approve {
		status, typ = ReviewApproved, ActivityReviewApproved
	}
	err := s.inTx(ctx, func(tx pgx.Tx) error {
		r, err := lockReview(ctx, tx, sessionID, reviewID)
		switch {
		case err != nil:
			return err
		case r.Reviewer.ID != user.ID:
			return ErrForbidden
		case r.Status != ReviewPending:
			return ErrConflict
		}
		if _, err := tx.Exec(ctx, `UPDATE review_requests SET status = $2, completed_at = now() WHERE id = $1`,
			reviewID, status); err != nil {
			return err
		}
		return insertActivity(ctx, tx, NewActivity{
			SessionID: sessionID, UserID: user.ID, Type: typ, Body: ReviewCompletedBody(user.Name, approve),
		})
	})
	if err != nil {
		return fmt.Errorf("complete review: %w", err)
	}
	return nil
}

// RevokeReview deletes a pending review request of a session as userID,
// its requester (spec §8.9; legacy: RevokeReviewForm). Nothing is logged
// (Q4). It is ErrNotFound if the request is not the session's, ErrForbidden
// if userID did not request it, and ErrConflict if it is no longer pending.
func (s *Store) RevokeReview(ctx context.Context, sessionID, reviewID, userID int64) error {
	err := s.inTx(ctx, func(tx pgx.Tx) error {
		r, err := lockReview(ctx, tx, sessionID, reviewID)
		switch {
		case err != nil:
			return err
		case r.Requester.ID != userID:
			return ErrForbidden
		case r.Status != ReviewPending:
			return ErrConflict
		}
		_, err = tx.Exec(ctx, `DELETE FROM review_requests WHERE id = $1`, reviewID)
		return err
	})
	if err != nil {
		return fmt.Errorf("revoke review: %w", err)
	}
	return nil
}

func lockReview(ctx context.Context, tx pgx.Tx, sessionID, reviewID int64) (ReviewRequest, error) {
	var r ReviewRequest
	err := tx.QueryRow(ctx, `
		SELECT id, session_id, requester_id, reviewer_id, status FROM review_requests
		WHERE id = $1 AND session_id = $2 FOR UPDATE`, reviewID, sessionID).
		Scan(&r.ID, &r.SessionID, &r.Requester.ID, &r.Reviewer.ID, &r.Status)
	return r, notFound(err)
}
