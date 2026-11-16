package dupl_test

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/chrisrob11/gosloplint/dupl"
)

func TestCopiesAreReportedAtTheLaterOccurrence(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), dupl.Analyzer, "copied", "renamed", "bare")
}

func TestSimilarButDifferentCodeIsClean(t *testing.T) {
	for _, pkg := range []string{"differ", "kinds", "overlap", "switch", "table", "exempt", "generated"} {
		if got := diagnostics(t, dupl.Analyzer, pkg); len(got) != 0 {
			t.Errorf("%s: %q, want none", pkg, got)
		}
	}
}

func TestTestFilesAreReadOnlyWhenTestsIsSet(t *testing.T) {
	if got := diagnostics(t, dupl.Analyzer, "tests"); len(got) != 0 {
		t.Errorf("default: %q, want none", got)
	}
	got := diagnostics(t, dupl.New(dupl.Settings{Tests: true}), "tests")
	if len(got) != 1 || !strings.HasPrefix(got[0], "lines 4-16 duplicate lines 8-20 of a.go (first)") {
		t.Errorf("tests: %q, want a_test.go's second", got)
	}
}

func TestMinimums(t *testing.T) {
	if got := diagnostics(t, dupl.New(dupl.Settings{MinNodes: 1000}), "copied"); len(got) != 0 {
		t.Errorf("min-nodes 1000: %q, want none", got)
	}
	if got := diagnostics(t, dupl.New(dupl.Settings{MinStatements: 10}), "copied"); len(got) != 0 {
		t.Errorf("min-statements 10: %q, want none", got)
	}
	if got := diagnostics(t, dupl.New(dupl.Settings{MinStatements: 9}), "copied"); len(got) != 1 {
		t.Errorf("min-statements 9: %q, want the copy of its 9 statements", got)
	}
	// differ's copies share runs under the default size: the last four
	// statements are alike.
	if got := diagnostics(t, dupl.New(dupl.Settings{MinNodes: 20}), "differ"); len(got) != 1 {
		t.Errorf("min-nodes 20: %q, want differ's shared tail", got)
	}
}

func TestExempt(t *testing.T) {
	for _, p := range []string{"b.go", "cop...", "copied"} {
		if got := diagnostics(t, dupl.New(dupl.Settings{Exempt: []string{p}}), "copied"); len(got) != 0 {
			t.Errorf("exempt %q: %q, want none", p, got)
		}
	}
	if got := diagnostics(t, dupl.New(dupl.Settings{Exempt: []string{"c.go", "other/..."}}), "copied"); len(got) != 1 {
		t.Errorf("exempt c.go, other/...: %q, want the copy", got)
	}
}

func TestFlags(t *testing.T) {
	setFlag(t, "min-nodes", "1000")
	if got := diagnostics(t, dupl.Analyzer, "copied"); len(got) != 0 {
		t.Errorf("-min-nodes=1000: %q", got)
	}
	setFlag(t, "min-nodes", "0")
	setFlag(t, "tests", "true")
	if got := diagnostics(t, dupl.Analyzer, "tests"); len(got) != 1 {
		t.Errorf("-tests: %q, want one", got)
	}
	setFlag(t, "exempt", "*_test.go")
	if got := diagnostics(t, dupl.Analyzer, "tests"); len(got) != 0 {
		t.Errorf("-exempt=*_test.go: %q", got)
	}
}

func TestBadSettings(t *testing.T) {
	for _, tc := range []struct {
		s    dupl.Settings
		want string
	}{
		{dupl.Settings{MinNodes: -1}, "min-nodes: -1 is negative"},
		{dupl.Settings{MinStatements: -2}, "min-statements: -2 is negative"},
		{dupl.Settings{Exempt: []string{" "}}, "exempt[0]: empty pattern"},
		{dupl.Settings{Exempt: []string{"x", "[x"}}, `exempt[1]: "[x": syntax error in pattern`},
	} {
		err := tc.s.Validate()
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("Validate(%+v) = %v, want %q", tc.s, err, tc.want)
		}
		func() {
			defer func() {
				if r := recover(); r == nil || !strings.Contains(fmt.Sprint(r), tc.want) {
					t.Errorf("New(%+v) panic = %v, want %q", tc.s, r, tc.want)
				}
			}()
			dupl.New(tc.s)
		}()
	}
}

func setFlag(t *testing.T, name, v string) {
	t.Helper()
	f := dupl.Analyzer.Flags.Lookup(name)
	if f == nil {
		t.Fatalf("no flag %s", name)
	}
	old := f.Value.String()
	if err := f.Value.Set(v); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := f.Value.Set(old); err != nil {
			t.Error(err)
		}
	})
}

// diagnostics runs a over the fixture pkg and returns its messages, sorted
// and without duplicates (a non-test file is read by the package's pass and
// its test variant's), without matching want comments.
func diagnostics(t *testing.T, a *analysis.Analyzer, pkg string) []string {
	t.Helper()
	var rec recorder
	var out []string
	for _, r := range analysistest.Run(&rec, analysistest.TestData(), a, pkg) {
		if r.Err != nil {
			t.Fatalf("%s: %v", pkg, r.Err)
		}
		for _, d := range r.Diagnostics {
			out = append(out, r.Pass.Fset.Position(d.Pos).String()+"\x00"+d.Message)
		}
	}
	slices.Sort(out)
	out = slices.Compact(out)
	for i, d := range out {
		_, out[i], _ = strings.Cut(d, "\x00")
	}
	return out
}

type recorder []string

func (r *recorder) Errorf(format string, args ...any) {
	*r = append(*r, fmt.Sprintf(format, args...))
}
