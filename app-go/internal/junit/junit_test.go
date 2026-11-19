package junit

import (
	"bytes"
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var update = flag.Bool("update", false, "rewrite the golden files in testdata/")

// TestGolden parses every testdata/*.xml and compares the report and its
// totals with the .golden.json next to it. Run with -update after checking
// a change by hand.
func TestGolden(t *testing.T) {
	files, err := filepath.Glob("testdata/*.xml")
	if err != nil || len(files) == 0 {
		t.Fatalf("no golden inputs: %v", err)
	}
	for _, in := range files {
		t.Run(filepath.Base(in), func(t *testing.T) {
			data, err := os.ReadFile(in)
			if err != nil {
				t.Fatal(err)
			}
			rep, err := Parse(data)
			if err != nil {
				t.Fatal(err)
			}
			got, err := json.MarshalIndent(struct {
				Report *Report
				Totals Totals
			}{rep, rep.Totals()}, "", "  ")
			if err != nil {
				t.Fatal(err)
			}
			got = append(got, '\n')
			golden := strings.TrimSuffix(in, ".xml") + ".golden.json"
			if *update {
				if err := os.WriteFile(golden, got, 0o644); err != nil {
					t.Fatal(err)
				}
				return
			}
			want, err := os.ReadFile(golden)
			if err != nil {
				t.Fatalf("%v (run go test ./internal/junit -update)", err)
			}
			if !bytes.Equal(got, want) {
				t.Errorf("%s differs from the parsed report:\n%s", golden, got)
			}
		})
	}
}

func TestFind(t *testing.T) {
	rep := &Report{Suites: []Suite{
		{Name: "a", Cases: []Case{{Name: "Checkout pay with card", ClassName: "pay with card", Status: StatusPass}}},
		{Name: "b", Cases: []Case{
			{Name: "Login_Flow", ClassName: "x", Status: StatusFail},
			{Name: "login flow", ClassName: "y", Status: StatusPass}, // shadowed by the first match
		}},
	}}
	tests := []struct {
		test   string
		want   string // class name of the match, "" for none
		status Status
	}{
		{"pay_with_card.cy.js", "pay with card", StatusPass}, // matches the class name
		{"PAY_WITH_CARD.cy.js", "pay with card", StatusPass},
		{"login_flow.yaml", "x", StatusFail}, // matches the name; first match wins
		{"login_flow", "x", StatusFail},      // no extension
		{"pay.with.card.cy.js", "", ""},      // cut at the first "."
		{"checkout.cy.js", "", ""},           // no substring matching
		{"", "", ""},
	}
	for _, tt := range tests {
		c, ok := rep.Find(tt.test)
		if ok != (tt.want != "") || c.ClassName != tt.want || c.Status != tt.status {
			t.Errorf("Find(%q) = %+v, %v; want class %q", tt.test, c, ok, tt.want)
		}
	}
}

func TestFindInParsedReport(t *testing.T) {
	data, err := os.ReadFile("testdata/maestro.xml")
	if err != nil {
		t.Fatal(err)
	}
	rep, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	c, ok := rep.Find("checkout.yaml")
	if !ok || c.Status != StatusFail || c.FailureMessage != "Element not found: Text matching regex: Pay" {
		t.Errorf("checkout = %+v, %v", c, ok)
	}
}

func TestParseErrors(t *testing.T) {
	for name, in := range map[string]string{
		"empty":      "",
		"only text":  "hello",
		"truncated":  `<testsuites><testsuite name="x"><testcase name="a">`,
		"wrong root": `<results><testsuite/></results>`,
		"html":       `<!doctype html><html><body>404</body></html>`,
	} {
		if rep, err := Parse([]byte(in)); err == nil {
			t.Errorf("%s: want an error, got %+v", name, rep)
		}
	}
	rep, err := Parse([]byte(`<testsuites/>`))
	if err != nil || len(rep.Suites) != 0 || rep.Totals() != (Totals{}) {
		t.Errorf("empty report = %+v, %v", rep, err)
	}
}

func TestTotalsNeverNegative(t *testing.T) {
	rep, err := Parse([]byte(`<testsuites><testsuite tests="1" failures="3" skipped="1" time="2"/></testsuites>`))
	if err != nil {
		t.Fatal(err)
	}
	if got := rep.Totals(); got.Passed != 0 || got.Failed != 3 || got.Tests != 1 {
		t.Errorf("totals = %+v", got)
	}
}

func TestNumbers(t *testing.T) {
	for in, want := range map[string]float64{
		"1.5": 1.5, " 2 ": 2, "": 0, "x": 0, "-1": 0, "NaN": 0, "Inf": 0, "1e400": 0, "1,5": 0,
	} {
		if got := number(in); got != want {
			t.Errorf("number(%q) = %v, want %v", in, got, want)
		}
	}
	big := "99999999999999999999"
	if got := count(&big, 0); got != maxCount {
		t.Errorf("count(%s) = %d, want the cap", big, got)
	}
	if got := count(nil, 4); got != 4 {
		t.Errorf("missing attribute = %d, want the counted 4", got)
	}
}

func TestStatusAttr(t *testing.T) {
	for in, want := range map[string]Status{
		"": StatusPass, "pass": StatusPass, "SUCCESS": StatusPass, "whatever": StatusPass,
		"fail": StatusFail, "FAILED": StatusFail, "Failure": StatusFail,
		"error": StatusError, "ERROR": StatusError,
		"skipped": StatusSkipped, "pending": StatusSkipped, "CANCELED": StatusSkipped,
	} {
		if got := statusAttr(in); got != want {
			t.Errorf("statusAttr(%q) = %q, want %q", in, got, want)
		}
	}
}
