package steer_test

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/chrisrob11/gosloplint/steer"
)

func TestReturnChecks(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), steer.Analyzer, "fixture", "dictated", "gen", "api", "example.com/hooks")
}

// TestReturnChecksReadTheModule runs in module mode, where the driver names
// the package's module: a path sharing three elements is not enough.
func TestReturnChecksReadTheModule(t *testing.T) {
	analysistest.Run(t, filepath.Join(analysistest.TestData(), "mod"), steer.Analyzer, "example.com/app/flags")
}

func TestCleanPackagesReportNothing(t *testing.T) {
	for _, pkg := range []string{"clean", "shadow", "example.com/other"} {
		if got := diagnostics(t, steer.Analyzer, pkg); len(got) != 0 {
			t.Errorf("%s: %q, want none", pkg, got)
		}
	}
}

func TestRecursionInARecursionPackage(t *testing.T) {
	a := steer.New(steer.Settings{RecursionPackages: []string{"rec"}})
	analysistest.Run(t, analysistest.TestData(), a, "rec")
}

func TestRecursionPatternsMatchLikeTheGoCommand(t *testing.T) {
	for _, p := range []string{"...", "re...", "rec/..."} {
		a := steer.New(steer.Settings{RecursionPackages: []string{p}})
		if got := diagnostics(t, a, "rec"); len(got) != 6 {
			t.Errorf("pattern %q: %d findings, want 6", p, len(got))
		}
	}
	a := steer.New(steer.Settings{RecursionPackages: []string{"re", "fixture"}})
	if got := diagnostics(t, a, "rec"); len(got) != 0 {
		t.Errorf("patterns naming other packages: %q, want none", got)
	}
}

func TestRecursionIsOffByDefault(t *testing.T) {
	if got := diagnostics(t, steer.Analyzer, "rec"); len(got) != 0 {
		t.Errorf("default settings over rec: %q, want none", got)
	}
}

func TestRecursionPackagesFlag(t *testing.T) {
	setFlag(t, "recursion-packages", "rec")
	analysistest.Run(t, analysistest.TestData(), steer.Analyzer, "rec")
}

func TestDisableDropsOnlyThatCheck(t *testing.T) {
	all := diagnostics(t, steer.Analyzer, "fixture")
	for _, check := range steer.Checks[:4] {
		got := diagnostics(t, steer.New(steer.Settings{Disable: []string{check}}), "fixture")
		n := 0
		for _, d := range all {
			if !strings.HasPrefix(d, check+":") {
				n++
			}
		}
		if len(got) != n || n == len(all) {
			t.Errorf("disable %s: %d findings, want %d of %d", check, len(got), n, len(all))
		}
		for _, d := range got {
			if strings.HasPrefix(d, check+":") {
				t.Errorf("disable %s: still reported %q", check, d)
			}
		}
	}
	a := steer.New(steer.Settings{Disable: []string{steer.UnguardedRecursion}})
	if got := diagnostics(t, a, "rec"); len(got) != 0 {
		t.Errorf("unguarded-recursion disabled: %q", got)
	}
}

func TestDisableFlag(t *testing.T) {
	setFlag(t, "disable", "returns-any, returns-interface,returns-one-concrete,sole-implementation")
	if got := diagnostics(t, steer.Analyzer, "fixture"); len(got) != 0 {
		t.Errorf("every return check disabled: %q", got)
	}
}

func TestBadSettings(t *testing.T) {
	for _, tc := range []struct {
		s    steer.Settings
		want string
	}{
		{steer.Settings{Disable: []string{"returns-anything"}}, `disable[0]: unknown check "returns-anything" (known: returns-any,`},
		{steer.Settings{RecursionPackages: []string{" "}}, "recursion-packages[0]: empty pattern"},
		{steer.Settings{Disable: []string{"unguarded-recursion"}, RecursionPackages: []string{"x"}}, "recursion-packages is set but unguarded-recursion is disabled"},
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
			steer.New(tc.s)
		}()
	}
}

func TestEveryMessageNamesItsCheck(t *testing.T) {
	a := steer.New(steer.Settings{RecursionPackages: []string{"..."}})
	seen := map[string]bool{}
	for _, pkg := range []string{"fixture", "rec"} {
		for _, d := range diagnostics(t, a, pkg) {
			check, _, _ := strings.Cut(d, ":")
			seen[check] = true
		}
	}
	for _, c := range steer.Checks {
		if !seen[c] {
			t.Errorf("no fixture finding starts %q", c+":")
		}
	}
}

func setFlag(t *testing.T, name, v string) {
	t.Helper()
	if err := steer.Analyzer.Flags.Set(name, v); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := steer.Analyzer.Flags.Set(name, ""); err != nil {
			t.Error(err)
		}
	})
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
