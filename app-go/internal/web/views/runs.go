package views

import (
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/laposa/testflow/app-go/internal/store"
	"github.com/laposa/testflow/app-go/internal/web/views/hx"
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// RunsID is the id of the run list block, which htmx replaces when it
// polls, starts runs or refreshes them. RunsFormID is its form, whose
// fields (selected and expanded services) every request includes, so a
// swap keeps them. RunsErrorsID is the message area above the table.
const (
	RunsID       = "runs"
	RunsFormID   = "runs-form"
	RunsErrorsID = "runs-errors"
)

// PollInterval is how often the session page polls the run list while
// runs are syncing (spec §8.5).
const PollInterval = "30s"

// RunListView is the run list of the session page (spec §8.4, §8.5).
type RunListView struct {
	SessionID int64
	Services  []store.Service
	Runs      []store.Run // by service, newest first
	Selected  map[int64]bool
	Expanded  map[int64]bool // services whose older runs are shown
	Now       time.Time
	Messages  []string // what went wrong starting or refreshing runs
}

// Syncing reports whether any run is still being synced, i.e. whether the
// list polls.
func (v RunListView) Syncing() bool {
	for _, r := range v.Runs {
		if r.Syncing(v.Now) {
			return true
		}
	}
	return false
}

func (v RunListView) base() string { return "/sessions/" + strconv.FormatInt(v.SessionID, 10) }

// RunList ports livewire/runs/list-runs.blade.php. It is a normal form:
// "Run" posts the selected services (all if none), and the toggle and
// "Refresh now" buttons work without JS through formaction. With htmx they
// replace this block instead. While runs are syncing, the block polls
// itself every 30 s and stops once nothing is left to sync (spec §8.5).
func RunList(v RunListView) g.Node {
	base := v.base()
	selected := 0
	for _, svc := range v.Services {
		if v.Selected[svc.ID] {
			selected++
		}
	}
	return h.Div(h.ID(RunsID), h.Class("runs"),
		g.If(v.Syncing(), g.Group{
			hx.Get(base + "/runs"), hx.Trigger("every " + PollInterval), hx.Include("#" + RunsFormID),
			hx.Swap("outerHTML"),
		}),
		h.Form(h.ID(RunsFormID), h.Method("post"), h.Action(base+"/runs"),
			hx.Post(base+"/runs"), hx.Target("#"+RunsID), hx.Swap("outerHTML"), hx.Sync("this:drop"),
			h.Div(h.ID(RunsErrorsID), h.Class("field-error"), h.Role("alert"), g.Map(v.Messages, ErrorText)),
			g.Map(v.Services, func(svc store.Service) g.Node {
				return g.If(v.Expanded[svc.ID], h.Input(h.Type("hidden"), h.Name("expand"), h.Value(id(svc.ID))))
			}),
			h.Table(h.Class("runs-list"),
				h.THead(h.Tr(
					h.Th(), h.Th(g.Text("Service")), h.Th(g.Text("Run ID")), h.Th(g.Text("Initiated")),
					h.Th(g.Text("Duration")), h.Th(g.Text("Results")), h.Th(),
				)),
				h.TBody(g.Map(v.Services, func(svc store.Service) g.Node { return serviceRows(v, svc) })),
			),
			h.Div(h.Class("list-runs-actions"),
				h.Button(h.Type("submit"), h.Class("button filled"), g.Attr("data-run-button"), g.Text(runLabel(selected))),
				h.Button(h.Type("submit"), h.Class("button"), h.FormAction(base+"/runs/sync"), h.FormNoValidate(),
					hx.Post(base+"/runs/sync"), hx.Target("#"+RunsID), hx.Swap("outerHTML"),
					h.Title("Check GitHub for news about the runs now"), g.Text("Refresh now")),
			),
		),
	)
}

// runLabel is the legacy button text; app.js updates it as boxes change.
func runLabel(selected int) string {
	if selected == 0 {
		return "Run All Tests"
	}
	return "Run Selected Tests (" + strconv.Itoa(selected) + ")"
}

// serviceRows is a service's latest run and, when expanded, its older runs.
func serviceRows(v RunListView, svc store.Service) g.Node {
	var runs []store.Run
	for _, r := range v.Runs {
		if r.ServiceID == svc.ID {
			runs = append(runs, r)
		}
	}
	manual := svc.Type == store.RunManual
	box := h.Input(h.Type("checkbox"), h.Name("services"), h.Value(id(svc.ID)), h.Class("run-selector"),
		g.Attr("aria-label", "Select "+svc.DisplayName()),
		g.If(v.Selected[svc.ID] && !manual, h.Checked()),
		// Manual runs come in iteration 10.
		g.If(manual, g.Group{h.Disabled(), h.Title("Manual runs are not available yet.")}),
	)
	if len(runs) == 0 {
		return h.Tr(h.Td(box), h.Td(g.Text(svc.DisplayName())),
			h.Td(g.Text("No runs")), h.Td(g.Text("-")), h.Td(g.Text("-")), h.Td(g.Text("-")), h.Td())
	}
	expanded := v.Expanded[svc.ID]
	var toggle g.Node
	if len(runs) > 1 {
		label, icon := "Show older runs", "plus.svg"
		if expanded {
			label, icon = "Hide older runs", "minus.svg"
		}
		toggle = h.Button(h.Type("submit"), h.Class("expand-icon"), h.Name("toggle"), h.Value(id(svc.ID)),
			h.FormMethod("get"), h.FormAction(v.base()), h.FormNoValidate(),
			hx.Get(v.base()+"/runs"), hx.Include("#"+RunsFormID), hx.Target("#"+RunsID), hx.Swap("outerHTML"),
			g.Attr("aria-expanded", strconv.FormatBool(expanded)), h.Title(label),
			h.Img(h.Src("/static/icons/"+icon), h.Alt(label), h.Class("icon")),
		)
	}
	rows := g.Group{h.Tr(h.Td(box), h.Td(g.Text(svc.DisplayName())), runCells(v, runs[0]), h.Td(toggle))}
	if expanded {
		for _, r := range runs[1:] {
			rows = append(rows, h.Tr(h.Class("more-runs-list"), h.Td(), h.Td(), runCells(v, r), h.Td()))
		}
	}
	return rows
}

// runCells are the run id, initiated, duration and results cells.
func runCells(v RunListView, r store.Run) g.Node {
	var duration g.Node
	switch {
	case r.FinishedAt != nil:
		duration = g.Text(HumanDuration(r.FinishedAt.Sub(runStart(r))))
	case r.Syncing(v.Now):
		duration = h.Span(h.Class("running"), g.Text("Running"))
	default:
		duration = g.Text("-")
	}
	var results g.Node
	switch {
	case r.HasResults && r.FinishedAt != nil:
		results = PassedFailed(r.Passed, r.Failed, r.Skipped, false)
	case !r.Syncing(v.Now) || r.FinishedAt != nil:
		// Finished without a report, or given up on: say how it ended.
		results = h.Span(h.Class("run-status run-status--"+r.Status), g.Text(r.Status))
	}
	return g.Group{
		h.Td(h.A(h.Href(v.base()+"/runs/"+id(r.ID)), h.Title("Show logs"), g.Text(id(r.ID)))),
		h.Td(Timestamp(r.CreatedAt)),
		h.Td(g.Iff(r.StartedAt != nil, func() g.Node {
			return h.Title("Picked by runner at " + r.StartedAt.Format("2 Jan 2006 15:04:05"))
		}), duration),
		h.Td(results),
	}
}

// runStart is when a runner picked the run up, or when it was dispatched.
func runStart(r store.Run) time.Time {
	if r.StartedAt != nil {
		return *r.StartedAt
	}
	return r.CreatedAt
}

// HumanDuration is d in words, largest unit first: "1 hour 2 minutes 5
// seconds" (legacy: CarbonInterval::cascade()->forHumans()).
func HumanDuration(d time.Duration) string {
	if d < time.Second {
		return "0 seconds"
	}
	units := []struct {
		size      time.Duration
		one, many string
	}{
		{24 * time.Hour, "day", "days"},
		{time.Hour, "hour", "hours"},
		{time.Minute, "minute", "minutes"},
		{time.Second, "second", "seconds"},
	}
	var parts []string
	for _, u := range units {
		if n := int(d / u.size); n > 0 {
			parts = append(parts, Plural(n, u.one, u.many))
			d -= time.Duration(n) * u.size
		}
	}
	return strings.Join(parts, " ")
}

// RunPage ports resources/views/sessions/run.blade.php: one run with each
// test's result, the JUnit report and the run log.
func RunPage(p Page, ts store.TestSession, svc store.Service, d store.RunDetail, now time.Time) g.Node {
	base := "/sessions/" + id(ts.ID)
	var when g.Node
	switch {
	case d.FinishedAt != nil:
		when = g.Text(" and ran for " + HumanDuration(d.FinishedAt.Sub(runStart(d.Run))) + ".")
	case d.Syncing(now):
		when = g.Text(" and is still running …")
	default:
		when = g.Text(" and never reported back.")
	}
	return Layout(p,
		Section(ts.Name, true,
			h.Div(h.Class("session-environment"),
				g.Text("Environment: "), h.Span(g.Text(Capitalize(ts.Environment))),
			),
			h.P(h.A(h.Href(base), g.Text("← Back to the session"))),
			h.P(g.Text("Run #"+id(d.ID)+" for "+svc.DisplayName()+".")),
			h.P(g.Text("Started at "), Timestamp(d.CreatedAt), when),
			h.P(g.Text("Status: "), h.Span(h.Class("run-status run-status--"+d.Status), g.Text(d.Status))),
			g.If(d.HasResults, h.P(PassedFailed(d.Passed, d.Failed, d.Skipped, true))),
			h.Section(
				h.Ul(h.Class("session-selected-suites run-tests"),
					g.Map(svc.Suites, func(suite store.Suite) g.Node {
						return h.Li(g.Text(suite.Name),
							h.Ul(g.Map(suite.Tests, func(t store.Test) g.Node { return runTest(base, d, t) })),
						)
					}),
				),
				g.If(d.HasResults, h.Details(h.Class("run-output"),
					h.Summary(g.Text("Results in JUnit format")),
					h.Pre(g.Text(d.ResultLog)),
				)),
			),
			g.If(d.RunLog != "", h.Details(h.Class("run-output"),
				h.Summary(g.Text("Detailed run log")),
				h.Pre(g.Text(StripANSI(d.RunLog))),
			)),
		),
	)
}

// runTest is one test of the run page: its name links to the file at the
// run's commit, followed by its result, or "not found" if the report does
// not mention it.
func runTest(base string, d store.RunDetail, t store.Test) g.Node {
	res, ok := d.Results[t.ID]
	var status g.Node
	switch {
	case ok:
		title := "Execution time " + strconv.FormatFloat(res.Duration, 'f', 1, 64) + "s"
		if res.Status == "error" {
			title = "Error. " + title
		}
		status = h.Span(h.Class("test-status "+res.Status), h.Title(title), g.Attr("aria-label", res.Status))
	case d.HasResults:
		status = h.Span(h.Class("test-status not-found"), g.Text("not found"))
	}
	return g.Group{
		h.Li(h.Class("test-file"),
			h.A(h.Href(base+"/runs/"+id(d.ID)+"/tests/"+id(t.ID)), h.Target("_blank"), h.Rel("noopener"),
				h.Title("Show file contents"), g.Text(t.Name)),
			g.Text(" "), status,
		),
		g.If(ok && res.Comment != "", h.Div(h.Class("comment"), h.I(g.Text(res.Comment)))),
	}
}

// ansiEscape matches ANSI escape sequences (colours, cursor moves) in logs.
var ansiEscape = regexp.MustCompile(`\x1b\[[0-?]*[ -/]*[@-~]|\x1b[@-Z\\-_]`)

// StripANSI removes ANSI escape sequences, which GitHub run logs use for
// colour. Legacy turned them into coloured HTML with ansi_up; plain text
// keeps the CSP self-only and needs no script (D17).
func StripANSI(s string) string { return ansiEscape.ReplaceAllString(s, "") }

func id(n int64) string { return strconv.FormatInt(n, 10) }
