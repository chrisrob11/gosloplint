// Package cannotfail defines an Analyzer that reports tests which cannot
// fail: they stay green whatever the code under test does.
//
// Checks, all in _test.go files:
//
//   - const-assert: a testify assertion on literals only (assert.True(t,
//     true), assert.NoError(t, nil)), or an if whose condition is literals
//     only and whose body fails the test.
//   - empty-loop: in a Test function, a range whose body asserts, over a
//     collection nothing else in the test reads, when no assertion runs
//     outside the loop: an empty collection passes having checked nothing.
//     Off unless Settings.Enable names it.
//   - no-assert: a Test function that can neither fail nor skip, directly or
//     through a helper of its package.
//   - empty-subtest: a t.Run literal that can neither fail nor skip.
//   - dead-err: an error variable assigned and never read before it is
//     overwritten or the function ends, or one never assigned and asserted
//     nil.
//   - literal-field: an assertion that a field equals the literal it was
//     set to, with nothing in between that could change it.
//
// "Can fail" is decided on resolved calls: a failing method of package
// testing, any testify call, panic, log.Fatal, os.Exit, a function of
// Settings.AssertFuncs, a function of the package that can fail, and any
// other call handed a testing value. A testing value stored anywhere also
// counts, so a fake holding t is never mistaken for silence. Benchmarks,
// examples and fuzz targets are not read by no-assert, and a test whose
// parameter is _ declares itself a smoke test. A self-comparison (x == x)
// is selfassert's, not this analyzer's. Generated files are skipped.
package cannotfail

import (
	"golang.org/x/tools/go/analysis"

	"github.com/chrisrob11/gosloplint/internal/listflag"
)

// Analyzer reports tests that cannot fail. Its settings come from the
// -disable, -enable and -assert-funcs flags; New builds one with explicit
// settings.
var Analyzer = newAnalyzer(&Settings{}, true)

// New returns a cannotfail Analyzer configured by s rather than by flags. It
// panics if s does not pass Validate.
func New(s Settings) *analysis.Analyzer {
	if err := s.Validate(); err != nil {
		panic("cannotfail.New: " + err.Error())
	}
	return newAnalyzer(&s, false)
}

func newAnalyzer(s *Settings, withFlags bool) *analysis.Analyzer {
	a := &analysis.Analyzer{
		Name: "cannotfail",
		Doc:  "cannotfail: report tests that cannot fail\n\nChecks: const-assert, empty-loop, no-assert, empty-subtest, dead-err, literal-field. empty-loop is off unless -enable names it. -disable turns checks off; -assert-funcs names functions that fail a test without being passed one.",
		URL:  "https://github.com/chrisrob11/gosloplint#cannotfail",
	}
	a.Run = func(pass *analysis.Pass) (any, error) {
		if err := s.Validate(); err != nil {
			return nil, err
		}
		newChecker(pass, *s).run()
		return nil, nil
	}
	if withFlags {
		a.Flags.Var(listflag.List{Items: &s.Disable}, "disable",
			"comma-separated checks not to run: const-assert, empty-loop, no-assert, empty-subtest, dead-err, literal-field")
		a.Flags.Var(listflag.List{Items: &s.Enable}, "enable",
			"comma-separated checks that are off by default to run: empty-loop")
		a.Flags.Var(listflag.List{Items: &s.AssertFuncs}, "assert-funcs",
			`comma-separated functions that fail a test without being passed one: importpath.Func, importpath.Type.Method, or an import path pattern ("..." matches anything)`)
	}
	return a
}
