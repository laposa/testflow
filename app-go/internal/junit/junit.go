// Package junit parses JUnit XML test reports and matches TestFlow tests to
// their test cases (spec §8.6; legacy: app/Services/JUnitLogParser.php).
//
// Reports come from the tested repositories' workflows (Cypress via
// mocha-junit-reporter, Maestro, Newman, ...), so parsing is lenient: a
// missing or malformed number counts as 0, like PHP's casts in legacy.
package junit

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"

	"golang.org/x/text/encoding/htmlindex"
)

// Status is a test case result. The values are those of
// run_test_results.status (spec §6.8).
type Status string

const (
	StatusPass    Status = "pass"
	StatusFail    Status = "fail"
	StatusSkipped Status = "skipped"
	StatusError   Status = "error"
)

// Report is a parsed JUnit file.
type Report struct {
	Suites []Suite
}

// Suite is a <testsuite>. The counts are its attributes; a missing attribute
// is counted from its test cases instead.
type Suite struct {
	Name     string
	Tests    int
	Failures int
	Errors   int
	Skipped  int
	Time     float64 // seconds
	Cases    []Case
}

// Case is a <testcase>.
type Case struct {
	Name           string
	ClassName      string
	Time           float64 // seconds
	Status         Status
	FailureMessage string // text of <failure> or <error>, or its message attribute
	Comment        string // text of <system-out>
}

// Totals are the run's counts (spec §8.6).
type Totals struct {
	Tests    int
	Passed   int
	Failed   int // failures and errors
	Skipped  int
	Duration float64 // seconds
}

// Parse reads a JUnit report. The root is <testsuites>, or a single
// <testsuite> as some reporters write it. Nested suites are flattened.
// A declared non-UTF-8 encoding (ISO-8859-1, windows-1252, ...) is decoded.
func Parse(data []byte) (*Report, error) {
	d := xml.NewDecoder(bytes.NewReader(data))
	d.CharsetReader = charsetReader
	for {
		tok, err := d.Token()
		if errors.Is(err, io.EOF) {
			return nil, errors.New("junit: no root element")
		}
		if err != nil {
			return nil, fmt.Errorf("junit: %w", err)
		}
		start, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}
		var rep Report
		switch start.Name.Local {
		case "testsuites":
			var root struct {
				Suites []xmlSuite `xml:"testsuite"`
			}
			if err := d.DecodeElement(&root, &start); err != nil {
				return nil, fmt.Errorf("junit: %w", err)
			}
			for _, s := range root.Suites {
				rep.Suites = s.appendSuites(rep.Suites, 0)
			}
		case "testsuite":
			var s xmlSuite
			if err := d.DecodeElement(&s, &start); err != nil {
				return nil, fmt.Errorf("junit: %w", err)
			}
			rep.Suites = s.appendSuites(nil, 0)
		default:
			return nil, fmt.Errorf("junit: root element is <%s>, want <testsuites> or <testsuite>", start.Name.Local)
		}
		return &rep, nil
	}
}

// Totals adds up the suites: passed = tests − failed − skipped, never below
// zero. Duration is the sum of the suite times; the root's time is not
// added, because reporters put the overall total there (legacy summed the
// suites only, too). The counts are capped at maxTotal so they fit the
// database's integer columns; a duration that overflows to infinity is 0.
func (r *Report) Totals() Totals {
	var t Totals
	for _, s := range r.Suites {
		t.Tests += s.Tests
		t.Failed += s.Failures + s.Errors
		t.Skipped += s.Skipped
		t.Duration += s.Time
	}
	t.Tests, t.Failed, t.Skipped = min(t.Tests, maxTotal), min(t.Failed, maxTotal), min(t.Skipped, maxTotal)
	t.Passed = max(t.Tests-t.Failed-t.Skipped, 0)
	if math.IsInf(t.Duration, 0) {
		t.Duration = 0
	}
	return t
}

// Find returns the first test case for a TestFlow test file name (legacy:
// getTestCase). Both sides are lower-cased with "_" read as a space, and the
// file name is cut at its first "."; it matches a case's name or class name.
// So "pay_with_card.cy.js" matches a case named "Pay with card".
// An empty name (say ".gitkeep") matches nothing, and neither does a case's
// empty name or class name.
func (r *Report) Find(testName string) (Case, bool) {
	name, _, _ := strings.Cut(normalize(testName), ".")
	if name == "" {
		return Case{}, false
	}
	for _, s := range r.Suites {
		for _, c := range s.Cases {
			if normalize(c.Name) == name || normalize(c.ClassName) == name {
				return c, true
			}
		}
	}
	return Case{}, false
}

func normalize(s string) string { return strings.ToLower(strings.ReplaceAll(s, "_", " ")) }

type xmlSuite struct {
	Name     string     `xml:"name,attr"`
	Tests    *string    `xml:"tests,attr"`
	Failures *string    `xml:"failures,attr"`
	Errors   *string    `xml:"errors,attr"`
	Skipped  *string    `xml:"skipped,attr"`
	Time     string     `xml:"time,attr"`
	Cases    []xmlCase  `xml:"testcase"`
	Suites   []xmlSuite `xml:"testsuite"` // nested suites (mocha, some others)
}

type xmlCase struct {
	Name      string      `xml:"name,attr"`
	ClassName string      `xml:"classname,attr"`
	Time      string      `xml:"time,attr"`
	Status    string      `xml:"status,attr"`
	Skipped   string      `xml:"skipped,attr"` // legacy's manual reports: skipped="1"
	Failure   *xmlProblem `xml:"failure"`
	Error     *xmlProblem `xml:"error"`
	SkippedEl *struct{}   `xml:"skipped"`
	SystemOut string      `xml:"system-out"`
}

type xmlProblem struct {
	Message string `xml:"message,attr"`
	Text    string `xml:",chardata"`
}

func (p *xmlProblem) text() string {
	if t := strings.TrimSpace(p.Text); t != "" {
		return t
	}
	return strings.TrimSpace(p.Message)
}

// maxDepth bounds suite nesting, so a hostile report cannot recurse deeply.
const maxDepth = 64

// appendSuites flattens x and its nested suites onto dst, parent first. A
// parent's count and time attributes usually total its children, so a suite
// with children is counted from its own cases only, and left out when it
// has none.
func (x xmlSuite) appendSuites(dst []Suite, depth int) []Suite {
	if len(x.Suites) == 0 {
		return append(dst, x.suite())
	}
	if len(x.Cases) > 0 {
		own := x
		own.Tests, own.Failures, own.Errors, own.Skipped, own.Time = nil, nil, nil, nil, ""
		s := own.suite()
		for _, c := range s.Cases {
			s.Time += c.Time
		}
		dst = append(dst, s)
	}
	if depth >= maxDepth {
		return dst
	}
	for _, c := range x.Suites {
		dst = c.appendSuites(dst, depth+1)
	}
	return dst
}

func (x xmlSuite) suite() Suite {
	s := Suite{Name: x.Name, Time: number(x.Time)}
	var failures, errs, skipped int
	for _, xc := range x.Cases {
		c := xc.testCase()
		s.Cases = append(s.Cases, c)
		switch c.Status {
		case StatusFail:
			failures++
		case StatusError:
			errs++
		case StatusSkipped:
			skipped++
		}
	}
	s.Tests = count(x.Tests, len(s.Cases))
	s.Failures = count(x.Failures, failures)
	s.Errors = count(x.Errors, errs)
	s.Skipped = count(x.Skipped, skipped)
	return s
}

// testCase works out the status like legacy, in rising precedence: the
// status attribute (pass if missing), a <failure> (fail), and a skip. An
// <error> element, which legacy ignored, is an error unless it failed too.
func (x xmlCase) testCase() Case {
	c := Case{
		Name:      x.Name,
		ClassName: x.ClassName,
		Time:      number(x.Time),
		Status:    statusAttr(x.Status),
		Comment:   strings.TrimSpace(x.SystemOut),
	}
	if x.Error != nil {
		c.Status, c.FailureMessage = StatusError, x.Error.text()
	}
	if x.Failure != nil {
		c.Status, c.FailureMessage = StatusFail, x.Failure.text()
	}
	if x.SkippedEl != nil || number(x.Skipped) > 0 {
		c.Status = StatusSkipped
	}
	return c
}

// statusAttr maps a status attribute to a Status. Legacy stored the value
// as is; reporters differ (Maestro writes SUCCESS and ERROR), and the
// database accepts only the Status values. Unknown values count as passed,
// like a missing one: failures are reported by <failure> and <error>.
func statusAttr(s string) Status {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "fail", "failed", "failure":
		return StatusFail
	case "error", "errored", "broken":
		return StatusError
	case "skip", "skipped", "pending", "ignored", "disabled", "canceled", "cancelled":
		return StatusSkipped
	default:
		return StatusPass
	}
}

// maxCount caps count attributes, and maxTotal the summed totals, so a
// bogus value cannot overflow the database's integer columns.
const (
	maxCount = 1 << 30
	maxTotal = math.MaxInt32
)

// charsetReader decodes a declared non-UTF-8 encoding for encoding/xml.
func charsetReader(label string, input io.Reader) (io.Reader, error) {
	enc, err := htmlindex.Get(label)
	if err != nil {
		return nil, fmt.Errorf("unsupported encoding %q", label)
	}
	return enc.NewDecoder().Reader(input), nil
}

// count is a count attribute, or n if it is missing.
func count(attr *string, n int) int {
	if attr == nil {
		return n
	}
	return int(min(number(*attr), maxCount))
}

// number parses a numeric attribute. Anything malformed, negative, NaN or
// infinite is 0.
func number(s string) float64 {
	f, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil || f < 0 || math.IsNaN(f) || math.IsInf(f, 0) {
		return 0
	}
	return f
}
