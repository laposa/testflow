package store

import (
	"context"
	"fmt"
	"strconv"

	"github.com/jackc/pgx/v5"
)

// Manual run results the tester can choose (spec §8.7).
const (
	ResultPass    = "pass"
	ResultFail    = "fail"
	ResultSkipped = "skipped"
)

// ManualRunStartedBody is the manual_run_started activity text (spec §8.7).
func ManualRunStartedBody(runID int64, svc Service, user string) string {
	return "Manual run " + strconv.FormatInt(runID, 10) + " for " + svc.DisplayName() + " started by " + user
}

// ManualRunFinishedBody is the manual_run_finished activity text (spec §8.7).
func ManualRunFinishedBody(runID int64, svc Service, passed, failed int) string {
	return "Finished run " + strconv.FormatInt(runID, 10) + " for " + svc.DisplayName() + " with " +
		strconv.Itoa(passed) + " passed and " + strconv.Itoa(failed) + " failed tests"
}

// StartManualRun starts a manual run of svc for user (spec §8.7; legacy:
// CreateManualSessionRun and ListRuns::createRun). A manual run of svc
// still in progress is finished first, as FinishManualRun would. The new
// run is in_progress from now, at the session's commit, and
// manual_run_started is logged, all in one transaction.
func (s *Store) StartManualRun(ctx context.Context, svc Service, user User) (Run, error) {
	var run Run
	err := s.inTx(ctx, func(tx pgx.Tx) error {
		// The service row serialises starts, so two at once cannot both
		// leave a run in progress.
		if _, err := tx.Exec(ctx, `SELECT 1 FROM session_services WHERE id = $1 FOR UPDATE`, svc.ID); err != nil {
			return err
		}
		rows, err := tx.Query(ctx, `
			SELECT id FROM service_runs
			WHERE service_id = $1 AND type = 'manual' AND status = 'in_progress'
			ORDER BY id FOR UPDATE`, svc.ID)
		if err != nil {
			return err
		}
		open, err := pgx.CollectRows(rows, pgx.RowTo[int64])
		if err != nil {
			return err
		}
		for _, id := range open {
			if _, err := finishManual(ctx, tx, id, user.ID); err != nil {
				return err
			}
		}
		run, err = scanRun(tx.QueryRow(ctx, `
			INSERT INTO service_runs AS r (service_id, type, status, started_at, commit_sha)
			VALUES ($1, 'manual', 'in_progress', now(), NULLIF($2, ''))
			RETURNING `+runColumns, svc.ID, svc.CommitSHA))
		if err != nil {
			return err
		}
		return insertActivity(ctx, tx, NewActivity{
			SessionID: svc.SessionID, UserID: user.ID, Type: ActivityManualRunStarted,
			Body:     ManualRunStartedBody(run.ID, svc, user.Name),
			Metadata: map[string]any{MetaRunID: run.ID},
		})
	})
	if err != nil {
		return Run{}, fmt.Errorf("start manual run: %w", err)
	}
	return run, nil
}

// SaveManualResult stores the tester's result for one test of a manual run
// in progress, replacing an earlier one, and recounts the run (spec §8.7).
// It is ErrConflict if the run is not a manual run in progress, and
// ErrNotFound if the test is not one of the run's service.
func (s *Store) SaveManualResult(ctx context.Context, runID int64, r TestResult) error {
	switch r.Status {
	case ResultPass, ResultFail, ResultSkipped:
	default:
		return fmt.Errorf("save manual result: invalid status %q", r.Status)
	}
	err := s.inTx(ctx, func(tx pgx.Tx) error {
		serviceID, err := lockManualRun(ctx, tx, runID)
		if err != nil {
			return err
		}
		tag, err := tx.Exec(ctx, `
			INSERT INTO run_test_results (run_id, test_id, status, comment)
			SELECT $1, t.id, $3, NULLIF($4, '')
			FROM suite_tests t JOIN service_suites su ON su.id = t.suite_id
			WHERE t.id = $2 AND su.service_id = $5
			ON CONFLICT (run_id, test_id) DO UPDATE SET status = EXCLUDED.status, comment = EXCLUDED.comment`,
			runID, r.TestID, r.Status, r.Comment, serviceID)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return ErrNotFound
		}
		_, err = recount(ctx, tx, runID, "")
		return err
	})
	if err != nil {
		return fmt.Errorf("save manual result: %w", err)
	}
	return nil
}

// FinishManualRun finishes a manual run in progress (spec §8.7; legacy:
// ManualTestRun::markRunAsFinished): tests without a result are skipped,
// the run is success from now, and manual_run_finished is logged by userID.
// It is ErrConflict if the run is not a manual run in progress.
func (s *Store) FinishManualRun(ctx context.Context, runID, userID int64) (RunTotals, error) {
	var totals RunTotals
	err := s.inTx(ctx, func(tx pgx.Tx) error {
		if _, err := lockManualRun(ctx, tx, runID); err != nil {
			return err
		}
		var err error
		totals, err = finishManual(ctx, tx, runID, userID)
		return err
	})
	if err != nil {
		return RunTotals{}, fmt.Errorf("finish manual run: %w", err)
	}
	return totals, nil
}

// lockManualRun locks a manual run in progress and returns its service.
func lockManualRun(ctx context.Context, tx pgx.Tx, runID int64) (int64, error) {
	var serviceID int64
	var typ, status string
	err := tx.QueryRow(ctx, `SELECT service_id, type, status FROM service_runs WHERE id = $1 FOR UPDATE`, runID).
		Scan(&serviceID, &typ, &status)
	if err != nil {
		return 0, notFound(err)
	}
	if typ != RunManual || status != RunInProgress {
		return 0, ErrConflict
	}
	return serviceID, nil
}

// finishManual finishes a locked manual run in progress.
func finishManual(ctx context.Context, tx pgx.Tx, runID, userID int64) (RunTotals, error) {
	var svc Service
	err := tx.QueryRow(ctx, `
		SELECT ss.id, ss.session_id, ss.type, ss.name, ss.repository_name
		FROM service_runs r JOIN session_services ss ON ss.id = r.service_id WHERE r.id = $1`, runID).
		Scan(&svc.ID, &svc.SessionID, &svc.Type, &svc.Name, &svc.RepositoryName)
	if err != nil {
		return RunTotals{}, err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO run_test_results (run_id, test_id, status)
		SELECT $1, t.id, 'skipped'
		FROM suite_tests t JOIN service_suites su ON su.id = t.suite_id
		WHERE su.service_id = $2
		ON CONFLICT (run_id, test_id) DO NOTHING`, runID, svc.ID); err != nil {
		return RunTotals{}, err
	}
	totals, err := recount(ctx, tx, runID, `, status = 'success', finished_at = now()`)
	if err != nil {
		return RunTotals{}, err
	}
	err = insertActivity(ctx, tx, NewActivity{
		SessionID: svc.SessionID, UserID: userID, Type: ActivityManualRunFinished,
		Body:     ManualRunFinishedBody(runID, svc, totals.Passed, totals.Failed),
		Metadata: map[string]any{MetaRunID: runID},
	})
	return totals, err
}

// recount sets a manual run's counts from its results, plus any extra SET
// clauses (a fixed string, never user input).
func recount(ctx context.Context, tx pgx.Tx, runID int64, extra string) (RunTotals, error) {
	var t RunTotals
	err := tx.QueryRow(ctx, `
		UPDATE service_runs r SET passed = c.passed, failed = c.failed, skipped = c.skipped`+extra+`
		FROM (SELECT count(*) FILTER (WHERE status = 'pass') AS passed,
		             count(*) FILTER (WHERE status IN ('fail', 'error')) AS failed,
		             count(*) FILTER (WHERE status = 'skipped') AS skipped
		      FROM run_test_results WHERE run_id = $1) c
		WHERE r.id = $1
		RETURNING r.passed, r.failed, r.skipped`, runID).Scan(&t.Passed, &t.Failed, &t.Skipped)
	return t, err
}
