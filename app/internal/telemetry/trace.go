package telemetry

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
)

// TraceID is a W3C trace id (16 bytes, printed as 32 hex digits).
type TraceID [16]byte

// SpanID is a W3C span (parent) id (8 bytes, printed as 16 hex digits).
type SpanID [8]byte

func (t TraceID) String() string { return hex.EncodeToString(t[:]) }
func (s SpanID) String() string  { return hex.EncodeToString(s[:]) }

// IsValid reports whether t is not all zeros.
func (t TraceID) IsValid() bool { return t != TraceID{} }

// IsValid reports whether s is not all zeros.
func (s SpanID) IsValid() bool { return s != SpanID{} }

// SpanContext identifies the current span. Log records carry its ids.
type SpanContext struct {
	TraceID TraceID
	SpanID  SpanID
	Sampled bool
}

// Flags returns the W3C trace flags as two hex digits ("01" when sampled).
func (sc SpanContext) Flags() string {
	if sc.Sampled {
		return "01"
	}
	return "00"
}

// Traceparent formats sc as a W3C traceparent header value (version 00).
func (sc SpanContext) Traceparent() string {
	return fmt.Sprintf("00-%s-%s-%s", sc.TraceID, sc.SpanID, sc.Flags())
}

// ParseTraceparent parses a W3C traceparent header. Only the fields defined
// by version 00 are read; later versions may append fields, which are ignored.
func ParseTraceparent(h string) (SpanContext, bool) {
	h = strings.TrimSpace(h)
	if len(h) < 55 || h[2] != '-' || h[35] != '-' || h[52] != '-' || (len(h) > 55 && h[55] != '-') {
		return SpanContext{}, false
	}
	version, err := hex.DecodeString(h[0:2])
	if err != nil || version[0] == 0xff || (version[0] == 0 && len(h) != 55) {
		return SpanContext{}, false
	}
	var sc SpanContext
	if _, err := hex.Decode(sc.TraceID[:], []byte(h[3:35])); err != nil || !isLowerHex(h[3:35]) {
		return SpanContext{}, false
	}
	if _, err := hex.Decode(sc.SpanID[:], []byte(h[36:52])); err != nil || !isLowerHex(h[36:52]) {
		return SpanContext{}, false
	}
	flags, err := hex.DecodeString(h[53:55])
	if err != nil {
		return SpanContext{}, false
	}
	if !sc.TraceID.IsValid() || !sc.SpanID.IsValid() {
		return SpanContext{}, false
	}
	sc.Sampled = flags[0]&1 == 1
	return sc, true
}

func isLowerHex(s string) bool {
	return !strings.ContainsAny(s, "ABCDEF")
}

// NewTraceID returns a random, valid trace id.
func NewTraceID() TraceID {
	var t TraceID
	for !t.IsValid() {
		_, _ = rand.Read(t[:])
	}
	return t
}

// NewSpanID returns a random, valid span id.
func NewSpanID() SpanID {
	var s SpanID
	for !s.IsValid() {
		_, _ = rand.Read(s[:])
	}
	return s
}

type spanKey struct{}

// ContextWithSpan returns a copy of ctx carrying sc. Records logged with that
// context include its trace_id and span_id.
func ContextWithSpan(ctx context.Context, sc SpanContext) context.Context {
	return context.WithValue(ctx, spanKey{}, sc)
}

// SpanFromContext returns the span stored in ctx, if any.
func SpanFromContext(ctx context.Context) (SpanContext, bool) {
	if ctx == nil {
		return SpanContext{}, false
	}
	sc, ok := ctx.Value(spanKey{}).(SpanContext)
	return sc, ok
}

// StartTrace returns a context with a new root span, for work that does not
// come from a request, such as one tick of the background syncer.
func StartTrace(ctx context.Context) context.Context {
	return ContextWithSpan(ctx, SpanContext{TraceID: NewTraceID(), SpanID: NewSpanID(), Sampled: true})
}
