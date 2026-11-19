package runs

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/laposa/testflow/app-go/internal/github"
	"github.com/laposa/testflow/app-go/internal/store"
)

var t0 = time.Date(2026, 11, 19, 10, 0, 0, 0, time.UTC)

func ptr[T any](v T) *T { return &v }

// fakeGitHub answers from maps keyed by run or artifact id.
type fakeGitHub struct {
	mu          sync.Mutex
	dispatched  []dispatch
	dispatchErr map[string]error // by repository
	runs        []github.WorkflowRun
	runsErr     error
	filters     []github.RunsFilter
	jobs        map[int64][]github.Job
	artifacts   map[int64][]github.Artifact
	artifactZip map[int64][]byte
	logs        map[int64][]byte
	logsErr     error
}

type dispatch struct {
	repo     string
	workflow int64
	ref      string
	inputs   map[string]string
}

func (f *fakeGitHub) DispatchWorkflow(_ context.Context, repo string, workflowID int64, ref string, inputs map[string]string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err := f.dispatchErr[repo]; err != nil {
		return err
	}
	f.dispatched = append(f.dispatched, dispatch{repo, workflowID, ref, inputs})
	return nil
}

func (f *fakeGitHub) WorkflowRuns(_ context.Context, _ string, _ int64, flt github.RunsFilter) ([]github.WorkflowRun, error) {
	f.filters = append(f.filters, flt)
	return f.runs, f.runsErr
}

func (f *fakeGitHub) RunJobs(_ context.Context, _ string, runID int64) ([]github.Job, error) {
	return f.jobs[runID], nil
}

func (f *fakeGitHub) RunArtifacts(_ context.Context, _ string, runID int64) ([]github.Artifact, error) {
	return f.artifacts[runID], nil
}

func (f *fakeGitHub) DownloadArtifact(_ context.Context, _ string, id int64) ([]byte, error) {
	b, ok := f.artifactZip[id]
	if !ok {
		return nil, errors.New("artifact download failed")
	}
	return b, nil
}

func (f *fakeGitHub) DownloadRunLogs(_ context.Context, _ string, runID int64) ([]byte, error) {
	if f.logsErr != nil {
		return nil, f.logsErr
	}
	return f.logs[runID], nil
}

// fakeStore records writes and returns the configured runs.
type fakeStore struct {
	mu        sync.Mutex
	nextID    int64
	created   []store.NewRun
	activity  []store.NewActivity
	pending   []store.PendingRun
	stale     []store.PendingRun
	tests     map[int64][]store.Test
	updates   []store.RunUpdate
	updateErr error
	since     time.Time
	taken     map[int64]bool // GitHub run ids of runs in the store
	takenFrom time.Time
}

func (f *fakeStore) CreateRun(_ context.Context, nr store.NewRun) (store.Run, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.nextID++
	f.created = append(f.created, nr)
	return store.Run{ID: 100 + f.nextID, ServiceID: nr.ServiceID, Type: nr.Type, Status: nr.Status}, nil
}

func (f *fakeStore) AddActivity(_ context.Context, a store.NewActivity) error {
	f.activity = append(f.activity, a)
	return nil
}

func (f *fakeStore) PendingRuns(_ context.Context, since time.Time) ([]store.PendingRun, error) {
	f.since = since
	return f.pending, nil
}

func (f *fakeStore) MatchedGitHubRuns(_ context.Context, since time.Time) (map[int64]bool, error) {
	f.takenFrom = since
	return f.taken, nil
}

func (f *fakeStore) StaleRuns(context.Context, time.Time) ([]store.PendingRun, error) {
	return f.stale, nil
}

func (f *fakeStore) ServiceTests(_ context.Context, serviceID int64) ([]store.Test, error) {
	return f.tests[serviceID], nil
}

func (f *fakeStore) UpdateRun(_ context.Context, u store.RunUpdate, a *store.NewActivity) error {
	if f.updateErr != nil {
		return f.updateErr
	}
	f.updates = append(f.updates, u)
	if a != nil {
		f.activity = append(f.activity, *a)
	}
	return nil
}

func quietLogger() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

func service(id int64, name string, suites ...store.Suite) store.Service {
	return store.Service{ID: id, SessionID: 1, Type: store.RunAutomated, Name: name, RepositoryName: "laposa/shop",
		WorkflowID: 55, Branch: "master", Suites: suites}
}

func TestTestFilter(t *testing.T) {
	suites := []store.Suite{
		{Name: "checkout", Tests: []store.Test{{Name: "pay.cy.js"}, {Name: "refund.cy.js"}}},
		{Name: "login", Tests: []store.Test{{Name: "sso.v2.cy.js"}}},
	}
	for name, want := range map[string]string{
		"mobile": "checkout,login",
		"web":    "tests/checkout/pay.cy.js,tests/checkout/refund.cy.js,tests/login/sso.v2.cy.js",
		"api":    "pay|refund|sso",
		"other":  "",
	} {
		if got := TestFilter(service(1, name, suites...)); got != want {
			t.Errorf("%s: TestFilter = %q, want %q", name, got, want)
		}
	}
}

func TestStatus(t *testing.T) {
	for _, tt := range []struct{ status, conclusion, want string }{
		{"queued", "", "queued"},
		{"in_progress", "", "in_progress"},
		{"waiting", "", "queued"},
		{"requested", "", "queued"},
		{"completed", "success", "success"},
		{"completed", "timed_out", "timed_out"},
		{"completed", "startup_failure", "failure"},
		{"completed", "", "failure"},
	} {
		if got := Status(github.WorkflowRun{Status: tt.status, Conclusion: tt.conclusion}); got != tt.want {
			t.Errorf("Status(%s, %s) = %q, want %q", tt.status, tt.conclusion, got, tt.want)
		}
	}
}

func pendingRun(id int64, created time.Time, ghID int64) store.PendingRun {
	return store.PendingRun{
		Run:     store.Run{ID: id, Type: store.RunAutomated, Status: store.RunUnknown, CreatedAt: created, GitHubRunID: ghID},
		Service: service(10, "web"),
	}
}

func TestMatch(t *testing.T) {
	gh := []github.WorkflowRun{
		{ID: 1, CreatedAt: t0.Add(1 * time.Second)},
		{ID: 2, CreatedAt: t0.Add(3 * time.Second)},
		{ID: 3, CreatedAt: t0.Add(30 * time.Second)},
		{ID: 4, CreatedAt: t0.Add(-time.Hour)},
	}
	pending := []store.PendingRun{
		pendingRun(1, t0, 0),                           // closest is GitHub run 1
		pendingRun(2, t0.Add(500*time.Millisecond), 0), // 1 is taken, so 2
		pendingRun(3, t0.Add(10*time.Second), 0),       // nothing within 5 s
		pendingRun(4, t0.Add(-2*time.Hour), 4),         // already matched by id
		pendingRun(5, t0.Add(-time.Hour), 0),           // 4 is claimed by run 4
		pendingRun(6, t0.Add(29*time.Second), 0),       // 3 belongs to a finished run
	}
	got := Match(pending, gh, map[int64]bool{3: true})
	want := map[int64]int64{1: 1, 2: 2, 4: 4}
	if len(got) != len(want) {
		t.Fatalf("matched %d runs, want %d: %+v", len(got), len(want), got)
	}
	for _, m := range got {
		if want[m.Pending.ID] != m.WorkflowRun.ID {
			t.Errorf("run %d matched GitHub run %d, want %d", m.Pending.ID, m.WorkflowRun.ID, want[m.Pending.ID])
		}
	}
}

func TestDispatch(t *testing.T) {
	gh := &fakeGitHub{dispatchErr: map[string]error{"laposa/broken": errors.New("422 workflow has no workflow_dispatch")}}
	st := &fakeStore{}
	web := service(1, "web", store.Suite{Name: "checkout", Tests: []store.Test{{Name: "pay.cy.js"}}})
	api := service(2, "api", store.Suite{Name: "orders", Tests: []store.Test{{Name: "list.json"}}})
	broken := service(3, "web")
	broken.RepositoryName = "laposa/broken"
	noWorkflow := service(4, "mobile")
	noWorkflow.WorkflowID = 0
	manual := service(5, "web")
	manual.Type = store.RunManual
	ts := store.TestSession{ID: 1, Environment: "beta", Services: []store.Service{web, api, broken, noWorkflow, manual}}
	ada := store.User{ID: 7, Name: "Ada"}

	d := &Dispatcher{GitHub: gh, Store: st, Logger: quietLogger()}
	res, err := d.Dispatch(t.Context(), ts, ada, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Started) != 2 || len(res.Failed) != 2 || res.Manual != 1 {
		t.Fatalf("result = %+v", res)
	}
	if !errors.Is(res.Failed[1].Err, ErrNoWorkflow) {
		t.Errorf("failure = %v, want ErrNoWorkflow", res.Failed[1].Err)
	}
	if got := gh.dispatched[0]; got.repo != "laposa/shop" || got.workflow != 55 || got.ref != "master" ||
		got.inputs["environment"] != "beta" || got.inputs["test_filter"] != "tests/checkout/pay.cy.js" {
		t.Errorf("dispatch = %+v", got)
	}
	if st.created[0] != (store.NewRun{ServiceID: 1, Type: "automated", Status: "unknown"}) {
		t.Errorf("created run = %+v", st.created[0])
	}
	if len(st.activity) != 1 {
		t.Fatalf("activity = %+v", st.activity)
	}
	a := st.activity[0]
	if a.Type != store.ActivityRunDispatched || a.UserID != 7 ||
		a.Body != "Ada has executed: shop/web (automated) with run ID 101, shop/api (automated) with run ID 102" {
		t.Errorf("activity = %+v", a)
	}

	// Only the selected service; nothing started means no activity.
	st2 := &fakeStore{}
	d.Store = st2
	if res, err := d.Dispatch(t.Context(), ts, ada, []int64{3}); err != nil || len(res.Started) != 0 || len(res.Failed) != 1 {
		t.Errorf("selected broken = %+v, %v", res, err)
	}
	if len(st2.activity) != 0 {
		t.Errorf("activity without started runs: %+v", st2.activity)
	}
}

func zipOf(t *testing.T, files ...string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for i := 0; i < len(files); i += 2 {
		w, err := zw.Create(files[i])
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(files[i+1])); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

const report = `<testsuites><testsuite name="checkout" tests="3" failures="1" time="4.5">
<testcase name="Pay" classname="pay" time="1.5"/>
<testcase name="Refund" classname="refund" time="3"><failure message="boom"/><system-out>see video</system-out></testcase>
<testcase name="Other" classname="other"/>
</testsuite></testsuites>`

func syncEnv(t *testing.T, svcName string) (*Syncer, *fakeGitHub, *fakeStore) {
	t.Helper()
	gh := &fakeGitHub{
		runs: []github.WorkflowRun{{ID: 900, Status: "completed", Conclusion: "failure", HeadSHA: "sha1", CreatedAt: t0.Add(2 * time.Second)}},
		jobs: map[int64][]github.Job{900: {
			{StartedAt: ptr(t0.Add(20 * time.Second)), CompletedAt: ptr(t0.Add(80 * time.Second))},
			{StartedAt: ptr(t0.Add(10 * time.Second)), CompletedAt: ptr(t0.Add(90 * time.Second))},
		}},
		artifacts: map[int64][]github.Artifact{900: {
			{ID: 1, Name: "coverage"},
			{ID: 2, Name: "test-results", Expired: false},
		}},
		artifactZip: map[int64][]byte{2: zipOf(t, "readme.txt", "x", "results/junit.xml", report)},
		logs: map[int64][]byte{900: zipOf(t,
			"build/1_setup.txt", "short",
			"0_build.txt", "all the logs\x00 https://console.mobile.dev/uploads/abc?x=1 done\xff")},
	}
	p := pendingRun(1, t0, 0)
	p.Service = service(10, svcName)
	st := &fakeStore{
		pending: []store.PendingRun{p},
		tests:   map[int64][]store.Test{10: {{ID: 31, Name: "pay.cy.js"}, {ID: 32, Name: "refund.cy.js"}, {ID: 33, Name: "missing.cy.js"}}},
	}
	s := NewSyncer(SyncerConfig{GitHub: gh, Store: st, Logger: quietLogger(), Now: func() time.Time { return t0.Add(time.Hour) }})
	return s, gh, st
}

func TestSyncCompletedRun(t *testing.T) {
	s, gh, st := syncEnv(t, "mobile")
	s.Tick(t.Context())

	if got := gh.filters[0]; got.Event != "workflow_dispatch" || got.Branch != "master" || !got.Created.Equal(t0.Add(-time.Minute)) {
		t.Errorf("runs filter = %+v", got)
	}
	if !st.since.Equal(t0.Add(time.Hour - store.RunSyncWindow)) {
		t.Errorf("pending since = %v", st.since)
	}
	if len(st.updates) != 1 {
		t.Fatalf("updates = %+v", st.updates)
	}
	u := st.updates[0]
	if u.Status != "failure" || u.OldStatus != "unknown" || u.GitHubRunID != 900 || u.CommitSHA != "sha1" ||
		!u.StartedAt.Equal(t0.Add(10*time.Second)) {
		t.Errorf("update = %+v", u)
	}
	c := u.Completion
	if c == nil || !c.FinishedAt.Equal(t0.Add(90*time.Second)) || c.ResultLog == nil || *c.ResultLog != report {
		t.Fatalf("completion = %+v", c)
	}
	if *c.Totals != (store.RunTotals{Passed: 2, Failed: 1, Duration: 4.5}) {
		t.Errorf("totals = %+v", *c.Totals)
	}
	wantResults := []store.TestResult{
		{TestID: 31, Status: "pass", Duration: 1.5},
		{TestID: 32, Status: "fail", Comment: "see video", FailureMessage: "boom", Duration: 3},
	}
	if len(c.Results) != len(wantResults) {
		t.Fatalf("results = %+v", c.Results)
	}
	for i, r := range wantResults {
		if c.Results[i] != r {
			t.Errorf("result %d = %+v, want %+v", i, c.Results[i], r)
		}
	}
	// The largest file, without NUL and with invalid UTF-8 replaced.
	if c.RunLog != "all the logs https://console.mobile.dev/uploads/abc?x=1 done�" {
		t.Errorf("run log = %q", c.RunLog)
	}
	if len(st.activity) != 1 {
		t.Fatalf("activity = %+v", st.activity)
	}
	a := st.activity[0]
	if a.Type != store.ActivityRunStatusChanged || a.UserID != 0 || a.SessionID != 1 ||
		a.Body != "Run 1 for shop/mobile (automated) has changed status from unknown to failure with 2 passed and 1 failed tests." ||
		a.Metadata[store.MetaRunID] != int64(1) || a.Metadata[store.MetaMaestroURL] != "https://console.mobile.dev/uploads/abc?x=1" {
		t.Errorf("activity = %+v", a)
	}
}

func TestSyncNoMaestroLinkForWeb(t *testing.T) {
	s, _, st := syncEnv(t, "web")
	s.Tick(t.Context())
	if len(st.activity) != 1 || st.activity[0].Metadata[store.MetaMaestroURL] != nil {
		t.Errorf("activity = %+v", st.activity)
	}
}

func TestSyncInProgress(t *testing.T) {
	s, gh, st := syncEnv(t, "web")
	gh.runs[0].Status, gh.runs[0].Conclusion = "in_progress", ""
	s.Tick(t.Context())
	if len(st.updates) != 1 || st.updates[0].Status != "in_progress" || st.updates[0].Completion != nil {
		t.Fatalf("updates = %+v", st.updates)
	}
	if st.activity[0].Body != "Run 1 for shop/web (automated) has changed status from unknown to in_progress." {
		t.Errorf("activity = %q", st.activity[0].Body)
	}

	// Nothing new on the next tick: no write, no entry.
	st.pending[0].Status, st.pending[0].GitHubRunID, st.pending[0].CommitSHA = "in_progress", 900, "sha1"
	st.pending[0].StartedAt = ptr(t0.Add(10 * time.Second))
	st.updates, st.activity = nil, nil
	s.Tick(t.Context())
	if len(st.updates) != 0 || len(st.activity) != 0 {
		t.Errorf("unchanged run was written: %+v %+v", st.updates, st.activity)
	}
}

func TestSyncDownloadFailureRetries(t *testing.T) {
	s, gh, st := syncEnv(t, "web")
	gh.logsErr = errors.New("connection reset")
	s.Tick(t.Context())
	if len(st.updates) != 0 || len(st.activity) != 0 {
		t.Errorf("a failed download must store nothing: %+v %+v", st.updates, st.activity)
	}
}

func TestSyncWithoutArtifactAndLogs(t *testing.T) {
	s, gh, st := syncEnv(t, "web")
	gh.artifacts = nil
	gh.logsErr = &github.APIError{StatusCode: http.StatusGone}
	s.Tick(t.Context())
	if len(st.updates) != 1 {
		t.Fatalf("updates = %+v", st.updates)
	}
	c := st.updates[0].Completion
	if c.ResultLog != nil || c.Totals != nil || c.RunLog != "" {
		t.Errorf("completion = %+v", c)
	}
	if st.activity[0].Body != "Run 1 for shop/web (automated) has changed status from unknown to failure." {
		t.Errorf("activity = %q", st.activity[0].Body)
	}
}

func TestSyncUnparsableReport(t *testing.T) {
	s, gh, st := syncEnv(t, "web")
	gh.artifactZip[2] = zipOf(t, "junit.xml", "<html>not junit</html>")
	s.Tick(t.Context())
	c := st.updates[0].Completion
	if c.ResultLog == nil || *c.ResultLog != "<html>not junit</html>" || c.Totals != nil || len(c.Results) != 0 {
		t.Errorf("completion = %+v", c)
	}
}

func TestSyncStale(t *testing.T) {
	s, gh, st := syncEnv(t, "web")
	gh.runs = nil
	old := pendingRun(5, t0.Add(-25*time.Hour), 0)
	old.Status = store.RunQueued
	st.stale = []store.PendingRun{old}
	s.Tick(t.Context())
	if len(st.updates) != 1 || st.updates[0] != (store.RunUpdate{ID: 5, OldStatus: "queued", Status: "stale"}) {
		t.Fatalf("updates = %+v", st.updates)
	}
	if st.activity[0].Body != "Run 5 for shop/web (automated) has changed status from queued to stale." {
		t.Errorf("activity = %q", st.activity[0].Body)
	}
}

func TestSyncConflictIsIgnored(t *testing.T) {
	s, _, st := syncEnv(t, "web")
	st.updateErr = store.ErrConflict
	s.Tick(t.Context()) // must not panic or log at WARN; nothing to assert beyond that
}

func TestSyncSession(t *testing.T) {
	s, gh, st := syncEnv(t, "web") // its run belongs to session 1
	if err := s.SyncSession(t.Context(), 42); err != nil {
		t.Fatal(err)
	}
	if len(gh.filters) != 0 || len(st.updates) != 0 {
		t.Errorf("another session's runs were synced: filters = %d, updates = %d", len(gh.filters), len(st.updates))
	}
	if err := s.SyncSession(t.Context(), 1); err != nil {
		t.Fatal(err)
	}
	if len(st.updates) != 1 || !st.takenFrom.Equal(st.since.Add(-time.Minute)) {
		t.Errorf("updates = %d, taken from %v", len(st.updates), st.takenFrom)
	}

	// A sync in progress blocks; the caller's deadline ends the wait.
	s.sem <- struct{}{}
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Millisecond)
	defer cancel()
	if err := s.SyncSession(ctx, 42); !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("busy syncer: err = %v", err)
	}
}

// Refreshing one session matches its group with every session's runs:
// session 2 dispatched the same workflow 2 s after session 1, and GitHub
// lists only session 1's run so far. Session 2's run must not take it.
func TestSyncSessionMatchesWholeGroup(t *testing.T) {
	s, gh, st := syncEnv(t, "web")
	gh.runs = []github.WorkflowRun{{ID: 900, Status: "queued", CreatedAt: t0.Add(500 * time.Millisecond)}}
	other := pendingRun(2, t0.Add(2*time.Second), 0)
	other.Service.SessionID = 2
	st.pending = append(st.pending, other)

	if err := s.SyncSession(t.Context(), 2); err != nil {
		t.Fatal(err)
	}
	if len(st.updates) != 1 || st.updates[0].ID != 1 || st.updates[0].GitHubRunID != 900 {
		t.Errorf("updates = %+v, want only run 1 matched to 900", st.updates)
	}
}

func TestMaestroLink(t *testing.T) {
	for log, want := range map[string]string{
		"see https://console.mobile.dev/uploads/abc?x=1 now":     "https://console.mobile.dev/uploads/abc?x=1",
		"\x1b[32mhttps://console.mobile.dev/uploads/abc\x1b[0m":  "https://console.mobile.dev/uploads/abc",
		"(https://console.mobile.dev/uploads/abc).":              "https://console.mobile.dev/uploads/abc",
		`<a href="https://console.mobile.dev/uploads/abc">x</a>`: "https://console.mobile.dev/uploads/abc",
		"no link here": "",
	} {
		if got := maestroLink(log); got != want {
			t.Errorf("maestroLink(%q) = %q, want %q", log, got, want)
		}
	}
}

type fakeCleaner struct{ calls int }

func (f *fakeCleaner) DeleteExpired(context.Context) (int64, error) { f.calls++; return 1, nil }

func TestTickDeletesExpiredSessions(t *testing.T) {
	s, _, _ := syncEnv(t, "web")
	c := &fakeCleaner{}
	s.sessions = c
	s.Tick(t.Context())
	if c.calls != 1 {
		t.Errorf("DeleteExpired calls = %d", c.calls)
	}
}

func TestLargestFileTruncates(t *testing.T) {
	long := strings.Repeat("a", maxRunLog) + "THE END"
	got, err := largestFile(zipOf(t, "small.txt", "x", "big.txt", long))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(got, "[TestFlow: this log is too long") || !strings.HasSuffix(got, "THE END") ||
		len(got) > maxRunLog+100 {
		t.Errorf("log = %q…%q (%d bytes)", got[:60], got[len(got)-10:], len(got))
	}
	if got, err := largestFile(zipOf(t)); err != nil || got != "" {
		t.Errorf("empty zip = %q, %v", got, err)
	}
	if _, err := largestFile([]byte("not a zip")); err == nil {
		t.Error("not a zip: want an error")
	}
}

func TestFirstXML(t *testing.T) {
	got, err := firstXML(zipOf(t, "a.txt", "x", "dir/b.XML", "<b/>", "c.xml", "<c/>"))
	if err != nil || string(got) != "<b/>" {
		t.Errorf("firstXML = %q, %v", got, err)
	}
	if got, err := firstXML(zipOf(t, "a.txt", "x")); err != nil || got != nil {
		t.Errorf("no xml = %q, %v", got, err)
	}
}
