// Package emptyskip defines an Analyzer that reports tests whose whole body
// is a skip, and so pass while running nothing.
//
// A test named for a behaviour but holding only t.Skip("...") is worse than
// an absent test: an absent test is a visible gap, while a skipped one
// occupies the name, appears in the count, and reads as coverage.
//
// Not reported:
//   - a skip guarded by a condition, the standard environment gate;
//   - a skip with statements behind it, which is pending work: a written test
//     that passes the day the bug is fixed;
//   - a subtest created inside a conditional, which documents a per-case N/A.
//
// A `//emptyskip:ok <reason>` comment on the line above the skip records a
// stub that is empty on purpose. The reason is required: a bare marker is
// itself reported.
package emptyskip

import (
	"go/ast"
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
	"golang.org/x/tools/go/types/typeutil"
)

// Analyzer reports tests and subtests whose body is only a skip.
var Analyzer = &analysis.Analyzer{
	Name:     "emptyskip",
	Doc:      "emptyskip: report tests whose whole body is t.Skip\n\nA test or unconditional subtest whose only statement is t.Skip, t.Skipf or t.SkipNow reports green and runs nothing. A `//emptyskip:ok <reason>` comment on the line above the skip exempts it.",
	URL:      "https://github.com/chrisrob11/gosloplint#emptyskip",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

const marker = "//emptyskip:ok"

func run(pass *analysis.Pass) (any, error) {
	ins := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	filter := []ast.Node{(*ast.FuncDecl)(nil), (*ast.CallExpr)(nil)}
	ins.WithStack(filter, func(n ast.Node, push bool, stack []ast.Node) bool {
		if !push || !isTestFile(pass, n) {
			return true
		}
		switch node := n.(type) {
		case *ast.FuncDecl:
			if isTestFunc(pass, node) {
				check(pass, stack[0].(*ast.File), node.Body, node.Name.Name)
			}
		case *ast.CallExpr:
			if name, body, ok := subtest(pass, node); ok && !conditionallyCreated(stack) {
				check(pass, stack[0].(*ast.File), body, name)
			}
		}
		return true
	})
	return nil, nil
}

func isTestFile(pass *analysis.Pass, n ast.Node) bool {
	return strings.HasSuffix(pass.Fset.File(n.Pos()).Name(), "_test.go")
}

// isTestFunc reports whether decl is `func TestXxx(t *testing.T)`.
func isTestFunc(pass *analysis.Pass, decl *ast.FuncDecl) bool {
	if decl.Body == nil || decl.Recv != nil || !strings.HasPrefix(decl.Name.Name, "Test") {
		return false
	}
	fn, ok := pass.TypesInfo.Defs[decl.Name].(*types.Func)
	if !ok {
		return false
	}
	params := fn.Signature().Params()
	return params.Len() == 1 && isTestingT(params.At(0).Type())
}

// subtest returns the name and body of a t.Run("name", func(t *testing.T){...})
// call. A computed name is reported as "subtest".
func subtest(pass *analysis.Pass, call *ast.CallExpr) (name string, body *ast.BlockStmt, ok bool) {
	if !isTestingMethod(pass, call, "Run") || len(call.Args) != 2 {
		return "", nil, false
	}
	fn, isFn := call.Args[1].(*ast.FuncLit)
	if !isFn {
		return "", nil, false
	}
	name = "subtest"
	if lit, isLit := call.Args[0].(*ast.BasicLit); isLit {
		name = strings.Trim(lit.Value, "\"`")
	}
	return name, fn.Body, true
}

// check reports body when it holds nothing but a skip. A body of more than
// one statement is pending work, not a stub, and is left alone.
func check(pass *analysis.Pass, file *ast.File, body *ast.BlockStmt, name string) {
	if body == nil || len(body.List) != 1 {
		return
	}
	expr, ok := body.List[0].(*ast.ExprStmt)
	if !ok {
		return
	}
	call, ok := expr.X.(*ast.CallExpr)
	if !ok || !isTestingMethod(pass, call, "Skip", "Skipf", "SkipNow") {
		return
	}
	reason, marked := exemption(pass, file, expr)
	switch {
	case !marked:
		pass.Reportf(expr.Pos(), "%s: test body is only a skip: it reports green and runs nothing", name)
	case reason == "":
		pass.Reportf(expr.Pos(), "%s: %s needs a reason after it: why can this behaviour not be tested?", name, marker)
	}
}

// conditionallyCreated reports whether the nearest enclosing statement that
// is not a block is a conditional. A subtest created inside one exists only
// for some cases, so an empty body there is a documented N/A.
func conditionallyCreated(stack []ast.Node) bool {
	for i := len(stack) - 1; i >= 0; i-- {
		switch stack[i].(type) {
		case *ast.IfStmt, *ast.SwitchStmt, *ast.TypeSwitchStmt, *ast.SelectStmt, *ast.CaseClause, *ast.CommClause:
			return true
		case *ast.FuncDecl, *ast.FuncLit:
			return false
		}
	}
	return false
}

// isTestingMethod reports whether call invokes one of the named methods of a
// type in package testing (T, B, F or TB).
func isTestingMethod(pass *analysis.Pass, call *ast.CallExpr, names ...string) bool {
	fn, ok := typeutil.Callee(pass.TypesInfo, call).(*types.Func)
	if !ok || fn.Pkg() == nil || fn.Pkg().Path() != "testing" || fn.Signature().Recv() == nil {
		return false
	}
	for _, n := range names {
		if fn.Name() == n {
			return true
		}
	}
	return false
}

// isTestingT reports whether t is *testing.T.
func isTestingT(t types.Type) bool {
	ptr, ok := t.(*types.Pointer)
	if !ok {
		return false
	}
	named, ok := ptr.Elem().(*types.Named)
	if !ok {
		return false
	}
	obj := named.Obj()
	return obj.Pkg() != nil && obj.Pkg().Path() == "testing" && obj.Name() == "T"
}

// exemption looks for the marker in a comment ending on the line above stmt,
// returning the reason written after it. marked is true whenever the marker
// is present, so a bare marker can be reported rather than honoured.
func exemption(pass *analysis.Pass, file *ast.File, stmt ast.Stmt) (reason string, marked bool) {
	line := pass.Fset.Position(stmt.Pos()).Line
	for _, group := range file.Comments {
		if pass.Fset.Position(group.End()).Line != line-1 {
			continue
		}
		for _, c := range group.List {
			text := strings.TrimSpace(c.Text)
			if !strings.HasPrefix(text, marker) {
				continue
			}
			rest := strings.TrimSpace(strings.TrimPrefix(text, marker))
			return strings.TrimSpace(strings.TrimLeft(rest, "-—:")), true
		}
	}
	return "", false
}
