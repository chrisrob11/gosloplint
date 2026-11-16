package cannotfail

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/chrisrob11/gosloplint/internal/testapi"
)

// emptyLoops reports, in the Test function fd, each range whose body checks
// its elements when the loop holds every assertion of its test or subtest
// and nothing shows the ranged collection is non-empty: an empty one passes
// having checked nothing.
//
// A collection is shown non-empty by a non-empty composite literal, by any
// read of it outside range statements (len(xs), require.Len, passing it on),
// or by being an element of an enclosing range, which is judged instead.
func emptyLoops(c *checker, fd *ast.FuncDecl) {
	info := c.pass.TypesInfo
	ast.Inspect(fd.Body, func(n ast.Node) bool {
		rng, ok := n.(*ast.RangeStmt)
		if !ok || !collection(info.TypeOf(rng.X)) || !checksElements(c, rng.Body) {
			return true
		}
		root := rootVar(info, rng.X)
		if root == nil || shownNonEmpty(c, fd, root) || c.failsBesides(scope(c, fd, rng), rng) {
			return true
		}
		c.report(rng, EmptyLoop, "every assertion here is inside this range over "+testapi.Render(c.pass.Fset, rng.X)+
			", and nothing shows it is non-empty: if it is empty the test passes having checked nothing. Check its length first")
		return true
	})
}

// collection reports whether t is a slice, map or string, which may be empty.
// An array has its length in its type; a channel, integer or function is
// not a collection a test asserts over.
func collection(t types.Type) bool {
	if t == nil {
		return false
	}
	switch u := t.Underlying().(type) {
	case *types.Slice, *types.Map:
		return true
	case *types.Basic:
		return u.Info()&types.IsString != 0
	}
	return false
}

// checksElements reports whether body checks each element: it holds an
// assertion other than a bare t.Error or t.Fatal. A loop of unconditional
// t.Errorf reports what it ranges, and an empty collection is its pass.
func checksElements(c *checker, body *ast.BlockStmt) bool {
	found := false
	var walk func(n ast.Node, guarded bool)
	walk = func(n ast.Node, guarded bool) {
		ast.Inspect(n, func(m ast.Node) bool {
			if found || m == nil {
				return false
			}
			switch m := m.(type) {
			case *ast.IfStmt, *ast.SwitchStmt, *ast.TypeSwitchStmt, *ast.SelectStmt:
				if !guarded && m != n {
					walk(m, true)
					return false
				}
			case *ast.CallExpr:
				if _, testing := testapi.TestingMethod(c.pass.TypesInfo, m); c.callFails(m) && (guarded || !testing) {
					found = true
				}
			}
			return !found
		})
	}
	walk(body, false)
	return found
}

// rootVar returns the variable e is read from: xs, xs.f, xs[i], *xs.
func rootVar(info *types.Info, e ast.Expr) *types.Var {
	for {
		switch x := e.(type) {
		case *ast.Ident:
			v, _ := info.Uses[x].(*types.Var)
			return v
		case *ast.SelectorExpr:
			if _, ok := info.Selections[x]; !ok {
				return nil
			}
			e = x.X
		case *ast.IndexExpr:
			e = x.X
		case *ast.StarExpr:
			e = x.X
		case *ast.ParenExpr:
			e = x.X
		default:
			return nil
		}
	}
}

// shownNonEmpty reports whether v is a non-empty literal, an element of a
// range, a closure's parameter, or read anywhere in fd other than as a
// range's collection or an assignment's target.
func shownNonEmpty(c *checker, fd *ast.FuncDecl, v *types.Var) bool {
	info := c.pass.TypesInfo
	if v.Pkg() != c.pass.Pkg || v.Parent() == c.pass.Pkg.Scope() && packageLiteral(c, v) {
		return true
	}
	skip := map[*ast.Ident]bool{}
	found := false
	ast.Inspect(fd.Body, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.FuncLit:
			// A closure's parameter is filled by its callers.
			found = found || (n.Type.Params.Pos() <= v.Pos() && v.Pos() < n.Type.Params.End())
		case *ast.RangeStmt:
			for _, kv := range []ast.Expr{n.Key, n.Value} {
				if id, ok := kv.(*ast.Ident); ok && info.ObjectOf(id) == v {
					found = true
				}
			}
			if rootVar(info, n.X) == v {
				ast.Inspect(n.X, func(m ast.Node) bool {
					if id, ok := m.(*ast.Ident); ok {
						skip[id] = true
					}
					return true
				})
			}
		case *ast.AssignStmt:
			for i, l := range n.Lhs {
				id, ok := l.(*ast.Ident)
				if !ok || info.ObjectOf(id) != v {
					continue
				}
				skip[id] = true
				if len(n.Lhs) == len(n.Rhs) && nonEmpty(c, n.Rhs[i]) {
					found = true
				}
			}
		case *ast.ValueSpec:
			for i, id := range n.Names {
				found = found || (info.Defs[id] == v && nonEmpty(c, valueAt(n, i)))
			}
		case *ast.Ident:
			found = found || (!skip[n] && info.Uses[n] == v)
		}
		return !found
	})
	return found
}

// nonEmpty reports whether e is visibly non-empty: a composite literal
// with elements, a non-empty constant string, a conversion or an append of
// one, an append of elements, or a package variable that is one.
func nonEmpty(c *checker, e ast.Expr) bool {
	info := c.pass.TypesInfo
	e = ast.Unparen(e)
	if tv, ok := info.Types[e]; ok && tv.Value != nil {
		return tv.Value.String() != `""`
	}
	switch x := e.(type) {
	case *ast.UnaryExpr:
		return x.Op == token.AND && nonEmpty(c, x.X)
	case *ast.CompositeLit:
		return len(x.Elts) > 0
	case *ast.Ident:
		v, ok := info.Uses[x].(*types.Var)
		return ok && (v.Pkg() != c.pass.Pkg || v.Parent() == c.pass.Pkg.Scope() && packageLiteral(c, v))
	case *ast.CallExpr:
		if tv, ok := info.Types[x.Fun]; ok && tv.IsType() && len(x.Args) == 1 {
			return nonEmpty(c, x.Args[0])
		}
		if id, ok := ast.Unparen(x.Fun).(*ast.Ident); !ok || info.Uses[id] != types.Universe.Lookup("append") || len(x.Args) < 2 {
			return false
		}
		if !x.Ellipsis.IsValid() {
			return true
		}
		return nonEmpty(c, x.Args[0]) || nonEmpty(c, x.Args[1])
	}
	return false
}

// packageLiteral reports whether the package-level v is declared with a
// non-empty composite literal, or assigned anywhere in the package, where
// this function cannot follow it.
func packageLiteral(c *checker, v *types.Var) bool {
	found := false
	for _, f := range c.pass.Files {
		ast.Inspect(f, func(n ast.Node) bool {
			switch n := n.(type) {
			case *ast.ValueSpec:
				for i, name := range n.Names {
					if c.pass.TypesInfo.Defs[name] != v {
						continue
					}
					found = found || nonEmpty(c, valueAt(n, i))
				}
			case *ast.AssignStmt:
				for _, l := range n.Lhs {
					found = found || rootVar(c.pass.TypesInfo, l) == v
				}
			case *ast.UnaryExpr:
				found = found || rootVar(c.pass.TypesInfo, n.X) == v
			}
			return !found
		})
	}
	return found
}

func valueAt(s *ast.ValueSpec, i int) ast.Expr {
	if len(s.Values) != len(s.Names) {
		return nil
	}
	return s.Values[i]
}

// scope returns the body whose assertions count for rng: the innermost
// t.Run literal holding it, else fd's body. A closure is no boundary: the
// check that makes the loop meaningful often follows the callback it runs
// in.
func scope(c *checker, fd *ast.FuncDecl, rng *ast.RangeStmt) ast.Node {
	var body ast.Node = fd.Body
	ast.Inspect(fd.Body, func(n ast.Node) bool {
		if call, ok := n.(*ast.CallExpr); ok {
			if lit, ok := c.subtest(call); ok && lit.Pos() <= rng.Pos() && rng.End() <= lit.End() {
				body = lit.Body
			}
		}
		return true
	})
	return body
}
