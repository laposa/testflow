// Package telemetry provides structured logging in the OpenTelemetry log data
// model, W3C trace context propagation and HTTP request logging (spec §11).
//
// Logs are written as one JSON object per line to stdout and shipped by an
// OpenTelemetry Collector. Call sites use plain log/slog; only this package
// knows about the output shape.
package telemetry

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"slices"
	"sync"
	"time"
)

// Resource describes the process emitting the logs.
type Resource struct {
	ServiceName    string // service.name
	ServiceVersion string // service.version
	Environment    string // deployment.environment
}

// Handler is a slog.Handler that writes one JSON object per record using the
// field names of the OpenTelemetry log data model:
//
//	{"timestamp":"…","severity_text":"INFO","severity_number":9,"body":"…",
//	 "trace_id":"…","span_id":"…","trace_flags":"01","resource":{…},"attributes":{…}}
type Handler struct {
	out      *output
	level    slog.Leveler
	resource []byte // pre-encoded JSON object

	attrs  []groupedAttr // from WithAttrs, in order
	groups []string      // from WithGroup, applies to later attrs
}

type output struct {
	mu sync.Mutex
	w  io.Writer
}

type groupedAttr struct {
	groups []string
	attr   slog.Attr
}

// NewHandler returns a Handler writing to w for records at or above level.
func NewHandler(w io.Writer, level slog.Leveler, res Resource) *Handler {
	resource, _ := json.Marshal(map[string]string{
		"service.name":           res.ServiceName,
		"service.version":        res.ServiceVersion,
		"deployment.environment": res.Environment,
	})
	return &Handler{out: &output{w: w}, level: level, resource: resource}
}

// Enabled implements slog.Handler.
func (h *Handler) Enabled(_ context.Context, l slog.Level) bool {
	return l >= h.level.Level()
}

// WithAttrs implements slog.Handler.
func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return h
	}
	h2 := *h
	h2.attrs = slices.Clip(h.attrs)
	for _, a := range attrs {
		h2.attrs = append(h2.attrs, groupedAttr{groups: h.groups, attr: a})
	}
	return &h2
}

// WithGroup implements slog.Handler.
func (h *Handler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	h2 := *h
	h2.groups = append(slices.Clip(h.groups), name)
	return &h2
}

// Handle implements slog.Handler.
func (h *Handler) Handle(ctx context.Context, r slog.Record) error {
	attrs := &object{}
	for _, ga := range h.attrs {
		attrs.add(ga.groups, ga.attr)
	}
	r.Attrs(func(a slog.Attr) bool {
		attrs.add(h.groups, a)
		return true
	})

	var buf bytes.Buffer
	buf.WriteByte('{')
	// Like slog's own handlers, a zero time is omitted; the Collector then
	// fills in observed_timestamp.
	if !r.Time.IsZero() {
		buf.WriteString(`"timestamp":`)
		writeJSON(&buf, r.Time.UTC().Format(time.RFC3339Nano))
		buf.WriteByte(',')
	}
	buf.WriteString(`"severity_text":`)
	writeJSON(&buf, r.Level.String())
	fmt.Fprintf(&buf, `,"severity_number":%d,"body":`, SeverityNumber(r.Level))
	writeJSON(&buf, r.Message)
	if sc, ok := SpanFromContext(ctx); ok {
		fmt.Fprintf(&buf, `,"trace_id":"%s","span_id":"%s","trace_flags":"%s"`, sc.TraceID, sc.SpanID, sc.Flags())
	}
	buf.WriteString(`,"resource":`)
	buf.Write(h.resource)
	if !attrs.empty() {
		buf.WriteString(`,"attributes":`)
		attrs.writeTo(&buf)
	}
	buf.WriteString("}\n")

	h.out.mu.Lock()
	defer h.out.mu.Unlock()
	_, err := h.out.w.Write(buf.Bytes())
	return err
}

// SeverityNumber maps a slog level to the OpenTelemetry severity number:
// DEBUG 5, INFO 9, WARN 13, ERROR 17. Levels in between keep their offset
// (slog.LevelInfo+2 is 11), clamped to the valid range 1–24.
func SeverityNumber(l slog.Level) int {
	return min(max(int(l)+9, 1), 24)
}

// object is an insertion-ordered JSON object used to build "attributes".
type object struct {
	keys []string
	vals map[string]any // string key -> json value or *object
}

func (o *object) empty() bool { return len(o.keys) == 0 }

func (o *object) set(key string, v any) {
	if o.vals == nil {
		o.vals = map[string]any{}
	}
	if _, exists := o.vals[key]; !exists {
		o.keys = append(o.keys, key)
	}
	o.vals[key] = v
}

func (o *object) child(key string) *object {
	if c, ok := o.vals[key].(*object); ok {
		return c
	}
	c := &object{}
	o.set(key, c)
	return c
}

// add inserts a under the nested groups path, following slog's rules: empty
// attrs are ignored, groups are nested, and empty-keyed groups are inlined.
func (o *object) add(groups []string, a slog.Attr) {
	a.Value = a.Value.Resolve()
	if a.Equal(slog.Attr{}) {
		return
	}
	if a.Value.Kind() == slog.KindGroup {
		members := a.Value.Group()
		if len(members) == 0 {
			return
		}
		path := groups
		if a.Key != "" {
			path = append(slices.Clip(groups), a.Key)
		}
		for _, m := range members {
			o.add(path, m)
		}
		return
	}
	target := o
	for _, g := range groups {
		target = target.child(g)
	}
	target.set(a.Key, jsonValue(a.Value))
}

func (o *object) writeTo(buf *bytes.Buffer) {
	buf.WriteByte('{')
	for i, k := range o.keys {
		if i > 0 {
			buf.WriteByte(',')
		}
		writeJSON(buf, k)
		buf.WriteByte(':')
		if c, ok := o.vals[k].(*object); ok {
			c.writeTo(buf)
		} else {
			writeJSON(buf, o.vals[k])
		}
	}
	buf.WriteByte('}')
}

func jsonValue(v slog.Value) any {
	switch v.Kind() {
	case slog.KindString:
		return v.String()
	case slog.KindInt64:
		return v.Int64()
	case slog.KindUint64:
		return v.Uint64()
	case slog.KindFloat64:
		return v.Float64()
	case slog.KindBool:
		return v.Bool()
	case slog.KindDuration:
		return v.Duration().String()
	case slog.KindTime:
		return v.Time().UTC().Format(time.RFC3339Nano)
	}
	switch x := v.Any().(type) {
	case error:
		return x.Error()
	case json.Marshaler:
		return x
	case fmt.Stringer:
		return x.String()
	default:
		if _, err := json.Marshal(x); err != nil {
			return fmt.Sprintf("%+v", x)
		}
		return x
	}
}

func writeJSON(buf *bytes.Buffer, v any) {
	b, err := json.Marshal(v)
	if err != nil {
		b, _ = json.Marshal(fmt.Sprintf("!ERROR: %v", err))
	}
	buf.Write(b)
}
