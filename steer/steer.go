// Package steer defines an Analyzer that reports the design shapes generated
// code keeps reaching for, each message saying what to write instead:
//
//   - returns-any: a result of type any or interface{}, or a container of it
//     (slice, array, map value, channel, pointer, iter.Seq or iter.Seq2).
//   - returns-interface: a result of any other interface type but error, or
//     a container of one. A slice, array or map of another module's
//     interface is left alone when the body fills it with two or more
//     concrete types, or a call is passed straight to another module's
//     field or parameter of that type: that API dictates it.
//   - returns-one-concrete: an interface result where every return statement
//     returns the same concrete type, so the interface hides nothing.
//   - sole-implementation: a New* function returning an interface its own
//     package declares and only one of its package-level types implements.
//   - unguarded-recursion: a self- or mutually recursive group of functions
//     in which no function names a depth. Off unless RecursionPackages (the
//     -recursion-packages flag) names the package.
//
// A result is that of any func type in the package: functions, methods,
// closures, func-typed fields, parameters and named func types, interface
// methods. A method whose signature is fixed by an interface declared in
// another package, which its receiver implements and which this package can
// see through its imports, is not reported: the fix is not in this package.
//
// Every check reads one package, through its type information. An
// implementation of an interface declared in another package, or in a
// package importing this one, is not seen; sole-implementation counts the
// package's own non-test types only. Generated files are skipped.
package steer

import (
	"golang.org/x/tools/go/analysis"

	"github.com/chrisrob11/gosloplint/internal/listflag"
)

// Analyzer reports the steer shapes. Its settings come from the -disable and
// -recursion-packages flags; New builds one with explicit settings instead.
var Analyzer = newAnalyzer(&Settings{}, true)

// New returns a steer Analyzer configured by s rather than by flags. It
// panics if s does not pass Validate.
func New(s Settings) *analysis.Analyzer {
	if err := s.Validate(); err != nil {
		panic("steer.New: " + err.Error())
	}
	return newAnalyzer(&s, false)
}

func newAnalyzer(s *Settings, withFlags bool) *analysis.Analyzer {
	a := &analysis.Analyzer{
		Name: "steer",
		Doc:  "steer: report results typed any or an interface, interfaces with one implementation, and unbounded recursion\n\nChecks: returns-any, returns-interface, returns-one-concrete, sole-implementation, unguarded-recursion (only in -recursion-packages). -disable turns checks off.",
		URL:  "https://github.com/chrisrob11/gosloplint#steer",
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
			"comma-separated checks not to run: returns-any, returns-interface, returns-one-concrete, sole-implementation, unguarded-recursion")
		a.Flags.Var(listflag.List{Items: &s.RecursionPackages}, "recursion-packages",
			`comma-separated import path patterns whose recursion unguarded-recursion reads ("..." matches anything)`)
	}
	return a
}
