package store

import (
	"context"
	"encoding/json"
	"fmt"
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
