// Package testhelper defines an Analyzer that reports must* test helpers
// which hide an error from the call site.
//
// A helper that wraps the code under test and calls t.Fatal on its error
// moves the check away from the line that can fail, so the failure names the
// helper rather than the call. Check an error where the call is made: return
// (T, error) from the helper, or write an assertion helper with no result.
//
// Reported: a function in a _test.go file whose name starts with must or
// Must, that takes a *testing.T, *testing.B or testing.TB, returns at least
// one result and no error, and calls Fatal, Fatalf or FailNow on a testing
// value, even inside a nested closure. Setup fixtures such as openDB(t) are
// left alone: they build the world around the code under test.
package testhelper

import (
	"go/ast"
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/types/typeutil"
)

// Analyzer reports must* test helpers that fail the test on a hidden error.
var Analyzer = &analysis.Analyzer{
	Name: "testhelper",
	Doc:  "testhelper: report must* test helpers that hide an error from the call site\n\nReports a must/Must function in a _test.go file that takes a *testing.T, *testing.B or testing.TB, returns values but no error, and calls t.Fatal, t.Fatalf or t.FailNow. Return (T, error) and check it at the call site instead.",
	URL:  "https://github.com/chrisrob11/gosloplint#testhelper",
	Run:  run,
}

func run(pass *analysis.Pass) (any, error) {
	for _, file := range pass.Files {
		if !strings.HasSuffix(pass.Fset.File(file.Pos()).Name(), "_test.go") {
			continue
		}
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil || !isMust(fn.Name.Name) {
				continue
			}
			obj, ok := pass.TypesInfo.Defs[fn.Name].(*types.Func)
			if !ok {
				continue
			}
			sig := obj.Signature()
			if !takesTesting(sig.Params()) || !returnsNonError(sig.Results()) || !callsFatal(pass, fn.Body) {
				continue
			}
			pass.Reportf(fn.Name.Pos(), "%s fails the test on an error it hides: return (T, error) and check it at the call site", fn.Name.Name)
		}
	}
	return nil, nil
}

// isMust reports whether name is mustX or MustX. No Test, Benchmark, Fuzz or
// Example function starts that way.
func isMust(name string) bool {
	return strings.HasPrefix(name, "must") || strings.HasPrefix(name, "Must")
}

// takesTesting reports whether a parameter is *testing.T, *testing.B or
// testing.TB.
func takesTesting(params *types.Tuple) bool {
	for p := range params.Variables() {
		t := types.Unalias(p.Type())
		ptr, isPtr := t.(*types.Pointer)
		if isPtr {
			t = types.Unalias(ptr.Elem())
		}
		named, ok := t.(*types.Named)
		if !ok || named.Obj().Pkg() == nil || named.Obj().Pkg().Path() != "testing" {
			continue
		}
		switch name := named.Obj().Name(); {
		case isPtr && (name == "T" || name == "B"), !isPtr && name == "TB":
			return true
		}
	}
	return false
}

// returnsNonError reports whether there is at least one result and none is
// an error.
func returnsNonError(results *types.Tuple) bool {
	if results.Len() == 0 {
		return false
	}
	errType := types.Universe.Lookup("error").Type()
	for r := range results.Variables() {
		if types.Identical(r.Type(), errType) {
			return false
		}
	}
	return true
}

// callsFatal reports whether body calls Fatal, Fatalf or FailNow on a value
// from package testing.
func callsFatal(pass *analysis.Pass, body *ast.BlockStmt) bool {
	found := false
	ast.Inspect(body, func(n ast.Node) bool {
		if found {
			return false
		}
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		fn, ok := typeutil.Callee(pass.TypesInfo, call).(*types.Func)
		if !ok || fn.Pkg() == nil || fn.Pkg().Path() != "testing" || fn.Signature().Recv() == nil {
			return true
		}
		switch fn.Name() {
		case "Fatal", "Fatalf", "FailNow":
			found = true
		}
		return !found
	})
	return found
}
