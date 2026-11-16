// Package web wires HTTP routes, middleware and handlers (spec §7, §9, §10).
// It depends on the domain packages; they never import it.
package web

import (
	"context"
	"encoding/json"
	"io/fs"
	"log/slog"
	"net/http"
	"path"
	"strings"
	"time"

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
const contentSecurityPolicy = "default-src 'self'; base-uri 'none'; object-src 'none'; " +
	"frame-ancestors 'none'; form-action 'self'"

// Pinger checks a dependency for GET /healthz, e.g. the database pool.
type Pinger interface {
	Ping(ctx context.Context) error
}

// Options configures the HTTP handler.
type Options struct {
	Logger      *slog.Logger
	Version     string // build version, used to bust asset caches
	Development bool   // disables long-lived caching of static files
	DB          Pinger // optional until the database lands (iteration 2)
}

type server struct {
	Options
	static fs.FS
}

// NewHandler returns the root handler with all routes and middleware.
func NewHandler(o Options) http.Handler {
	if o.Logger == nil {
		o.Logger = slog.Default()
	}
	s := &server{Options: o, static: static.FS}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.healthz)
	mux.HandleFunc("GET /static/{path...}", s.staticFile)
	mux.HandleFunc("GET /{$}", s.home)

	// Telemetry wraps the mux with handlers that keep the same *http.Request,
	// so it can log the matched route pattern.
	return telemetry.Middleware(o.Logger, securityHeaders(limitBody(mux)))
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

func (s *server) page(title string) views.Page {
	return views.Page{Title: title, AssetVersion: s.Version}
}

// render writes a Gomponents node as an HTML response.
func (s *server) render(w http.ResponseWriter, r *http.Request, status int, n g.Node) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	if err := n.Render(w); err != nil {
		s.Logger.ErrorContext(r.Context(), "render failed", "err", err)
	}
}

func (s *server) home(w http.ResponseWriter, r *http.Request) {
	s.render(w, r, http.StatusOK, views.HomePage(s.page("")))
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
