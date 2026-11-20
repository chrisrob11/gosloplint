package narration_test

import (
	"fmt"
	"strings"
	"testing"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/chrisrob11/gosloplint/narration"
)

func TestHistory(t *testing.T) {
	a := narration.New(narration.Settings{Disable: []string{narration.Warnsign}})
	analysistest.Run(t, analysistest.TestData(), a, "history", "tests")
}

func TestWarnsign(t *testing.T) {
	a := narration.New(narration.Settings{Disable: []string{narration.History}})
	analysistest.Run(t, analysistest.TestData(), a, "warnsign")
}

func TestNearMissesHeadersDirectivesAndGeneratedFilesAreClean(t *testing.T) {
	for _, pkg := range []string{"nearmiss", "skipped"} {
		if got := diagnostics(t, narration.Analyzer, pkg); len(got) != 0 {
			t.Errorf("%s: %q, want none", pkg, got)
		}
	}
}

func TestDisable(t *testing.T) {
	a := narration.New(narration.Settings{Disable: []string{narration.History, narration.Warnsign}})
	for _, pkg := range []string{"history", "warnsign"} {
		if got := diagnostics(t, a, pkg); len(got) != 0 {
			t.Errorf("both disabled, %s: %q", pkg, got)
		}
	}
	f := narration.Analyzer.Flags.Lookup("disable")
	if err := f.Value.Set("history"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := f.Value.Set(""); err != nil {
			t.Error(err)
		}
	})
	got := diagnostics(t, narration.Analyzer, "history")
	if len(got) != 0 {
		t.Errorf("-disable=history: %q, want none", got)
	}
	if got := diagnostics(t, narration.Analyzer, "warnsign"); len(got) != 6 {
		t.Errorf("-disable=history over warnsign: %d findings %q, want 6", len(got), got)
	}
}

func TestBadSettings(t *testing.T) {
	const want = `disable[0]: unknown check "histories" (known: history, warnsign)`
	s := narration.Settings{Disable: []string{"histories"}}
	if err := s.Validate(); err == nil || err.Error() != want {
		t.Errorf("Validate = %v, want %q", err, want)
	}
	defer func() {
		if r := recover(); r == nil || !strings.Contains(fmt.Sprint(r), want) {
			t.Errorf("New panic = %v, want %q", r, want)
		}
	}()
	narration.New(s)
}

// diagnostics runs a over the fixture pkg and returns its messages, without
// matching want comments.
func diagnostics(t *testing.T, a *analysis.Analyzer, pkg string) []string {
	t.Helper()
	var rec recorder
	var out []string
	for _, r := range analysistest.Run(&rec, analysistest.TestData(), a, pkg) {
		if r.Err != nil {
			t.Fatalf("%s: %v", pkg, r.Err)
		}
		for _, d := range r.Diagnostics {
			out = append(out, d.Message)
		}
	}
	return out
}

type recorder []string

func (r *recorder) Errorf(format string, args ...any) {
	*r = append(*r, fmt.Sprintf(format, args...))
}
