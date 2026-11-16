package cannotfail_test

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/chrisrob11/gosloplint/cannotfail"
)

func TestChecks(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), cannotfail.Analyzer, "a")
}

func TestGeneratedFilesAreSkipped(t *testing.T) {
	if got := diagnostics(t, cannotfail.Analyzer, "gen"); len(got) != 0 {
		t.Errorf("gen: %q, want none", got)
	}
}

func TestEmptyLoopIsOffByDefault(t *testing.T) {
	if got := diagnostics(t, cannotfail.Analyzer, "loops"); len(got) != 0 {
		t.Errorf("default settings over loops: %q, want none", got)
	}
}

func TestEmptyLoop(t *testing.T) {
	a := cannotfail.New(cannotfail.Settings{Enable: []string{cannotfail.EmptyLoop}})
	analysistest.Run(t, analysistest.TestData(), a, "loops")
}

func TestEnableFlag(t *testing.T) {
	setFlag(t, "enable", "empty-loop")
	analysistest.Run(t, analysistest.TestData(), cannotfail.Analyzer, "loops")
}

func TestDisableDropsOnlyThatCheck(t *testing.T) {
	all := diagnostics(t, cannotfail.Analyzer, "a")
	for _, check := range cannotfail.Checks {
		if check == cannotfail.EmptyLoop {
			continue
		}
		got := diagnostics(t, cannotfail.New(cannotfail.Settings{Disable: []string{check}}), "a")
		n := 0
		for _, d := range all {
			if !strings.HasPrefix(d, check+":") {
				n++
			}
		}
		if len(got) != n || n == len(all) {
			t.Errorf("disable %s: %d findings, want %d of %d", check, len(got), n, len(all))
		}
	}
}

func TestDisableFlag(t *testing.T) {
	setFlag(t, "disable", "const-assert, no-assert,empty-subtest,dead-err,literal-field")
	if got := diagnostics(t, cannotfail.Analyzer, "a"); len(got) != 0 {
		t.Errorf("every check disabled: %q", got)
	}
}

func TestAssertFuncs(t *testing.T) {
	const fn, method = "TestAssertFuncs calls no", "TestAssertMethods calls no"
	for _, tc := range []struct {
		funcs []string
		gone  []string
	}{
		{[]string{"mylib/check.That"}, []string{fn}},
		{[]string{"mylib/check.Suite.Holds"}, []string{method}},
		{[]string{"mylib/check"}, []string{fn, method}},
		{[]string{"mylib/..."}, []string{fn, method}},
		{[]string{"mylib/check.Other", "other/check.That"}, nil},
	} {
		got := strings.Join(diagnostics(t, cannotfail.New(cannotfail.Settings{AssertFuncs: tc.funcs}), "a"), "\n")
		for _, want := range []string{fn, method} {
			wantReported := !slices.Contains(tc.gone, want)
			if reported := strings.Contains(got, want); reported != wantReported {
				t.Errorf("assert-funcs %v: %q reported = %v, want %v", tc.funcs, want, reported, wantReported)
			}
		}
	}
}

func TestAssertFuncsFlag(t *testing.T) {
	setFlag(t, "assert-funcs", "mylib/check")
	if got := strings.Join(diagnostics(t, cannotfail.Analyzer, "a"), "\n"); strings.Contains(got, "TestAssertFuncs") {
		t.Errorf("-assert-funcs mylib/check: still reported TestAssertFuncs")
	}
}

func TestBadSettings(t *testing.T) {
	for _, tc := range []struct {
		s    cannotfail.Settings
		want string
	}{
		{cannotfail.Settings{Disable: []string{"no-asserts"}}, `disable[0]: unknown check "no-asserts" (known: const-assert,`},
		{cannotfail.Settings{Enable: []string{"no-assert"}}, `enable[0]: "no-assert" is not a check that is off by default (those are: empty-loop)`},
		{cannotfail.Settings{Enable: []string{"empty-loop"}, Disable: []string{"empty-loop"}}, `enable[0]: "empty-loop" is also disabled`},
		{cannotfail.Settings{AssertFuncs: []string{" "}}, "assert-funcs[0]: empty entry"},
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
			cannotfail.New(tc.s)
		}()
	}
}

func TestEveryMessageNamesItsCheck(t *testing.T) {
	a := cannotfail.New(cannotfail.Settings{Enable: []string{cannotfail.EmptyLoop}})
	seen := map[string]bool{}
	for _, pkg := range []string{"a", "loops"} {
		for _, d := range diagnostics(t, a, pkg) {
			check, _, _ := strings.Cut(d, ":")
			seen[check] = true
		}
	}
	for _, c := range cannotfail.Checks {
		if !seen[c] {
			t.Errorf("no fixture finding starts %q", c+":")
		}
	}
}

func setFlag(t *testing.T, name, v string) {
	t.Helper()
	if err := cannotfail.Analyzer.Flags.Set(name, v); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := cannotfail.Analyzer.Flags.Set(name, ""); err != nil {
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
