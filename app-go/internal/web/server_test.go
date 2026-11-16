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
	"strings"
	"testing"

	"github.com/laposa/testflow/app-go/internal/telemetry"
	"github.com/laposa/testflow/app-go/internal/web/views"
)

type fakePinger struct{ err error }

func (f fakePinger) Ping(context.Context) error { return f.err }

func newTestHandler(t *testing.T, o Options) (http.Handler, *bytes.Buffer) {
	t.Helper()
	var logs bytes.Buffer
	o.Logger = slog.New(telemetry.NewHandler(&logs, slog.LevelInfo, telemetry.Resource{ServiceName: "testflow"}))
	if o.Version == "" {
		o.Version = "test"
	}
	return NewHandler(o), &logs
}

func get(h http.Handler, target string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, target, nil))
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

func TestHomePage(t *testing.T) {
	h, _ := newTestHandler(t, Options{Version: "abc123"})
	rec := get(h, "/")
	if rec.Code != 200 {
		t.Fatalf("status = %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "text/html; charset=utf-8" {
		t.Errorf("Content-Type = %q", ct)
	}
	body := rec.Body.String()
	for _, want := range []string{"<!doctype html>", "/static/" + views.HTMXFile, "/static/app.css?v=abc123"} {
		if !strings.Contains(body, want) {
			t.Errorf("home page missing %q", want)
		}
	}
	if rec := get(h, "/does-not-exist"); rec.Code != http.StatusNotFound {
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
