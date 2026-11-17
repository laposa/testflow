package views

import (
	"crypto/sha512"
	"encoding/base64"
	"strings"
	"testing"

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

func TestHx(t *testing.T) {
	got := render(t, h.Div(hx.Get("/sessions/1/runs"), hx.Trigger("every 30s"), hx.Swap("outerHTML"), hx.Attr("vals", `{"a":1}`)))
	want := `<div hx-get="/sessions/1/runs" hx-trigger="every 30s" hx-swap="outerHTML" hx-vals="{&#34;a&#34;:1}"></div>`
	if got != want {
		t.Errorf("got  %s\nwant %s", got, want)
	}
}
