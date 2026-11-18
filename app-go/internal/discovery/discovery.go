// Package discovery finds the tests in the GitHub App installation's
// repositories and turns a selection of them into session services
// (spec §8.1, §8.2; legacy: app/Actions/Github/FetchTests.php,
// app/Actions/Session/CreateSession.php).
//
// The layout it expects is tests/{service}/tests/{suite}/{test}. Results are
// cached in memory for a short TTL, and every result is kept as a snapshot
// for a while longer, so a form rendered from it can still be submitted
// after the cache has moved on.
package discovery

import (
	"cmp"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/laposa/testflow/app-go/internal/github"
)

// Defaults for Config.
const (
	DefaultTTL         = 5 * time.Minute
	DefaultRetention   = time.Hour
	DefaultConcurrency = 4
	// DefaultTimeout bounds one discovery. It stays below the HTTP server's
	// WriteTimeout, since the test picker waits for it.
	DefaultTimeout = 45 * time.Second
	// Branch is discovered from and dispatched to (open question Q6).
	Branch = "master"
)

// Test types (spec §6.4).
const (
	TypeAutomated = "automated"
	TypeManual    = "manual"
)

// GitHub is the part of the GitHub client discovery uses.
type GitHub interface {
	Repositories(ctx context.Context) ([]github.Repository, error)
	CommitSHA(ctx context.Context, repo, ref string) (string, error)
	Dir(ctx context.Context, repo, path, ref string) ([]github.ContentEntry, error)
	Workflows(ctx context.Context, repo string) ([]github.Workflow, error)
}

// Result is one discovery: the test tree of every repository that has tests.
type Result struct {
	ID           string // identifies the snapshot, see Discoverer.Snapshot
	FetchedAt    time.Time
	Repositories []Repository // sorted by name
	Problems     []Problem    // repositories that could not be read fully
}

// Problem says what went wrong with one repository. Message is meant for
// users; details are logged.
type Problem struct {
	Repository string
	Message    string
}

// Repository is a repository with at least one test.
type Repository struct {
	ID        int64
	Name      string // owner/repo
	CommitSHA string // latest commit of Branch; pins everything below
	Services  []Service
}

// Service is tests/{service}.
type Service struct {
	Name       string
	Path       string // tests/{service}
	WorkflowID int64  // 0 if no run-tests-{service}.yaml workflow exists
	Suites     []Suite
}

// Suite is tests/{service}/tests/{suite}.
type Suite struct {
	Name  string
	Path  string
	Tests []Test
}

// Test is one test file of a suite.
type Test struct {
	Key  string // unique within a Result; the form value that selects it
	Name string // file name
	Path string
}

// Type is TypeManual if the file name contains "manual" (in practice
// *.manual.yaml), otherwise TypeAutomated (spec §8.1 step 5).
func (t Test) Type() string { return TestType(t.Name) }

// TestType returns the type of a test with the given file name.
func TestType(name string) string {
	if strings.Contains(name, "manual") {
		return TypeManual
	}
	return TypeAutomated
}

// Selectable reports whether t can be part of a session. Automated tests
// need the service's workflow to be dispatched; manual tests never use it.
func (s Service) Selectable(t Test) bool {
	return s.WorkflowID != 0 || t.Type() == TypeManual
}

// TestKey is the key of the test at path in repository repoID.
func TestKey(repoID int64, path string) string {
	return strconv.FormatInt(repoID, 10) + ":" + path
}

// WorkflowFile is the workflow file name that runs a service's tests
// (legacy: helpers.php getWorkflowFilename).
func WorkflowFile(service string) string { return "run-tests-" + service + ".yaml" }

// Config configures a Discoverer. GitHub is required.
type Config struct {
	GitHub      GitHub
	TTL         time.Duration // how long a result is reused, default DefaultTTL
	Retention   time.Duration // how long a snapshot stays available, default DefaultRetention
	Concurrency int           // repositories walked at once, default DefaultConcurrency
	Timeout     time.Duration // per discovery, default DefaultTimeout
	Logger      *slog.Logger
	Now         func() time.Time
}

// Discoverer runs and caches discoveries. It is safe for concurrent use.
type Discoverer struct {
	cfg Config

	fetchMu sync.Mutex // one discovery at a time

	mu        sync.Mutex
	current   *Result
	snapshots map[string]*Result
	lastErr   error  // the last failed discovery
	failures  uint64 // number of failed discoveries
}

// New returns a Discoverer.
func New(c Config) *Discoverer {
	if c.TTL <= 0 {
		c.TTL = DefaultTTL
	}
	if c.Retention < c.TTL {
		c.Retention = max(DefaultRetention, c.TTL)
	}
	if c.Concurrency <= 0 {
		c.Concurrency = DefaultConcurrency
	}
	if c.Timeout <= 0 {
		c.Timeout = DefaultTimeout
	}
	if c.Logger == nil {
		c.Logger = slog.Default()
	}
	if c.Now == nil {
		c.Now = time.Now
	}
	return &Discoverer{cfg: c, snapshots: map[string]*Result{}}
}

// Tests returns the cached result if it is younger than the TTL, otherwise
// it discovers again. refresh skips the cache, unless another request
// finished a discovery while this one waited for it.
//
// If the discovery fails, Tests returns the error together with the last
// successful result while its snapshot is still retained, so callers can
// offer it as possibly outdated. The result is nil if there is none.
//
// The discovery itself is not cancelled with ctx, so a user who navigates
// away does not waste a nearly finished run; it is bounded by the timeout.
func (d *Discoverer) Tests(ctx context.Context, refresh bool) (*Result, error) {
	asked := d.cfg.Now()
	d.mu.Lock()
	failures := d.failures
	d.mu.Unlock()
	d.fetchMu.Lock()
	defer d.fetchMu.Unlock()

	d.mu.Lock()
	cur, lastErr, failedMeanwhile := d.current, d.lastErr, d.failures != failures
	d.mu.Unlock()
	if cur != nil {
		fresh := d.cfg.Now().Sub(cur.FetchedAt) < d.cfg.TTL
		if (!refresh && fresh) || (refresh && !cur.FetchedAt.Before(asked)) {
			return cur, nil
		}
	}
	// A discovery that failed while this request waited has just spent up
	// to Timeout; failing fast keeps queued requests within the server's
	// write timeout instead of each retrying in turn.
	if failedMeanwhile {
		return d.fallback(cur), lastErr
	}

	fctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), d.cfg.Timeout)
	defer cancel()
	res, err := d.discover(fctx)
	if err == nil && fctx.Err() != nil {
		// Every repository still pending failed with the deadline; a tree
		// with them missing must not be cached as if it were complete.
		err = fmt.Errorf("discovery: %w", fctx.Err())
	}

	d.mu.Lock()
	defer d.mu.Unlock()
	if err != nil {
		d.lastErr = err
		d.failures++
		return d.fallback(cur), err
	}
	d.current = res
	d.snapshots[res.ID] = res
	for id, s := range d.snapshots {
		if res.FetchedAt.Sub(s.FetchedAt) > d.cfg.Retention {
			delete(d.snapshots, id)
		}
	}
	return res, nil
}

// fallback returns cur if its snapshot is still retained, so a form
// rendered from it can be submitted.
func (d *Discoverer) fallback(cur *Result) *Result {
	if cur == nil || d.cfg.Now().Sub(cur.FetchedAt) > d.cfg.Retention {
		return nil
	}
	return cur
}

// Snapshot returns the result with id if it is still retained.
func (d *Discoverer) Snapshot(id string) (*Result, bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	s, ok := d.snapshots[id]
	if !ok || d.cfg.Now().Sub(s.FetchedAt) > d.cfg.Retention {
		return nil, false
	}
	return s, true
}

// discover walks every repository of the installation, at most
// Concurrency at a time.
func (d *Discoverer) discover(ctx context.Context) (*Result, error) {
	repos, err := d.cfg.GitHub.Repositories(ctx)
	if err != nil {
		return nil, err
	}

	type outcome struct {
		repo     *Repository
		problems []Problem
	}
	out := make([]outcome, len(repos))
	sem := make(chan struct{}, d.cfg.Concurrency)
	var wg sync.WaitGroup
	for i, r := range repos {
		wg.Go(func() {
			sem <- struct{}{}
			defer func() { <-sem }()
			repo, problems := d.repository(ctx, r)
			out[i] = outcome{repo, problems}
		})
	}
	wg.Wait()

	res := &Result{ID: newID(), FetchedAt: d.cfg.Now()}
	for _, o := range out {
		if o.repo != nil {
			res.Repositories = append(res.Repositories, *o.repo)
		}
		res.Problems = append(res.Problems, o.problems...)
	}
	slices.SortFunc(res.Repositories, func(a, b Repository) int { return cmp.Compare(a.Name, b.Name) })
	slices.SortFunc(res.Problems, func(a, b Problem) int { return cmp.Compare(a.Repository, b.Repository) })
	return res, nil
}

// repository returns r's test tree, or nil if it has no tests. A
// repository without the branch, without commits or without tests/ is
// skipped silently.
func (d *Discoverer) repository(ctx context.Context, r github.Repository) (*Repository, []Problem) {
	name := r.FullName
	fail := func(msg string, err error) (*Repository, []Problem) {
		d.cfg.Logger.WarnContext(ctx, "discovery: repository skipped", "repository", name, "problem", msg, "err", err)
		return nil, []Problem{{Repository: name, Message: msg}}
	}

	sha, err := d.cfg.GitHub.CommitSHA(ctx, name, Branch)
	if skippable(err) {
		return nil, nil
	}
	if err != nil {
		return fail("could not read the latest commit of "+Branch, err)
	}

	repo := &Repository{ID: r.ID, Name: name, CommitSHA: sha}
	services, err := d.dirs(ctx, name, "tests", sha)
	if errors.Is(err, github.ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return fail("could not read tests/", err)
	}
	for _, se := range services {
		svc := Service{Name: se.Name, Path: se.Path}
		// tests/{service}/tests; a service without it has no tests.
		suites, err := d.dirs(ctx, name, se.Path+"/tests", sha)
		if errors.Is(err, github.ErrNotFound) {
			continue
		}
		if err != nil {
			return fail("could not read "+se.Path+"/tests", err)
		}
		for _, su := range suites {
			entries, err := d.cfg.GitHub.Dir(ctx, name, su.Path, sha)
			if err != nil {
				return fail("could not read "+su.Path, err)
			}
			suite := Suite{Name: su.Name, Path: su.Path}
			for _, e := range entries {
				if hidden(e.Name) {
					continue
				}
				suite.Tests = append(suite.Tests, Test{Key: TestKey(r.ID, e.Path), Name: e.Name, Path: e.Path})
			}
			if len(suite.Tests) > 0 {
				svc.Suites = append(svc.Suites, suite)
			}
		}
		if len(svc.Suites) > 0 {
			repo.Services = append(repo.Services, svc)
		}
	}
	if len(repo.Services) == 0 {
		return nil, nil
	}

	var problems []Problem
	workflows, err := d.cfg.GitHub.Workflows(ctx, name)
	if err != nil {
		// Keep the tree without workflows: manual tests stay selectable.
		workflows = nil
		d.cfg.Logger.WarnContext(ctx, "discovery: could not read workflows", "repository", name, "err", err)
		problems = append(problems, Problem{Repository: name, Message: "could not read workflows, so its automated tests cannot be selected"})
	}
	for i := range repo.Services {
		repo.Services[i].WorkflowID = matchWorkflow(workflows, repo.Services[i].Name)
	}
	return repo, problems
}

// dirs lists the subdirectories of path, without hidden ones.
func (d *Discoverer) dirs(ctx context.Context, repo, path, ref string) ([]github.ContentEntry, error) {
	entries, err := d.cfg.GitHub.Dir(ctx, repo, path, ref)
	if err != nil {
		return nil, err
	}
	var out []github.ContentEntry
	for _, e := range entries {
		if e.Type == "dir" && !hidden(e.Name) {
			out = append(out, e)
		}
	}
	return out, nil
}

// hidden reports entries to skip, such as _utils (legacy parity).
func hidden(name string) bool { return name == "" || name[0] == '_' }

// matchWorkflow returns the id of the workflow whose path ends with
// run-tests-{service}.yaml, or 0 (spec §8.1 step 4).
func matchWorkflow(workflows []github.Workflow, service string) int64 {
	file := WorkflowFile(service)
	for _, w := range workflows {
		if strings.HasSuffix(w.Path, "/"+file) || w.Path == file {
			return w.ID
		}
	}
	return 0
}

// skippable reports errors that mean "no tests here": a missing branch
// (404) or an empty repository (409).
func skippable(err error) bool {
	if errors.Is(err, github.ErrNotFound) {
		return true
	}
	var apiErr *github.APIError
	return errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusConflict
}

func newID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
