package web

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/laposa/testflow/app-go/internal/auth"
	"github.com/laposa/testflow/app-go/internal/auth/authtest"
	"github.com/laposa/testflow/app-go/internal/store"
	"github.com/laposa/testflow/app-go/internal/telemetry"
	"github.com/laposa/testflow/app-go/internal/web/views"
)

type fakePinger struct{ err error }

func (f fakePinger) Ping(context.Context) error { return f.err }

// fakeOAuth stands in for GitHub: code "good" signs in as gh.
type fakeOAuth struct{ gh store.GitHubUser }

func (f fakeOAuth) AuthCodeURL(state string) string {
	return "https://github.test/authorize?state=" + state
}

func (f fakeOAuth) Authenticate(_ context.Context, code string) (store.GitHubUser, error) {
	if code != "good" {
		return store.GitHubUser{}, errors.New("bad_verification_code")
	}
	return f.gh, nil
}

// testEnv is a handler with in-memory users and sessions.
type testEnv struct {
	h            http.Handler
	logs         *bytes.Buffer
	store        *authtest.Store
	discovery    *fakeDiscovery    // set unless Options.Discovery was given
	testSessions *fakeTestSessions // set unless Options.TestSessions was given
	activity     *fakeActivity     // set unless Options.Activity was given
}

func newEnv(t *testing.T, o Options) *testEnv {
	t.Helper()
	var logs bytes.Buffer
	o.Logger = slog.New(telemetry.NewHandler(&logs, slog.LevelInfo, telemetry.Resource{ServiceName: "testflow"}))
	if o.Version == "" {
		o.Version = "test"
	}
	st := authtest.NewStore()
	if o.Sessions == nil {
		o.Sessions = &auth.Sessions{Store: st, Lifetime: time.Hour, Secure: true, Logger: o.Logger}
	}
	if o.OAuth == nil {
		o.OAuth = fakeOAuth{store.GitHubUser{ID: 99, Login: "ada", Name: "Ada Lovelace"}}
	}
	if o.Users == nil {
		o.Users = st
	}
	e := &testEnv{logs: &logs, store: st}
	if o.Discovery == nil {
		e.discovery = &fakeDiscovery{res: sampleTests()}
		o.Discovery = e.discovery
	}
	if o.TestSessions == nil {
		e.testSessions = newFakeTestSessions(st)
		o.TestSessions = e.testSessions
	}
	if o.Activity == nil {
		e.activity = &fakeActivity{sessions: e.testSessions, entries: map[int64][]store.Activity{}}
		o.Activity = e.activity
	}
	if o.Environments == nil {
		o.Environments = []string{"preprod", "beta", "prod"}
	}
	h, err := NewHandler(o)
	if err != nil {
		t.Fatal(err)
	}
	e.h = h
	return e
}

func newTestHandler(t *testing.T, o Options) (http.Handler, *bytes.Buffer) {
	t.Helper()
	e := newEnv(t, o)
	return e.h, e.logs
}

// signIn stores a user with a session and returns its cookie.
func (e *testEnv) signIn(name string) *http.Cookie {
	id := int64(len(e.store.Users) + 1)
	e.store.Users[id] = store.User{ID: id, GitHubID: 1000 + id, Name: name, Login: "u"}
	return &http.Cookie{Name: auth.CookieName, Value: e.store.Add(id, time.Now().Add(time.Hour))}
}

func get(h http.Handler, target string, cookies ...*http.Cookie) *httptest.ResponseRecorder {
	r := httptest.NewRequest(http.MethodGet, target, nil)
	for _, c := range cookies {
		r.AddCookie(c)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, r)
	return rec
}

func TestHealthz(t *testing.T) {
	tests := []struct {
		name       string
		db         Pinger
		wantStatus int
		wantBody   string
	}{
		{"no database configured", nil, 200, "ok"},
		{"database up", fakePinger{}, 200, "ok"},
		{"database down", fakePinger{errors.New("connection refused")}, 503, "unavailable"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, _ := newTestHandler(t, Options{DB: tt.db})
			rec := get(h, "/healthz")
			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			var body map[string]string
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if body["status"] != tt.wantBody {
				t.Errorf("body = %v", body)
			}
			if strings.Contains(rec.Body.String(), "refused") {
				t.Error("health response leaks error details")
			}
		})
	}
}

func TestHealthzRejectsOtherMethods(t *testing.T) {
	h, _ := newTestHandler(t, Options{})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/healthz", nil))
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("POST /healthz = %d", rec.Code)
	}
}

func TestSecurityHeaders(t *testing.T) {
	h, _ := newTestHandler(t, Options{})
	rec := get(h, "/")
	for name, want := range map[string]string{
		"Content-Security-Policy": contentSecurityPolicy,
		"X-Content-Type-Options":  "nosniff",
		"Referrer-Policy":         "same-origin",
	} {
		if got := rec.Header().Get(name); got != want {
			t.Errorf("%s = %q, want %q", name, got, want)
		}
	}
	if !strings.Contains(contentSecurityPolicy, "default-src 'self'") {
		t.Error("CSP must be self only")
	}
}

func TestNewHandlerRequiresAuth(t *testing.T) {
	if _, err := NewHandler(Options{}); err == nil {
		t.Error("NewHandler without Sessions/OAuth/Users should fail")
	}
}

func TestHomePage(t *testing.T) {
	e := newEnv(t, Options{Version: "abc123"})
	h := e.h
	rec := get(h, "/", e.signIn("Ada <Lovelace>"))
	if rec.Code != 200 {
		t.Fatalf("status = %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "text/html; charset=utf-8" {
		t.Errorf("Content-Type = %q", ct)
	}
	body := rec.Body.String()
	for _, want := range []string{"<!doctype html>", "/static/" + views.HTMXFile, "/static/app.css?v=abc123",
		"Ada &lt;Lovelace&gt;", `action="/auth/logout"`, `<h2>All Sessions</h2>`} {
		if !strings.Contains(body, want) {
			t.Errorf("home page missing %q", want)
		}
	}
	if rec := get(h, "/does-not-exist", e.signIn("x")); rec.Code != http.StatusNotFound {
		t.Errorf("unknown path = %d, want 404", rec.Code)
	}
}

func TestStatic(t *testing.T) {
	h, _ := newTestHandler(t, Options{})
	tests := []struct {
		target, wantType, wantCache string
		wantStatus                  int
	}{
		{"/static/" + views.HTMXFile, "text/javascript; charset=utf-8", "public, max-age=31536000, immutable", 200},
		{"/static/app.css?v=1", "text/css; charset=utf-8", "public, max-age=31536000, immutable", 200},
		{"/static/app.js", "text/javascript; charset=utf-8", "public, max-age=86400", 200},
		{"/static/fonts/DMSans-VariableFont.ttf", "font/ttf", "public, max-age=86400", 200},
		{"/static/icons/logo_check.svg", "image/svg+xml", "public, max-age=86400", 200},
		{"/static/", "", "", 404},
		{"/static/fonts", "", "", 404},
		{"/static/missing.css", "", "", 404},
		{"/static/missing/../../static.go", "", "", http.StatusTemporaryRedirect}, // ServeMux cleans the path to /static.go, which is a 404
		{"/static.go", "", "", 404},
	}
	for _, tt := range tests {
		t.Run(tt.target, func(t *testing.T) {
			rec := get(h, tt.target)
			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if tt.wantStatus != 200 {
				return
			}
			if got := rec.Header().Get("Content-Type"); got != tt.wantType {
				t.Errorf("Content-Type = %q, want %q", got, tt.wantType)
			}
			if got := rec.Header().Get("Cache-Control"); got != tt.wantCache {
				t.Errorf("Cache-Control = %q, want %q", got, tt.wantCache)
			}
			if rec.Body.Len() == 0 {
				t.Error("empty body")
			}
		})
	}
}

func TestStaticDevelopmentNoCache(t *testing.T) {
	h, _ := newTestHandler(t, Options{Development: true})
	if got := get(h, "/static/app.css?v=1").Header().Get("Cache-Control"); got != "no-cache" {
		t.Errorf("Cache-Control = %q, want no-cache", got)
	}
}

func TestStaticDir(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "app.css"), []byte("body{color:red}"), 0o600); err != nil {
		t.Fatal(err)
	}
	h, _ := newTestHandler(t, Options{Development: true, StaticDir: dir})
	rec := get(h, "/static/app.css?v=1")
	if rec.Code != http.StatusOK || rec.Body.String() != "body{color:red}" {
		t.Fatalf("GET = %d %q, want the file from STATIC_DIR", rec.Code, rec.Body.String())
	}
	// An edit shows on the next request, without a restart.
	if err := os.WriteFile(filepath.Join(dir, "app.css"), []byte("body{color:blue}"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := get(h, "/static/app.css").Body.String(); got != "body{color:blue}" {
		t.Errorf("after an edit = %q", got)
	}
	if rec := get(h, "/static/missing.css"); rec.Code != http.StatusNotFound {
		t.Errorf("missing file = %d, want 404", rec.Code)
	}

	o := Options{StaticDir: filepath.Join(dir, "nope")}
	o.Sessions, o.OAuth, o.Users = &auth.Sessions{}, fakeOAuth{}, authtest.NewStore()
	o.Discovery, o.TestSessions, o.Activity, o.Environments = &fakeDiscovery{}, &fakeTestSessions{}, &fakeActivity{}, []string{"x"}
	if _, err := NewHandler(o); err == nil {
		t.Error("a missing STATIC_DIR should fail at startup")
	}
}

func TestRequestLogUsesRoutePattern(t *testing.T) {
	h, logs := newTestHandler(t, Options{})
	get(h, "/static/app.css")
	var line struct {
		Attributes map[string]any `json:"attributes"`
	}
	if err := json.Unmarshal(logs.Bytes(), &line); err != nil {
		t.Fatalf("log line: %v\n%s", err, logs)
	}
	if got := line.Attributes["http.route"]; got != "GET /static/{path...}" {
		t.Errorf("http.route = %v", got)
	}
}

func TestBodyLimit(t *testing.T) {
	var readErr error
	h := limitBody(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, readErr = io.ReadAll(r.Body)
	}))
	big := strings.NewReader(strings.Repeat("x", maxBodyBytes+1))
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/", big))
	var mbe *http.MaxBytesError
	if !errors.As(readErr, &mbe) {
		t.Errorf("reading an oversized body: err = %v, want MaxBytesError", readErr)
	}
}
