// Package clone finds copied code: runs of statements with the same syntax
// tree shape, names and literal values aside, long enough that one function
// both places call was the better choice. The dupl analyzer runs it over one
// package; `gosloplint dupl` runs it over many, which no analysis pass can.
//
// The shape of a statement is the sequence of its node kinds, with each
// operator, assignment and branch token and each literal kind kept, so
// `a := x + 1` and `b := y + 2` match and `a := x - 1` does not. A composite
// literal is one node, whatever it holds, and so is a switch's or select's
// block of clauses: a clause's body is compared on its own. A clone is a run
// of consecutive statements of one block whose shapes, summed, reach the
// minimum node and statement counts (a run of bare calls is compared by
// its literal values and called names too); a pair of such runs is
// reported once per pair of enclosing declarations, each side widened to
// cover every run the pair shares without the two sides overlapping.
package clone

import (
	"cmp"
	"encoding/binary"
	"fmt"
	"go/ast"
	"go/token"
	"hash/fnv"
	"slices"
	"strings"
)

// Directive is the line that, followed by a reason, in a function's doc
// comment leaves the function out. Without a reason it does nothing.
const Directive = "//gosloplint:dupl-ok"

// Detector collects the statement runs of the files added to it.
type Detector struct {
	fset   *token.FileSet
	cfg    Config
	byHash map[uint64][]Span
}

// NewDetector returns a Detector reading positions from fset, with cfg's
// minimums.
func NewDetector(fset *token.FileSet, cfg Config) *Detector {
	return &Detector{fset: fset, cfg: cfg, byHash: map[uint64][]Span{}}
}

// Add reads the function bodies of f. A generated file is skipped; which
// files to add is otherwise the caller's choice (Config.Includes).
func (d *Detector) Add(f *ast.File) {
	if ast.IsGenerated(f) {
		return
	}
	for _, decl := range f.Decls {
		fd, ok := decl.(*ast.FuncDecl)
		if !ok || fd.Body == nil || exempt(fd) {
			continue
		}
		name := declName(fd)
		eachStmtList(fd.Body, func(list []ast.Stmt) { d.windows(name, list) })
	}
}

// Clones returns every clone among the files added, sorted by the later
// occurrence's position.
//
// The pairs of one pair of declarations fold into one clone, each side
// widened, in order of position. Within one declaration a pair that would
// make the two sides overlap is left out: three copies there are pairs
// that interleave, and their union would name lines as copies of
// themselves.
func (d *Detector) Clones() []Clone {
	var pairs []Clone
	for _, ws := range d.byHash {
		for x := range ws {
			for y := x + 1; y < len(ws); y++ {
				a, b := ws[x], ws[y]
				if b.before(a) {
					a, b = b, a
				}
				// Overlapping runs are a statement repeated in place.
				if !a.overlaps(b) {
					pairs = append(pairs, Clone{A: a, B: b})
				}
			}
		}
	}
	slices.SortFunc(pairs, func(p, q Clone) int {
		return cmp.Or(p.A.compare(q.A), p.B.compare(q.B))
	})
	found := map[string]int{}
	var out []Clone
	for _, c := range pairs {
		k, ok := found[c.key()]
		if !ok {
			found[c.key()] = len(out)
			out = append(out, c)
			continue
		}
		if a, b := out[k].A.widen(c.A), out[k].B.widen(c.B); !a.overlaps(b) {
			out[k].A, out[k].B = a, b
		}
	}
	slices.SortFunc(out, func(p, q Clone) int {
		return cmp.Or(p.B.compare(q.B), p.A.compare(q.A))
	})
	return out
}

// windows records, for each statement of list, the shortest run starting
// there that reaches both minimums. A run whose every statement is a bare
// call or a guarded one is recorded by its literal values and the names it
// calls as well as its shape: two flag blocks match only when they register
// the same flags.
func (d *Detector) windows(decl string, list []ast.Stmt) {
	sums := make([]uint64, len(list))
	sizes := make([]int, len(list))
	// calls[k] counts the plain calls among the first k statements; spelled
	// holds each one's literal values and called names.
	calls := make([]int, len(list)+1)
	spelled := make([]uint64, len(list))
	for k, s := range list {
		sums[k], sizes[k] = shape(s)
		calls[k+1] = calls[k]
		if bareCall(s) || guardedCall(s) {
			calls[k+1]++
			spelled[k] = spelling(s)
		}
	}
	minNodes, minStmts := d.cfg.minNodes(), d.cfg.minStatements()
	for i := range list {
		h, n := fnv.New64a(), 0
		for j := i; j < len(list); j++ {
			h.Write(binary.LittleEndian.AppendUint64(nil, sums[j]))
			if n += sizes[j]; n < minNodes || j-i+1 < minStmts {
				continue
			}
			if calls[j+1]-calls[i] == j-i+1 {
				for _, v := range spelled[i : j+1] {
					h.Write(binary.LittleEndian.AppendUint64(nil, v))
				}
			}
			start := d.fset.Position(list[i].Pos())
			span := Span{File: start.Filename, Decl: decl, Pos: list[i].Pos(), Start: start.Line, End: d.fset.Position(list[j].End()).Line}
			k := h.Sum64()
			d.byHash[k] = append(d.byHash[k], span)
			break
		}
	}
}

// spelling hashes the literal values in s and the name of each function it
// calls; a statement without a literal, `r.Unregister(m.hits)`, is a list
// of names, so it hashes every name in it.
func spelling(s ast.Stmt) uint64 {
	spelled, names := fnv.New64a(), fnv.New64a()
	literal := false
	ast.Inspect(s, func(x ast.Node) bool {
		switch x := x.(type) {
		case *ast.BasicLit:
			literal = true
			spelled.Write([]byte(x.Value + ";"))
		case *ast.Ident:
			names.Write([]byte(x.Name + ";"))
		case *ast.CallExpr:
			switch f := x.Fun.(type) {
			case *ast.Ident:
				spelled.Write([]byte(f.Name + "("))
			case *ast.SelectorExpr:
				spelled.Write([]byte(f.Sel.Name + "("))
			}
		}
		return true
	})
	if literal {
		return spelled.Sum64()
	}
	return names.Sum64()
}

// bareCall reports whether s is a call made for its effect whose arguments
// are each a name, a literal, a field or the address of one: a flag
// registration, a route, a setter. A run of those matches another of the
// same length on node kinds alone: it is data spelled as calls, a copy only
// when it is spelled alike.
func bareCall(s ast.Stmt) bool {
	es, ok := s.(*ast.ExprStmt)
	if !ok {
		return false
	}
	call, ok := es.X.(*ast.CallExpr)
	return ok && plainCall(call)
}

// guardedCall reports whether s is `if err := f(x); err != nil { return err }`
// around a call bareCall accepts, with nothing else in it. A chain of those,
// one per step, matches any other chain of as many steps.
func guardedCall(s ast.Stmt) bool {
	is, ok := s.(*ast.IfStmt)
	if !ok || is.Else != nil || len(is.Body.List) != 1 {
		return false
	}
	init, ok := is.Init.(*ast.AssignStmt)
	if !ok || len(init.Lhs) != 1 || len(init.Rhs) != 1 {
		return false
	}
	call, ok := init.Rhs[0].(*ast.CallExpr)
	if !ok || !plainCall(call) {
		return false
	}
	cond, ok := is.Cond.(*ast.BinaryExpr)
	if !ok || cond.Op != token.NEQ || !isIdent(cond.X, identName(init.Lhs[0])) || !isIdent(cond.Y, "nil") {
		return false
	}
	ret, ok := is.Body.List[0].(*ast.ReturnStmt)
	return ok && !slices.ContainsFunc(ret.Results, func(r ast.Expr) bool {
		c, isCall := r.(*ast.CallExpr)
		return !plainArg(r) && (!isCall || !plainCall(c))
	})
}

func identName(e ast.Expr) string {
	if id, ok := e.(*ast.Ident); ok {
		return id.Name
	}
	return ""
}

func isIdent(e ast.Expr, name string) bool { return name != "" && identName(e) == name }

// plainCall reports whether call's function is a name or a chain of fields
// and plain calls, and its arguments are plain.
func plainCall(call *ast.CallExpr) bool {
	return callee(call.Fun) && !slices.ContainsFunc(call.Args, func(a ast.Expr) bool { return !plainArg(a) })
}

func callee(e ast.Expr) bool {
	switch e := e.(type) {
	case *ast.Ident:
		return true
	case *ast.SelectorExpr:
		return callee(e.X)
	case *ast.CallExpr:
		return plainCall(e)
	}
	return false
}

func plainArg(e ast.Expr) bool {
	switch e := e.(type) {
	case *ast.BasicLit, *ast.Ident, *ast.CompositeLit, *ast.FuncLit:
		return true
	case *ast.SelectorExpr:
		return plainArg(e.X)
	case *ast.UnaryExpr:
		return e.Op == token.AND && plainArg(e.X)
	}
	return false
}

// shape hashes a statement's node kinds, operators and literal kinds, and
// counts its nodes.
func shape(s ast.Stmt) (uint64, int) {
	h, n := fnv.New64a(), 0
	put := func(s string) { h.Write([]byte(s)) }
	tables := clauseBlocks(s)
	ast.Inspect(s, func(x ast.Node) bool {
		if x == nil {
			return false
		}
		n++
		put(fmt.Sprintf("%T", x))
		switch x := x.(type) {
		case *ast.BlockStmt:
			if tables[x] {
				put(";")
				return false
			}
		case *ast.BinaryExpr:
			put(x.Op.String())
		case *ast.UnaryExpr:
			put(x.Op.String())
		case *ast.AssignStmt:
			put(x.Tok.String())
		case *ast.IncDecStmt:
			put(x.Tok.String())
		case *ast.BranchStmt:
			put(x.Tok.String())
		case *ast.BasicLit:
			put(x.Kind.String())
		case *ast.CompositeLit:
			// A literal is data: two tables of one shape are not one
			// decision made twice.
			put(";")
			return false
		}
		put(";")
		return true
	})
	return h.Sum64(), n
}

// eachStmtList calls visit with every statement list under body: blocks,
// including those of function literals, and the bodies of switch and select
// clauses. A switch's own block, its list of clauses, is a dispatch table
// rather than a run of statements, and is left out.
func eachStmtList(body *ast.BlockStmt, visit func([]ast.Stmt)) {
	clauses := clauseBlocks(body)
	ast.Inspect(body, func(x ast.Node) bool {
		switch x := x.(type) {
		case *ast.BlockStmt:
			if !clauses[x] {
				visit(x.List)
			}
		case *ast.CaseClause:
			visit(x.Body)
		case *ast.CommClause:
			visit(x.Body)
		}
		return true
	})
}

// clauseBlocks returns the blocks under n that hold a switch's or a
// select's clauses.
func clauseBlocks(n ast.Node) map[*ast.BlockStmt]bool {
	out := map[*ast.BlockStmt]bool{}
	ast.Inspect(n, func(x ast.Node) bool {
		switch x := x.(type) {
		case *ast.SwitchStmt:
			out[x.Body] = true
		case *ast.TypeSwitchStmt:
			out[x.Body] = true
		case *ast.SelectStmt:
			out[x.Body] = true
		}
		return true
	})
	return out
}

// exempt reports whether fd's doc comment carries Directive with a reason.
func exempt(fd *ast.FuncDecl) bool {
	if fd.Doc == nil {
		return false
	}
	for _, c := range fd.Doc.List {
		if why, ok := strings.CutPrefix(c.Text, Directive); ok && strings.TrimSpace(why) != "" {
			return true
		}
	}
	return false
}

// declName is a function's name, or Type.Method for a method.
func declName(fd *ast.FuncDecl) string {
	if fd.Recv == nil || len(fd.Recv.List) == 0 {
		return fd.Name.Name
	}
	t := fd.Recv.List[0].Type
	for {
		switch x := t.(type) {
		case *ast.StarExpr:
			t = x.X
		case *ast.IndexExpr:
			t = x.X
		case *ast.IndexListExpr:
			t = x.X
		case *ast.Ident:
			return x.Name + "." + fd.Name.Name
		default:
			return fd.Name.Name
		}
	}
}
