package discovery

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"path"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/laposa/testflow/app-go/internal/github"
)

// fakeGitHub serves repositories from a map of file paths per repository.
type fakeGitHub struct {
	repos     []github.Repository
	files     map[string][]string // repo → file paths
	shas      map[string]string   // repo → master SHA; missing = 404
	shaErr    map[string]error
	dirErr    map[string]error // "repo:path" → error
	workflows map[string][]github.Workflow
	wfErr     map[string]error
	reposErr  error

	calls    atomic.Int64
	inFlight atomic.Int64
	maxSeen  atomic.Int64
	delay    time.Duration

	mu   sync.Mutex
	refs []string // refs passed to Dir
}

func (f *fakeGitHub) enter() func() {
	f.calls.Add(1)
	n := f.inFlight.Add(1)
	for {
		m := f.maxSeen.Load()
		if n <= m || f.maxSeen.CompareAndSwap(m, n) {
			break
		}
	}
	if f.delay > 0 {
		time.Sleep(f.delay)
	}
	return func() { f.inFlight.Add(-1) }
}

func (f *fakeGitHub) Repositories(context.Context) ([]github.Repository, error) {
	defer f.enter()()
	return f.repos, f.reposErr
}

func (f *fakeGitHub) CommitSHA(_ context.Context, repo, ref string) (string, error) {
	defer f.enter()()
	if ref != Branch {
		return "", errors.New("unexpected ref " + ref)
	}
	if err := f.shaErr[repo]; err != nil {
		return "", err
	}
	sha, ok := f.shas[repo]
	if !ok {
		return "", &github.APIError{StatusCode: http.StatusNotFound}
	}
	return sha, nil
}

func (f *fakeGitHub) Dir(_ context.Context, repo, dir, ref string) ([]github.ContentEntry, error) {
	defer f.enter()()
	f.mu.Lock()
	f.refs = append(f.refs, ref)
	f.mu.Unlock()
	if err := f.dirErr[repo+":"+dir]; err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	var out []github.ContentEntry
	for _, p := range f.files[repo] {
		rest, ok := strings.CutPrefix(p, dir+"/")
		if !ok {
			continue
		}
		name, _, isDir := strings.Cut(rest, "/")
		if seen[name] {
			continue
		}
		seen[name] = true
		typ := "file"
		if isDir {
			typ = "dir"
		}
		out = append(out, github.ContentEntry{Type: typ, Name: name, Path: dir + "/" + name})
	}
	if len(out) == 0 {
		return nil, &github.APIError{StatusCode: http.StatusNotFound}
	}
	slices.SortFunc(out, func(a, b github.ContentEntry) int { return strings.Compare(a.Name, b.Name) })
	return out, nil
}

func (f *fakeGitHub) Workflows(_ context.Context, repo string) ([]github.Workflow, error) {
	defer f.enter()()
	return f.workflows[repo], f.wfErr[repo]
}

func quietLogger() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

func shopFake() *fakeGitHub {
	return &fakeGitHub{
		repos: []github.Repository{
			{ID: 2, FullName: "laposa/shop-tests"},
			{ID: 1, FullName: "laposa/app"},   // tests/ but no service tests
			{ID: 3, FullName: "laposa/empty"}, // no master
			{ID: 4, FullName: "laposa/no-tests"},
		},
		shas: map[string]string{"laposa/shop-tests": "sha-shop", "laposa/app": "sha-app", "laposa/no-tests": "sha-nt"},
		files: map[string][]string{
			"laposa/shop-tests": {
				"tests/README.md",
				"tests/web/package.json",
				"tests/web/tests/checkout/pay.cy.js",
				"tests/web/tests/checkout/pay.manual.yaml",
				"tests/web/tests/checkout/_helpers.js",
				"tests/web/tests/_utils/x.js",
				"tests/web/tests/account/login.cy.js",
				"tests/mobile/tests/login/flow.yaml",
				"tests/_shared/tests/a/b.js",
				"tests/api/README.md", // no tests/ inside
			},
			"laposa/app":      {"tests/unit/foo_test.go"},
			"laposa/no-tests": {"src/main.go"},
		},
		workflows: map[string][]github.Workflow{
			"laposa/shop-tests": {
				{ID: 10, Path: ".github/workflows/build.yaml"},
				{ID: 11, Path: ".github/workflows/run-tests-web.yaml"},
				{ID: 12, Path: ".github/workflows/e2e-run-tests-mobile.yaml"},
			},
		},
	}
}

func TestDiscover(t *testing.T) {
	gh := shopFake()
	d := New(Config{GitHub: gh, Logger: quietLogger()})
	res, err := d.Tests(t.Context(), false)
	if err != nil {
		t.Fatal(err)
	}
	if res.ID == "" || res.FetchedAt.IsZero() {
		t.Errorf("result id/time not set: %+v", res)
	}
	if len(res.Problems) != 0 {
		t.Errorf("problems = %+v", res.Problems)
	}
	if len(res.Repositories) != 1 {
		t.Fatalf("repositories = %+v, want only shop-tests", res.Repositories)
	}
	repo := res.Repositories[0]
	if repo.ID != 2 || repo.Name != "laposa/shop-tests" || repo.CommitSHA != "sha-shop" {
		t.Errorf("repo = %+v", repo)
	}

	var got []string
	for _, svc := range repo.Services {
		for _, su := range svc.Suites {
			for _, tc := range su.Tests {
				got = append(got, svc.Name+"|"+su.Name+"|"+tc.Name+"|"+tc.Type())
			}
		}
	}
	want := []string{
		"mobile|login|flow.yaml|automated",
		"web|account|login.cy.js|automated",
		"web|checkout|pay.cy.js|automated",
		"web|checkout|pay.manual.yaml|manual",
	}
	if !slices.Equal(got, want) {
		t.Errorf("tests =\n%v\nwant\n%v", got, want)
	}

	web := repo.Services[1]
	if web.Path != "tests/web" || web.WorkflowID != 11 {
		t.Errorf("web service = %+v", web)
	}
	if su := web.Suites[1]; su.Path != "tests/web/tests/checkout" || su.Tests[0].Path != "tests/web/tests/checkout/pay.cy.js" {
		t.Errorf("checkout suite = %+v", su)
	}
	if k := web.Suites[1].Tests[0].Key; k != "2:tests/web/tests/checkout/pay.cy.js" {
		t.Errorf("key = %q", k)
	}
	// Only a file named exactly run-tests-mobile.yaml matches.
	if mobile := repo.Services[0]; mobile.WorkflowID != 0 {
		t.Errorf("mobile workflow = %d, want 0", mobile.WorkflowID)
	}
	// Everything is read at the discovered commit.
	for _, ref := range gh.refs {
		if !slices.Contains([]string{"sha-shop", "sha-app", "sha-nt"}, ref) {
			t.Errorf("Dir called with ref %q", ref)
		}
	}
}

func TestDiscoverProblems(t *testing.T) {
	gh := shopFake()
	gh.repos = append(gh.repos,
		github.Repository{ID: 5, FullName: "laposa/forbidden"},
		github.Repository{ID: 6, FullName: "laposa/emptyrepo"},
	)
	gh.shas["laposa/forbidden"] = "sha-f"
	gh.files["laposa/forbidden"] = []string{"tests/web/tests/a/b.js"}
	gh.dirErr = map[string]error{"laposa/forbidden:tests/web/tests": &github.APIError{StatusCode: 403, Message: "nope"}}
	gh.shaErr = map[string]error{"laposa/emptyrepo": &github.APIError{StatusCode: http.StatusConflict}}
	gh.wfErr = map[string]error{"laposa/shop-tests": errors.New("timeout")}

	res, err := New(Config{GitHub: gh, Logger: quietLogger()}).Tests(t.Context(), false)
	if err != nil {
		t.Fatal(err)
	}
	var probs []string
	for _, p := range res.Problems {
		probs = append(probs, p.Repository+": "+p.Message)
	}
	want := []string{
		"laposa/forbidden: could not read tests/web/tests",
		"laposa/shop-tests: could not read workflows, so its automated tests cannot be selected",
	}
	if !slices.Equal(probs, want) {
		t.Errorf("problems = %q, want %q", probs, want)
	}
	// The tree is kept without workflows; manual tests stay selectable.
	if len(res.Repositories) != 1 {
		t.Fatalf("repositories = %d", len(res.Repositories))
	}
	web := res.Repositories[0].Services[1]
	manual := web.Suites[1].Tests[1]
	if web.WorkflowID != 0 || !web.Selectable(manual) || web.Selectable(web.Suites[1].Tests[0]) {
		t.Errorf("selectable without workflows: %+v", web)
	}
}

func TestDiscoverRepositoriesError(t *testing.T) {
	gh := &fakeGitHub{reposErr: errors.New("boom")}
	if _, err := New(Config{GitHub: gh, Logger: quietLogger()}).Tests(t.Context(), false); err == nil {
		t.Error("want error")
	}
}

func TestDiscoverTimeoutIsNotCached(t *testing.T) {
	gh := shopFake()
	gh.delay = 20 * time.Millisecond
	d := New(Config{GitHub: gh, Timeout: 30 * time.Millisecond, Logger: quietLogger()})
	if _, err := d.Tests(t.Context(), false); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("err = %v, want deadline exceeded", err)
	}
	gh.delay = 0
	res, err := d.Tests(t.Context(), false)
	if err != nil || len(res.Repositories) != 1 {
		t.Errorf("after a timeout the next request should discover again: %v %+v", err, res)
	}
}

func TestQueuedRequestsShareFailure(t *testing.T) {
	gh := &fakeGitHub{reposErr: errors.New("boom"), delay: 50 * time.Millisecond}
	d := New(Config{GitHub: gh, Logger: quietLogger()})
	var wg sync.WaitGroup
	for range 3 {
		wg.Go(func() {
			if _, err := d.Tests(t.Context(), false); err == nil {
				t.Error("want error")
			}
		})
		time.Sleep(5 * time.Millisecond) // the first one starts the discovery
	}
	wg.Wait()
	if n := gh.calls.Load(); n != 1 {
		t.Errorf("GitHub called %d times, want 1: queued requests should get the same error", n)
	}
	// A later request tries again.
	_, _ = d.Tests(t.Context(), false)
	if n := gh.calls.Load(); n != 2 {
		t.Errorf("GitHub called %d times, want 2", n)
	}
}

func TestFailureFallsBackToLastResult(t *testing.T) {
	gh := shopFake()
	now := time.Date(2026, 11, 18, 9, 0, 0, 0, time.UTC)
	var mu sync.Mutex
	clock := func() time.Time { mu.Lock(); defer mu.Unlock(); return now }
	advance := func(d time.Duration) { mu.Lock(); now = now.Add(d); mu.Unlock() }
	d := New(Config{GitHub: gh, TTL: 5 * time.Minute, Retention: time.Hour, Now: clock, Logger: quietLogger()})
	ctx := t.Context()

	first, err := d.Tests(ctx, false)
	if err != nil {
		t.Fatal(err)
	}
	gh.reposErr = errors.New("boom")

	advance(10 * time.Minute)
	res, err := d.Tests(ctx, false)
	if err == nil || res != first {
		t.Errorf("after the TTL: res = %p, err = %v; want the last result with the error", res, err)
	}
	if res, err := d.Tests(ctx, true); err == nil || res != first {
		t.Errorf("refresh: res = %p, err = %v; want the last result with the error", res, err)
	}
	if _, ok := d.Snapshot(first.ID); !ok {
		t.Error("the fallback result must stay submittable")
	}

	// Once its snapshot has expired, a form rendered from it could not be
	// submitted, so it is not offered any more.
	advance(time.Hour)
	if res, err := d.Tests(ctx, false); err == nil || res != nil {
		t.Errorf("beyond retention: res = %p, err = %v; want only the error", res, err)
	}
}

func TestDiscoverConcurrencyBound(t *testing.T) {
	gh := &fakeGitHub{shas: map[string]string{}, files: map[string][]string{}, delay: 5 * time.Millisecond}
	for i := range 10 {
		name := "o/r" + string(rune('a'+i))
		gh.repos = append(gh.repos, github.Repository{ID: int64(i + 1), FullName: name})
		gh.shas[name] = "s"
	}
	if _, err := New(Config{GitHub: gh, Concurrency: 3, Logger: quietLogger()}).Tests(t.Context(), false); err != nil {
		t.Fatal(err)
	}
	if m := gh.maxSeen.Load(); m > 3 || m < 2 {
		t.Errorf("max concurrent calls = %d, want 2..3", m)
	}
}

func TestCacheAndSnapshots(t *testing.T) {
	gh := shopFake()
	now := time.Date(2026, 11, 18, 9, 0, 0, 0, time.UTC)
	var mu sync.Mutex
	clock := func() time.Time { mu.Lock(); defer mu.Unlock(); return now }
	advance := func(d time.Duration) { mu.Lock(); now = now.Add(d); mu.Unlock() }
	d := New(Config{GitHub: gh, TTL: 5 * time.Minute, Retention: time.Hour, Now: clock, Logger: quietLogger()})
	ctx := t.Context()

	if d.Cached() {
		t.Error("Cached before the first discovery")
	}
	first, err := d.Tests(ctx, false)
	if err != nil {
		t.Fatal(err)
	}
	calls := gh.calls.Load()

	advance(4 * time.Minute)
	if !d.Cached() {
		t.Error("not Cached within the TTL")
	}
	if again, _ := d.Tests(ctx, false); again != first || gh.calls.Load() != calls {
		t.Error("cached result not reused within the TTL")
	}

	advance(2 * time.Minute)
	if d.Cached() {
		t.Error("Cached after the TTL")
	}
	second, _ := d.Tests(ctx, false)
	if second == first || second.ID == first.ID {
		t.Error("result not refreshed after the TTL")
	}
	if s, ok := d.Snapshot(first.ID); !ok || s != first {
		t.Error("previous snapshot not retained")
	}

	advance(time.Minute)
	third, _ := d.Tests(ctx, true)
	if third == second {
		t.Error("refresh did not discover again")
	}

	advance(time.Hour)
	if _, ok := d.Snapshot(first.ID); ok {
		t.Error("snapshot kept beyond retention")
	}
	if _, ok := d.Snapshot("nope"); ok {
		t.Error("unknown snapshot found")
	}
}

func TestDiscoveryNotCancelledWithRequest(t *testing.T) {
	gh := shopFake()
	d := New(Config{GitHub: gh, Logger: quietLogger()})
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := d.Tests(ctx, false); err != nil {
		t.Errorf("discovery stopped with the request context: %v", err)
	}
}

func TestTestType(t *testing.T) {
	for name, want := range map[string]string{
		"login.manual.yaml": TypeManual,
		"manual-check.js":   TypeManual,
		"login.cy.js":       TypeAutomated,
		"Login.MANUAL.yaml": TypeAutomated, // case-sensitive, like legacy Str::contains
	} {
		if got := TestType(name); got != want {
			t.Errorf("TestType(%q) = %q, want %q", name, got, want)
		}
	}
}

func selectFixture(t *testing.T) *Result {
	t.Helper()
	res, err := New(Config{GitHub: shopFake(), Logger: quietLogger()}).Tests(t.Context(), false)
	if err != nil {
		t.Fatal(err)
	}
	return res
}

func TestSelect(t *testing.T) {
	res := selectFixture(t)
	k := func(p string) string { return TestKey(2, path.Join("tests", p)) }
	groups, err := res.Select([]string{
		k("web/tests/checkout/pay.manual.yaml"),
		k("web/tests/checkout/pay.cy.js"),
		k("web/tests/account/login.cy.js"),
		k("web/tests/checkout/pay.cy.js"), // duplicate
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(groups) != 2 {
		t.Fatalf("groups = %+v", groups)
	}
	auto, manual := groups[0], groups[1]
	if auto.Type != TypeAutomated || auto.Service.Name != "web" || auto.Repository.CommitSHA != "sha-shop" || auto.Service.WorkflowID != 11 {
		t.Errorf("automated group = %+v", auto)
	}
	if len(auto.Service.Suites) != 2 || auto.Service.Suites[0].Name != "account" ||
		len(auto.Service.Suites[1].Tests) != 1 || auto.Service.Suites[1].Tests[0].Name != "pay.cy.js" {
		t.Errorf("automated suites = %+v", auto.Service.Suites)
	}
	if manual.Type != TypeManual || len(manual.Service.Suites) != 1 || manual.Service.Suites[0].Tests[0].Name != "pay.manual.yaml" {
		t.Errorf("manual group = %+v", manual)
	}
	if auto.Repository.Services != nil {
		t.Error("group repository should not carry the whole tree")
	}
}

func TestSelectErrors(t *testing.T) {
	res := selectFixture(t)
	if _, err := res.Select(nil); !errors.Is(err, ErrNoTests) {
		t.Errorf("empty selection: %v", err)
	}
	if _, err := res.Select([]string{"2:tests/web/tests/checkout/nope.js"}); err == nil {
		t.Error("unknown key accepted")
	}
	if _, err := res.Select([]string{"9:tests/web/tests/checkout/pay.cy.js"}); err == nil {
		t.Error("key of another repository accepted")
	}
	if _, err := res.Select([]string{"2:tests/mobile/tests/login/flow.yaml"}); err == nil {
		t.Error("automated test without workflow accepted")
	}
}
