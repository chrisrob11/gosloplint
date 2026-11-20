// Package sideeffect decides whether a statement can change a value a test
// already holds: the "nothing between" rule several checks share.
package sideeffect

import (
	"go/ast"
	"go/token"
	"go/types"
)

// Harmless reports whether s cannot change a value a test holds: a simple
// statement with no call but to a builtin or a conversion, no receive, no
// address taken, and no assignment but to a plain variable that guarded
// does not hold. A function literal's body is not read: defining a closure
// runs nothing.
func Harmless(info *types.Info, s ast.Stmt, guarded func(types.Object) bool) bool {
	switch s := s.(type) {
	case *ast.ExprStmt, *ast.DeclStmt, *ast.EmptyStmt:
	case *ast.AssignStmt:
		for _, l := range s.Lhs {
			id, ok := l.(*ast.Ident)
			if !ok || guarded(info.ObjectOf(id)) {
				return false
			}
		}
	default:
		return false
	}
	ok := true
	ast.Inspect(s, func(n ast.Node) bool {
		if !ok {
			return false
		}
		switch n := n.(type) {
		case *ast.CallExpr:
			ok = info.Types[n.Fun].IsType() || pureBuiltin(info, n)
		case *ast.UnaryExpr:
			ok = n.Op != token.AND && n.Op != token.ARROW
		case *ast.FuncLit:
			return false
		}
		return ok
	})
	return ok
}

// pureBuiltin reports whether call is to a builtin that returns a value and
// does nothing else a test could see.
func pureBuiltin(info *types.Info, call *ast.CallExpr) bool {
	id, ok := ast.Unparen(call.Fun).(*ast.Ident)
	if !ok {
		return false
	}
	b, ok := info.Uses[id].(*types.Builtin)
	if !ok {
		return false
	}
	switch b.Name() {
	case "panic", "recover", "print", "println":
		return false
	}
	return true
}
