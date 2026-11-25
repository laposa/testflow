package runs

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"

	"github.com/laposa/testflow/app-go/internal/store"
)

// Dispatcher starts automated runs (spec §8.4).
type Dispatcher struct {
	GitHub GitHub
	Store  Store
	Logger *slog.Logger // default slog.Default()
}

// Started is a run that was dispatched and stored.
type Started struct {
	Service store.Service
	Run     store.Run
}

// Failure is a service that could not be started.
type Failure struct {
	Service store.Service
	Err     error
}

// Result says what Dispatch did.
type Result struct {
	Started []Started // automated runs
	Manual  []Started // manual runs, filled in with the wizard (spec §8.7)
	Failed  []Failure
}

// ErrNoWorkflow is the failure of an automated service without a workflow.
var ErrNoWorkflow = errors.New("the service has no workflow")

// Dispatch starts a run of each selected service of ts, or of every one if
// serviceIDs is empty (legacy: "Run All Tests"). Each automated service
// gets a workflow_dispatch on its branch, then a service_runs row with
// status unknown, which the syncer matches to the GitHub run by time. One
// run_dispatched entry lists the started automated runs. Each manual
// service gets a manual run in progress, which finishes one still open
// (store.StartManualRun).
//
// A service GitHub refuses is reported in Result.Failed and the others are
// still started. The error is only for storage failures. The dispatches
// are not cancelled with ctx: once the first workflow runs, the rest of
// the batch and its activity entry should follow.
func (d *Dispatcher) Dispatch(ctx context.Context, ts store.TestSession, user store.User, serviceIDs []int64) (Result, error) {
	ctx = context.WithoutCancel(ctx)
	logger := d.Logger
	if logger == nil {
		logger = slog.Default()
	}
	var res Result
	for _, svc := range ts.Services {
		if len(serviceIDs) > 0 && !slices.Contains(serviceIDs, svc.ID) {
			continue
		}
		if svc.Type == store.RunManual {
			run, err := d.Store.StartManualRun(ctx, svc, user)
			if err != nil {
				d.logDispatched(ctx, ts.ID, user, res.Started, logger)
				return res, fmt.Errorf("start %s: %w", svc.DisplayName(), err)
			}
			logger.InfoContext(ctx, "manual run started", "session.id", ts.ID, "service.id", svc.ID, "run.id", run.ID)
			res.Manual = append(res.Manual, Started{svc, run})
			continue
		}
		if svc.WorkflowID == 0 {
			res.Failed = append(res.Failed, Failure{svc, ErrNoWorkflow})
			continue
		}
		inputs := map[string]string{"environment": ts.Environment, "test_filter": TestFilter(svc)}
		if err := d.GitHub.DispatchWorkflow(ctx, svc.RepositoryName, svc.WorkflowID, svc.Branch, inputs); err != nil {
			logger.WarnContext(ctx, "workflow dispatch failed", "session.id", ts.ID, "service.id", svc.ID, "err", err)
			res.Failed = append(res.Failed, Failure{svc, err})
			continue
		}
		// Inserted right after the dispatch: its created_at is what the
		// syncer matches GitHub's run by.
		run, err := d.Store.CreateRun(ctx, store.NewRun{ServiceID: svc.ID, Type: store.RunAutomated, Status: store.RunUnknown})
		if err != nil {
			// The workflow runs, but TestFlow will not track it.
			d.logDispatched(ctx, ts.ID, user, res.Started, logger)
			return res, fmt.Errorf("dispatch %s: %w", svc.DisplayName(), err)
		}
		logger.InfoContext(ctx, "run dispatched", "session.id", ts.ID, "service.id", svc.ID, "run.id", run.ID)
		res.Started = append(res.Started, Started{svc, run})
	}
	if err := d.logDispatched(ctx, ts.ID, user, res.Started, logger); err != nil {
		return res, err
	}
	return res, nil
}

func (d *Dispatcher) logDispatched(ctx context.Context, sessionID int64, user store.User, started []Started, logger *slog.Logger) error {
	if len(started) == 0 {
		return nil
	}
	err := d.Store.AddActivity(ctx, store.NewActivity{
		SessionID: sessionID,
		UserID:    user.ID,
		Type:      store.ActivityRunDispatched,
		Body:      DispatchedBody(user.Name, started),
	})
	if err != nil {
		logger.ErrorContext(ctx, "log run_dispatched failed", "session.id", sessionID, "err", err)
		return fmt.Errorf("log run_dispatched: %w", err)
	}
	return nil
}
