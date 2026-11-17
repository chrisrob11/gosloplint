// Package testapi resolves, through type information, the calls and types of
// package testing and of testify's assert and require, for the analyzers that
// read test code.
package testapi

import (
	"bytes"
	"go/ast"
	"go/printer"
	"go/token"
	"go/types"
	"strings"

	"golang.org/x/tools/go/types/typeutil"
)

// TestifyPkgs are testify's assertion packages.
var TestifyPkgs = map[string]bool{
	"github.com/stretchr/testify/assert":  true,
	"github.com/stretchr/testify/require": true,
}

// EqualityAsserts are the testify assertions of two equal values.
var EqualityAsserts = map[string]bool{
	"Equal": true, "Equalf": true,
	"EqualValues": true, "EqualValuesf": true,
	"Same": true, "Samef": true,
	"Exactly": true, "Exactlyf": true,
}

// IsTestFile reports whether n is in a _test.go file.
func IsTestFile(fset *token.FileSet, n ast.Node) bool {
	return strings.HasSuffix(fset.File(n.Pos()).Name(), "_test.go")
}

// Testify returns the callee of a testify assert or require call, as
// assert.Equal(t, x, y) or a.Equal(x, y), and its arguments after t.
func Testify(info *types.Info, call *ast.CallExpr) (fn *types.Func, args []ast.Expr, ok bool) {
	fn, isFn := typeutil.Callee(info, call).(*types.Func)
	if !isFn || fn.Pkg() == nil || !TestifyPkgs[fn.Pkg().Path()] {
		return nil, nil, false
	}
	if fn.Signature().Recv() != nil {
		return fn, call.Args, true
	}
	if len(call.Args) == 0 {
		return nil, nil, false
	}
	return fn, call.Args[1:], true
}

// EqualityAssert returns the name (pkg.Func) and the two compared arguments
// of a testify equality assertion.
func EqualityAssert(info *types.Info, call *ast.CallExpr) (name string, x, y ast.Expr, ok bool) {
	fn, args, ok := Testify(info, call)
	if !ok || !EqualityAsserts[fn.Name()] || len(args) < 2 {
		return "", nil, nil, false
	}
	return fn.Pkg().Name() + "." + fn.Name(), args[0], args[1], true
}

// TestingMethod returns the method of a package testing type call invokes.
func TestingMethod(info *types.Info, call *ast.CallExpr) (*types.Func, bool) {
	fn, ok := typeutil.Callee(info, call).(*types.Func)
	if !ok || fn.Pkg() == nil || fn.Pkg().Path() != "testing" || fn.Signature().Recv() == nil {
		return nil, false
	}
	return fn, true
}

// IsTestingMethod reports whether call invokes one of the named methods of a
// type in package testing.
func IsTestingMethod(info *types.Info, call *ast.CallExpr, names ...string) bool {
	fn, ok := TestingMethod(info, call)
	if !ok {
		return false
	}
	for _, n := range names {
		if fn.Name() == n {
			return true
		}
	}
	return false
}

// TestingName returns the name of the testing type t is or points to:
// ("T", true) for *testing.T, ("TB", false) for testing.TB, else "".
func TestingName(t types.Type) (name string, ptr bool) {
	t = types.Unalias(t)
	if p, isPtr := t.(*types.Pointer); isPtr {
		t, ptr = types.Unalias(p.Elem()), true
	}
	named, ok := t.(*types.Named)
	if !ok || named.Obj().Pkg() == nil || named.Obj().Pkg().Path() != "testing" {
		return "", false
	}
	return named.Obj().Name(), ptr
}

// Render prints e in normalised form, which is what makes "the same
// expression" decidable. A blank result means it could not be printed, and
// callers never treat two blanks as equal.
func Render(fset *token.FileSet, e ast.Expr) string {
	var buf bytes.Buffer
	if err := printer.Fprint(&buf, fset, e); err != nil {
		return ""
	}
	return strings.Join(strings.Fields(buf.String()), " ")
}
