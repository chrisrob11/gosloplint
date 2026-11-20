package gosloplint_test

import (
	"slices"
	"strings"
	"testing"

	"golang.org/x/tools/go/analysis"

	"github.com/chrisrob11/gosloplint"
)

func TestAnalyzersAreValidAndDocumented(t *testing.T) {
	all := gosloplint.Analyzers()
	if err := analysis.Validate(all); err != nil {
		t.Fatalf("Validate: %v", err)
	}
	for _, a := range all {
		if a.Doc == "" {
			t.Errorf("%s: no Doc", a.Name)
		}
		if want := "https://github.com/chrisrob11/gosloplint#" + a.Name; a.URL != want {
			t.Errorf("%s: URL = %q, want %q", a.Name, a.URL, want)
		}
		if !strings.HasPrefix(a.Doc, a.Name+":") {
			t.Errorf("%s: Doc should start with %q", a.Name, a.Name+":")
		}
	}
}

func TestChecksNamesEveryAnalyzer(t *testing.T) {
	checks := gosloplint.Checks()
	for _, a := range gosloplint.Analyzers() {
		if _, ok := checks[a.Name]; !ok {
			t.Errorf("Checks has no entry for %s", a.Name)
		}
	}
	if len(checks) != len(gosloplint.Analyzers()) {
		t.Errorf("Checks has %d entries for %d analyzers", len(checks), len(gosloplint.Analyzers()))
	}
	for name, want := range map[string]string{"comment": "density", "fileplace": "testname", "narration": "warnsign", "steer": "returns-interface"} {
		if !slices.Contains(checks[name], want) {
			t.Errorf("Checks()[%s] = %v, want it to hold %s", name, checks[name], want)
		}
	}
	checks["steer"][0] = "changed"
	if gosloplint.Checks()["steer"][0] == "changed" {
		t.Error("Checks shares its slices with the analyzers' packages")
	}
}
