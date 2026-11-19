package runs

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/laposa/testflow/app-go/internal/github"
	"github.com/laposa/testflow/app-go/internal/junit"
	"github.com/laposa/testflow/app-go/internal/store"
	"github.com/laposa/testflow/app-go/internal/telemetry"
)

// MatchWindow is how far apart our run's created_at and GitHub's may be
// for them to match (spec §8.4, Q1).
const MatchWindow = 5 * time.Second

// ResultsArtifact is the name of the artifact holding the JUnit report.
const ResultsArtifact = "test-results"

// dispatchEvent is the event of runs started by TestFlow.
const dispatchEvent = "workflow_dispatch"

// maestroURL finds the Maestro Cloud Console link in a mobile run's log.
// It stops at whitespace, control characters (the log's ANSI colour codes
// start with ESC), quotes and angle brackets; maestroLink drops trailing
// punctuation.
var maestroURL = regexp.MustCompile(`https://console\.mobile\.dev/uploads[^\s\x00-\x1f\x7f"'<>]*`)

// maestroLink is the first Maestro Cloud Console link in log, or "".
func maestroLink(log string) string {
	return strings.TrimRight(maestroURL.FindString(log), ".,;:!?)]}")
}

// SessionCleaner deletes expired web sessions (auth.Sessions).
type SessionCleaner interface {
	DeleteExpired(ctx context.Context) (int64, error)
}

// SyncerConfig configures a Syncer. GitHub and Store are required.
type SyncerConfig struct {
	GitHub   GitHub
	Store    Store
	Sessions SessionCleaner // optional: expired web sessions are deleted each tick
	Logger   *slog.Logger   // default slog.Default()
	Now      func() time.Time
}

// Syncer brings automated runs up to date with GitHub (spec §8.5). It
// replaces legacy's Livewire polling and app:sync-github-runs. Only one
// sync runs at a time, so a "refresh now" never races the ticker.
type Syncer struct {
	gh       GitHub
	store    Store
	sessions SessionCleaner
	logger   *slog.Logger
	now      func() time.Time
	sem      chan struct{}
}

// NewSyncer returns a Syncer.
func NewSyncer(c SyncerConfig) *Syncer {
	s := &Syncer{gh: c.GitHub, store: c.Store, sessions: c.Sessions, logger: c.Logger, now: c.Now,
		sem: make(chan struct{}, 1)}
	if s.logger == nil {
		s.logger = slog.Default()
	}
	if s.now == nil {
		s.now = time.Now
	}
	return s
}

// Run syncs every interval until ctx is cancelled, starting straight away.
// Each tick is its own trace (spec §11).
func (s *Syncer) Run(ctx context.Context, interval time.Duration) {
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		s.Tick(telemetry.StartTrace(ctx))
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}

// Tick runs one round: it marks runs stale that the syncer gave up on,
// syncs every pending run and deletes expired web sessions. Failures are
// logged; the next tick tries again.
func (s *Syncer) Tick(ctx context.Context) {
	if !s.acquire(ctx) {
		return
	}
	defer s.release()
	now := s.now()
	s.markStale(ctx, now)
	if err := s.sync(ctx, now, 0); err != nil && ctx.Err() == nil {
		s.logger.WarnContext(ctx, "run sync failed", "err", err)
	}
	if s.sessions != nil {
		n, err := s.sessions.DeleteExpired(ctx)
		switch {
		case err != nil && ctx.Err() == nil:
			s.logger.WarnContext(ctx, "delete expired web sessions failed", "err", err)
		case n > 0:
			s.logger.DebugContext(ctx, "expired web sessions deleted", "count", n)
		}
	}
}

// SyncSession syncs the pending runs of one session now ("refresh now",
// spec §8.5). It waits for a sync in progress to finish, up to ctx's
// deadline.
func (s *Syncer) SyncSession(ctx context.Context, sessionID int64) error {
	if !s.acquire(ctx) {
		return ctx.Err()
	}
	defer s.release()
	return s.sync(ctx, s.now(), sessionID)
}

func (s *Syncer) acquire(ctx context.Context) bool {
	select {
	case s.sem <- struct{}{}:
		return true
	case <-ctx.Done():
		return false
	}
}

func (s *Syncer) release() { <-s.sem }

// markStale gives up on runs that never finished within RunSyncWindow and
// sets them to stale (Q2), logging the status change.
func (s *Syncer) markStale(ctx context.Context, now time.Time) {
	list, err := s.store.StaleRuns(ctx, now.Add(-store.RunSyncWindow))
	if err != nil {
		if ctx.Err() == nil {
			s.logger.WarnContext(ctx, "list stale runs failed", "err", err)
		}
		return
	}
	for _, p := range list {
		u := store.RunUpdate{ID: p.ID, OldStatus: p.Status, Status: store.RunStale}
		s.update(ctx, p, u, nil)
	}
}

type groupKey struct {
	repo     string
	workflow int64
	branch   string
}

// sync matches the pending runs to GitHub's workflow runs and applies what
// it finds. Workflow runs are fetched once per repository, workflow and
// branch. A non-zero sessionID syncs only the groups holding a run of that
// session, but each of them whole: matching only that session's runs could
// hand one of them another session's GitHub run.
func (s *Syncer) sync(ctx context.Context, now time.Time, sessionID int64) error {
	since := now.Add(-store.RunSyncWindow)
	pending, err := s.store.PendingRuns(ctx, since)
	if err != nil {
		return err
	}
	groups := map[groupKey][]store.PendingRun{}
	wanted := map[groupKey]bool{}
	var order []groupKey
	for _, p := range pending {
		if p.Service.WorkflowID == 0 {
			continue // cannot be matched; it goes stale
		}
		k := groupKey{p.Service.RepositoryName, p.Service.WorkflowID, p.Service.Branch}
		if _, ok := groups[k]; !ok {
			order = append(order, k)
		}
		groups[k] = append(groups[k], p)
		if sessionID == 0 || p.Service.SessionID == sessionID {
			wanted[k] = true
		}
	}
	if len(wanted) == 0 {
		return nil
	}
	// GitHub runs that finished runs hold are taken too. The window starts
	// a minute earlier, like the runs filter below.
	taken, err := s.store.MatchedGitHubRuns(ctx, since.Add(-time.Minute))
	if err != nil {
		return err
	}
	for _, k := range order {
		if !wanted[k] {
			continue
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		runs := groups[k]
		// Oldest first, so the filter covers every run of the group; the
		// extra minute allows for clock skew.
		ghRuns, err := s.gh.WorkflowRuns(ctx, k.repo, k.workflow, github.RunsFilter{
			Event: dispatchEvent, Branch: k.branch, Created: runs[0].CreatedAt.Add(-time.Minute),
		})
		if err != nil {
			s.logger.WarnContext(ctx, "list workflow runs failed", "repository", k.repo, "workflow.id", k.workflow, "err", err)
			continue
		}
		for _, m := range Match(runs, ghRuns, taken) {
			if err := s.syncRun(ctx, m.Pending, m.WorkflowRun); err != nil && ctx.Err() == nil {
				s.logger.WarnContext(ctx, "sync run failed", "run.id", m.Pending.ID, "github.run_id", m.WorkflowRun.ID, "err", err)
			}
		}
	}
	return nil
}

// Matched pairs one of our runs with its GitHub workflow run.
type Matched struct {
	Pending     store.PendingRun
	WorkflowRun github.WorkflowRun
}

// Match pairs pending runs with workflow runs (spec §8.4). A run already
// matched keeps its github_run_id. Otherwise it takes the workflow run
// created closest to it within ±MatchWindow that no other run has, so two
// dispatches of the same workflow a few seconds apart get one run each
// (legacy took the first in range for both). taken holds GitHub run ids
// that other runs, e.g. finished ones, already have.
func Match(pending []store.PendingRun, ghRuns []github.WorkflowRun, taken map[int64]bool) []Matched {
	byID := make(map[int64]github.WorkflowRun, len(ghRuns))
	for _, wr := range ghRuns {
		byID[wr.ID] = wr
	}
	claimed := make(map[int64]bool, len(taken))
	for id := range taken {
		claimed[id] = true
	}
	for _, p := range pending {
		if p.GitHubRunID != 0 {
			claimed[p.GitHubRunID] = true
		}
	}
	var out []Matched
	for _, p := range pending {
		if p.GitHubRunID != 0 {
			if wr, ok := byID[p.GitHubRunID]; ok {
				out = append(out, Matched{p, wr})
			}
			continue
		}
		best, bestDist := -1, MatchWindow+1
		for i, wr := range ghRuns {
			d := wr.CreatedAt.Sub(p.CreatedAt).Abs()
			if d <= MatchWindow && d < bestDist && !claimed[wr.ID] {
				best, bestDist = i, d
			}
		}
		if best >= 0 {
			claimed[ghRuns[best].ID] = true
			out = append(out, Matched{p, ghRuns[best]})
		}
	}
	return out
}

// syncRun applies one matched workflow run: its id, commit and start time
// and status, and once completed its results and logs. If a download fails
// nothing is stored, so the next tick retries with the status change still
// to log.
func (s *Syncer) syncRun(ctx context.Context, p store.PendingRun, wr github.WorkflowRun) error {
	jobs, err := s.gh.RunJobs(ctx, p.Service.RepositoryName, wr.ID)
	if err != nil {
		return err
	}
	u := store.RunUpdate{
		ID: p.ID, OldStatus: p.Status, Status: Status(wr),
		GitHubRunID: wr.ID, CommitSHA: wr.HeadSHA, StartedAt: minStarted(jobs),
	}
	if wr.Status == "completed" {
		if u.Completion, err = s.complete(ctx, p, wr, jobs); err != nil {
			return err
		}
	}
	if u.Completion == nil && u.Status == p.Status && u.GitHubRunID == p.GitHubRunID &&
		u.CommitSHA == p.CommitSHA && sameTime(u.StartedAt, p.StartedAt) {
		return nil
	}
	s.update(ctx, p, u, u.Completion)
	return nil
}

// update stores u with a run_status_changed entry if the status changed.
func (s *Syncer) update(ctx context.Context, p store.PendingRun, u store.RunUpdate, c *store.RunCompletion) {
	var a *store.NewActivity
	if u.Status != p.Status {
		var totals *store.RunTotals
		meta := map[string]any{store.MetaRunID: p.ID}
		if c != nil {
			totals = c.Totals
			if p.Service.Name == "mobile" {
				if url := maestroLink(c.RunLog); url != "" {
					meta[store.MetaMaestroURL] = url
				}
			}
		}
		a = &store.NewActivity{
			SessionID: p.Service.SessionID,
			Type:      store.ActivityRunStatusChanged,
			Body:      StatusChangedBody(p.ID, p.Service, p.Status, u.Status, totals),
			Metadata:  meta,
		}
	}
	err := s.store.UpdateRun(ctx, u, a)
	switch {
	case errors.Is(err, store.ErrConflict):
		s.logger.DebugContext(ctx, "run changed meanwhile, skipped", "run.id", p.ID)
	case err != nil:
		if ctx.Err() == nil {
			s.logger.WarnContext(ctx, "update run failed", "run.id", p.ID, "err", err)
		}
	case u.Status != p.Status:
		s.logger.InfoContext(ctx, "run status changed", "run.id", p.ID, "from", p.Status, "to", u.Status)
	}
}

// complete fetches a completed run's JUnit report and run log (spec §8.4).
func (s *Syncer) complete(ctx context.Context, p store.PendingRun, wr github.WorkflowRun, jobs []github.Job) (*store.RunCompletion, error) {
	repo := p.Service.RepositoryName
	c := &store.RunCompletion{FinishedAt: maxCompleted(jobs)}

	artifacts, err := s.gh.RunArtifacts(ctx, repo, wr.ID)
	if err != nil {
		return nil, err
	}
	for _, a := range artifacts {
		if a.Name != ResultsArtifact || a.Expired {
			continue
		}
		zipped, err := s.gh.DownloadArtifact(ctx, repo, a.ID)
		if err != nil {
			return nil, err
		}
		xml, err := firstXML(zipped)
		if errors.Is(err, errTooLarge) {
			// Retrying would not help: keep the run without results.
			s.logger.WarnContext(ctx, "JUnit report skipped", "run.id", p.ID, "err", err)
		} else if err != nil {
			return nil, err
		}
		if xml != nil {
			if err := s.results(ctx, p, xml, c); err != nil {
				return nil, err
			}
		}
		break
	}

	zipped, err := s.gh.DownloadRunLogs(ctx, repo, wr.ID)
	switch {
	case gone(err):
		// GitHub keeps logs for a limited time; store "" to stop asking.
	case err != nil:
		return nil, err
	default:
		if c.RunLog, err = largestFile(zipped); err != nil {
			return nil, err
		}
	}
	return c, nil
}

// results stores the raw report and, if it parses, the totals and the
// result of each of the service's tests that the report has (spec §8.6).
// A test the report does not mention gets no row and shows "not found".
func (s *Syncer) results(ctx context.Context, p store.PendingRun, xml []byte, c *store.RunCompletion) error {
	raw := text(xml)
	c.ResultLog = &raw
	rep, err := junit.Parse(xml)
	if err != nil {
		s.logger.WarnContext(ctx, "JUnit report does not parse", "run.id", p.ID, "err", err)
		return nil
	}
	t := rep.Totals()
	c.Totals = &store.RunTotals{Passed: t.Passed, Failed: t.Failed, Skipped: t.Skipped, Duration: t.Duration}
	tests, err := s.store.ServiceTests(ctx, p.Service.ID)
	if err != nil {
		return err
	}
	for _, test := range tests {
		tc, ok := rep.Find(test.Name)
		if !ok {
			continue
		}
		c.Results = append(c.Results, store.TestResult{
			TestID:         test.ID,
			Status:         string(tc.Status),
			Comment:        text([]byte(tc.Comment)),
			FailureMessage: text([]byte(tc.FailureMessage)),
			Duration:       tc.Time,
		})
	}
	return nil
}

// gone reports a download GitHub no longer has (404 or 410).
func gone(err error) bool {
	if errors.Is(err, github.ErrNotFound) {
		return true
	}
	apiErr, ok := errors.AsType[*github.APIError](err)
	return ok && apiErr.StatusCode == http.StatusGone
}

func minStarted(jobs []github.Job) *time.Time {
	var t *time.Time
	for _, j := range jobs {
		if j.StartedAt != nil && (t == nil || j.StartedAt.Before(*t)) {
			t = j.StartedAt
		}
	}
	return t
}

func maxCompleted(jobs []github.Job) *time.Time {
	var t *time.Time
	for _, j := range jobs {
		if j.CompletedAt != nil && (t == nil || j.CompletedAt.After(*t)) {
			t = j.CompletedAt
		}
	}
	return t
}

func sameTime(a, b *time.Time) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return a.Equal(*b)
}
