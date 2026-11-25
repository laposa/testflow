package views

import (
	"strconv"

	"github.com/laposa/testflow/app-go/internal/store"
	"github.com/laposa/testflow/app-go/internal/testcase"
	"github.com/laposa/testflow/app-go/internal/web/views/hx"
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// ManualDialogID is the session page's dialog that holds the manual run
// wizard; ManualContentID is the block inside it (and on the wizard page)
// that the wizard's requests replace.
const (
	ManualDialogID  = "manual-run"
	ManualContentID = "manual-run-content"
)

// ManualRunView is the manual run wizard at one test (spec §8.7).
type ManualRunView struct {
	SessionID int64
	Run       store.RunDetail
	Service   store.Service
	Tests     []store.Test // in wizard order
	Index     int          // the current test

	Case       *testcase.TestCase // nil if it could not be loaded or parsed
	Warnings   []string           // e.g. the name does not match the file name
	LoadError  string             // the file could not be read from GitHub
	Raw        string             // the file, when it does not parse
	ParseError string

	Draft  *store.TestResult // what the tester sent, when it is re-shown with Errors
	Errors []string
}

func (v ManualRunView) base() string {
	return "/sessions/" + id(v.SessionID) + "/runs/" + id(v.Run.ID)
}

// ManualRunPage is the wizard as a page, used without JS. The dialog on the
// session page shows the same block.
func ManualRunPage(p Page, ts store.TestSession, v ManualRunView) g.Node {
	return Layout(p,
		Section(ts.Name, true,
			h.P(h.A(h.Href("/sessions/"+id(ts.ID)), g.Text("← Back to the session"))),
			h.Div(h.ID(ManualContentID), ManualRun(v)),
		),
	)
}

// ManualDialog is the empty wizard dialog of the session page. A "Complete
// manual tests" link loads the wizard into it with htmx and app.js opens
// it; closing it refreshes the run list.
func ManualDialog() g.Node {
	return h.Dialog(h.ID(ManualDialogID), h.Class("manual-dialog"), g.Attr("aria-label", "Manual run"),
		h.Div(h.Class("manual-dialog-body"),
			h.Form(h.Method("dialog"), h.Class("manual-dialog-close"),
				h.Button(h.Type("submit"), h.Class("close"), g.Attr("aria-label", "Close"), g.Text("×")),
			),
			// Shown until the wizard arrives; app.js puts it back on open.
			h.Div(h.ID(ManualContentID), h.Div(h.Class("loading"), LoadingOverlay("", "Loading the test", true))),
		),
	)
}

// ManualRun ports livewire/runs/manual-test-run (the pagination) and
// manual-test-form: one test case with its steps, and the result form.
// Every link and form works without JS; htmx replaces #manual-run-content
// with the answer instead.
func ManualRun(v ManualRunView) g.Node {
	if v.Run.Status != store.RunInProgress {
		return h.Div(h.Class("manual-wizard"),
			g.If(len(v.Errors) > 0, ErrorList(v.Errors)),
			h.P(h.Role("status"), g.Text("Run #"+id(v.Run.ID)+" for "+v.Service.DisplayName()+" is finished. ")),
			h.P(h.A(h.Href("/sessions/"+id(v.SessionID)+"/runs/"+id(v.Run.ID)), g.Text("Show the results"))),
		)
	}
	t := v.Tests[v.Index]
	res, answered := v.Run.Results[t.ID]
	if v.Draft != nil {
		res, answered = *v.Draft, true
	}
	return h.Div(h.Class("manual-wizard"),
		pagination(v),
		h.Div(h.Class("manual-test"),
			testCaseHeading(v, t),
			g.If(len(v.Errors) > 0, ErrorList(v.Errors)),
			testCaseBody(v),
			h.Form(h.Method("post"), h.Action(v.base()+"/manual/"+id(t.ID)),
				hx.Post(v.base()+"/manual/"+id(t.ID)), hx.Target("#"+ManualContentID), hx.Swap("innerHTML"),
				hx.Sync("this:drop"),
				h.Textarea(h.Name("comment"), h.Placeholder("Additional information"), g.Attr("aria-label", "Comment"),
					h.MaxLength("10000"), g.Text(res.Comment)),
				h.Div(h.Class("test-status"),
					h.Div(h.Class("statuses"), h.Role("radiogroup"), g.Attr("aria-label", "Result"),
						resultChoice(store.ResultPass, "Pass", answered && res.Status == store.ResultPass),
						resultChoice(store.ResultFail, "Fail", answered && res.Status == store.ResultFail),
					),
					h.Div(h.Class("actions"),
						h.Button(h.Type("submit"), h.Name("skip"), h.Value("1"), h.Class("skip"), h.FormNoValidate(),
							g.Text("Skip")),
						h.Button(h.Type("submit"), h.Class("submit"), g.Text(submitLabel(v))),
					),
				),
			),
		),
	)
}

func submitLabel(v ManualRunView) string {
	if v.Index == len(v.Tests)-1 {
		return "Submit and finish"
	}
	return "Submit"
}

// pagination is the legacy "‹ page / total ›" bar: links, and a GET form
// whose number field jumps to a test.
func pagination(v ManualRunView) g.Node {
	url := v.base() + "/manual"
	link := func(class, label string, index int) g.Node {
		if index < 0 || index >= len(v.Tests) {
			return h.Span(h.Class(class+" disabled"), g.Attr("aria-hidden", "true"))
		}
		href := url + "?test=" + strconv.Itoa(index+1)
		return h.A(h.Class(class), h.Href(href), hx.Get(href), hx.Target("#"+ManualContentID), hx.Swap("innerHTML"),
			g.Attr("aria-label", label))
	}
	return h.Nav(h.Class("test-pagination"), g.Attr("aria-label", "Tests"),
		link("prev", "Previous test", v.Index-1),
		// Enter fires change and submit: htmx handles both (and cancels the
		// normal submit, which would leave the dialog), and replace keeps
		// only the last request.
		h.Form(h.Method("get"), h.Action(url), h.Class("pages"),
			hx.Get(url), hx.Target("#"+ManualContentID), hx.Swap("innerHTML"), hx.Trigger("submit, change"),
			hx.Sync("this:replace"),
			h.Input(h.Type("number"), h.Name("test"), h.Class("test-pagination-input"), h.Min("1"),
				h.Max(strconv.Itoa(len(v.Tests))), h.Value(strconv.Itoa(v.Index+1)), g.Attr("aria-label", "Test number")),
			g.Text(" / "+strconv.Itoa(len(v.Tests))),
		),
		link("next", "Next test", v.Index+1),
	)
}

// testCaseHeading is "suite / name" and the description, from the file
// when it parsed, otherwise from the session.
func testCaseHeading(v ManualRunView, t store.Test) g.Node {
	suite, name, description := suiteOf(v.Service, t.ID), t.Name, ""
	if c := v.Case; c != nil {
		suite, name, description = c.Suite, c.Name, c.Description
	}
	return h.Div(h.Class("heading"),
		h.H2(h.Span(h.Title("suite"), g.Text(suite)), g.Text(" / "), h.Span(h.Title("test name"), g.Text(name))),
		h.Div(h.Class("description"),
			g.If(description != "", h.Span(h.Title("description"), g.Text(description))),
			g.If(len(v.Warnings) > 0, h.Div(h.Class("error"), h.Role("alert"),
				g.Text("This test contains the following errors, which may affect the result of the run:"),
				h.Ul(g.Map(v.Warnings, func(w string) g.Node { return h.Li(g.Text(w)) })),
			)),
		),
	)
}

func suiteOf(svc store.Service, testID int64) string {
	for _, suite := range svc.Suites {
		for _, t := range suite.Tests {
			if t.ID == testID {
				return suite.Name
			}
		}
	}
	return ""
}

// testCaseBody is the steps table, or what went wrong reading the file.
func testCaseBody(v ManualRunView) g.Node {
	switch {
	case v.LoadError != "":
		return h.P(h.Class("alert"), h.Role("alert"), g.Text(v.LoadError))
	case v.Case == nil:
		return g.Group{
			h.P(h.Class("alert"), h.Role("alert"),
				g.Text("The test case could not be read ("+v.ParseError+"). This is the file as it is:")),
			h.Pre(h.Class("manual-test-raw"), g.Text(v.Raw)),
		}
	}
	return h.Table(
		h.THead(h.Tr(h.Th(g.Text("Step")), h.Th(g.Text("Action")), h.Th(g.Text("Expected result")), h.Th())),
		h.TBody(g.Map(indexed(v.Case.Steps), func(s indexedStep) g.Node {
			n := strconv.Itoa(s.i + 1)
			return h.Tr(
				h.Td(g.Text(n)),
				h.Td(lines(s.Action), g.If(len(s.Input) > 0, h.Div(h.Class("input"),
					g.Map(s.Input, func(in testcase.Input) g.Node { return inputLine(in) }),
				))),
				h.Td(h.Ul(h.Class("bullet-list"), g.Map(s.Result, func(r string) g.Node { return h.Li(g.Text(r)) }))),
				// Aids only: app.js picks Pass when every step passed and
				// Fail when one failed. They are not sent.
				h.Td(h.Div(h.Class("radio-wrapper"),
					stepRadio(n, "pass", "Mark step "+n+" as passed"),
					stepRadio(n, "fail", "Mark step "+n+" as failed"),
				)),
			)
		})),
	)
}

type indexedStep struct {
	i int
	testcase.Step
}

func indexed(steps []testcase.Step) []indexedStep {
	out := make([]indexedStep, len(steps))
	for i, s := range steps {
		out[i] = indexedStep{i, s}
	}
	return out
}

// lines is one action as text, several as a bullet list (legacy).
func lines(items []string) g.Node {
	switch len(items) {
	case 0:
		return nil
	case 1:
		return g.Text(items[0])
	}
	return h.Ul(h.Class("bullet-list"), g.Map(items, func(s string) g.Node { return h.Li(g.Text(s)) }))
}

// inputLine is "name: description: value" with a copy button on the value,
// or "name: value" for a plain value (legacy).
func inputLine(in testcase.Input) g.Node {
	if in.Plain {
		return h.Div(g.Text(in.Name + ": " + in.Value))
	}
	return h.Div(g.Text(in.Name+": "+in.Description+": "),
		h.Button(h.Type("button"), h.Class("copy-to-clipboard"), h.Title("Copy to clipboard"), g.Text(in.Value)))
}

func stepRadio(n, result, label string) g.Node {
	return h.Label(h.Class("custom-radio"), h.Title(label),
		h.Input(h.Type("radio"), h.Name("step-"+n), h.Class("radio-"+result), g.Attr("aria-label", label)),
		h.Div(h.Class("radio "+result)),
	)
}

func resultChoice(value, label string, checked bool) g.Node {
	return h.Label(
		h.Input(h.Type("radio"), h.Name("result"), h.Value(value), h.Required(), h.Class("result-"+value),
			g.If(checked, h.Checked())),
		h.Div(h.Class(value), g.Text(label)),
	)
}
