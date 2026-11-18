package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// ActivityType is activity_log.type (spec §6.9; legacy:
// app/Enums/SessionActivityType.php).
type ActivityType string

const (
	ActivitySessionCreated    ActivityType = "session_created"
	ActivityRunDispatched     ActivityType = "run_dispatched"
	ActivityRunStatusChanged  ActivityType = "run_status_changed"
	ActivityManualRunStarted  ActivityType = "manual_run_started"
	ActivityManualRunFinished ActivityType = "manual_run_finished"
	ActivityReviewRequested   ActivityType = "review_requested"
	ActivityReviewApproved    ActivityType = "review_approved"
	ActivityReviewRejected    ActivityType = "review_rejected"
	ActivityComment           ActivityType = "comment"
)

// NewActivity is an activity_log row to insert. Body is plain text (D13).
type NewActivity struct {
	SessionID int64
	UserID    int64 // 0 means the system, e.g. the syncer
	Type      ActivityType
	Body      string
	Metadata  map[string]any // nil is stored as {}
}

func insertActivity(ctx context.Context, db DB, a NewActivity) error {
	meta := []byte("{}")
	if len(a.Metadata) > 0 {
		var err error
		if meta, err = json.Marshal(a.Metadata); err != nil {
			return fmt.Errorf("insert activity: encode metadata: %w", err)
		}
	}
	_, err := db.Exec(ctx, `
		INSERT INTO activity_log (session_id, user_id, type, body, metadata)
		VALUES ($1, NULLIF($2::bigint, 0), $3, $4, $5)`,
		a.SessionID, a.UserID, string(a.Type), a.Body, meta)
	if err != nil {
		return fmt.Errorf("insert activity: %w", err)
	}
	return nil
}

// Activity is an activity_log row with its user's name.
type Activity struct {
	ID        int64
	UserID    int64  // 0 for the system
	UserName  string // "" for the system
	Type      ActivityType
	Body      string // plain text (D13)
	CreatedAt time.Time
}

// SessionActivity returns the activity of a session, newest first (spec
// §8.9). Entries written in the same transaction share created_at, so the
// id decides.
func (s *Store) SessionActivity(ctx context.Context, sessionID int64) ([]Activity, error) {
	rows, err := s.db.Query(ctx, `
		SELECT a.id, COALESCE(a.user_id, 0), COALESCE(u.name, ''), a.type, a.body, a.created_at
		FROM activity_log a
		LEFT JOIN users u ON u.id = a.user_id
		WHERE a.session_id = $1
		ORDER BY a.id DESC`, sessionID)
	if err != nil {
		return nil, fmt.Errorf("session activity: %w", err)
	}
	list, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (Activity, error) {
		var a Activity
		err := row.Scan(&a.ID, &a.UserID, &a.UserName, &a.Type, &a.Body, &a.CreatedAt)
		return a, err
	})
	if err != nil {
		return nil, fmt.Errorf("session activity: %w", err)
	}
	return list, nil
}

// AddComment logs a comment by userID on a session (spec §8.9; legacy:
// Livewire/Comments/CreateComment). It returns ErrNotFound if the session
// does not exist. The body is validated by the caller.
func (s *Store) AddComment(ctx context.Context, sessionID, userID int64, body string) error {
	err := insertActivity(ctx, s.db, NewActivity{SessionID: sessionID, UserID: userID, Type: ActivityComment, Body: body})
	if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok &&
		pgErr.Code == "23503" && pgErr.ConstraintName == "activity_log_session_id_fkey" {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("add comment: %w", err)
	}
	return nil
}
