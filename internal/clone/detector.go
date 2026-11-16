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
// minimum node and statement counts; a pair of such runs is reported once
// per pair of enclosing declarations, each side widened to cover every run
// the pair shares.
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
	byHash map[uint64][]window
	blocks int
}

// window is the run of statements i..j of one block, the shortest from i
// that reaches the minimums.
type window struct {
	span  Span
	block int
	i, j  int
}

// NewDetector returns a Detector reading positions from fset, with cfg's
// minimums.
func NewDetector(fset *token.FileSet, cfg Config) *Detector {
	return &Detector{fset: fset, cfg: cfg, byHash: map[uint64][]window{}}
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
		eachStmtList(fd.Body, func(list []ast.Stmt) {
			d.windows(name, list)
			d.blocks++
		})
	}
}

// Clones returns every clone among the files added, sorted by the later
// occurrence's position.
func (d *Detector) Clones() []Clone {
	found := map[string]*Clone{}
	for _, ws := range d.byHash {
		for x := range ws {
			for y := x + 1; y < len(ws); y++ {
				a, b := ws[x], ws[y]
				// Two overlapping windows of one block are a statement
				// repeated in place, not a copy.
				if a.block == b.block && a.i <= b.j && b.i <= a.j {
					continue
				}
				if b.span.before(a.span) {
					a, b = b, a
				}
				c := Clone{A: a.span, B: b.span}
				if prev, ok := found[c.key()]; ok {
					prev.A = prev.A.widen(a.span)
					prev.B = prev.B.widen(b.span)
					continue
				}
				found[c.key()] = &c
			}
		}
	}
	out := make([]Clone, 0, len(found))
	for _, c := range found {
		out = append(out, *c)
	}
	slices.SortFunc(out, func(p, q Clone) int {
		return cmp.Or(p.B.compare(q.B), p.A.compare(q.A))
	})
	return out
}

// windows records, for each statement of list, the shortest run starting
// there that reaches both minimums.
func (d *Detector) windows(decl string, list []ast.Stmt) {
	sums := make([]uint64, len(list))
	sizes := make([]int, len(list))
	for k, s := range list {
		sums[k], sizes[k] = shape(s)
	}
	minNodes, minStmts := d.cfg.minNodes(), d.cfg.minStatements()
	for i := range list {
		h, n := fnv.New64a(), 0
		for j := i; j < len(list); j++ {
			h.Write(binary.LittleEndian.AppendUint64(nil, sums[j]))
			if n += sizes[j]; n < minNodes || j-i+1 < minStmts {
				continue
			}
			start := d.fset.Position(list[i].Pos())
			span := Span{File: start.Filename, Decl: decl, Pos: list[i].Pos(), Start: start.Line, End: d.fset.Position(list[j].End()).Line}
			k := h.Sum64()
			d.byHash[k] = append(d.byHash[k], window{span: span, block: d.blocks, i: i, j: j})
			break
		}
	}
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
