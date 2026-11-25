// Package views holds the Gomponents layouts, components and pages (spec §10).
// Every page is a full HTML document; htmx fragments are rendered by the same
// components.
package views

import (
	"strconv"
	"time"

	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// AppName is shown in the header and the page title.
const AppName = "TestFlow"

// The vendored htmx file (spec D4, §12). `make update-htmx` prints the new
// hash; update both constants in the same PR as the file.
const (
	HTMXVersion   = "4.0.1"
	HTMXIntegrity = "sha384-Qp9rdC8H7EqOoX+nR0P3RPG2Noji+qdwJlGpfzwQJsfOpp3sGwIdSP9xO3+UlEhn"
)

// HTMXFile is the file name of the vendored htmx inside internal/web/static.
const HTMXFile = "htmx-" + HTMXVersion + ".min.js"

// htmxConfig turns off the indicator CSS that htmx would otherwise add at
// runtime; app.css carries those rules so the CSP can stay self-only (§9).
const htmxConfig = `{"includeIndicatorCSS":false}`

// Page carries what the layout needs besides the page content.
type Page struct {
	Title        string // page-specific part of <title>; empty for the bare app name
	UserName     string // signed-in user; empty hides the account block
	AssetVersion string // appended to CSS/JS URLs to bust caches on deploy
}

// Layout renders a full HTML document with header, main container and footer
// (legacy: resources/views/components/layout.blade.php).
func Layout(p Page, children ...g.Node) g.Node {
	title := AppName
	if p.Title != "" {
		title = p.Title + " · " + AppName
	}
	return h.Doctype(
		h.HTML(h.Lang("en"),
			h.Head(
				h.Meta(h.Charset("utf-8")),
				h.Meta(h.Name("viewport"), h.Content("width=device-width, initial-scale=1")),
				h.Meta(h.Name("htmx-config"), h.Content(htmxConfig)),
				h.TitleEl(g.Text(title)),
				h.Link(h.Rel("icon"), h.Type("image/svg+xml"), h.Href("/static/icons/logo_check.svg")),
				h.Link(h.Rel("stylesheet"), h.Href(asset("app.css", p.AssetVersion))),
				h.Script(h.Src("/static/"+HTMXFile), h.Integrity(HTMXIntegrity), h.Defer()),
				h.Script(h.Src(asset("app.js", p.AssetVersion)), h.Defer()),
			),
			h.Body(
				header(p.UserName),
				h.Main(h.Div(h.Class("container"), g.Group(children))),
				footer(),
			),
		),
	)
}

// header ports resources/views/components/header.blade.php. Logout is a POST
// form (spec §7) instead of the legacy GET link.
func header(userName string) g.Node {
	return h.Header(
		h.Div(h.Class("header"),
			h.A(h.Href("/"), h.Class("logo"),
				h.Img(h.Src("/static/icons/logo_check.svg"), h.Alt("")),
				g.Text(AppName),
			),
			g.If(userName != "",
				h.Div(h.Class("account"),
					h.Img(h.Src("/static/icons/person.svg"), h.Class("account__image"), h.Alt("")),
					h.Span(g.Text(userName)),
					h.Form(h.Method("post"), h.Action("/auth/logout"),
						h.Button(h.Type("submit"), h.Class("logout"), h.Title("Logout"),
							h.Img(h.Src("/static/icons/logout.png"), h.Alt("Logout")),
						),
					),
				),
			),
		),
	)
}

// footer ports resources/views/components/footer.blade.php.
func footer() g.Node {
	return h.Footer(
		g.Text("©"+strconv.Itoa(time.Now().Year())+" Laposa Limited |"),
		h.A(h.Href("https://github.com/laposa/testflow/issues"), g.Text("Report issue")),
	)
}

// Section ports resources/views/components/portal-section.blade.php: a titled
// block, optionally followed by a divider.
func Section(title string, border bool, children ...g.Node) g.Node {
	class := "portal-section"
	if !border {
		class += " no-border"
	}
	return h.Section(h.Class(class),
		h.H2(g.Text(title)),
		g.Group(children),
		h.Hr(),
	)
}

func asset(name, version string) string {
	u := "/static/" + name
	if version != "" {
		u += "?v=" + version
	}
	return u
}

// LoadingOverlay ports components/loading-overlay.blade.php: the spinning
// TestFlow check mark over a white veil covering the page, or with
// relative only its positioned container. With an id it starts hidden as
// an htmx indicator, shown while a request is in flight.
func LoadingOverlay(id, label string, relative bool) g.Node {
	class := "loading-overlay"
	if relative {
		class += " relative"
	}
	if id != "" {
		class += " htmx-indicator"
	}
	return h.Div(g.If(id != "", h.ID(id)), h.Class(class),
		h.Role("status"), g.Attr("aria-label", label),
		g.Raw(loadingSVG),
	)
}

// loadingSVG is the legacy spinner: three dots circling a check mark.
// app.css spins the .loading-spin groups. Legacy used SMIL
// (<animateTransform>), which Chromium leaves frozen when the SVG starts
// out hidden, as in the test file dialog; a CSS animation starts whenever
// the element is shown.
const loadingSVG = `<svg width="75" height="75" viewBox="0 0 100 100" xmlns="http://www.w3.org/2000/svg" aria-hidden="true">` +
	`<mask id="loading-mask"><rect width="100" height="100" fill="white"/>` +
	`<g class="loading-spin">` + loadingDotsBlack + `</g></mask>` +
	`<circle cx="50" cy="50" r="45" stroke="#5f9ea0" fill="none" stroke-width="5" mask="url(#loading-mask)"/>` +
	`<g class="loading-spin">` + loadingDotsTeal + `</g>` +
	`<path d="M 30,50 L 45,65 L 70,35" stroke="#5f9ea0" stroke-width="5" fill="none"/></svg>`

// The dots, 30° apart. The black ones cut the gaps into the ring.
const (
	loadingDotsBlack = `<circle cx="50" cy="5" r="15" fill="black"/>` +
		`<circle cx="50" cy="5" r="15" fill="black" transform="rotate(30 50 50)"/>` +
		`<circle cx="50" cy="5" r="15" fill="black" transform="rotate(60 50 50)"/>`
	loadingDotsTeal = `<circle cx="50" cy="5" r="5" fill="#5f9ea0"/>` +
		`<circle cx="50" cy="5" r="5" fill="#5f9ea0" transform="rotate(30 50 50)"/>` +
		`<circle cx="50" cy="5" r="5" fill="#5f9ea0" transform="rotate(60 50 50)"/>`
)
