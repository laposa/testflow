package github

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// fakeAPI is an httptest GitHub: it issues installation tokens and serves
// the routes a test registers, checking the standard API headers on each.
type fakeAPI struct {
	t      *testing.T
	mux    *http.ServeMux
	srv    *httptest.Server
	client *Client
	issued atomic.Int64
}

func newFakeAPI(t *testing.T) *fakeAPI {
	t.Helper()
	f := &fakeAPI{t: t, mux: http.NewServeMux()}
	f.mux.HandleFunc("POST /app/installations/42/access_tokens", func(w http.ResponseWriter, r *http.Request) {
		n := f.issued.Add(1)
		w.WriteHeader(http.StatusCreated)
		fmt.Fprintf(w, `{"token":"tok-%d","expires_at":%q}`, n, time.Now().Add(time.Hour).UTC().Format(time.RFC3339))
	})
	f.srv = httptest.NewServer(f.mux)
	t.Cleanup(f.srv.Close)
	f.client = newTestClient(t, f.srv, nil)
	return f
}

// handle registers an API route that requires the current installation
// token and the standard headers.
func (f *fakeAPI) handle(pattern string, h http.HandlerFunc) {
	f.mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
		if want := fmt.Sprintf("Bearer tok-%d", f.issued.Load()); r.Header.Get("Authorization") != want {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"message":"Bad credentials"}`))
			return
		}
		if r.Header.Get("X-GitHub-Api-Version") != APIVersion {
			f.t.Errorf("%s: X-GitHub-Api-Version = %q", pattern, r.Header.Get("X-GitHub-Api-Version"))
		}
		h(w, r)
	})
}

// fixture serves a testdata file as JSON.
func fixture(t *testing.T, name string) http.HandlerFunc {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Accept") != "application/vnd.github+json" {
			t.Errorf("%s: Accept = %q", r.URL.Path, r.Header.Get("Accept"))
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Write(b)
	}
}

func TestRepositoriesPaginates(t *testing.T) {
	f := newFakeAPI(t)
	page1, page2 := fixture(t, "installation_repositories_page1.json"), fixture(t, "installation_repositories_page2.json")
	f.handle("GET /installation/repositories", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("per_page") != "100" {
			t.Errorf("per_page = %q", r.URL.Query().Get("per_page"))
		}
		switch r.URL.Query().Get("page") {
		case "1":
			w.Header().Set("Link", `<`+f.srv.URL+`/installation/repositories?per_page=100&page=2>; rel="next", <`+f.srv.URL+`/installation/repositories?per_page=100&page=2>; rel="last"`)
			page1(w, r)
		case "2":
			w.Header().Set("Link", `<`+f.srv.URL+`/installation/repositories?per_page=100&page=1>; rel="prev", <`+f.srv.URL+`/installation/repositories?per_page=100&page=1>; rel="first"`)
			page2(w, r)
		default:
			t.Errorf("unexpected page %q", r.URL.Query().Get("page"))
		}
	})
	repos, err := f.client.Repositories(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	want := []Repository{
		{ID: 1296269, Name: "shop-tests", FullName: "laposa/shop-tests", DefaultBranch: "master"},
		{ID: 1296270, Name: "app-tests", FullName: "laposa/app-tests", DefaultBranch: "main"},
		{ID: 1296271, Name: "old-tests", FullName: "laposa/old-tests", DefaultBranch: "master", Archived: true},
	}
	if !reflect.DeepEqual(repos, want) {
		t.Errorf("repos = %+v", repos)
	}
}

func TestCommitSHA(t *testing.T) {
	f := newFakeAPI(t)
	f.handle("GET /repos/laposa/shop-tests/commits/{ref...}", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Accept") != "application/vnd.github.sha" {
			t.Errorf("Accept = %q", r.Header.Get("Accept"))
		}
		if r.PathValue("ref") != "release/1.0" {
			w.WriteHeader(http.StatusUnprocessableEntity)
			w.Write([]byte(`{"message":"No commit found for SHA: x"}`))
			return
		}
		w.Write([]byte("acb5820ced9479c074f688cc328bf03f341a511d"))
	})
	ctx := context.Background()
	sha, err := f.client.CommitSHA(ctx, "laposa/shop-tests", "release/1.0")
	if err != nil || sha != "acb5820ced9479c074f688cc328bf03f341a511d" {
		t.Fatalf("CommitSHA = %q, %v", sha, err)
	}
	_, err = f.client.CommitSHA(ctx, "laposa/shop-tests", "master")
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != 422 || apiErr.Message != "No commit found for SHA: x" {
		t.Errorf("err = %v", err)
	}
}

func TestDir(t *testing.T) {
	f := newFakeAPI(t)
	f.handle("GET /repos/laposa/shop-tests/contents/tests", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("ref") != "acb5820" {
			t.Errorf("ref = %q", r.URL.Query().Get("ref"))
		}
		fixture(t, "contents_dir.json")(w, r)
	})
	f.handle("GET /repos/laposa/shop-tests/contents/tests/README.md", fixture(t, "contents_file.json"))
	f.handle("GET /repos/laposa/no-tests/contents/tests", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"message":"Not Found","documentation_url":"https://docs.github.com/rest/repos/contents#get-repository-content","status":"404"}`))
	})
	ctx := context.Background()

	entries, err := f.client.Dir(ctx, "laposa/shop-tests", "tests/", "acb5820")
	if err != nil {
		t.Fatal(err)
	}
	want := []ContentEntry{
		{Type: "dir", Name: "web", Path: "tests/web", SHA: "a84d88e7554fc1fa21bcbc4efae3c782a70d2b9d"},
		{Type: "file", Name: "README.md", Path: "tests/README.md", SHA: "3d21ec53a331a6f037a91c368710b99387d012c1", Size: 625},
	}
	if !reflect.DeepEqual(entries, want) {
		t.Errorf("entries = %+v", entries)
	}

	if _, err := f.client.Dir(ctx, "laposa/shop-tests", "tests/README.md", ""); err == nil || !strings.Contains(err.Error(), "not a directory") {
		t.Errorf("file as dir: err = %v", err)
	}
	if _, err := f.client.Dir(ctx, "laposa/no-tests", "tests", ""); !errors.Is(err, ErrNotFound) {
		t.Errorf("missing dir: err = %v, want ErrNotFound", err)
	}
}

func TestFile(t *testing.T) {
	f := newFakeAPI(t)
	content := "name: login with valid credentials\nsuite: auth\n"
	f.handle("GET /repos/laposa/shop-tests/contents/tests/web/tests/auth/{file}", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Accept") != "application/vnd.github.raw+json" {
			t.Errorf("Accept = %q", r.Header.Get("Accept"))
		}
		if r.PathValue("file") != "login ok.manual.yaml" || r.URL.Query().Get("ref") != "abc" {
			t.Errorf("file = %q, ref = %q", r.PathValue("file"), r.URL.Query().Get("ref"))
		}
		w.Header().Set("Content-Type", "application/vnd.github.raw+json")
		w.Write([]byte(content))
	})
	b, err := f.client.File(context.Background(), "laposa/shop-tests", "tests/web/tests/auth/login ok.manual.yaml", "abc")
	if err != nil || string(b) != content {
		t.Fatalf("File = %q, %v", b, err)
	}
}

func TestWorkflows(t *testing.T) {
	f := newFakeAPI(t)
	f.handle("GET /repos/laposa/shop-tests/actions/workflows", fixture(t, "workflows.json"))
	wfs, err := f.client.Workflows(context.Background(), "laposa/shop-tests")
	if err != nil {
		t.Fatal(err)
	}
	want := []Workflow{
		{ID: 161335, Name: "Run web tests", Path: ".github/workflows/run-tests-web.yaml", State: "active"},
		{ID: 161336, Name: "Run api tests", Path: ".github/workflows/run-tests-api.yaml", State: "active"},
	}
	if !reflect.DeepEqual(wfs, want) {
		t.Errorf("workflows = %+v", wfs)
	}
}

func TestDispatchWorkflow(t *testing.T) {
	f := newFakeAPI(t)
	f.handle("POST /repos/laposa/shop-tests/actions/workflows/{id}/dispatches", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Content-Type = %q", r.Header.Get("Content-Type"))
		}
		var body struct {
			Ref    string            `json:"ref"`
			Inputs map[string]string `json:"inputs"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if r.PathValue("id") != "161335" {
			w.WriteHeader(http.StatusUnprocessableEntity)
			w.Write([]byte(`{"message":"Unexpected inputs provided: [\"bogus\"]"}`))
			return
		}
		want := map[string]string{"environment": "beta", "test_filter": "tests/auth/login.spec.ts"}
		if body.Ref != "master" || !reflect.DeepEqual(body.Inputs, want) {
			t.Errorf("body = %+v", body)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	ctx := context.Background()
	err := f.client.DispatchWorkflow(ctx, "laposa/shop-tests", 161335, "master",
		map[string]string{"environment": "beta", "test_filter": "tests/auth/login.spec.ts"})
	if err != nil {
		t.Fatal(err)
	}
	err = f.client.DispatchWorkflow(ctx, "laposa/shop-tests", 1, "master", nil)
	if err == nil || !strings.Contains(err.Error(), "Unexpected inputs") {
		t.Errorf("err = %v", err)
	}
}

func TestWorkflowRuns(t *testing.T) {
	f := newFakeAPI(t)
	f.handle("GET /repos/laposa/shop-tests/actions/workflows/161335/runs", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("event") != "workflow_dispatch" || q.Get("branch") != "master" || q.Get("created") != ">=2026-11-17T10:00:00Z" {
			t.Errorf("query = %v", q)
		}
		fixture(t, "workflow_runs.json")(w, r)
	})
	created := time.Date(2026, 11, 17, 11, 0, 0, 0, time.FixedZone("CET", 3600))
	runs, err := f.client.WorkflowRuns(context.Background(), "laposa/shop-tests", 161335,
		RunsFilter{Event: "workflow_dispatch", Branch: "master", Created: created})
	if err != nil {
		t.Fatal(err)
	}
	if len(runs) != 2 {
		t.Fatalf("runs = %d", len(runs))
	}
	r0, r1 := runs[0], runs[1]
	if r0.ID != 30433642 || r0.Status != "in_progress" || r0.Conclusion != "" || r0.HeadSHA != "acb5820ced9479c074f688cc328bf03f341a511d" ||
		!r0.CreatedAt.Equal(time.Date(2026, 11, 18, 10, 15, 3, 0, time.UTC)) {
		t.Errorf("run 0 = %+v", r0)
	}
	if r1.Status != "completed" || r1.Conclusion != "failure" || r1.RunAttempt != 2 || r1.Event != "workflow_dispatch" {
		t.Errorf("run 1 = %+v", r1)
	}
}

func TestRunJobs(t *testing.T) {
	f := newFakeAPI(t)
	f.handle("GET /repos/laposa/shop-tests/actions/runs/30433600/jobs", fixture(t, "jobs.json"))
	jobs, err := f.client.RunJobs(context.Background(), "laposa/shop-tests", 30433600)
	if err != nil {
		t.Fatal(err)
	}
	if len(jobs) != 2 {
		t.Fatalf("jobs = %d", len(jobs))
	}
	if j := jobs[0]; j.StartedAt == nil || !j.StartedAt.Equal(time.Date(2026, 11, 18, 9, 5, 10, 0, time.UTC)) ||
		j.CompletedAt == nil || j.Conclusion != "success" {
		t.Errorf("job 0 = %+v", j)
	}
	if j := jobs[1]; j.StartedAt != nil || j.CompletedAt != nil || j.Status != "queued" {
		t.Errorf("job 1 = %+v", j)
	}
}

func TestRunArtifacts(t *testing.T) {
	f := newFakeAPI(t)
	f.handle("GET /repos/laposa/shop-tests/actions/runs/30433600/artifacts", fixture(t, "artifacts.json"))
	arts, err := f.client.RunArtifacts(context.Background(), "laposa/shop-tests", 30433600)
	if err != nil {
		t.Fatal(err)
	}
	want := []Artifact{
		{ID: 11, Name: "test-results", SizeInBytes: 556},
		{ID: 13, Name: "screenshots", SizeInBytes: 453012, Expired: true},
	}
	if !reflect.DeepEqual(arts, want) {
		t.Errorf("artifacts = %+v", arts)
	}
}

// Downloads redirect to signed storage URLs on another host; the token must
// not be sent there (spec §9).
func TestDownloadsFollowRedirectWithoutToken(t *testing.T) {
	storage := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if auth := r.Header.Get("Authorization"); auth != "" {
			t.Errorf("storage got Authorization %q", auth)
		}
		w.Header().Set("Content-Type", "application/zip")
		w.Write([]byte("PK-" + r.URL.Path))
	}))
	defer storage.Close()

	f := newFakeAPI(t)
	redirect := func(to string) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, storage.URL+to+"?sig=abc", http.StatusFound)
		}
	}
	f.handle("GET /repos/laposa/shop-tests/actions/artifacts/11/zip", redirect("/artifact"))
	f.handle("GET /repos/laposa/shop-tests/actions/runs/30433600/logs", redirect("/logs"))
	ctx := context.Background()

	b, err := f.client.DownloadArtifact(ctx, "laposa/shop-tests", 11)
	if err != nil || string(b) != "PK-/artifact" {
		t.Errorf("DownloadArtifact = %q, %v", b, err)
	}
	b, err = f.client.DownloadRunLogs(ctx, "laposa/shop-tests", 30433600)
	if err != nil || string(b) != "PK-/logs" {
		t.Errorf("DownloadRunLogs = %q, %v", b, err)
	}
}

func TestDownloadTooLarge(t *testing.T) {
	f := newFakeAPI(t)
	f.handle("GET /repos/laposa/shop-tests/actions/runs/1/logs", func(w http.ResponseWriter, r *http.Request) {
		w.Write(bytes.Repeat([]byte("x"), 11))
	})
	f.client.maxDownload = 10
	if _, err := f.client.DownloadRunLogs(context.Background(), "laposa/shop-tests", 1); err == nil || !strings.Contains(err.Error(), "larger than 10 bytes") {
		t.Errorf("err = %v", err)
	}
}

// A 401 (e.g. the token was revoked) drops the cached token and retries
// once with a fresh one.
func TestUnauthorizedRefreshesToken(t *testing.T) {
	f := newFakeAPI(t)
	f.handle("GET /repos/laposa/shop-tests/actions/workflows", fixture(t, "workflows.json"))
	ctx := context.Background()
	if _, err := f.client.tokens.Token(ctx); err != nil {
		t.Fatal(err)
	}
	f.issued.Add(1) // GitHub now only accepts tok-2, which we don't have yet

	if _, err := f.client.Workflows(ctx, "laposa/shop-tests"); err != nil {
		t.Fatal(err)
	}
	if n := f.issued.Load(); n != 3 {
		t.Errorf("tokens issued = %d, want 3 (one refresh after 401)", n)
	}
}

func TestUnauthorizedTwiceFails(t *testing.T) {
	f := newFakeAPI(t)
	f.mux.HandleFunc("GET /repos/laposa/shop-tests/actions/workflows", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	})
	_, err := f.client.Workflows(context.Background(), "laposa/shop-tests")
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusUnauthorized {
		t.Fatalf("err = %v", err)
	}
	if n := f.issued.Load(); n != 2 {
		t.Errorf("tokens issued = %d, want 2", n)
	}
}

func TestInvalidArguments(t *testing.T) {
	f := newFakeAPI(t)
	ctx := context.Background()
	for _, repo := range []string{"", "laposa", "/x", "laposa/", "a/b/c", "../x", "laposa/.."} {
		if _, err := f.client.Workflows(ctx, repo); err == nil || !strings.Contains(err.Error(), "invalid repository") {
			t.Errorf("repo %q: err = %v", repo, err)
		}
	}
	for _, p := range []string{"", "tests/../../../user", "tests//x", "./tests"} {
		if _, err := f.client.File(ctx, "laposa/shop-tests", p, ""); err == nil || !strings.Contains(err.Error(), "invalid path") {
			t.Errorf("path %q: err = %v", p, err)
		}
	}
	if _, err := f.client.CommitSHA(ctx, "laposa/shop-tests", ""); err == nil {
		t.Error("empty ref accepted")
	}
	if n := f.issued.Load(); n != 0 {
		t.Errorf("invalid arguments reached GitHub (%d token exchanges)", n)
	}
}

func TestRequestsAreLoggedAtDebug(t *testing.T) {
	f := newFakeAPI(t)
	f.handle("GET /repos/laposa/shop-tests/actions/workflows", fixture(t, "workflows.json"))
	var buf bytes.Buffer
	f.client.logger = slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	if _, err := f.client.Workflows(context.Background(), "laposa/shop-tests"); err != nil {
		t.Fatal(err)
	}
	var lines []map[string]any
	for line := range strings.SplitSeq(strings.TrimSpace(buf.String()), "\n") {
		var m map[string]any
		if err := json.Unmarshal([]byte(line), &m); err != nil {
			t.Fatal(err)
		}
		lines = append(lines, m)
	}
	if len(lines) != 2 {
		t.Fatalf("log lines = %d, want 2 (token exchange, workflows)", len(lines))
	}
	l := lines[1]
	if l["level"] != "DEBUG" || l["msg"] != "github request" || l["http.request.method"] != "GET" ||
		l["url.path"] != "/repos/laposa/shop-tests/actions/workflows" || l["http.response.status_code"] != float64(200) {
		t.Errorf("log = %v", l)
	}
	if _, ok := l["http.client.request.duration"]; !ok {
		t.Error("no duration logged")
	}
	if strings.Contains(buf.String(), "tok-") {
		t.Error("token appears in logs")
	}
}

func TestHasNextPage(t *testing.T) {
	for link, want := range map[string]bool{
		"": false,
		`<https://api.github.com/x?page=2>; rel="next", <https://api.github.com/x?page=5>; rel="last"`:  true,
		`<https://api.github.com/x?page=1>; rel="prev", <https://api.github.com/x?page=1>; rel="first"`: false,
	} {
		if got := hasNextPage(link); got != want {
			t.Errorf("hasNextPage(%q) = %v", link, got)
		}
	}
}

// A list response without the expected key is an error, not an empty list.
func TestListMissingKeyIsError(t *testing.T) {
	f := newFakeAPI(t)
	f.handle("GET /repos/laposa/shop-tests/actions/workflows", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"total_count":0,"items":[]}`))
	})
	_, err := f.client.Workflows(context.Background(), "laposa/shop-tests")
	if err == nil || !strings.Contains(err.Error(), `response has no "workflows"`) {
		t.Fatalf("err = %v", err)
	}
}
