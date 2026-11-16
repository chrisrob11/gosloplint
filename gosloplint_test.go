package gosloplint_test

import (
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
