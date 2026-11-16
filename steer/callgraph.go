package steer

import (
	"go/ast"
	"go/types"
	"slices"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/types/typeutil"
)

// fnode is one function or method of the package in its call graph.
type fnode struct {
	fn      *types.Func
	decl    *ast.FuncDecl
	callees []*fnode
	// index and low are Tarjan's; onStack marks the open component.
	index, low int
	onStack    bool
}

// callGraph is the static call graph of a package's non-test functions. A
// call through an interface method is an edge to every method of that name
// whose receiver implements the interface.
type callGraph struct {
	nodes []*fnode
	byFn  map[*types.Func]*fnode
}

func newCallGraph(pass *analysis.Pass) *callGraph {
	g := &callGraph{byFn: map[*types.Func]*fnode{}}
	byName := map[string][]*fnode{}
	for _, f := range pass.Files {
		if ast.IsGenerated(f) || strings.HasSuffix(pass.Fset.Position(f.Pos()).Filename, "_test.go") {
			continue
		}
		for _, d := range f.Decls {
			fd, ok := d.(*ast.FuncDecl)
			if !ok || fd.Body == nil {
				continue
			}
			fn, ok := pass.TypesInfo.Defs[fd.Name].(*types.Func)
			if !ok {
				continue
			}
			n := &fnode{fn: fn, decl: fd, index: -1}
			g.nodes = append(g.nodes, n)
			g.byFn[fn] = n
			if fd.Recv != nil {
				byName[fn.Name()] = append(byName[fn.Name()], n)
			}
		}
	}
	for _, n := range g.nodes {
		ast.Inspect(n.decl.Body, func(x ast.Node) bool {
			call, ok := x.(*ast.CallExpr)
			if !ok {
				return true
			}
			fn, ok := typeutil.Callee(pass.TypesInfo, call).(*types.Func)
			if !ok {
				return true
			}
			fn = fn.Origin()
			if m := g.byFn[fn]; m != nil {
				n.callees = append(n.callees, m)
				return true
			}
			recv := fn.Signature().Recv()
			if recv == nil {
				return true
			}
			it, ok := recv.Type().Underlying().(*types.Interface)
			if !ok {
				return true
			}
			for _, m := range byName[fn.Name()] {
				rt := m.fn.Signature().Recv().Type()
				if types.Implements(rt, it) || types.Implements(types.NewPointer(rt), it) {
					n.callees = append(n.callees, m)
				}
			}
			return true
		})
	}
	return g
}

// components are the strongly connected components of the graph, by
// Tarjan's algorithm run with an explicit stack.
func (g *callGraph) components() [][]*fnode {
	type frame struct {
		n    *fnode
		next int
	}
	var out [][]*fnode
	var open []*fnode
	index := 0
	enter := func(n *fnode) {
		n.index, n.low, n.onStack = index, index, true
		index++
		open = append(open, n)
	}
	for _, root := range g.nodes {
		if root.index >= 0 {
			continue
		}
		enter(root)
		work := []frame{{n: root}}
		for len(work) > 0 {
			top := &work[len(work)-1]
			if top.next < len(top.n.callees) {
				m := top.n.callees[top.next]
				top.next++
				switch {
				case m.index < 0:
					enter(m)
					work = append(work, frame{n: m})
				case m.onStack:
					top.n.low = min(top.n.low, m.index)
				}
				continue
			}
			n := top.n
			work = work[:len(work)-1]
			if len(work) > 0 {
				p := work[len(work)-1].n
				p.low = min(p.low, n.low)
			}
			if n.low != n.index {
				continue
			}
			var scc []*fnode
			for {
				m := open[len(open)-1]
				open = open[:len(open)-1]
				m.onStack = false
				scc = append(scc, m)
				if m == n {
					break
				}
			}
			out = append(out, scc)
		}
	}
	return out
}

// unguarded are the recursive components in which no function names a
// depth, each sorted by position.
func (g *callGraph) unguarded() [][]*fnode {
	var out [][]*fnode
	for _, scc := range g.components() {
		recursive := len(scc) > 1 || slices.Contains(scc[0].callees, scc[0])
		if !recursive || slices.ContainsFunc(scc, func(n *fnode) bool { return guarded(n.decl) }) {
			continue
		}
		slices.SortFunc(scc, func(a, b *fnode) int { return int(a.decl.Pos() - b.decl.Pos()) })
		out = append(out, scc)
	}
	slices.SortFunc(out, func(a, b []*fnode) int { return int(a[0].decl.Pos() - b[0].decl.Pos()) })
	return out
}

// guarded reports whether d bounds its depth: a parameter, or any name it
// uses, says depth.
func guarded(d *ast.FuncDecl) bool {
	found := false
	ast.Inspect(d, func(n ast.Node) bool {
		if id, ok := n.(*ast.Ident); ok && strings.Contains(strings.ToLower(id.Name), "depth") {
			found = true
		}
		return !found
	})
	return found
}
