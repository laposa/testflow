package telemetry

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"sync/atomic"
	"time"
)

// requestInfo is mutable per-request state that inner handlers fill in and
// the middleware logs after the request finishes.
type requestInfo struct {
	userID atomic.Int64
}

type requestInfoKey struct{}

// SetUserID records the authenticated user for the request log line. It is a
// no-op outside a request handled by Middleware.
func SetUserID(ctx context.Context, id int64) {
	if ri, ok := ctx.Value(requestInfoKey{}).(*requestInfo); ok {
		ri.userID.Store(id)
	}
}

// Middleware continues or starts a W3C trace for each request, stores it in
// the request context, returns the server span in the traceparent response
// header and logs one line per request.
//
// It must wrap the http.ServeMux directly (or through handlers that pass the
// same *http.Request on), so the matched route pattern can be read after the
// request is served.
func Middleware(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		sc := SpanContext{SpanID: NewSpanID(), Sampled: true}
		if parent, ok := ParseTraceparent(r.Header.Get("traceparent")); ok {
			sc.TraceID, sc.Sampled = parent.TraceID, parent.Sampled
		} else {
			sc.TraceID = NewTraceID()
		}
		w.Header().Set("traceparent", sc.Traceparent())

		ri := &requestInfo{}
		ctx := context.WithValue(ContextWithSpan(r.Context(), sc), requestInfoKey{}, ri)
		r = r.WithContext(ctx)
		rec := &recorder{ResponseWriter: w}

		defer func() {
			// A panicking handler is logged as a 500 at ERROR, then the panic
			// continues so net/http aborts the response and logs the stack.
			// http.ErrAbortHandler is a deliberate abort, not an error.
			p := recover()
			panicked := p != nil && p != http.ErrAbortHandler

			status := rec.status
			switch {
			case panicked && status == 0:
				status = http.StatusInternalServerError
			case status == 0:
				status = http.StatusOK
			}
			level := slog.LevelInfo
			if status >= 500 || panicked {
				level = slog.LevelError
			}
			attrs := []slog.Attr{
				slog.String("http.request.method", r.Method),
				slog.String("http.route", r.Pattern),
				slog.String("url.path", r.URL.Path),
				slog.Int("http.response.status_code", status),
				slog.Float64("http.server.request.duration", time.Since(start).Seconds()),
				slog.Int64("http.response.body.size", rec.bytes),
			}
			if id := ri.userID.Load(); id != 0 {
				attrs = append(attrs, slog.Int64("user.id", id))
			}
			if panicked {
				attrs = append(attrs, slog.String("panic", fmt.Sprint(p)))
			}
			logger.LogAttrs(ctx, level, "http request", attrs...)
			if p != nil {
				panic(p)
			}
		}()

		next.ServeHTTP(rec, r)
	})
}

// recorder captures the status code and body size of a response.
type recorder struct {
	http.ResponseWriter
	status int
	bytes  int64
}

func (r *recorder) WriteHeader(code int) {
	if r.status == 0 {
		r.status = code
	}
	r.ResponseWriter.WriteHeader(code)
}

func (r *recorder) Write(b []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	n, err := r.ResponseWriter.Write(b)
	r.bytes += int64(n)
	return n, err
}

// Unwrap lets http.ResponseController reach the underlying writer
// (flushing, deadlines).
func (r *recorder) Unwrap() http.ResponseWriter { return r.ResponseWriter }
