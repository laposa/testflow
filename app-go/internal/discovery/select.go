package discovery

import (
	"errors"
	"fmt"
)

// ErrNoTests is returned by Select for an empty selection.
var ErrNoTests = errors.New("no tests selected")

// Group is one session service to create: the selected tests of one type
// from one repository's service (spec §8.2).
type Group struct {
	Type       string // TypeAutomated or TypeManual
	Repository Repository
	Service    Service // Suites hold only the selected tests of Type
}

// Select turns the selected test keys into session services: grouped by
// type (automated first), then by repository and service, keeping the
// tree's order. Duplicate keys count once. Unknown keys and tests that
// cannot be selected are an error.
func (r *Result) Select(keys []string) ([]Group, error) {
	want := make(map[string]bool, len(keys))
	for _, k := range keys {
		want[k] = true
	}
	if len(want) == 0 {
		return nil, ErrNoTests
	}

	var groups []Group
	found := 0
	for _, typ := range []string{TypeAutomated, TypeManual} {
		for _, repo := range r.Repositories {
			for _, svc := range repo.Services {
				g := Group{Type: typ, Repository: repo, Service: svc}
				g.Repository.Services = nil
				g.Service.Suites = nil
				for _, suite := range svc.Suites {
					picked := suite
					picked.Tests = nil
					for _, t := range suite.Tests {
						if !want[t.Key] || t.Type() != typ {
							continue
						}
						if !svc.Selectable(t) {
							return nil, fmt.Errorf("test %s in %s cannot be selected: service %s has no %s workflow", t.Path, repo.Name, svc.Name, WorkflowFile(svc.Name))
						}
						picked.Tests = append(picked.Tests, t)
						found++
					}
					if len(picked.Tests) > 0 {
						g.Service.Suites = append(g.Service.Suites, picked)
					}
				}
				if len(g.Service.Suites) > 0 {
					groups = append(groups, g)
				}
			}
		}
	}
	if found != len(want) {
		return nil, fmt.Errorf("%d selected test(s) are not part of this test list", len(want)-found)
	}
	return groups, nil
}
