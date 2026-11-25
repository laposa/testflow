// Package testcase reads the *.manual.yaml test cases shown in the manual
// run wizard (spec §8.7, §8.8; legacy: app/Data/ManualTest*.php,
// FetchManualTestContent, helpers.php validateManualTest).
//
// The schema is owned by tests-validator/validate-test-cases.js, which the
// tested repositories run in CI. This package only reads what the wizard
// shows and checks that the name matches the file name, like legacy.
package testcase

import (
	"errors"
	"fmt"
	"strings"

	"go.yaml.in/yaml/v3"
)

// FileSuffix ends the file name of every manual test case.
const FileSuffix = ".manual.yaml"

// TestCase is one manual test case.
type TestCase struct {
	Name        string `json:"name"`
	Suite       string `json:"suite"`
	Description string `json:"description"`
	Steps       []Step `json:"steps"`
}

// Step is what the tester does and what they should see.
type Step struct {
	Action []string `json:"action"`
	Input  []Input  `json:"input,omitempty"` // in file order
	Result []string `json:"result"`
}

// Input is a value the tester enters in a step. The usual form is
// `name: {description, value}`; a plain `name: value` is kept as Value with
// Plain set (legacy showed it without the copy button).
type Input struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Value       string `json:"value,omitempty"`
	Plain       bool   `json:"plain,omitempty"`
}

// Parse reads a test case. It fails on YAML that is not a mapping or whose
// fields have the wrong shape; missing fields are left empty, since full
// validation is tests-validator's job.
func Parse(data []byte) (TestCase, error) {
	var root yaml.Node
	if err := yaml.Unmarshal(data, &root); err != nil {
		return TestCase{}, fmt.Errorf("test case: %w", err)
	}
	if root.Kind != yaml.DocumentNode || len(root.Content) == 0 {
		return TestCase{}, errors.New("test case: the file is empty")
	}
	var raw struct {
		Name        string `yaml:"name"`
		Suite       string `yaml:"suite"`
		Description string `yaml:"description"`
		Steps       []struct {
			Action stringList `yaml:"action"`
			Input  yaml.Node  `yaml:"input"`
			Result stringList `yaml:"result"`
		} `yaml:"steps"`
	}
	if root.Content[0].Kind != yaml.MappingNode {
		return TestCase{}, errors.New("test case: the file is not a YAML mapping")
	}
	if err := root.Content[0].Decode(&raw); err != nil {
		return TestCase{}, fmt.Errorf("test case: %w", err)
	}
	tc := TestCase{Name: raw.Name, Suite: raw.Suite, Description: raw.Description}
	for i, s := range raw.Steps {
		inputs, err := parseInputs(&s.Input)
		if err != nil {
			return TestCase{}, fmt.Errorf("test case: step %d: %w", i+1, err)
		}
		tc.Steps = append(tc.Steps, Step{Action: s.Action, Input: inputs, Result: s.Result})
	}
	return tc, nil
}

// stringList is a list of strings that also accepts a single string.
type stringList []string

func (s *stringList) UnmarshalYAML(n *yaml.Node) error {
	if n.Kind == yaml.ScalarNode {
		if n.Tag == "!!null" {
			*s = nil
			return nil
		}
		*s = []string{n.Value}
		return nil
	}
	var list []string
	if err := n.Decode(&list); err != nil {
		return err
	}
	*s = list
	return nil
}

// parseInputs reads a step's input map, keeping the file's order.
func parseInputs(n *yaml.Node) ([]Input, error) {
	switch {
	case n.Kind == 0 || (n.Kind == yaml.ScalarNode && n.Tag == "!!null"):
		return nil, nil
	case n.Kind != yaml.MappingNode:
		return nil, fmt.Errorf("line %d: input must be a mapping", n.Line)
	}
	var out []Input
	for i := 0; i+1 < len(n.Content); i += 2 {
		key, val := n.Content[i], n.Content[i+1]
		in := Input{Name: key.Value}
		switch val.Kind {
		case yaml.MappingNode:
			var v struct {
				Description string `yaml:"description"`
				Value       string `yaml:"value"`
			}
			if err := val.Decode(&v); err != nil {
				return nil, fmt.Errorf("input %q: %w", key.Value, err)
			}
			in.Description, in.Value = v.Description, v.Value
		case yaml.ScalarNode:
			if val.Tag != "!!null" {
				in.Value = val.Value
			}
			in.Plain = true
		default:
			return nil, fmt.Errorf("line %d: input %q must be a value or {description, value}", val.Line, key.Value)
		}
		out = append(out, in)
	}
	return out, nil
}

// Warnings are problems the wizard shows above a test case without
// blocking the run (legacy: validateManualTest). fileName is the test's file
// name, e.g. login.manual.yaml.
func (tc TestCase) Warnings(fileName string) []string {
	var w []string
	if want := strings.TrimSuffix(fileName, FileSuffix); tc.Name != want {
		w = append(w, fmt.Sprintf("The name “%s” does not match the file name “%s”.", tc.Name, fileName))
	}
	return w
}
