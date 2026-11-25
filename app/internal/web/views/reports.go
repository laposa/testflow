package views

import (
	"cmp"
	"slices"
	"time"

	"github.com/laposa/testflow/app-go/internal/store"
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// ReportView is a session report (spec §8.11).
type ReportView struct {
	Session store.TestSession
	Runs    []store.Run // by service, newest first (store.SessionRuns)
	Full    bool
	Results map[int64]map[int64]store.TestResult // by run and test; Full only
	Now     time.Time
}

// ReportPage ports sessions/report.blade.php. The executive report is the
// session's name, environment and run list (each service's latest run,
// as legacy printed it); the full report adds every run with each test's
// result, one run per printed page.
func ReportPage(p Page, v ReportView) g.Node {
	ts := v.Session
	return Layout(p,
		Section(ts.Name, true,
			h.Div(h.Class("report-actions"),
				h.A(h.Href("/sessions/"+id(ts.ID)), g.Text("← Back to the session")),
				// app.js shows it and prints on click (no inline script, §9).
				h.Button(h.Type("button"), h.Class("button filled"), g.Attr("data-print"), g.Attr("hidden"), g.Text("Print")),
			),
			h.Div(h.Class("session-environment"),
				g.Text("Environment: "), h.Span(g.Text(Capitalize(ts.Environment))),
			),
			reportRunList(v),
		),
		g.If(v.Full, g.Group(fullRuns(v))),
	)
}

// reportRunList is the run list without its form: no checkboxes, actions,
// older runs or polling.
func reportRunList(v ReportView) g.Node {
	rl := RunListView{SessionID: v.Session.ID, Services: v.Session.Services, Runs: v.Runs, Now: v.Now}
	return h.Table(h.Class("runs-list report-runs"),
		h.THead(h.Tr(
			h.Th(g.Text("Service")), h.Th(g.Text("Run ID")), h.Th(g.Text("Initiated")),
			h.Th(g.Text("Duration")), h.Th(g.Text("Results")),
		)),
		h.TBody(g.Map(v.Session.Services, func(svc store.Service) g.Node {
			i := slices.IndexFunc(v.Runs, func(r store.Run) bool { return r.ServiceID == svc.ID })
			if i < 0 {
				return h.Tr(h.Td(g.Text(svc.DisplayName())), h.Td(g.Text("No runs")),
					h.Td(g.Text("-")), h.Td(g.Text("-")), h.Td(g.Text("-")))
			}
			return h.Tr(h.Td(g.Text(svc.DisplayName())), runCells(rl, v.Runs[i], false))
		})),
	)
}

// fullRuns are the full report's runs, oldest first (legacy:
// $session->runs), each on its own printed page.
func fullRuns(v ReportView) []g.Node {
	runs := slices.Clone(v.Runs)
	slices.SortFunc(runs, func(a, b store.Run) int { return cmp.Compare(a.ID, b.ID) })
	var out []g.Node
	for _, r := range runs {
		i := slices.IndexFunc(v.Session.Services, func(s store.Service) bool { return s.ID == r.ServiceID })
		if i < 0 {
			continue
		}
		svc := v.Session.Services[i]
		out = append(out, h.Section(h.Class("page-break report-run"),
			h.H3(g.Text("Run #"+id(r.ID)+" for "+svc.DisplayName())),
			h.P(g.Text("Started at "), Timestamp(r.CreatedAt), g.Text(runWhen(r, v.Now))),
			h.P(g.Iff(r.HasTotals, func() g.Node { return PassedFailed(r.Passed, r.Failed, r.Skipped, true) }),
				g.If(!r.HasTotals, h.Span(h.Class("run-status run-status--"+r.Status), g.Text(r.Status)))),
			h.Ul(h.Class("session-selected-suites run-tests"),
				g.Map(svc.Suites, func(suite store.Suite) g.Node {
					return h.Li(g.Text(suite.Name),
						h.Ul(g.Map(suite.Tests, func(t store.Test) g.Node { return reportTest(r, v.Results[r.ID], t) })),
					)
				}),
			),
		))
	}
	return out
}

// reportTest is a test's name, result icon and time, written out since a
// printed page has no tooltips, and its comment. Manual tests have no
// time.
func reportTest(r store.Run, results map[int64]store.TestResult, t store.Test) g.Node {
	res, ok := results[t.ID]
	return h.Li(h.Class("test-file"), g.Text(t.Name+" "), testStatus(r, res, ok),
		g.If(ok && r.Type == store.RunAutomated, h.Span(h.Class("test-time"), g.Text(" "+seconds(res.Duration)))),
		testComment(res, ok),
	)
}
