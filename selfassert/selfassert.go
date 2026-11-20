// Package selfassert defines an Analyzer that reports test assertions which
// compare a value to itself, and so pass while testing nothing.
//
// The shape usually arrives through a rewrite: a change makes one spelling of
// a value illegal, both sides of an existing comparison are rewritten to the
// surviving spelling, and the assertion silently becomes x == x. It stays
// green and keeps its coverage, and the property it protected is gone.
//
// Three shapes are reported, all in _test.go files:
//   - a testify equality assertion whose two compared arguments are the same
//     expression: assert.Equal(t, f(x), f(x));
//   - x == x or x != x with the same expression on both sides (floating-point
//     operands are skipped: x != x is the NaN test);
//   - two locals given the same initialiser, each assigned once, then compared
//     to each other, directly or inside larger expressions that differ only in
//     which of the two they name: a := f(); b := f(); assert.Equal(t, a[k], b[k]).
//     Only when nothing between the two definitions and the comparison could
//     change either value (the rule of internal/sideeffect): a call between
//     them makes the pair a before/after snapshot, which is a real check.
//
// "The same expression" is decided on the printed syntax tree, so formatting
// does not matter. Two different expressions that happen to evaluate to the
// same value are not reported.
//
// The not-equal assertions (NotEqual, NotSame, ...) are deliberately left
// out: assert.NotEqual(t, newID(), newID()) is the strongest way to say a
// function does not return the same thing twice.
package selfassert

import (
	"go/ast"
	"go/token"
	"go/types"
	"slices"
	"strings"

	"golang.org/x/tools/go/analysis"

	"github.com/chrisrob11/gosloplint/internal/sideeffect"
	"github.com/chrisrob11/gosloplint/internal/testapi"
)

// Analyzer reports assertions that compare a value to itself.
var Analyzer = &analysis.Analyzer{
	Name: "selfassert",
	Doc:  "selfassert: report test assertions that compare a value to itself\n\nReports, in _test.go files, a testify equality assertion or an == / != whose two sides are the same expression, and a comparison of two locals built by the same initialiser. Such an assertion cannot fail.",
	URL:  "https://github.com/chrisrob11/gosloplint#selfassert",
	Run:  run,
}

func run(pass *analysis.Pass) (any, error) {
	for _, file := range pass.Files {
		if !testapi.IsTestFile(pass.Fset, file) {
			continue
		}
		ast.Inspect(file, func(n ast.Node) bool {
			switch node := n.(type) {
			case *ast.CallExpr:
				checkAssert(pass, node)
			case *ast.BinaryExpr:
				checkBinary(pass, node)
			}
			return true
		})
		for _, decl := range file.Decls {
			if fn, ok := decl.(*ast.FuncDecl); ok && fn.Body != nil {
				checkAliasedInits(pass, fn)
			}
		}
	}
	return nil, nil
}

func checkAssert(pass *analysis.Pass, call *ast.CallExpr) {
	name, x, y, ok := testapi.EqualityAssert(pass.TypesInfo, call)
	if !ok {
		return
	}
	if left := testapi.Render(pass.Fset, x); left != "" && left == testapi.Render(pass.Fset, y) {
		pass.Reportf(call.Pos(), "%s compares %s to itself", name, left)
	}
}

func checkBinary(pass *analysis.Pass, bin *ast.BinaryExpr) {
	if bin.Op != token.EQL && bin.Op != token.NEQ {
		return
	}
	if isFloat(pass.TypesInfo.TypeOf(bin.X)) {
		return
	}
	if left := testapi.Render(pass.Fset, bin.X); left != "" && left == testapi.Render(pass.Fset, bin.Y) {
		pass.Reportf(bin.Pos(), "%s compares %s to itself", bin.Op, left)
	}
}

// checkAliasedInits reports, inside one function, a comparison of two
// different locals that hold the same initialiser and are assigned once.
//
// Only an initialiser that calls something counts: a := 1; b := 1 is
// uninteresting, and two make(...) containers are usually filled by
// different routes before they are compared.
func checkAliasedInits(pass *analysis.Pass, fn *ast.FuncDecl) {
	inits, counts, defs := collectInits(pass, fn)
	parents := parentsOf(fn.Body)
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		var x, y ast.Expr
		var name string
		switch node := n.(type) {
		case *ast.CallExpr:
			var ok bool
			if name, x, y, ok = testapi.EqualityAssert(pass.TypesInfo, node); !ok {
				return true
			}
		case *ast.BinaryExpr:
			// Only ==: a != b over two names holding the same computation is
			// the determinism idiom, not a defect.
			if node.Op != token.EQL {
				return true
			}
			x, y, name = node.X, node.Y, node.Op.String()
		default:
			return true
		}
		rx, ry := testapi.Render(pass.Fset, x), testapi.Render(pass.Fset, y)
		for _, u := range objectsIn(pass, x) {
			for _, v := range objectsIn(pass, y) {
				if u == v || counts[u] != 1 || counts[v] != 1 || inits[u] == "" || inits[u] != inits[v] {
					continue
				}
				if alphaRename(rx, u.Name()) == alphaRename(ry, v.Name()) && quietBetween(pass, parents, defs[u], defs[v], n, u, v) {
					pass.Reportf(n.Pos(), "%s compares %s and %s, which hold the same initialiser %s",
						name, u.Name(), v.Name(), inits[u])
					return true
				}
			}
		}
		return true
	})
}

// collectInits maps each local assigned exactly once in fn to its rendered
// initialiser and the node assigning it, and counts assignments so a
// reassigned local is discarded.
func collectInits(pass *analysis.Pass, fn *ast.FuncDecl) (map[types.Object]string, map[types.Object]int, map[types.Object]ast.Node) {
	inits := map[types.Object]string{}
	counts := map[types.Object]int{}
	defs := map[types.Object]ast.Node{}
	obj := func(e ast.Expr) types.Object {
		id, ok := e.(*ast.Ident)
		if !ok || id.Name == "_" {
			return nil
		}
		return pass.TypesInfo.ObjectOf(id)
	}
	note := func(at ast.Node, lhs, rhs []ast.Expr) {
		if len(lhs) != 1 || len(rhs) != 1 {
			// One name's value is not a standalone expression here.
			for _, l := range lhs {
				if o := obj(l); o != nil {
					counts[o] += 2
				}
			}
			return
		}
		o := obj(lhs[0])
		if o == nil {
			return
		}
		counts[o]++
		defs[o] = at
		if containsCall(pass, rhs[0]) {
			inits[o] = testapi.Render(pass.Fset, rhs[0])
		}
	}
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		switch s := n.(type) {
		case *ast.AssignStmt:
			note(s, s.Lhs, s.Rhs)
		case *ast.ValueSpec:
			lhs := make([]ast.Expr, 0, len(s.Names))
			for _, id := range s.Names {
				lhs = append(lhs, id)
			}
			note(s, lhs, s.Values)
		case *ast.IncDecStmt:
			if o := obj(s.X); o != nil {
				counts[o] += 2
			}
		case *ast.UnaryExpr:
			// &x lets anything write x.
			if o := obj(s.X); s.Op == token.AND && o != nil {
				counts[o] += 2
			}
		}
		return true
	})
	return inits, counts, defs
}

// parentsOf maps each node under root to its parent.
func parentsOf(root ast.Node) map[ast.Node]ast.Node {
	parents := map[ast.Node]ast.Node{}
	var stack []ast.Node
	ast.Inspect(root, func(n ast.Node) bool {
		if n == nil {
			stack = stack[:len(stack)-1]
			return false
		}
		if len(stack) > 0 {
			parents[n] = stack[len(stack)-1]
		}
		stack = append(stack, n)
		return true
	})
	return parents
}

// listed returns the statement holding n that sits directly in a statement
// list, the node owning that list, and the list.
func listed(parents map[ast.Node]ast.Node, n ast.Node) (ast.Stmt, ast.Node, []ast.Stmt) {
	for p := parents[n]; p != nil; n, p = p, parents[p] {
		var list []ast.Stmt
		switch p := p.(type) {
		case *ast.BlockStmt:
			list = p.List
		case *ast.CaseClause:
			list = p.Body
		case *ast.CommClause:
			list = p.Body
		default:
			continue
		}
		if s, ok := n.(ast.Stmt); ok {
			return s, p, list
		}
	}
	return nil, nil, nil
}

// quietBetween reports whether the definitions du and dv of u and v share a
// statement list, the comparison cmp comes after both in it, and no
// statement between, nor any before cmp in a list nested between, could
// change u or v by sideeffect.Harmless.
func quietBetween(pass *analysis.Pass, parents map[ast.Node]ast.Node, du, dv, cmp ast.Node, u, v types.Object) bool {
	guarded := func(o types.Object) bool { return o == u || o == v }
	su, owner, list := listed(parents, du)
	sv, ownerV, _ := listed(parents, dv)
	if owner == nil || owner != ownerV {
		return false
	}
	iu, iv := slices.Index(list, su), slices.Index(list, sv)
	for n := cmp; ; {
		s, o, l := listed(parents, n)
		if o == nil {
			return false
		}
		between := l[:slices.Index(l, s)]
		if o == owner {
			if len(between) <= max(iu, iv) {
				return false
			}
			between = slices.Concat(l[min(iu, iv)+1:max(iu, iv)], l[max(iu, iv)+1:len(between)])
		}
		for _, b := range between {
			if !sideeffect.Harmless(pass.TypesInfo, b, guarded) {
				return false
			}
		}
		if o == owner {
			return true
		}
		n = o
	}
}

// objectsIn lists the local variables named anywhere in e, excluding the
// field half of a selector.
func objectsIn(pass *analysis.Pass, e ast.Expr) []types.Object {
	seen := map[types.Object]bool{}
	var out []types.Object
	ast.Inspect(e, func(n ast.Node) bool {
		if sel, ok := n.(*ast.SelectorExpr); ok {
			ast.Inspect(sel.X, func(m ast.Node) bool {
				out = appendVar(pass, m, seen, out)
				return true
			})
			return false
		}
		out = appendVar(pass, n, seen, out)
		return true
	})
	return out
}

func appendVar(pass *analysis.Pass, n ast.Node, seen map[types.Object]bool, out []types.Object) []types.Object {
	id, ok := n.(*ast.Ident)
	if !ok {
		return out
	}
	v, ok := pass.TypesInfo.Uses[id].(*types.Var)
	if !ok || v.IsField() || seen[v] {
		return out
	}
	seen[v] = true
	return append(out, v)
}

// alphaRename replaces whole-word occurrences of name in printed Go source
// with a placeholder, so two expressions differing only in which alias they
// name compare equal. The input comes from go/printer, so it holds no
// comments and identifiers are exactly the \w+ tokens.
func alphaRename(rendered, name string) string {
	var b strings.Builder
	for i := 0; i < len(rendered); {
		if !isIdentStart(rendered[i]) {
			b.WriteByte(rendered[i])
			i++
			continue
		}
		j := i
		for j < len(rendered) && isIdentPart(rendered[j]) {
			j++
		}
		if rendered[i:j] == name {
			b.WriteString("\x00alias")
		} else {
			b.WriteString(rendered[i:j])
		}
		i = j
	}
	return b.String()
}

func isIdentStart(c byte) bool {
	return c == '_' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

func isIdentPart(c byte) bool { return isIdentStart(c) || (c >= '0' && c <= '9') }

// containsCall reports whether e calls something other than the make and new
// builtins. Conversions such as string(b) are not calls.
func containsCall(pass *analysis.Pass, e ast.Expr) bool {
	found := false
	ast.Inspect(e, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok || found {
			return !found
		}
		if tv, ok := pass.TypesInfo.Types[call.Fun]; ok && tv.IsType() {
			return true
		}
		if id, ok := call.Fun.(*ast.Ident); ok {
			if b, ok := pass.TypesInfo.Uses[id].(*types.Builtin); ok && (b.Name() == "make" || b.Name() == "new") {
				return false
			}
		}
		found = true
		return false
	})
	return found
}

func isFloat(t types.Type) bool {
	if t == nil {
		return false
	}
	b, ok := t.Underlying().(*types.Basic)
	return ok && b.Info()&(types.IsFloat|types.IsComplex) != 0
}
