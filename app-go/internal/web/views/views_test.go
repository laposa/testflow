package views

import (
	"crypto/sha512"
	"encoding/base64"
	"strings"
	"testing"
	"time"

	"github.com/laposa/testflow/app-go/internal/web/static"
	"github.com/laposa/testflow/app-go/internal/web/views/hx"
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

func render(t *testing.T, n g.Node) string {
	t.Helper()
	var b strings.Builder
	if err := n.Render(&b); err != nil {
		t.Fatal(err)
	}
	return b.String()
}

// TestHTMXIntegrity guards the manual upgrade procedure (spec §12): the
// embedded file must exist under the pinned name and match the SRI hash.
func TestHTMXIntegrity(t *testing.T) {
	data, err := static.FS.ReadFile(HTMXFile)
	if err != nil {
		t.Fatalf("vendored htmx missing: %v", err)
	}
	sum := sha512.Sum384(data)
	if got := "sha384-" + base64.StdEncoding.EncodeToString(sum[:]); got != HTMXIntegrity {
		t.Errorf("HTMXIntegrity = %s, file hash is %s", HTMXIntegrity, got)
	}
}

func TestLayout(t *testing.T) {
	out := render(t, Layout(Page{Title: "Sessions", UserName: "Ada <Lovelace>", AssetVersion: "v1"},
		h.P(g.Text("content"))))

	for _, want := range []string{
		"<!doctype html>",
		"<title>Sessions · TestFlow</title>",
		`<script src="/static/htmx-4.0.1.min.js" integrity="` + HTMXIntegrity + `" defer></script>`,
		`<link rel="stylesheet" href="/static/app.css?v=v1">`,
		`<script src="/static/app.js?v=v1" defer></script>`,
		`<meta name="htmx-config" content="{&#34;includeIndicatorCSS&#34;:false}">`,
		`<div class="container"><p>content</p></div>`,
		"Ada &lt;Lovelace&gt;", // user content is escaped
		`<form method="post" action="/auth/logout">`,
		"Laposa Limited",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("layout missing %q\n%s", want, out)
		}
	}
	if strings.Contains(out, "<script>") || strings.Contains(out, " style=") {
		t.Error("layout contains inline script or style (breaks the CSP)")
	}
}

func TestLayoutAnonymous(t *testing.T) {
	out := render(t, Layout(Page{}))
	if !strings.Contains(out, "<title>TestFlow</title>") {
		t.Error("bare title missing")
	}
	if strings.Contains(out, "/auth/logout") {
		t.Error("logout shown without a user")
	}
	if !strings.Contains(out, `href="/static/app.css"`) {
		t.Error("asset URL without version wrong")
	}
}

func TestSection(t *testing.T) {
	if got := render(t, Section("Runs", true, g.Text("x"))); got != `<section class="portal-section"><h2>Runs</h2>x<hr></section>` {
		t.Errorf("bordered section = %s", got)
	}
	if got := render(t, Section("Runs", false)); !strings.Contains(got, `class="portal-section no-border"`) {
		t.Errorf("borderless section = %s", got)
	}
}

func TestPassedFailed(t *testing.T) {
	pass := func(n string) string { return `<span class="pass">` + n + ` passed</span>` }
	fail := func(n string) string { return `<span class="fail">` + n + ` failed</span>` }
	skip := func(n string) string { return `<span class="skip">` + n + ` skipped</span>` }
	tests := []struct {
		p, f, s  int
		showZero bool
		want     string
	}{
		{0, 0, 0, false, `<span class="no-runs">None</span>`},
		{3, 0, 0, false, pass("3")},
		{3, 1, 0, false, pass("3") + " and " + fail("1")},
		{3, 0, 2, false, pass("3") + " and " + skip("2")}, // legacy: "and and"
		{0, 1, 2, false, fail("1") + " and " + skip("2")},
		{3, 1, 2, false, pass("3") + ", " + fail("1") + " and " + skip("2")},
		{0, 0, 0, true, pass("0") + ", " + fail("0") + " and " + skip("0")},
	}
	for _, tt := range tests {
		if got := render(t, PassedFailed(tt.p, tt.f, tt.s, tt.showZero)); got != tt.want {
			t.Errorf("PassedFailed(%d, %d, %d, %v)\n got  %s\n want %s", tt.p, tt.f, tt.s, tt.showZero, got, tt.want)
		}
	}
}

func TestTimestamp(t *testing.T) {
	ts := time.Date(2026, 3, 4, 17, 5, 0, 0, time.UTC)
	if got := render(t, Timestamp(ts)); got != `<time datetime="2026-03-04T17:05:00Z">4 Mar 2026 17:05</time>` {
		t.Errorf("Timestamp = %s", got)
	}
}

func TestHx(t *testing.T) {
	got := render(t, h.Div(hx.Get("/sessions/1/runs"), hx.Trigger("every 30s"), hx.Swap("outerHTML"), hx.Attr("vals", `{"a":1}`)))
	want := `<div hx-get="/sessions/1/runs" hx-trigger="every 30s" hx-swap="outerHTML" hx-vals="{&#34;a&#34;:1}"></div>`
	if got != want {
		t.Errorf("got  %s\nwant %s", got, want)
	}
}
