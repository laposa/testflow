package telemetry

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/slogtest"
	"time"
)

var testResource = Resource{ServiceName: "testflow", ServiceVersion: "1.2.3", Environment: "test"}

func decodeLines(t *testing.T, buf *bytes.Buffer) []map[string]any {
	t.Helper()
	var out []map[string]any
	for line := range strings.SplitSeq(strings.TrimSpace(buf.String()), "\n") {
		if line == "" {
			continue
		}
		var m map[string]any
		if err := json.Unmarshal([]byte(line), &m); err != nil {
			t.Fatalf("invalid JSON line %q: %v", line, err)
		}
		out = append(out, m)
	}
	return out
}

// TestSlogContract runs the standard library's handler conformance tests.
// slogtest expects attributes at the top level, so each line's "attributes"
// object is lifted up before checking, together with the mapped core fields.
func TestSlogContract(t *testing.T) {
	var buf bytes.Buffer
	h := NewHandler(&buf, slog.LevelDebug, testResource)
	results := func() []map[string]any {
		var out []map[string]any
		for _, line := range decodeLines(t, &buf) {
			m := map[string]any{}
			if attrs, ok := line["attributes"].(map[string]any); ok {
				m = attrs
			}
			m[slog.MessageKey] = line["body"]
			m[slog.LevelKey] = line["severity_text"]
			if ts := line["timestamp"]; ts != nil {
				m[slog.TimeKey] = ts
			}
			out = append(out, m)
		}
		return out
	}
	if err := slogtest.TestHandler(h, results); err != nil {
		t.Fatal(err)
	}
}

func TestHandlerOutputShape(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(NewHandler(&buf, slog.LevelInfo, testResource))

	sc := SpanContext{TraceID: NewTraceID(), SpanID: NewSpanID(), Sampled: true}
	ctx := ContextWithSpan(context.Background(), sc)

	logger.With("component", "syncer").WithGroup("run").
		InfoContext(ctx, "run synced", "id", 7, "took", 1500*time.Millisecond, "err", errors.New("boom"))
	logger.Debug("not logged")
	logger.Warn("no context")

	lines := decodeLines(t, &buf)
	if len(lines) != 2 {
		t.Fatalf("got %d lines, want 2", len(lines))
	}
	first := lines[0]
	if _, err := time.Parse(time.RFC3339Nano, first["timestamp"].(string)); err != nil {
		t.Errorf("timestamp: %v", err)
	}
	checks := map[string]any{
		"severity_text":   "INFO",
		"severity_number": float64(9),
		"body":            "run synced",
		"trace_id":        sc.TraceID.String(),
		"span_id":         sc.SpanID.String(),
	}
	for k, want := range checks {
		if first[k] != want {
			t.Errorf("%s = %v, want %v", k, first[k], want)
		}
	}
	res := first["resource"].(map[string]any)
	if res["service.name"] != "testflow" || res["service.version"] != "1.2.3" || res["deployment.environment"] != "test" {
		t.Errorf("resource = %v", res)
	}
	attrs := first["attributes"].(map[string]any)
	if attrs["component"] != "syncer" {
		t.Errorf("attributes = %v", attrs)
	}
	run := attrs["run"].(map[string]any)
	if run["id"] != float64(7) || run["took"] != "1.5s" || run["err"] != "boom" {
		t.Errorf("attributes.run = %v", run)
	}

	second := lines[1]
	if second["severity_number"] != float64(13) {
		t.Errorf("WARN severity_number = %v", second["severity_number"])
	}
	if _, ok := second["trace_id"]; ok {
		t.Error("trace_id present without span in context")
	}
	if _, ok := second["attributes"]; ok {
		t.Error("attributes present although there are none")
	}
}

func TestSeverityNumber(t *testing.T) {
	tests := map[slog.Level]int{
		slog.LevelDebug:     5,
		slog.LevelInfo:      9,
		slog.LevelWarn:      13,
		slog.LevelError:     17,
		slog.LevelInfo + 2:  11,
		slog.LevelDebug - 9: 1,
		slog.LevelError + 9: 24,
	}
	for level, want := range tests {
		if got := SeverityNumber(level); got != want {
			t.Errorf("SeverityNumber(%v) = %d, want %d", level, got, want)
		}
	}
}

func TestParseTraceparent(t *testing.T) {
	const valid = "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01"
	sc, ok := ParseTraceparent(valid)
	if !ok {
		t.Fatal("valid header rejected")
	}
	if sc.TraceID.String() != "4bf92f3577b34da6a3ce929d0e0e4736" || sc.SpanID.String() != "00f067aa0ba902b7" || !sc.Sampled {
		t.Errorf("parsed %+v", sc)
	}
	if sc.Traceparent() != valid {
		t.Errorf("round trip = %q", sc.Traceparent())
	}
	if _, ok := ParseTraceparent("01-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-00-future"); !ok {
		t.Error("future version with extra fields rejected")
	}

	invalid := []string{
		"",
		"garbage",
		"00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01-extra", // v00 must be exact
		"00-00000000000000000000000000000000-00f067aa0ba902b7-01",       // zero trace id
		"00-4bf92f3577b34da6a3ce929d0e0e4736-0000000000000000-01",       // zero span id
		"00-4BF92F3577B34DA6A3CE929D0E0E4736-00f067aa0ba902b7-01",       // upper case
		"ff-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01",       // forbidden version
		"00-4bf92f3577b34da6a3ce929d0e0e47zz-00f067aa0ba902b7-01",       // not hex
	}
	for _, h := range invalid {
		if _, ok := ParseTraceparent(h); ok {
			t.Errorf("ParseTraceparent(%q) accepted", h)
		}
	}
}

func TestMiddleware(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(NewHandler(&buf, slog.LevelInfo, testResource))

	var seen SpanContext
	mux := http.NewServeMux()
	mux.HandleFunc("GET /items/{id}", func(w http.ResponseWriter, r *http.Request) {
		seen, _ = SpanFromContext(r.Context())
		SetUserID(r.Context(), 42)
		w.WriteHeader(http.StatusTeapot)
		_, _ = w.Write([]byte("hello"))
	})
	h := Middleware(logger, mux)

	t.Run("continues incoming trace", func(t *testing.T) {
		buf.Reset()
		req := httptest.NewRequest("GET", "/items/5", nil)
		req.Header.Set("traceparent", "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)

		if seen.TraceID.String() != "4bf92f3577b34da6a3ce929d0e0e4736" {
			t.Errorf("trace id not continued: %s", seen.TraceID)
		}
		if seen.SpanID.String() == "00f067aa0ba902b7" {
			t.Error("server span must be a new span, not the parent")
		}
		if got := rec.Header().Get("traceparent"); got != seen.Traceparent() {
			t.Errorf("response traceparent = %q, want %q", got, seen.Traceparent())
		}

		lines := decodeLines(t, &buf)
		if len(lines) != 1 {
			t.Fatalf("got %d log lines", len(lines))
		}
		line := lines[0]
		if line["trace_id"] != seen.TraceID.String() || line["span_id"] != seen.SpanID.String() {
			t.Errorf("log line not correlated: %v", line)
		}
		attrs := line["attributes"].(map[string]any)
		want := map[string]any{
			"http.request.method":       "GET",
			"http.route":                "GET /items/{id}",
			"url.path":                  "/items/5",
			"http.response.status_code": float64(418),
			"http.response.body.size":   float64(5),
			"user.id":                   float64(42),
		}
		for k, v := range want {
			if attrs[k] != v {
				t.Errorf("%s = %v, want %v", k, attrs[k], v)
			}
		}
		if _, ok := attrs["http.server.request.duration"].(float64); !ok {
			t.Error("duration missing")
		}
	})

	t.Run("starts new trace", func(t *testing.T) {
		buf.Reset()
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest("GET", "/nope", nil))
		sc, ok := ParseTraceparent(rec.Header().Get("traceparent"))
		if !ok {
			t.Fatalf("no valid traceparent in response: %q", rec.Header().Get("traceparent"))
		}
		line := decodeLines(t, &buf)[0]
		if line["trace_id"] != sc.TraceID.String() {
			t.Error("log trace id differs from response header")
		}
		attrs := line["attributes"].(map[string]any)
		if attrs["http.response.status_code"] != float64(404) {
			t.Errorf("status = %v", attrs["http.response.status_code"])
		}
		if _, ok := attrs["user.id"]; ok {
			t.Error("user.id logged for anonymous request")
		}
	})
}
