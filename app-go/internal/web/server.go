// Package web wires HTTP routes, middleware and handlers (spec §7, §9, §10).
// It depends on the domain packages; they never import it.
package web

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
	"time"

	"github.com/laposa/testflow/app-go/internal/auth"
	"github.com/laposa/testflow/app-go/internal/telemetry"
	"github.com/laposa/testflow/app-go/internal/web/static"
	"github.com/laposa/testflow/app-go/internal/web/views"
	g "maragu.dev/gomponents"
)

// maxBodyBytes limits request bodies. Forms and comments are small; nothing
// in TestFlow uploads files.
const maxBodyBytes = 1 << 20

// contentSecurityPolicy allows only same-origin resources (spec §9). There
// are no inline scripts or styles; htmx's runtime indicator CSS is switched
// off in the layout.
//
// form-action also allows github.com: browsers check form-action against
// redirects that follow a form submission, and POST /auth/redirect redirects
// to GitHub's OAuth page.
const contentSecurityPolicy = "default-src 'self'; base-uri 'none'; object-src 'none'; " +
	"frame-ancestors 'none'; form-action 'self' https://github.com"

// Pinger checks a dependency for GET /healthz, e.g. the database pool.
type Pinger interface {
	Ping(ctx context.Context) error
}

// Options configures the HTTP handler.
type Options struct {
	Logger      *slog.Logger
	Version     string // build version, used to bust asset caches
	Development bool   // disables long-lived caching of static files
	// StaticDir, if set, serves /static/ from this directory instead of the
	// embedded files, so edits show on reload (STATIC_DIR, development only).
	StaticDir string
	DB        Pinger // optional: /healthz pings it when set

	// BaseURL is the public origin (BASE_URL). Its origin is trusted by the
	// CSRF check, which matters when a proxy rewrites the Host header.
	BaseURL  *url.URL
	Sessions *auth.Sessions // web sessions and the auth middleware
	OAuth    OAuthProvider  // GitHub login
	Users    UserStore

	Discovery    TestDiscovery    // test picker (spec §8.1)
	TestSessions TestSessionStore // test sessions (spec §8.2)
	Environments []string         // TEST_SESSION_ENVIRONMENTS
	Activity     ActivityStore    // activity feed and comments (spec §8.9)

	Runs       RunStore      // run list and run detail (spec §8.4)
	Dispatcher RunDispatcher // starts automated runs
	Syncer     RunSyncer     // "refresh now" (spec §8.5)
	TestFiles  TestFiles     // raw test files at a run's commit
}

type server struct {
	Options
	static fs.FS
}

// NewHandler returns the root handler with all routes and middleware.
// Everything but Logger, Version, Development, StaticDir, DB and BaseURL
// is required.
func NewHandler(o Options) (http.Handler, error) {
	if o.Logger == nil {
		o.Logger = slog.Default()
	}
	if o.Sessions == nil || o.OAuth == nil || o.Users == nil {
		return nil, fmt.Errorf("web: Sessions, OAuth and Users are required")
	}
	if o.Discovery == nil || o.TestSessions == nil || len(o.Environments) == 0 || o.Activity == nil {
		return nil, fmt.Errorf("web: Discovery, TestSessions, Environments and Activity are required")
	}
	if o.Runs == nil || o.Dispatcher == nil || o.Syncer == nil || o.TestFiles == nil {
		return nil, fmt.Errorf("web: Runs, Dispatcher, Syncer and TestFiles are required")
	}
	s := &server{Options: o, static: static.FS}
	if o.StaticDir != "" {
		if info, err := os.Stat(o.StaticDir); err != nil || !info.IsDir() {
			return nil, fmt.Errorf("web: STATIC_DIR %q is not a directory", o.StaticDir)
		}
		s.static = os.DirFS(o.StaticDir)
	}

	csrf, err := s.crossOriginProtection()
	if err != nil {
		return nil, err
	}

	// Public routes (spec §7).
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.healthz)
	mux.HandleFunc("GET /static/{path...}", s.staticFile)
	mux.HandleFunc("GET /login", s.login)
	mux.HandleFunc("POST /auth/redirect", s.authRedirect)
	mux.HandleFunc("GET /auth/callback", s.authCallback)
	mux.HandleFunc("POST /auth/logout", s.logout)

	// Everything else requires a signed-in user.
	private := func(pattern string, h http.HandlerFunc) { mux.Handle(pattern, o.Sessions.Require(h)) }
	private("GET /{$}", s.listSessions)
	private("GET /sessions", s.listSessions)
	private("GET /sessions/new", s.newSession)
	private("POST /sessions", s.createSession)
	private("GET /sessions/{id}", s.showSession)
	private("POST /sessions/{id}/comments", s.addComment)
	private("GET /sessions/{id}/runs", s.showRuns)
	private("POST /sessions/{id}/runs", s.startRuns)
	private("POST /sessions/{id}/runs/sync", s.syncRuns)
	private("GET /sessions/{id}/runs/{run}", s.showRun)
	private("GET /sessions/{id}/runs/{run}/tests/{test}", s.testFile)

	// Telemetry wraps the mux with handlers that keep the same *http.Request,
	// so it can log the matched route pattern.
	return telemetry.Middleware(o.Logger, securityHeaders(limitBody(csrf.Handler(mux)))), nil
}

// crossOriginProtection rejects cross-origin non-GET requests (spec §9). It
// checks Sec-Fetch-Site, or Origin against Host for older browsers, so no
// CSRF tokens are needed in forms.
func (s *server) crossOriginProtection() (*http.CrossOriginProtection, error) {
	c := http.NewCrossOriginProtection()
	if s.BaseURL != nil && s.BaseURL.Host != "" {
		if err := c.AddTrustedOrigin(s.BaseURL.Scheme + "://" + s.BaseURL.Host); err != nil {
			return nil, fmt.Errorf("web: trusted origin: %w", err)
		}
	}
	c.SetDenyHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Logger.WarnContext(r.Context(), "cross-origin request rejected",
			"origin", r.Header.Get("Origin"), "sec_fetch_site", r.Header.Get("Sec-Fetch-Site"))
		http.Error(w, "Forbidden: cross-origin request", http.StatusForbidden)
	}))
	return c, nil
}

// NewServer returns an http.Server with timeouts suited to TestFlow's
// short, server-rendered requests (spec §9).
func NewServer(addr string, h http.Handler, logger *slog.Logger) *http.Server {
	return &http.Server{
		Addr:              addr,
		Handler:           h,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    64 << 10,
		ErrorLog:          slog.NewLogLogger(logger.Handler(), slog.LevelWarn),
	}
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hdr := w.Header()
		hdr.Set("Content-Security-Policy", contentSecurityPolicy)
		hdr.Set("X-Content-Type-Options", "nosniff")
		hdr.Set("Referrer-Policy", "same-origin")
		hdr.Set("X-Frame-Options", "DENY")
		next.ServeHTTP(w, r)
	})
}

// limitBody caps the request body. It replaces the body in place rather than
// using http.MaxBytesHandler, which copies the request and would hide the
// matched route pattern from the telemetry middleware.
func limitBody(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Body != nil {
			r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
		}
		next.ServeHTTP(w, r)
	})
}

// page returns the layout data for r, including the signed-in user's name.
func (s *server) page(r *http.Request, title string) views.Page {
	p := views.Page{Title: title, AssetVersion: s.Version}
	if u, ok := auth.UserFrom(r.Context()); ok {
		p.UserName = u.Name
	}
	return p
}

// render writes a Gomponents node as an HTML response.
func (s *server) render(w http.ResponseWriter, r *http.Request, status int, n g.Node) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	if err := n.Render(w); err != nil {
		s.Logger.ErrorContext(r.Context(), "render failed", "err", err)
	}
}

// healthz reports liveness and, once configured, database reachability.
func (s *server) healthz(w http.ResponseWriter, r *http.Request) {
	status, body := http.StatusOK, map[string]string{"status": "ok"}
	if s.DB != nil {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := s.DB.Ping(ctx); err != nil {
			s.Logger.WarnContext(r.Context(), "health check failed", "check", "database", "err", err)
			status, body = http.StatusServiceUnavailable, map[string]string{"status": "unavailable", "database": "unreachable"}
		}
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

// staticFile serves embedded assets. Versioned URLs (the htmx file name, or
// a ?v= query added by the layout) are cached for a year; others for a day.
// Directories are not listed.
func (s *server) staticFile(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("path")
	if !fs.ValidPath(name) || name == "." {
		http.NotFound(w, r)
		return
	}
	if info, err := fs.Stat(s.static, name); err != nil || info.IsDir() {
		http.NotFound(w, r)
		return
	}
	switch {
	case s.Development:
		w.Header().Set("Cache-Control", "no-cache")
	case r.URL.Query().Has("v") || strings.HasPrefix(path.Base(name), "htmx-"):
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	default:
		w.Header().Set("Cache-Control", "public, max-age=86400")
	}
	http.ServeFileFS(w, r, s.static, name)
}
