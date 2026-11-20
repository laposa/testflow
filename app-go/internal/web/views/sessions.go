package views

import (
	"strconv"
	"unicode"
	"unicode/utf8"

	"github.com/laposa/testflow/app-go/internal/discovery"
	"github.com/laposa/testflow/app-go/internal/store"
	"github.com/laposa/testflow/app-go/internal/web/views/hx"
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// NewSessionForm is what the test picker shows: the form values, the
// discovered tests and any errors.
type NewSessionForm struct {
	Environments []string
	Name         string
	Environment  string
	Selected     map[string]bool   // test keys
	Tests        *discovery.Result // nil if discovery failed
	LoadError    string            // why Tests is nil
	Warning      string            // set if Tests is a fallback after a failed discovery
	Errors       []string          // validation errors
}

// NewSessionPage ports resources/views/livewire/sessions/create-session.blade.php
// and components/tests/list-interactive.blade.php.
func NewSessionPage(p Page, f NewSessionForm) g.Node {
	return Layout(p, NewSessionContent(f))
}

// NewSessionID is the block that NewSessionLoadingPage loads and the
// Refresh button reloads with htmx.
const NewSessionID = "new-session"

// NewSessionLoadingPage is shown while the tests are being discovered
// (legacy: the lazy CreateSession component's placeholder, the heading and
// a loading overlay). Once loaded, htmx fetches the form and replaces the
// block with it. Without JS, the link in <noscript> loads the page the
// slow way.
func NewSessionLoadingPage(p Page) g.Node {
	return Layout(p,
		h.Div(h.ID(NewSessionID), hx.Get("/sessions/new"), hx.Trigger("load"), hx.Swap("outerHTML"),
			h.Section(h.H2(g.Text("New session"))),
			LoadingOverlay("", "Loading tests from GitHub", false),
			h.NoScript(h.P(h.A(h.Href("/sessions/new?refresh"), h.Class("button"), g.Text("Load the tests")))),
		),
	)
}

// NewSessionContent is the new session form: the page body, and the
// fragment htmx loads into the loading page or a refresh.
func NewSessionContent(f NewSessionForm) g.Node {
	return h.Div(h.ID(NewSessionID),
		g.If(len(f.Errors) > 0, ErrorList(f.Errors)),
		// data-submit-once: app.js disables the button on submit, so a
		// double click does not create two sessions.
		h.Form(h.Method("post"), h.Action("/sessions"), h.Class("new-session"), g.Attr("data-submit-once"),
			g.Iff(f.Tests != nil, func() g.Node { return h.Input(h.Type("hidden"), h.Name("discovery"), h.Value(f.Tests.ID)) }),
			h.Section(
				h.H2(g.Text("New session")),
				h.Div(h.Class("information"),
					h.Input(h.Type("text"), h.Name("name"), h.Required(), h.MaxLength("255"),
						h.Placeholder("Session name"), g.Attr("aria-label", "Session name"), h.Value(f.Name)),
					h.Select(h.Name("environment"), g.Attr("aria-label", "Environment"),
						g.Map(f.Environments, func(env string) g.Node {
							return h.Option(h.Value(env), g.If(env == f.Environment, h.Selected()), g.Text(Capitalize(env)))
						}),
					),
				),
			),
			h.Section(testPicker(f)),
			h.Section(
				h.Button(h.Type("submit"), h.Class("button filled"), g.If(f.Tests == nil, h.Disabled()), g.Text("Create")),
			),
		),
		// Shown while Refresh discovers again.
		LoadingOverlay(NewSessionID+"-loading", "Loading tests from GitHub", false),
	)
}

// testPicker renders the repository → service → suite → test tree. Each
// level is a <details> (D17); app.js keeps the "select all" checkboxes in
// the summaries in step with the tests.
func testPicker(f NewSessionForm) g.Node {
	// Without JS, ?refresh discovers and redirects back.
	refresh := h.A(h.Href("/sessions/new?refresh"), h.Class("button"),
		hx.Get("/sessions/new?refresh"), hx.Target("#"+NewSessionID), hx.Swap("outerHTML"),
		hx.Indicator("#"+NewSessionID+"-loading"), hx.Sync("this:drop"), g.Text("Refresh"))
	if f.Tests == nil {
		return g.Group{
			h.P(h.Class("alert"), h.Role("alert"), g.Text(f.LoadError)),
			refresh,
		}
	}
	res := f.Tests
	return g.Group{
		h.Div(h.Class("test-picker__header"),
			h.Span(g.Text("Tests from "+Plural(len(res.Repositories), "repository", "repositories")+
				", loaded at "+res.FetchedAt.Format("15:04")+".")),
			refresh,
		),
		g.If(f.Warning != "", h.P(h.Class("alert"), h.Role("alert"), g.Text(f.Warning))),
		g.If(len(res.Problems) > 0,
			h.Ul(h.Class("test-picker__problems"),
				g.Map(res.Problems, func(pr discovery.Problem) g.Node {
					return h.Li(h.Class("alert"), g.Text(pr.Repository+": "+pr.Message+"."))
				}),
			),
		),
		g.If(len(res.Repositories) == 0,
			h.P(g.Text("No tests were found in the repositories the GitHub App can access.")),
		),
		h.Div(h.Class("list-interactive"),
			g.Map(res.Repositories, func(repo discovery.Repository) g.Node {
				open, selectable := false, false
				for _, svc := range repo.Services {
					open = open || suitesSelected(f.Selected, svc.Suites)
					selectable = selectable || hasSelectable(svc, svc.Suites)
				}
				return treeLevel("list-repository", repo.Name, open, selectable,
					g.Map(repo.Services, func(svc discovery.Service) g.Node {
						return treeLevel("list-service", svc.Name, suitesSelected(f.Selected, svc.Suites), hasSelectable(svc, svc.Suites),
							g.Map(svc.Suites, func(suite discovery.Suite) g.Node {
								one := []discovery.Suite{suite}
								return treeLevel("list-suite", suite.Name, suitesSelected(f.Selected, one), hasSelectable(svc, one),
									h.Div(h.Class("list-test"),
										g.Map(suite.Tests, func(t discovery.Test) g.Node { return testItem(svc, t, f.Selected[t.Key]) }),
									),
								)
							}),
						)
					}),
				)
			}),
		),
	}
}

// treeLevel is one collapsible level of the test tree. It starts open when
// it holds selected tests, e.g. after a validation error.
func treeLevel(class, title string, open, selectable bool, children ...g.Node) g.Node {
	return h.Details(h.Class("list "+class), g.If(open, h.Open()),
		h.Summary(h.Class("title"),
			h.Label(h.Class("checkbox"),
				h.Input(h.Type("checkbox"), h.Class("group-selector"), g.If(!selectable, h.Disabled()),
					g.Attr("aria-label", "Select all tests in "+title)),
				h.Span(h.Class("checkmark")),
			),
			h.Span(h.Class("expand"), g.Text(title)),
		),
		g.Group(children),
	)
}

func testItem(svc discovery.Service, t discovery.Test, checked bool) g.Node {
	if !svc.Selectable(t) {
		return h.Div(h.Class("test"),
			h.Span(h.Class("alert-icon error"),
				h.Title("This test has no "+discovery.WorkflowFile(svc.Name)+" workflow and cannot be selected.")),
			g.Text(t.Name),
		)
	}
	return h.Div(h.Class("test"),
		h.Label(h.Class("checkbox"),
			h.Input(h.Type("checkbox"), h.Class("test-selector"), h.Name("tests"), h.Value(t.Key),
				g.If(checked, h.Checked()), g.Attr("aria-label", t.Name)),
			h.Span(h.Class("checkmark")),
		),
		g.Text(t.Name),
	)
}

func suitesSelected(selected map[string]bool, suites []discovery.Suite) bool {
	for _, suite := range suites {
		for _, t := range suite.Tests {
			if selected[t.Key] {
				return true
			}
		}
	}
	return false
}

func hasSelectable(svc discovery.Service, suites []discovery.Suite) bool {
	for _, suite := range suites {
		for _, t := range suite.Tests {
			if svc.Selectable(t) {
				return true
			}
		}
	}
	return false
}

// SessionPage is the session detail (legacy: resources/views/sessions/show.blade.php).
// Manual runs, reviews and reports come in later iterations.
func SessionPage(p Page, ts store.TestSession, runs RunListView, reviews ReviewsView, activity ActivityView) g.Node {
	return Layout(p,
		Section(ts.Name, true,
			h.Div(h.Class("session-environment"),
				g.Text("Environment: "), h.Span(g.Text(Capitalize(ts.Environment))),
			),
			h.P(h.Class("session-meta"),
				g.Text("Created by "+ts.IssuerName+" on "),
				Timestamp(ts.CreatedAt),
				g.Text("."),
			),
			RunList(runs),
			g.If(hasManual(ts), ManualDialog()),
			h.Details(h.Class("session-selected-tests"),
				h.Summary(g.Text("Selected tests ("+strconv.Itoa(countTests(ts))+")")),
				h.Ul(h.Class("session-selected-suites"),
					g.Map(ts.Services, func(svc store.Service) g.Node {
						return h.Li(h.B(g.Text(svc.DisplayName())),
							h.Ul(g.Map(svc.Suites, func(suite store.Suite) g.Node {
								return h.Li(h.B(g.Text(suite.Name)),
									h.Ul(g.Map(suite.Tests, func(t store.Test) g.Node {
										return h.Li(g.Text(t.Name))
									})),
								)
							})),
						)
					}),
				),
			),
		),
		Reviews(reviews),
		Activity(activity),
	)
}

func hasManual(ts store.TestSession) bool {
	for _, svc := range ts.Services {
		if svc.Type == store.RunManual {
			return true
		}
	}
	return false
}

func countTests(ts store.TestSession) int {
	n := 0
	for _, svc := range ts.Services {
		for _, suite := range svc.Suites {
			n += len(suite.Tests)
		}
	}
	return n
}

// ErrorList ports components/error-list: validation errors above a form.
func ErrorList(errs []string) g.Node {
	return h.Div(h.Class("error-list"), h.Role("alert"),
		h.Ul(g.Map(errs, func(e string) g.Node { return h.Li(g.Text(e)) })),
	)
}

// Capitalize upper-cases the first letter, like PHP's ucfirst.
func Capitalize(s string) string {
	r, n := utf8.DecodeRuneInString(s)
	if n == 0 {
		return s
	}
	return string(unicode.ToUpper(r)) + s[n:]
}

// Plural returns "1 repository" or "3 repositories".
func Plural(n int, one, many string) string {
	if n == 1 {
		return "1 " + one
	}
	return strconv.Itoa(n) + " " + many
}
