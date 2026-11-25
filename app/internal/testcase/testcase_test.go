package testcase

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

// golden is what a test case file parses to: the test case and its
// warnings, or the error.
type golden struct {
	TestCase *TestCase `json:"testCase,omitempty"`
	Warnings []string  `json:"warnings,omitempty"`
	Error    string    `json:"error,omitempty"`
}

// TestGolden parses every testdata/*.manual.yaml and compares the result
// with the .golden.json next to it. Run with -update after checking a
// change by hand.
func TestGolden(t *testing.T) {
	inputs, err := filepath.Glob("testdata/*" + FileSuffix)
	if err != nil || len(inputs) == 0 {
		t.Fatalf("no golden inputs: %v", err)
	}
	for _, in := range inputs {
		t.Run(filepath.Base(in), func(t *testing.T) {
			data, err := os.ReadFile(in)
			if err != nil {
				t.Fatal(err)
			}
			var g golden
			if tc, err := Parse(data); err != nil {
				g.Error = err.Error()
			} else {
				g.TestCase, g.Warnings = &tc, tc.Warnings(filepath.Base(in))
			}
			got, err := json.MarshalIndent(g, "", "  ")
			if err != nil {
				t.Fatal(err)
			}
			got = append(got, '\n')
			path := strings.TrimSuffix(in, FileSuffix) + ".golden.json"
			if *update {
				if err := os.WriteFile(path, got, 0o644); err != nil {
					t.Fatal(err)
				}
				return
			}
			want, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("%v (run go test ./internal/testcase -update)", err)
			}
			if !bytes.Equal(got, want) {
				t.Errorf("%s differs from the parsed test case:\n%s", path, got)
			}
		})
	}
}

func TestEmpty(t *testing.T) {
	for _, in := range []string{"", "# only a comment\n"} {
		if _, err := Parse([]byte(in)); err == nil {
			t.Errorf("Parse(%q) succeeded", in)
		}
	}
}

func TestWarnings(t *testing.T) {
	tc := TestCase{Name: "login"}
	if w := tc.Warnings("login.manual.yaml"); len(w) != 0 {
		t.Errorf("matching name: %v", w)
	}
	if w := tc.Warnings("logout.manual.yaml"); len(w) != 1 {
		t.Errorf("other name: %v", w)
	}
}
