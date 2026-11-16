package cannotfail

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/chrisrob11/gosloplint/internal/testapi"
)

// truthAsserts take one boolean, nilAsserts one value whose nil-ness they
// assert, and negatedAsserts two values that must differ.
var (
	truthAsserts   = map[string]bool{"True": true, "Truef": true, "False": true, "Falsef": true}
	nilAsserts     = map[string]bool{"Nil": true, "Nilf": true, "NotNil": true, "NotNilf": true, "NoError": true, "NoErrorf": true, "Error": true, "Errorf": true}
	negatedAsserts = map[string]bool{"NotEqual": true, "NotEqualf": true, "NotEqualValues": true, "NotEqualValuesf": true, "NotSame": true, "NotSamef": true}
)

// constAsserts reports the assertions of f whose outcome its literals
// decide. Only literals count: two named constants that agree are a real
// check that independent declarations match. An equality of two identical
// literals is selfassert's.
func constAsserts(c *checker, f *ast.File) {
	ast.Inspect(f, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.CallExpr:
			constCall(c, n)
		case *ast.IfStmt:
			if !literal(c, n.Cond) || sameSides(c, n.Cond) || !c.fails(n.Body, true) || skips(c, n.Body) {
				return true
			}
			c.report(n.Cond, ConstAssert, "the condition "+testapi.Render(c.pass.Fset, n.Cond)+
				" is literals only, so this check never changes outcome: compare a result of the code under test")
		}
		return true
	})
}

func constCall(c *checker, call *ast.CallExpr) {
	fn, args, ok := testapi.Testify(c.pass.TypesInfo, call)
	if !ok || len(args) == 0 {
		return
	}
	name := fn.Pkg().Name() + "." + fn.Name()
	var subject string
	switch n := fn.Name(); {
	case truthAsserts[n] || nilAsserts[n]:
		if !literal(c, args[0]) || sameSides(c, args[0]) {
			return
		}
		subject = testapi.Render(c.pass.Fset, args[0])
	case (testapi.EqualityAsserts[n] || negatedAsserts[n]) && len(args) >= 2:
		x, y := testapi.Render(c.pass.Fset, args[0]), testapi.Render(c.pass.Fset, args[1])
		if !literal(c, args[0]) || !literal(c, args[1]) || (x == y && testapi.EqualityAsserts[n]) {
			return
		}
		subject = x + " and " + y
	default:
		return
	}
	c.report(call, ConstAssert, name+" is given only literals ("+subject+
		"), so its outcome is fixed before the test runs: assert on a result of the code under test")
}

// literal reports whether e is built from literals alone: basic literals,
// true, false and nil, with operators.
func literal(c *checker, e ast.Expr) bool {
	ok := true
	ast.Inspect(e, func(n ast.Node) bool {
		if !ok {
			return false
		}
		switch n := n.(type) {
		case *ast.BasicLit, *ast.BinaryExpr, *ast.UnaryExpr, *ast.ParenExpr, nil:
		case *ast.Ident:
			obj := c.pass.TypesInfo.Uses[n]
			switch obj.(type) {
			case *types.Const, *types.Nil:
				ok = obj.Parent() == types.Universe && n.Name != "iota"
			default:
				ok = false
			}
		default:
			ok = false
		}
		return ok
	})
	return ok
}

// sameSides reports whether e is x == x or x != x, which selfassert reports.
func sameSides(c *checker, e ast.Expr) bool {
	for {
		p, ok := e.(*ast.ParenExpr)
		if !ok {
			break
		}
		e = p.X
	}
	b, ok := e.(*ast.BinaryExpr)
	return ok && (b.Op == token.EQL || b.Op == token.NEQ) &&
		testapi.Render(c.pass.Fset, b.X) == testapi.Render(c.pass.Fset, b.Y)
}

// skips reports whether body calls t.Skip: an if true { t.Skip(...) } turns
// a test off on purpose.
func skips(c *checker, body ast.Node) bool {
	found := false
	ast.Inspect(body, func(n ast.Node) bool {
		if call, ok := n.(*ast.CallExpr); ok && !found {
			found = testapi.IsTestingMethod(c.pass.TypesInfo, call, "Skip", "Skipf", "SkipNow")
		}
		return !found
	})
	return found
}
