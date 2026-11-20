package cannotfail

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"

	"github.com/chrisrob11/gosloplint/internal/sideeffect"
	"github.com/chrisrob11/gosloplint/internal/testapi"
)

// fieldKey is one field of one local variable.
type fieldKey struct {
	root, field *types.Var
}

// fieldFact is the literal a field was last set to.
type fieldFact struct {
	lit  string
	line int
}

// literalFields reports, in each statement list of f, an assertion that a
// field equals the literal it was set to, by an assignment or a composite
// literal, with only statements between that cannot change it: no call
// other than a builtin, no receive, no address taken, no assignment but to
// a plain variable, and no compound statement.
func literalFields(c *checker, f *ast.File) {
	ast.Inspect(f, func(n ast.Node) bool {
		if list := stmtList(n); list != nil {
			literalFieldsIn(c, list)
		}
		return true
	})
}

func literalFieldsIn(c *checker, list []ast.Stmt) {
	facts := map[fieldKey]fieldFact{}
	for _, s := range list {
		if key, lit, ok := fieldAssert(c, s); ok {
			if fact, ok := facts[key]; ok && fact.lit == lit {
				c.report(s, LiteralField, fmt.Sprintf("%s.%s was set to %s on line %d and nothing since could change it, "+
					"so this checks the assignment, not the code under test", key.root.Name(), key.field.Name(), lit, fact.line))
			}
			continue
		}
		if !setFacts(c, s, facts) && !harmless(c, s, facts) {
			clear(facts)
		}
	}
}

// fieldAssert returns the field and literal s asserts equal: a testify
// equality of the two, assert.True(t, x.f == lit), or an if x.f != lit that
// fails the test.
func fieldAssert(c *checker, s ast.Stmt) (fieldKey, string, bool) {
	var x, y ast.Expr
	switch s := s.(type) {
	case *ast.ExprStmt:
		call, ok := s.X.(*ast.CallExpr)
		if !ok {
			return fieldKey{}, "", false
		}
		if _, a, b, ok := testapi.EqualityAssert(c.pass.TypesInfo, call); ok {
			x, y = a, b
		} else if fn, args, ok := testapi.Testify(c.pass.TypesInfo, call); ok && len(args) > 0 {
			bin, ok := ast.Unparen(args[0]).(*ast.BinaryExpr)
			if !ok {
				return fieldKey{}, "", false
			}
			isTrue := fn.Name() == "True" && bin.Op == token.EQL
			isFalse := fn.Name() == "False" && bin.Op == token.NEQ
			if !isTrue && !isFalse {
				return fieldKey{}, "", false
			}
			x, y = bin.X, bin.Y
		}
	case *ast.IfStmt:
		bin, ok := s.Cond.(*ast.BinaryExpr)
		if s.Init != nil || s.Else != nil || !ok || bin.Op != token.NEQ || !c.fails(s.Body, true) {
			return fieldKey{}, "", false
		}
		x, y = bin.X, bin.Y
	}
	if x == nil {
		return fieldKey{}, "", false
	}
	if key, ok := fieldOf(c, x); ok && literal(c, y) {
		return key, testapi.Render(c.pass.Fset, y), true
	}
	if key, ok := fieldOf(c, y); ok && literal(c, x) {
		return key, testapi.Render(c.pass.Fset, x), true
	}
	return fieldKey{}, "", false
}

// fieldOf returns the field e selects directly from a local variable.
func fieldOf(c *checker, e ast.Expr) (fieldKey, bool) {
	sel, ok := e.(*ast.SelectorExpr)
	if !ok {
		return fieldKey{}, false
	}
	s, ok := c.pass.TypesInfo.Selections[sel]
	id, isIdent := sel.X.(*ast.Ident)
	if !ok || !isIdent || s.Kind() != types.FieldVal {
		return fieldKey{}, false
	}
	root, ok := c.pass.TypesInfo.Uses[id].(*types.Var)
	if !ok || root.IsField() || root.Parent() == c.pass.Pkg.Scope() {
		return fieldKey{}, false
	}
	return fieldKey{root, s.Obj().(*types.Var)}, true
}

// setFacts records what s sets, when s is x.f = lit or x := T{f: lit}, and
// reports whether it was.
func setFacts(c *checker, s ast.Stmt, facts map[fieldKey]fieldFact) bool {
	as, ok := s.(*ast.AssignStmt)
	if !ok || len(as.Lhs) != 1 || len(as.Rhs) != 1 {
		return false
	}
	line := c.pass.Fset.Position(s.Pos()).Line
	if key, ok := fieldOf(c, as.Lhs[0]); ok && as.Tok == token.ASSIGN && literal(c, as.Rhs[0]) {
		facts[key] = fieldFact{testapi.Render(c.pass.Fset, as.Rhs[0]), line}
		return true
	}
	id, isIdent := as.Lhs[0].(*ast.Ident)
	rhs := ast.Unparen(as.Rhs[0])
	if u, ok := rhs.(*ast.UnaryExpr); ok && u.Op == token.AND {
		rhs = u.X
	}
	cl, isLit := rhs.(*ast.CompositeLit)
	root, isVar := c.pass.TypesInfo.ObjectOf(id).(*types.Var)
	if !isIdent || !isLit || !isVar || !harmless(c, &ast.ExprStmt{X: cl}, nil) {
		return false
	}
	for k := range facts {
		if k.root == root {
			delete(facts, k)
		}
	}
	for _, e := range cl.Elts {
		kv, ok := e.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		key, isKey := kv.Key.(*ast.Ident)
		field, isField := c.pass.TypesInfo.Uses[key].(*types.Var)
		if isKey && isField && field.IsField() && literal(c, kv.Value) {
			facts[fieldKey{root, field}] = fieldFact{testapi.Render(c.pass.Fset, kv.Value), line}
		}
	}
	return true
}

// harmless reports whether s cannot change a field of facts, by the rule
// sideeffect.Harmless holds.
func harmless(c *checker, s ast.Stmt, facts map[fieldKey]fieldFact) bool {
	return sideeffect.Harmless(c.pass.TypesInfo, s, func(o types.Object) bool {
		for k := range facts {
			if o == k.root {
				return true
			}
		}
		return false
	})
}
