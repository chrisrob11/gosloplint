package cannotfail

import (
	"go/ast"
	"go/token"
	"go/types"
	"regexp"
	"slices"
	"strings"

	"golang.org/x/tools/go/types/typeutil"

	"github.com/chrisrob11/gosloplint/internal/camel"
)

// smokeWords in a comment on or in a test say it checks only that the code
// survives: a test for a panic, a race, a hang.
var smokeWords = regexp.MustCompile(`(?i)\b(panic|panics|panicked|panicking|crash|crashes|crashed|race|races|racy|` +
	`concurrent|concurrently|concurrency|hang|hangs|deadlock|deadlocks|infinite loop|loop forever)\b`)

// noAsserts reports the Test functions and t.Run literals of f that can
// neither fail nor skip. A body holding a subtest is judged by its subtests.
// A parameter spelled _ or left unnamed, or a comment or a test or subtest
// name saying the test is for a panic, crash, race or hang, says it only
// checks the code survives. A test that returns at once unless an
// environment variable is set is os/exec's helper process: the test that
// runs it as a subprocess asserts on what it did.
func noAsserts(c *checker, f *ast.File) {
	for _, d := range f.Decls {
		fd, ok := d.(*ast.FuncDecl)
		if !ok || fd.Body == nil || !c.isTest(fd) || !named(fd.Type) || judged(c, fd.Body) || smoke(c, f, fd, fd.Body) ||
			smokeName(fd.Name.Name) || helperProcess(c, fd.Body) {
			continue
		}
		c.report(fd.Name, NoAssert, fd.Name.Name+" calls no t.Error, t.Fatal or assertion, directly or through a helper, so it passes "+
			"whatever the code does: assert on a result, or name the parameter _ to say it only checks for a panic")
	}
	ast.Inspect(f, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		lit, ok := c.subtest(call)
		if !ok || !named(lit.Type) || judged(c, lit.Body) || feedsParent(c, lit) || smoke(c, f, call, lit.Body) {
			return true
		}
		name := "subtest"
		if b, ok := call.Args[0].(*ast.BasicLit); ok {
			if smokeName(b.Value) {
				return true
			}
			name += " " + b.Value
		}
		c.report(call, EmptySubtest, name+" calls no t.Error, t.Fatal or assertion, directly or through a helper, so it passes "+
			"whatever the code does: assert on a result, or drop it")
		return true
	})
}

// judged reports whether body can fail or skip, or holds a subtest that is
// judged on its own.
func judged(c *checker, body *ast.BlockStmt) bool {
	if c.fails(body, false) {
		return true
	}
	found := false
	ast.Inspect(body, func(n ast.Node) bool {
		if call, ok := n.(*ast.CallExpr); ok && !found {
			_, found = c.subtest(call)
		}
		return !found
	})
	return found
}

// feedsParent reports whether lit assigns a variable declared outside it or
// sends on a channel: the test around it then checks what the subtest did.
func feedsParent(c *checker, lit *ast.FuncLit) bool {
	found := false
	outer := func(e ast.Expr) {
		if v := rootVar(c.pass.TypesInfo, e); v != nil && (v.Pos() < lit.Pos() || v.Pos() >= lit.End()) {
			found = true
		}
	}
	ast.Inspect(lit.Body, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.AssignStmt:
			for _, l := range n.Lhs {
				outer(l)
			}
		case *ast.IncDecStmt:
			outer(n.X)
		case *ast.SendStmt:
			found = true
		}
		return !found
	})
	return found
}

// smoke reports whether a comment in body, or ending on the line above at,
// holds a smoke word.
func smoke(c *checker, f *ast.File, at ast.Node, body *ast.BlockStmt) bool {
	above := c.pass.Fset.Position(at.Pos()).Line - 1
	for _, g := range f.Comments {
		in := body.Pos() <= g.Pos() && g.End() <= body.End()
		if (in || c.pass.Fset.Position(g.End()).Line == above) && smokeWords.MatchString(g.Text()) {
			return true
		}
	}
	return false
}

// smokeName reports whether a test's name, read as words, holds a smoke
// word: TestConcurrentWrites, t.Run("no panic", ...).
func smokeName(name string) bool {
	return smokeWords.MatchString(strings.Join(camel.Words(name), " "))
}

// helperProcess reports whether body opens with if os.Getenv(...) == or !=
// something { return }, the entry of a helper process.
func helperProcess(c *checker, body *ast.BlockStmt) bool {
	if len(body.List) == 0 {
		return false
	}
	ifs, ok := body.List[0].(*ast.IfStmt)
	if !ok || ifs.Init != nil || ifs.Else != nil || len(ifs.Body.List) != 1 {
		return false
	}
	ret, isRet := ifs.Body.List[0].(*ast.ReturnStmt)
	bin, isBin := ifs.Cond.(*ast.BinaryExpr)
	if !isRet || len(ret.Results) != 0 || !isBin || bin.Op != token.EQL && bin.Op != token.NEQ {
		return false
	}
	return slices.ContainsFunc([]ast.Expr{bin.X, bin.Y}, func(e ast.Expr) bool {
		call, ok := ast.Unparen(e).(*ast.CallExpr)
		if !ok {
			return false
		}
		fn, ok := typeutil.Callee(c.pass.TypesInfo, call).(*types.Func)
		return ok && fn.Pkg() != nil && fn.Pkg().Path() == "os" && fn.Name() == "Getenv"
	})
}

// named reports whether the function's one parameter has a name other than _.
func named(ft *ast.FuncType) bool {
	ps := ft.Params.List
	return len(ps) == 1 && len(ps[0].Names) == 1 && ps[0].Names[0].Name != "_"
}
