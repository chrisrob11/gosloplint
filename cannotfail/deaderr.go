package cannotfail

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"

	"github.com/chrisrob11/gosloplint/internal/testapi"
)

// errRefs are the references to one local error variable.
type errRefs struct {
	decl   *ast.Ident   // its declaring name
	zero   bool         // declared by var with no value
	writes []*ast.Ident // plain assignments to it, := included
	reads  []*ast.Ident // every other use
	// hidden is set when a closure names it or its address is taken: a
	// write or read may then happen anywhere.
	hidden bool
}

// deadErrs reports, in fd, an error assigned and never read before it is
// overwritten or fd ends, and an error never assigned but asserted nil. A
// variable a closure names or whose address is taken is left alone, and so
// is every variable of a function holding a goto.
func deadErrs(c *checker, fd *ast.FuncDecl) {
	if hasGoto(fd.Body) {
		return
	}
	for v, r := range errVars(c, fd) {
		switch {
		case r.hidden || r.decl == nil:
		case r.zero && len(r.writes) == 0:
			for _, id := range r.reads {
				if what := nilCheck(c, id); what != "" {
					c.report(id, DeadErr, fmt.Sprintf("%s is declared and never assigned, so it is always nil: %s cannot fail", v.Name(), what))
				}
			}
		default:
			for _, w := range r.writes {
				if msg := deadWrite(c, v, w, r); msg != "" {
					c.report(w, DeadErr, msg)
				}
			}
		}
	}
}

// errVars collects the references to each error variable declared in fd's
// body.
func errVars(c *checker, fd *ast.FuncDecl) map[*types.Var]*errRefs {
	info := c.pass.TypesInfo
	errType := types.Universe.Lookup("error").Type()
	out := map[*types.Var]*errRefs{}
	ast.Inspect(fd.Body, func(n ast.Node) bool {
		id, ok := n.(*ast.Ident)
		if !ok || id.Name == "_" {
			return true
		}
		v, ok := info.ObjectOf(id).(*types.Var)
		if !ok || v.IsField() || !types.Identical(v.Type(), errType) || v.Pos() < fd.Body.Pos() || v.Pos() >= fd.Body.End() {
			return true
		}
		r := out[v]
		if r == nil {
			r = &errRefs{}
			out[v] = r
		}
		parent := c.parents[id]
		if _, isField := parent.(*ast.Field); isField {
			r.hidden = true // a closure's parameter or named result
		}
		if info.Defs[id] == v {
			r.decl = id
			if vs, ok := parent.(*ast.ValueSpec); ok && len(vs.Values) == 0 {
				r.zero = true
			} else {
				r.writes = append(r.writes, id)
			}
		} else if as, ok := parent.(*ast.AssignStmt); ok && containsNode(as.Lhs, id) {
			r.writes = append(r.writes, id)
		} else {
			r.reads = append(r.reads, id)
		}
		if u, ok := parent.(*ast.UnaryExpr); ok && u.Op == token.AND {
			r.hidden = true
		}
		return true
	})
	for _, r := range out {
		if r.decl == nil {
			continue
		}
		home := enclosingFunc(c, r.decl)
		for _, id := range append(r.writes, r.reads...) {
			r.hidden = r.hidden || enclosingFunc(c, id) != home
		}
	}
	return out
}

// deadWrite returns why the value w gives v is never read, or "".
func deadWrite(c *checker, v *types.Var, w *ast.Ident, r *errRefs) string {
	stmt := c.parents[w]
	switch s := stmt.(type) {
	case *ast.ValueSpec:
		stmt = c.parents[c.parents[s]] // the GenDecl's DeclStmt
	case *ast.AssignStmt:
	default:
		return "" // a range or a type switch binds it, and reads it within
	}
	list := stmtList(c.parents[stmt])
	i := indexOf(list, stmt)
	if i < 0 {
		return "" // an if or for header: its own statement reads it
	}
	if next := nextWrite(c, list[i+1:], v); next != nil {
		if readIn(r, stmt.End(), next.End()) {
			return ""
		}
		return fmt.Sprintf("the error assigned to %s here is overwritten on line %d before anything checks it",
			v.Name(), c.pass.Fset.Position(next.Pos()).Line)
	}
	if readIn(r, stmt.End(), enclosingFunc(c, stmt).End()) {
		return ""
	}
	// A loop around the write but not around v may read it on its next turn.
	for n := c.parents[stmt]; n != nil && n.Pos() > v.Pos(); n = c.parents[n] {
		switch n.(type) {
		case *ast.ForStmt, *ast.RangeStmt:
			if readIn(r, n.Pos(), n.End()) {
				return ""
			}
		}
	}
	return fmt.Sprintf("the error assigned to %s here is never checked: nothing reads %s after it", v.Name(), v.Name())
}

// nextWrite returns the first statement of list that assigns v, stopping at
// a statement that may jump away first.
func nextWrite(c *checker, list []ast.Stmt, v *types.Var) ast.Stmt {
	for _, s := range list {
		if as, ok := s.(*ast.AssignStmt); ok {
			for _, l := range as.Lhs {
				if id, ok := l.(*ast.Ident); ok && c.pass.TypesInfo.ObjectOf(id) == v {
					return s
				}
			}
		}
		jumps := false
		ast.Inspect(s, func(n ast.Node) bool {
			switch n.(type) {
			case *ast.BranchStmt, *ast.LabeledStmt:
				jumps = true
			case *ast.FuncLit:
				return false
			}
			return !jumps
		})
		if jumps {
			return nil
		}
	}
	return nil
}

func readIn(r *errRefs, from, to token.Pos) bool {
	for _, id := range r.reads {
		if from < id.Pos() && id.Pos() < to {
			return true
		}
	}
	return false
}

// nilCheck names the assertion id is the subject of when it asserts id nil
// or non-nil: a testify nil assertion or an if comparing id to nil that can
// fail the test.
func nilCheck(c *checker, id *ast.Ident) string {
	switch p := c.parents[id].(type) {
	case *ast.CallExpr:
		if fn, args, ok := testapi.Testify(c.pass.TypesInfo, p); ok && len(args) > 0 && args[0] == id && nilAsserts[fn.Name()] {
			return fn.Pkg().Name() + "." + fn.Name() + "(t, " + id.Name + ")"
		}
	case *ast.BinaryExpr:
		other := p.Y
		if p.Y == id {
			other = p.X
		}
		ifs, ok := c.parents[p].(*ast.IfStmt)
		if (p.Op == token.EQL || p.Op == token.NEQ) && ok && ifs.Cond == p && c.pass.TypesInfo.Types[other].IsNil() && c.fails(ifs.Body, true) {
			return "if " + id.Name + " " + p.Op.String() + " nil"
		}
	}
	return ""
}

func stmtList(n ast.Node) []ast.Stmt {
	switch n := n.(type) {
	case *ast.BlockStmt:
		return n.List
	case *ast.CaseClause:
		return n.Body
	case *ast.CommClause:
		return n.Body
	}
	return nil
}

func indexOf(list []ast.Stmt, n ast.Node) int {
	for i, s := range list {
		if s == n {
			return i
		}
	}
	return -1
}

// enclosingFunc returns the innermost function declaration or literal
// holding n.
func enclosingFunc(c *checker, n ast.Node) ast.Node {
	for p := c.parents[n]; p != nil; p = c.parents[p] {
		switch p.(type) {
		case *ast.FuncDecl, *ast.FuncLit:
			return p
		}
	}
	return nil
}

func hasGoto(body *ast.BlockStmt) bool {
	found := false
	ast.Inspect(body, func(n ast.Node) bool {
		if b, ok := n.(*ast.BranchStmt); ok && b.Tok == token.GOTO {
			found = true
		}
		return !found
	})
	return found
}
