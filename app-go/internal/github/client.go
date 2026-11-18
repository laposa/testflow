// Package github is the GitHub App REST client (spec §9): the app JWT, the
// cached installation token, and the REST endpoints used by discovery,
// dispatch and the run syncer (spec §8.1, §8.4, §8.5).
//
// Domain packages do not import this package's concrete Client in their
// logic; they declare the small interface they need, which *Client
// satisfies, so tests can use fakes (spec §4).
package github

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// DefaultAPIURL is GitHub's REST API.
const DefaultAPIURL = "https://api.github.com"

// apiVersion is sent as X-GitHub-Api-Version.
const apiVersion = "2022-11-28"

// Defaults for Config.
const (
	DefaultTimeout         = 30 * time.Second
	DefaultDownloadTimeout = 5 * time.Minute
	DefaultMaxDownload     = 100 << 20 // 100 MiB
)

// Limits for JSON responses and error bodies read into memory.
const (
	maxJSONBody  = 10 << 20
	maxErrorBody = 64 << 10
	maxRedirects = 10
	maxPages     = 50 // ×100 items: far beyond what TestFlow lists
)

// ErrNotFound matches (with errors.Is) an *APIError with status 404, e.g. a
// repository without a tests/ directory (spec §8.1).
var ErrNotFound = errors.New("github: not found")

// APIError is a non-2xx response from GitHub.
type APIError struct {
	Method     string
	Path       string
	StatusCode int
	Message    string // GitHub's "message" field, if any
}

func (e *APIError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("github: %s %s: status %d: %s", e.Method, e.Path, e.StatusCode, e.Message)
	}
	return fmt.Sprintf("github: %s %s: status %d", e.Method, e.Path, e.StatusCode)
}

// Is makes errors.Is(err, ErrNotFound) true for 404 responses.
func (e *APIError) Is(target error) bool {
	return target == ErrNotFound && e.StatusCode == http.StatusNotFound
}

// Config configures a Client. The first three fields are required.
type Config struct {
	ClientID       string // GitHub App client id, used as the JWT iss
	InstallationID int64
	PrivateKey     []byte // PEM

	// Optional.
	APIURL          string        // default DefaultAPIURL; tests use httptest
	HTTPClient      *http.Client  // default http.DefaultClient's settings
	Logger          *slog.Logger  // default slog.Default()
	Timeout         time.Duration // per API call, default DefaultTimeout
	DownloadTimeout time.Duration // per log/artifact download, default DefaultDownloadTimeout
	MaxDownload     int64         // bytes, default DefaultMaxDownload
	Now             func() time.Time
}

// Client calls the GitHub REST API as the app installation. It is safe for
// concurrent use.
type Client struct {
	apiURL          string
	http            *http.Client
	logger          *slog.Logger
	timeout         time.Duration
	downloadTimeout time.Duration
	maxDownload     int64
	tokens          *installationTokens
}

// New returns a Client. It fails if the private key cannot be parsed, so a
// bad key is found at startup, not on the first API call.
func New(c Config) (*Client, error) {
	if c.ClientID == "" || c.InstallationID <= 0 {
		return nil, errors.New("github: client id and installation id are required")
	}
	key, err := ParsePrivateKey(c.PrivateKey)
	if err != nil {
		return nil, err
	}
	cl := &Client{
		apiURL:          strings.TrimRight(orDefault(c.APIURL, DefaultAPIURL), "/"),
		logger:          c.Logger,
		timeout:         c.Timeout,
		downloadTimeout: c.DownloadTimeout,
		maxDownload:     c.MaxDownload,
	}
	if cl.logger == nil {
		cl.logger = slog.Default()
	}
	if cl.timeout <= 0 {
		cl.timeout = DefaultTimeout
	}
	if cl.downloadTimeout <= 0 {
		cl.downloadTimeout = DefaultDownloadTimeout
	}
	if cl.maxDownload <= 0 {
		cl.maxDownload = DefaultMaxDownload
	}

	// Copy the HTTP client so its redirect policy can be set without
	// changing the caller's. Log and artifact downloads redirect to signed
	// storage URLs on another host, which must not receive our token.
	hc := http.Client{}
	if c.HTTPClient != nil {
		hc = *c.HTTPClient
	}
	hc.CheckRedirect = stripAuthOnRedirect
	cl.http = &hc

	now := c.Now
	if now == nil {
		now = time.Now
	}
	cl.tokens = &installationTokens{
		clientID:       c.ClientID,
		installationID: c.InstallationID,
		key:            key,
		client:         cl,
		now:            now,
	}
	return cl, nil
}

// stripAuthOnRedirect follows up to maxRedirects redirects and drops the
// Authorization header whenever the target host (including port) differs
// from the original request's.
func stripAuthOnRedirect(req *http.Request, via []*http.Request) error {
	if len(via) >= maxRedirects {
		return fmt.Errorf("stopped after %d redirects", maxRedirects)
	}
	if req.URL.Host != via[0].URL.Host {
		req.Header.Del("Authorization")
	}
	return nil
}

// request describes one API call. path is the escaped URL path.
type request struct {
	method   string
	path     string
	query    url.Values
	body     any    // JSON-encoded when non-nil
	accept   string // default application/vnd.github+json
	bearer   string // explicit token (the app JWT); default the installation token
	download bool   // use the download timeout
}

// do sends r and, for a 2xx response, calls handle with it. The response
// body is closed afterwards. A 401 with the installation token invalidates
// the cached token and retries once with a new one.
func (c *Client) do(ctx context.Context, r request, handle func(*http.Response) error) error {
	var body []byte
	if r.body != nil {
		var err error
		if body, err = json.Marshal(r.body); err != nil {
			return fmt.Errorf("github: encode %s %s: %w", r.method, r.path, err)
		}
	}
	timeout := c.timeout
	if r.download {
		timeout = c.downloadTimeout
	}
	for attempt := 0; ; attempt++ {
		token := r.bearer
		if token == "" {
			var err error
			if token, err = c.tokens.Token(ctx); err != nil {
				return err
			}
		}
		err := c.send(ctx, timeout, r, body, token, handle)
		var apiErr *APIError
		if r.bearer == "" && attempt == 0 && errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusUnauthorized {
			c.tokens.invalidate(token)
			continue
		}
		return err
	}
}

func (c *Client) send(ctx context.Context, timeout time.Duration, r request, body []byte, token string, handle func(*http.Response) error) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	u := c.apiURL + r.path
	if len(r.query) > 0 {
		u += "?" + r.query.Encode()
	}
	var rb io.Reader
	if body != nil {
		rb = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, r.method, u, rb)
	if err != nil {
		return fmt.Errorf("github: %s %s: %w", r.method, r.path, err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", orDefault(r.accept, "application/vnd.github+json"))
	req.Header.Set("X-GitHub-Api-Version", apiVersion)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	start := time.Now()
	resp, err := c.http.Do(req)
	status := 0
	if resp != nil {
		status = resp.StatusCode
	}
	c.logger.LogAttrs(ctx, slog.LevelDebug, "github request",
		slog.String("http.request.method", r.method),
		slog.String("url.path", r.path),
		slog.Int("http.response.status_code", status),
		slog.Float64("http.client.request.duration", time.Since(start).Seconds()),
	)
	if err != nil {
		return fmt.Errorf("github: %s %s: %w", r.method, r.path, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode/100 != 2 {
		apiErr := &APIError{Method: r.method, Path: r.path, StatusCode: resp.StatusCode}
		var msg struct {
			Message string `json:"message"`
		}
		if json.NewDecoder(io.LimitReader(resp.Body, maxErrorBody)).Decode(&msg) == nil {
			apiErr.Message = msg.Message
		}
		return apiErr
	}
	if handle == nil {
		return nil
	}
	if err := handle(resp); err != nil {
		return fmt.Errorf("github: %s %s: %w", r.method, r.path, err)
	}
	return nil
}

// decodeJSON returns a handler that decodes the response body into v.
func decodeJSON(v any) func(*http.Response) error {
	return func(resp *http.Response) error {
		if err := json.NewDecoder(io.LimitReader(resp.Body, maxJSONBody)).Decode(v); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
		return nil
	}
}

// readAll returns a handler that reads at most limit bytes of the body into
// *dst and fails if the body is longer.
func readAll(dst *[]byte, limit int64) func(*http.Response) error {
	return func(resp *http.Response) error {
		b, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
		if err != nil {
			return fmt.Errorf("read response: %w", err)
		}
		if int64(len(b)) > limit {
			return fmt.Errorf("response larger than %d bytes", limit)
		}
		*dst = b
		return nil
	}
}

// listAll fetches every page (per_page=100) of a list endpoint whose items
// are under key in the response object (e.g. "repositories"), following the
// Link header's rel="next".
func listAll[T any](ctx context.Context, c *Client, path string, query url.Values, key string) ([]T, error) {
	q := url.Values{}
	for k, v := range query {
		q[k] = v
	}
	q.Set("per_page", "100")
	var all []T
	for page := 1; ; page++ {
		if page > maxPages {
			return nil, fmt.Errorf("github: GET %s: more than %d pages", path, maxPages)
		}
		q.Set("page", strconv.Itoa(page))
		var raw map[string]json.RawMessage
		var next bool
		err := c.do(ctx, request{method: http.MethodGet, path: path, query: q}, func(resp *http.Response) error {
			next = hasNextPage(resp.Header.Get("Link"))
			return decodeJSON(&raw)(resp)
		})
		if err != nil {
			return nil, err
		}
		var items []T
		if v, ok := raw[key]; ok {
			if err := json.Unmarshal(v, &items); err != nil {
				return nil, fmt.Errorf("github: GET %s: decode %s: %w", path, key, err)
			}
		}
		all = append(all, items...)
		if !next || len(items) == 0 {
			return all, nil
		}
	}
}

// hasNextPage reports whether a Link header has a rel="next" entry.
func hasNextPage(link string) bool {
	for part := range strings.SplitSeq(link, ",") {
		for param := range strings.SplitSeq(part, ";") {
			if strings.TrimSpace(param) == `rel="next"` {
				return true
			}
		}
	}
	return false
}

func orDefault(v, def string) string {
	if v != "" {
		return v
	}
	return def
}
