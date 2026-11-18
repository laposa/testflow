package views

import (
	"strconv"
	"time"

	"github.com/laposa/testflow/app-go/internal/store"
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// SessionListPage is the session list on GET / and GET /sessions (spec §8.3;
// legacy: resources/views/sessions/index.blade.php and
// components/sessions/list.blade.php). The legacy filter bar is not ported:
// its search, date and select inputs were never wired up.
func SessionListPage(p Page, sessions []store.SessionSummary) g.Node {
	return Layout(p,
		h.Section(h.Class("sessions"),
			h.Div(h.Class("sessions__header"),
				h.H2(g.Text("All Sessions")),
				h.A(h.Href("/sessions/new"), h.Class("button filled"), g.Text("New session")),
			),
			h.Table(
				h.THead(h.Tr(
					h.Th(g.Text("Name")),
					h.Th(g.Text("Status")),
					h.Th(g.Text("Environment")),
					h.Th(g.Text("Created")),
					h.Th(g.Text("Issuer")),
				)),
				h.TBody(
					g.If(len(sessions) == 0,
						h.Tr(h.Td(h.ColSpan("5"), g.Text("No sessions yet."))),
					),
					g.Map(sessions, func(s store.SessionSummary) g.Node {
						return h.Tr(
							h.Td(h.A(h.Href("/sessions/"+strconv.FormatInt(s.ID, 10)), g.Text(s.Name))),
							h.Td(sessionStatus(s)),
							h.Td(g.Text(Capitalize(s.Environment))),
							h.Td(Timestamp(s.CreatedAt)),
							h.Td(g.Text(s.IssuerName)),
						)
					}),
				),
			),
		),
	)
}

// sessionStatus is "No runs yet", the status of a run that is still going,
// or the added-up results of the latest runs.
func sessionStatus(s store.SessionSummary) g.Node {
	switch {
	case s.Runs == 0:
		return h.Span(g.Text("No runs yet"))
	case s.Running():
		return h.Span(g.Text(s.RunningStatus))
	default:
		return PassedFailed(s.Passed, s.Failed, s.Skipped, false)
	}
}

// PassedFailed ports components/passed-failed.blade.php: "3 passed, 1 failed
// and 2 skipped", leaving out zero counts unless showZero is set, or "None"
// if all are zero. Legacy printed "3 passed and and 2 skipped" when only
// failed was zero; the parts are joined properly here.
func PassedFailed(passed, failed, skipped int, showZero bool) g.Node {
	type part struct {
		n           int
		class, word string
	}
	var parts []part
	for _, p := range []part{{passed, "pass", "passed"}, {failed, "fail", "failed"}, {skipped, "skip", "skipped"}} {
		if p.n > 0 || showZero {
			parts = append(parts, p)
		}
	}
	if len(parts) == 0 {
		return h.Span(h.Class("no-runs"), g.Text("None"))
	}
	var nodes g.Group
	for i, p := range parts {
		switch {
		case i == 0:
		case i == len(parts)-1:
			nodes = append(nodes, g.Text(" and "))
		default:
			nodes = append(nodes, g.Text(", "))
		}
		nodes = append(nodes, h.Span(h.Class(p.class), g.Text(strconv.Itoa(p.n)+" "+p.word)))
	}
	return nodes
}

// Timestamp renders t as "2 Jan 2006 15:04" in a <time> element. Legacy's
// list used 'd M, Y h:m', which showed the month where the minutes belong.
func Timestamp(t time.Time) g.Node {
	return h.Time(h.DateTime(t.UTC().Format(time.RFC3339)), g.Text(t.Format("2 Jan 2006 15:04")))
}
