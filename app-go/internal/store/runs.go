package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// Run types (service_runs.type).
const (
	RunAutomated = "automated"
	RunManual    = "manual"
)

// Run statuses (service_runs.status, spec §8.4): GitHub's conclusion if
// set, otherwise its status, plus unknown (dispatched, not matched yet) and
// stale (given up on, Q2).
const (
	RunUnknown        = "unknown"
	RunQueued         = "queued"
	RunInProgress     = "in_progress"
	RunSuccess        = "success"
	RunFailure        = "failure"
	RunNeutral        = "neutral"
	RunCancelled      = "cancelled"
	RunSkipped        = "skipped"
	RunTimedOut       = "timed_out"
	RunActionRequired = "action_required"
	RunStale          = "stale"
)

// RunSyncWindow is how long the syncer keeps trying an automated run that
// has no run log yet; after that it is marked stale (spec §8.5, Q2).
const RunSyncWindow = 24 * time.Hour

// ErrConflict is returned when a row changed since it was read, e.g. a run
// whose status is no longer the one the syncer saw.
var ErrConflict = errors.New("conflict")

// Run is a service_runs row without its logs.
type Run struct {
	ID          int64
	ServiceID   int64
	Type        string
	Status      string
	HasResults  bool // result_log is set
	HasRunLog   bool // run_log is set
	Passed      int
	Failed      int
	Skipped     int
	Duration    float64 // seconds, from JUnit
	GitHubRunID int64   // 0 until matched
	CommitSHA   string
	StartedAt   *time.Time
	FinishedAt  *time.Time
	CreatedAt   time.Time
}

// Syncing reports whether the syncer still works on the run at now: an
// automated run without a run log, not stale and inside RunSyncWindow.
// The session page polls the run list while any run is syncing (§8.5).
func (r Run) Syncing(now time.Time) bool {
	return r.Type == RunAutomated && !r.HasRunLog && r.Status != RunStale &&
		now.Sub(r.CreatedAt) < RunSyncWindow
}

// RunDetail is a run with its logs and per-test results.
type RunDetail struct {
	Run
	ResultLog string               // raw JUnit XML; "" if none
	RunLog    string               // the largest file of the GitHub run log zip
	Results   map[int64]TestResult // by suite_tests id
}

// TestResult is a run_test_results row (spec §6.8).
type TestResult struct {
	TestID         int64
	Status         string // pass, fail, skipped, error or notRun
	Comment        string
	FailureMessage string
	Duration       float64 // seconds
}

const runColumns = `r.id, r.service_id, r.type, r.status, r.result_log IS NOT NULL, r.run_log IS NOT NULL,
	COALESCE(r.passed, 0), COALESCE(r.failed, 0), COALESCE(r.skipped, 0), COALESCE(r.duration, 0),
	COALESCE(r.github_run_id, 0), COALESCE(r.commit_sha, ''), r.started_at, r.finished_at, r.created_at`

func scanRun(row interface{ Scan(...any) error }, extra ...any) (Run, error) {
	var r Run
	dest := append([]any{&r.ID, &r.ServiceID, &r.Type, &r.Status, &r.HasResults, &r.HasRunLog,
		&r.Passed, &r.Failed, &r.Skipped, &r.Duration, &r.GitHubRunID, &r.CommitSHA,
		&r.StartedAt, &r.FinishedAt, &r.CreatedAt}, extra...)
	err := row.Scan(dest...)
	return r, err
}

// NewRun is a service_runs row to insert.
type NewRun struct {
	ServiceID int64
	Type      string // automated or manual
	Status    string
}

// CreateRun inserts a run and returns it.
func (s *Store) CreateRun(ctx context.Context, nr NewRun) (Run, error) {
	r, err := scanRun(s.db.QueryRow(ctx, `
		INSERT INTO service_runs AS r (service_id, type, status) VALUES ($1, $2, $3)
		RETURNING `+runColumns, nr.ServiceID, nr.Type, nr.Status))
	if err != nil {
		return Run{}, fmt.Errorf("create run: %w", err)
	}
	return r, nil
}

// AddActivity logs an activity entry, e.g. run_dispatched (spec §6.9).
func (s *Store) AddActivity(ctx context.Context, a NewActivity) error {
	return insertActivity(ctx, s.db, a)
}

// SessionRuns returns the runs of a session's services, by service and
// newest first within a service.
func (s *Store) SessionRuns(ctx context.Context, sessionID int64) ([]Run, error) {
	rows, err := s.db.Query(ctx, `
		SELECT `+runColumns+`
		FROM service_runs r JOIN session_services ss ON ss.id = r.service_id
		WHERE ss.session_id = $1
		ORDER BY r.service_id, r.id DESC`, sessionID)
	if err != nil {
		return nil, fmt.Errorf("session runs: %w", err)
	}
	list, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (Run, error) { return scanRun(row) })
	if err != nil {
		return nil, fmt.Errorf("session runs: %w", err)
	}
	return list, nil
}

// RunDetail returns a run of the session with its logs and test results,
// or ErrNotFound (also for a run of another session).
func (s *Store) RunDetail(ctx context.Context, sessionID, runID int64) (RunDetail, error) {
	var d RunDetail
	var err error
	d.Run, err = scanRun(s.db.QueryRow(ctx, `
		SELECT `+runColumns+`, COALESCE(r.result_log, ''), COALESCE(r.run_log, '')
		FROM service_runs r JOIN session_services ss ON ss.id = r.service_id
		WHERE r.id = $1 AND ss.session_id = $2`, runID, sessionID), &d.ResultLog, &d.RunLog)
	if err != nil {
		return RunDetail{}, notFound(err)
	}
	rows, err := s.db.Query(ctx, `
		SELECT test_id, status, COALESCE(comment, ''), COALESCE(failure_message, ''), COALESCE(duration, 0)
		FROM run_test_results WHERE run_id = $1`, runID)
	if err != nil {
		return RunDetail{}, fmt.Errorf("run test results: %w", err)
	}
	results, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (TestResult, error) {
		var t TestResult
		err := row.Scan(&t.TestID, &t.Status, &t.Comment, &t.FailureMessage, &t.Duration)
		return t, err
	})
	if err != nil {
		return RunDetail{}, fmt.Errorf("run test results: %w", err)
	}
	d.Results = make(map[int64]TestResult, len(results))
	for _, t := range results {
		d.Results[t.TestID] = t
	}
	return d, nil
}

// PendingRun is an automated run the syncer works on, with its service.
type PendingRun struct {
	Run
	Service Service // without suites
}

// PendingRuns returns the automated runs without a run log that are not
// stale and were created at or after since, oldest first (spec §8.5). A
// non-zero sessionID limits them to that session.
func (s *Store) PendingRuns(ctx context.Context, since time.Time, sessionID int64) ([]PendingRun, error) {
	return s.unsyncedRuns(ctx, `r.status <> 'stale' AND r.created_at >= $1 AND ($2::bigint = 0 OR ss.session_id = $2)`,
		since, sessionID)
}

// StaleRuns returns the automated runs without a run log that were created
// before before and never finished (unknown, queued or in_progress): the
// runs to mark stale (Q2).
func (s *Store) StaleRuns(ctx context.Context, before time.Time) ([]PendingRun, error) {
	return s.unsyncedRuns(ctx, `r.status IN ('unknown', 'queued', 'in_progress') AND r.created_at < $1`, before)
}

func (s *Store) unsyncedRuns(ctx context.Context, where string, args ...any) ([]PendingRun, error) {
	rows, err := s.db.Query(ctx, `
		SELECT `+runColumns+`,
		       ss.id, ss.session_id, ss.type, ss.name, ss.path, ss.repository_id, ss.repository_name,
		       COALESCE(ss.workflow_id, 0), ss.commit_sha, ss.branch
		FROM service_runs r JOIN session_services ss ON ss.id = r.service_id
		WHERE r.type = 'automated' AND r.run_log IS NULL AND `+where+`
		ORDER BY r.id`, args...)
	if err != nil {
		return nil, fmt.Errorf("unsynced runs: %w", err)
	}
	list, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (PendingRun, error) {
		var p PendingRun
		svc := &p.Service
		r, err := scanRun(row, &svc.ID, &svc.SessionID, &svc.Type, &svc.Name, &svc.Path, &svc.RepositoryID,
			&svc.RepositoryName, &svc.WorkflowID, &svc.CommitSHA, &svc.Branch)
		p.Run = r
		return p, err
	})
	if err != nil {
		return nil, fmt.Errorf("unsynced runs: %w", err)
	}
	return list, nil
}

// ServiceTests returns the tests of a service, in suite and test order.
func (s *Store) ServiceTests(ctx context.Context, serviceID int64) ([]Test, error) {
	rows, err := s.db.Query(ctx, `
		SELECT t.id, t.name, t.path
		FROM suite_tests t JOIN service_suites su ON su.id = t.suite_id
		WHERE su.service_id = $1
		ORDER BY su.id, t.id`, serviceID)
	if err != nil {
		return nil, fmt.Errorf("service tests: %w", err)
	}
	list, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (Test, error) {
		var t Test
		err := row.Scan(&t.ID, &t.Name, &t.Path)
		return t, err
	})
	if err != nil {
		return nil, fmt.Errorf("service tests: %w", err)
	}
	return list, nil
}

// RunUpdate is what one sync learned about a run (spec §8.4). Zero fields
// keep the stored value.
type RunUpdate struct {
	ID          int64
	OldStatus   string // the status the syncer read; the update fails with ErrConflict if it changed
	Status      string
	GitHubRunID int64
	CommitSHA   string
	StartedAt   *time.Time
	Completion  *RunCompletion // set once GitHub reports the run completed
}

// RunCompletion is the data of a completed run.
type RunCompletion struct {
	FinishedAt *time.Time
	ResultLog  *string    // nil: the run has no test-results artifact
	Totals     *RunTotals // nil: no result log, or it did not parse
	Results    []TestResult
	RunLog     string // "" if GitHub has no logs (any more); never NULL afterwards
}

// RunTotals are the counts of a run's JUnit report.
type RunTotals struct {
	Passed, Failed, Skipped int
	Duration                float64
}

// UpdateRun applies a sync in one transaction: the run's fields, on
// completion its logs, totals and test results (replacing any stored), and
// the activity entry a (if not nil). It returns ErrConflict, changing
// nothing, if the run's status is no longer u.OldStatus.
func (s *Store) UpdateRun(ctx context.Context, u RunUpdate, a *NewActivity) error {
	err := s.inTx(ctx, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `
			UPDATE service_runs SET
			    status = $2,
			    github_run_id = COALESCE(NULLIF($3::bigint, 0), github_run_id),
			    commit_sha = COALESCE(NULLIF($4::text, ''), commit_sha),
			    started_at = COALESCE($5, started_at)
			WHERE id = $1 AND status = $6`,
			u.ID, u.Status, u.GitHubRunID, u.CommitSHA, u.StartedAt, u.OldStatus)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return ErrConflict
		}
		if c := u.Completion; c != nil {
			if err := completeRun(ctx, tx, u.ID, c); err != nil {
				return err
			}
		}
		if a != nil {
			return insertActivity(ctx, tx, *a)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("update run %d: %w", u.ID, err)
	}
	return nil
}

func completeRun(ctx context.Context, tx pgx.Tx, runID int64, c *RunCompletion) error {
	var passed, failed, skipped *int
	var duration *float64
	if t := c.Totals; t != nil {
		passed, failed, skipped, duration = &t.Passed, &t.Failed, &t.Skipped, &t.Duration
	}
	if _, err := tx.Exec(ctx, `
		UPDATE service_runs SET finished_at = $2, result_log = $3, run_log = $4,
		    passed = $5, failed = $6, skipped = $7, duration = $8
		WHERE id = $1`,
		runID, c.FinishedAt, c.ResultLog, c.RunLog, passed, failed, skipped, duration); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM run_test_results WHERE run_id = $1`, runID); err != nil {
		return err
	}
	if len(c.Results) == 0 {
		return nil
	}
	n := len(c.Results)
	ids, statuses, comments, messages := make([]int64, n), make([]string, n), make([]string, n), make([]string, n)
	durations := make([]float64, n)
	for i, r := range c.Results {
		ids[i], statuses[i], comments[i], messages[i], durations[i] = r.TestID, r.Status, r.Comment, r.FailureMessage, r.Duration
	}
	_, err := tx.Exec(ctx, `
		INSERT INTO run_test_results (run_id, test_id, status, comment, failure_message, duration)
		SELECT $1, t.test_id, t.status, NULLIF(t.comment, ''), NULLIF(t.message, ''), t.duration
		FROM unnest($2::bigint[], $3::text[], $4::text[], $5::text[], $6::float8[])
		     AS t(test_id, status, comment, message, duration)`,
		runID, ids, statuses, comments, messages, durations)
	return err
}
