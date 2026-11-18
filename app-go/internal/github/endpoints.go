package github

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Repository is an entry of GET /installation/repositories.
type Repository struct {
	ID            int64  `json:"id"`
	Name          string `json:"name"`
	FullName      string `json:"full_name"` // owner/repo
	DefaultBranch string `json:"default_branch"`
	Archived      bool   `json:"archived"`
}

// ContentEntry is an entry of a directory listing from the contents API.
type ContentEntry struct {
	Type string `json:"type"` // file, dir, symlink or submodule
	Name string `json:"name"`
	Path string `json:"path"`
	SHA  string `json:"sha"`
	Size int64  `json:"size"`
}

// Workflow is an entry of GET /repos/{repo}/actions/workflows.
type Workflow struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Path  string `json:"path"` // e.g. .github/workflows/run-tests-web.yaml
	State string `json:"state"`
}

// WorkflowRun is an entry of GET /repos/{repo}/actions/workflows/{id}/runs.
type WorkflowRun struct {
	ID         int64     `json:"id"`
	Name       string    `json:"name"`
	Event      string    `json:"event"`
	Status     string    `json:"status"`     // queued, in_progress, completed, ...
	Conclusion string    `json:"conclusion"` // empty until completed
	HeadBranch string    `json:"head_branch"`
	HeadSHA    string    `json:"head_sha"`
	RunNumber  int64     `json:"run_number"`
	RunAttempt int       `json:"run_attempt"`
	HTMLURL    string    `json:"html_url"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// Job is an entry of GET /repos/{repo}/actions/runs/{id}/jobs.
type Job struct {
	ID          int64      `json:"id"`
	Name        string     `json:"name"`
	Status      string     `json:"status"`
	Conclusion  string     `json:"conclusion"`
	StartedAt   *time.Time `json:"started_at"`
	CompletedAt *time.Time `json:"completed_at"` // nil while running
}

// Artifact is an entry of GET /repos/{repo}/actions/runs/{id}/artifacts.
type Artifact struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	SizeInBytes int64  `json:"size_in_bytes"`
	Expired     bool   `json:"expired"`
}

// RunsFilter narrows WorkflowRuns. Zero fields are not sent.
type RunsFilter struct {
	Event   string    // e.g. workflow_dispatch
	Branch  string    // head branch
	Created time.Time // only runs created at or after this time
}

// Repositories lists all repositories the installation can access
// (spec §8.1 step 1).
func (c *Client) Repositories(ctx context.Context) ([]Repository, error) {
	return listAll[Repository](ctx, c, "/installation/repositories", nil, "repositories")
}

// CommitSHA returns the SHA of the commit ref points to, e.g. the latest
// commit of master (spec §8.1 step 2). It asks for the bare SHA media type,
// so GitHub does not send the full commit with its diff.
func (c *Client) CommitSHA(ctx context.Context, repo, ref string) (string, error) {
	rp, err := repoPath(repo)
	if err != nil {
		return "", err
	}
	ep, err := escapePath(ref)
	if err != nil {
		return "", err
	}
	var b []byte
	r := request{method: http.MethodGet, path: rp + "/commits/" + ep, accept: "application/vnd.github.sha"}
	if err := c.do(ctx, r, readAll(&b, 1<<10)); err != nil {
		return "", err
	}
	sha := strings.TrimSpace(string(b))
	if sha == "" {
		return "", fmt.Errorf("github: GET %s: empty sha", r.path)
	}
	return sha, nil
}

// Dir lists the directory at path at ref (spec §8.1 step 3). A missing path
// gives an error matching ErrNotFound. A path that is a file is an error.
func (c *Client) Dir(ctx context.Context, repo, path, ref string) ([]ContentEntry, error) {
	r, err := contentsRequest(repo, path, ref)
	if err != nil {
		return nil, err
	}
	var raw json.RawMessage
	if err := c.do(ctx, r, decodeJSON(&raw)); err != nil {
		return nil, err
	}
	if t := strings.TrimSpace(string(raw)); !strings.HasPrefix(t, "[") {
		return nil, fmt.Errorf("github: GET %s: not a directory", r.path)
	}
	var entries []ContentEntry
	if err := json.Unmarshal(raw, &entries); err != nil {
		return nil, fmt.Errorf("github: GET %s: decode response: %w", r.path, err)
	}
	return entries, nil
}

// File returns the raw content of the file at path at ref: manual test
// cases (spec §8.8) and the raw test file route (spec §7).
func (c *Client) File(ctx context.Context, repo, path, ref string) ([]byte, error) {
	r, err := contentsRequest(repo, path, ref)
	if err != nil {
		return nil, err
	}
	r.accept = "application/vnd.github.raw+json"
	var b []byte
	if err := c.do(ctx, r, readAll(&b, c.maxDownload)); err != nil {
		return nil, err
	}
	return b, nil
}

// Workflows lists the repository's workflows (spec §8.1 step 4).
func (c *Client) Workflows(ctx context.Context, repo string) ([]Workflow, error) {
	rp, err := repoPath(repo)
	if err != nil {
		return nil, err
	}
	return listAll[Workflow](ctx, c, rp+"/actions/workflows", nil, "workflows")
}

// DispatchWorkflow starts a workflow_dispatch run of workflowID on ref with
// the given inputs (spec §8.4). GitHub returns no run id.
func (c *Client) DispatchWorkflow(ctx context.Context, repo string, workflowID int64, ref string, inputs map[string]string) error {
	rp, err := repoPath(repo)
	if err != nil {
		return err
	}
	if inputs == nil {
		inputs = map[string]string{}
	}
	return c.do(ctx, request{
		method: http.MethodPost,
		path:   rp + "/actions/workflows/" + strconv.FormatInt(workflowID, 10) + "/dispatches",
		body:   map[string]any{"ref": ref, "inputs": inputs},
	}, nil)
}

// WorkflowRuns lists all runs of workflowID matching f, newest first
// (spec §8.4 matching, §8.5).
func (c *Client) WorkflowRuns(ctx context.Context, repo string, workflowID int64, f RunsFilter) ([]WorkflowRun, error) {
	rp, err := repoPath(repo)
	if err != nil {
		return nil, err
	}
	q := url.Values{}
	if f.Event != "" {
		q.Set("event", f.Event)
	}
	if f.Branch != "" {
		q.Set("branch", f.Branch)
	}
	if !f.Created.IsZero() {
		q.Set("created", ">="+f.Created.UTC().Format(time.RFC3339))
	}
	return listAll[WorkflowRun](ctx, c, rp+"/actions/workflows/"+strconv.FormatInt(workflowID, 10)+"/runs", q, "workflow_runs")
}

// RunJobs lists the jobs of a run's latest attempt; started_at and
// finished_at come from them (spec §8.4).
func (c *Client) RunJobs(ctx context.Context, repo string, runID int64) ([]Job, error) {
	rp, err := repoPath(repo)
	if err != nil {
		return nil, err
	}
	return listAll[Job](ctx, c, rp+"/actions/runs/"+strconv.FormatInt(runID, 10)+"/jobs", nil, "jobs")
}

// RunArtifacts lists a run's artifacts, e.g. to find test-results (spec §8.4).
func (c *Client) RunArtifacts(ctx context.Context, repo string, runID int64) ([]Artifact, error) {
	rp, err := repoPath(repo)
	if err != nil {
		return nil, err
	}
	return listAll[Artifact](ctx, c, rp+"/actions/runs/"+strconv.FormatInt(runID, 10)+"/artifacts", nil, "artifacts")
}

// DownloadArtifact returns an artifact's zip file. GitHub redirects to a
// signed storage URL, which is followed without our token.
func (c *Client) DownloadArtifact(ctx context.Context, repo string, artifactID int64) ([]byte, error) {
	rp, err := repoPath(repo)
	if err != nil {
		return nil, err
	}
	return c.download(ctx, rp+"/actions/artifacts/"+strconv.FormatInt(artifactID, 10)+"/zip")
}

// DownloadRunLogs returns the zip of a run's logs (spec §8.4 run_log).
func (c *Client) DownloadRunLogs(ctx context.Context, repo string, runID int64) ([]byte, error) {
	rp, err := repoPath(repo)
	if err != nil {
		return nil, err
	}
	return c.download(ctx, rp+"/actions/runs/"+strconv.FormatInt(runID, 10)+"/logs")
}

func (c *Client) download(ctx context.Context, path string) ([]byte, error) {
	var b []byte
	r := request{method: http.MethodGet, path: path, download: true}
	if err := c.do(ctx, r, readAll(&b, c.maxDownload)); err != nil {
		return nil, err
	}
	return b, nil
}

func contentsRequest(repo, path, ref string) (request, error) {
	rp, err := repoPath(repo)
	if err != nil {
		return request{}, err
	}
	ep, err := escapePath(strings.Trim(path, "/"))
	if err != nil {
		return request{}, err
	}
	r := request{method: http.MethodGet, path: rp + "/contents/" + ep}
	if ref != "" {
		r.query = url.Values{"ref": {ref}}
	}
	return r, nil
}

// repoPath returns /repos/{owner}/{repo} for "owner/repo", escaped.
func repoPath(repo string) (string, error) {
	owner, name, ok := strings.Cut(repo, "/")
	if !ok || owner == "" || name == "" || strings.Contains(name, "/") || owner == ".." || name == ".." || owner == "." || name == "." {
		return "", fmt.Errorf("github: invalid repository %q, want owner/repo", repo)
	}
	return "/repos/" + url.PathEscape(owner) + "/" + url.PathEscape(name), nil
}

// escapePath escapes each segment of a slash-separated path. Empty, "." and
// ".." segments are rejected so a stored path cannot address another
// endpoint.
func escapePath(p string) (string, error) {
	segs := strings.Split(p, "/")
	for i, s := range segs {
		if s == "" || s == "." || s == ".." {
			return "", fmt.Errorf("github: invalid path %q", p)
		}
		segs[i] = url.PathEscape(s)
	}
	return strings.Join(segs, "/"), nil
}
