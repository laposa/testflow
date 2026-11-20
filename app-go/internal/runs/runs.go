// Package runs dispatches automated runs to GitHub Actions and keeps them
// in sync with GitHub (spec §8.4, §8.5; legacy:
// app/Actions/Github/DispatchSessionRun.php, FetchSessionWorkflowRuns.php,
// Console/Commands/SyncGithubRuns.php), and starts manual runs (spec §8.7).
package runs

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/laposa/testflow/app-go/internal/github"
	"github.com/laposa/testflow/app-go/internal/store"
)

// GitHub is the part of the GitHub App client runs needs (*github.Client).
type GitHub interface {
	DispatchWorkflow(ctx context.Context, repo string, workflowID int64, ref string, inputs map[string]string) error
	WorkflowRuns(ctx context.Context, repo string, workflowID int64, f github.RunsFilter) ([]github.WorkflowRun, error)
	RunJobs(ctx context.Context, repo string, runID int64) ([]github.Job, error)
	RunArtifacts(ctx context.Context, repo string, runID int64) ([]github.Artifact, error)
	DownloadArtifact(ctx context.Context, repo string, artifactID int64) ([]byte, error)
	DownloadRunLogs(ctx context.Context, repo string, runID int64) ([]byte, error)
}

// Store is the part of store.Store runs needs.
type Store interface {
	CreateRun(ctx context.Context, nr store.NewRun) (store.Run, error)
	AddActivity(ctx context.Context, a store.NewActivity) error
	PendingRuns(ctx context.Context, since time.Time) ([]store.PendingRun, error)
	MatchedGitHubRuns(ctx context.Context, since time.Time) (map[int64]bool, error)
	StaleRuns(ctx context.Context, before time.Time) ([]store.PendingRun, error)
	ServiceTests(ctx context.Context, serviceID int64) ([]store.Test, error)
	UpdateRun(ctx context.Context, u store.RunUpdate, a *store.NewActivity) error
	StartManualRun(ctx context.Context, svc store.Service, user store.User) (store.Run, error)
}

// TestFilter is the test_filter input of a service's workflow (spec §8.4):
// mobile gets the unique suite names joined with ",", web
// "tests/{suite}/{test}" joined with ",", api each test name up to its
// first "." joined with "|", and any other service an empty filter (Q3).
func TestFilter(svc store.Service) string {
	var parts []string
	seen := map[string]bool{}
	for _, suite := range svc.Suites {
		for _, t := range suite.Tests {
			switch svc.Name {
			case "mobile":
				if !seen[suite.Name] {
					seen[suite.Name] = true
					parts = append(parts, suite.Name)
				}
			case "web":
				parts = append(parts, "tests/"+suite.Name+"/"+t.Name)
			case "api":
				name, _, _ := strings.Cut(t.Name, ".")
				parts = append(parts, name)
			}
		}
	}
	if svc.Name == "api" {
		return strings.Join(parts, "|")
	}
	return strings.Join(parts, ",")
}

// DispatchedBody is the run_dispatched activity text: "{user} has executed:
// {service} with run ID {id}, …" (legacy: DispatchSessionRun).
func DispatchedBody(user string, started []Started) string {
	var b strings.Builder
	b.WriteString(user + " has executed: ")
	for i, s := range started {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(s.Service.DisplayName() + " with run ID " + strconv.FormatInt(s.Run.ID, 10))
	}
	return b.String()
}

// StatusChangedBody is the run_status_changed activity text: "Run {id} for
// {service} has changed status from {old} to {new}", with "with {p} passed
// and {f} failed tests" when the run has results (legacy:
// FetchSessionWorkflowRuns::logRunStatusChange).
func StatusChangedBody(runID int64, svc store.Service, old, status string, totals *store.RunTotals) string {
	body := "Run " + strconv.FormatInt(runID, 10) + " for " + svc.DisplayName() +
		" has changed status from " + old + " to " + status
	if totals != nil {
		body += " with " + strconv.Itoa(totals.Passed) + " passed and " + strconv.Itoa(totals.Failed) + " failed tests"
	}
	return body + "."
}

// githubStatuses are the conclusions and statuses stored as they are
// (service_runs.status accepts only §8.4 values).
var githubStatuses = map[string]bool{
	store.RunQueued: true, store.RunInProgress: true, store.RunSuccess: true, store.RunFailure: true,
	store.RunNeutral: true, store.RunCancelled: true, store.RunSkipped: true, store.RunTimedOut: true,
	store.RunActionRequired: true, store.RunStale: true,
}

// Status is a workflow run's status for service_runs: the conclusion if
// set, otherwise the status (spec §8.4). Values outside §8.4 are mapped: an
// unknown conclusion (e.g. startup_failure) is failure, and a run that has
// not started (waiting, requested, pending) is queued.
func Status(wr github.WorkflowRun) string {
	if wr.Conclusion != "" {
		if githubStatuses[wr.Conclusion] {
			return wr.Conclusion
		}
		return store.RunFailure
	}
	if wr.Status == "completed" {
		return store.RunFailure
	}
	if githubStatuses[wr.Status] {
		return wr.Status
	}
	return store.RunQueued
}
