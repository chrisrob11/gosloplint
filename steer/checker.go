package steer

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"strconv"
	"strings"

	"golang.org/x/tools/go/analysis"

	"github.com/chrisrob11/gosloplint/internal/pkgpattern"
)

const (
	msgAny = "%s returns %s: every caller must type-assert to use it. " +
		"Return the concrete type, or a struct with a kind tag for a choice of types; take any only as a parameter"
	msgInterface = "%s returns %s, %s: accept interfaces, return structs. " +
		"Return the concrete type; for several implementations, return a struct wrapping the choice. " +
		"Callers that need the interface accept it as a parameter%s"
	msgOneConcrete = "%s returns the interface %s, but every return statement returns %s: return %s. " +
		"The interface hides nothing and costs callers a type assertion%s"
	msgDictated = "; the interface %s fixes this signature, so change it there first"
	msgSoleImpl = "%s returns the interface %s, which only %s implements in this package: return %s, " +
		"and declare the interface where a caller accepts it, once there is a second implementation"
	msgRecursion = "%s %s with no depth bound: a stack overflow is fatal and cannot be recovered. " +
		"Thread a depth parameter and return an error past a limit before descending"
)

// kind is what a result type is, for the return checks.
type kind int

const (
	concrete kind = iota
	anyKind
	ifaceKind
	// openKind is an interface returns-interface leaves alone: the
	// standard library's, or a sealed one. returns-one-concrete reads it.
	openKind
)

// checker runs the checks over one pass.
type checker struct {
	pass *analysis.Pass
	s    Settings
	// ifaces are the non-generic named interfaces with methods declared in
	// the package and the packages it imports, transitively; nil until read.
	ifaces []*types.Named
	// asValue are the package's functions and methods used other than by a
	// call, as a callback is: the type it is passed as fixes its signature.
	asValue map[*types.Func]bool
}

func newChecker(pass *analysis.Pass, s Settings) *checker {
	return &checker{pass: pass, s: s}
}

func (c *checker) run() {
	c.asValue = usedAsValue(c.pass)
	for _, f := range c.pass.Files {
		if ast.IsGenerated(f) {
			continue
		}
		var stack []ast.Node
		ast.Inspect(f, func(n ast.Node) bool {
			if n == nil {
				stack = stack[:len(stack)-1]
				return false
			}
			stack = append(stack, n)
			if ft, ok := n.(*ast.FuncType); ok {
				c.results(ft, stack)
			}
			return true
		})
	}
	if c.s.enabled(UnguardedRecursion) && pkgpattern.Any(c.s.RecursionPackages, c.pass.Pkg.Path()) {
		c.recursion()
	}
}

// recursion reports every recursive group of functions naming no depth.
func (c *checker) recursion() {
	for _, scc := range newCallGraph(c.pass).unguarded() {
		names := make([]string, len(scc))
		for i, n := range scc {
			names[i] = funcName(n.decl)
		}
		verb := "recurses"
		if len(names) > 1 {
			verb = "recurse"
		}
		c.report(scc[0].decl.Name, UnguardedRecursion, fmt.Sprintf(msgRecursion, strings.Join(names, ","), verb))
	}
}

func (c *checker) report(n ast.Node, check, msg string) {
	if !c.s.enabled(check) {
		return
	}
	c.pass.Report(analysis.Diagnostic{Pos: n.Pos(), Category: check, Message: check + ": " + msg})
}

// results reports ft's any and interface results and, when ft is a
// declaration's or a literal's, the checks its body allows.
func (c *checker) results(ft *ast.FuncType, stack []ast.Node) {
	if ft.Results == nil {
		return
	}
	var decl *ast.FuncDecl
	var body *ast.BlockStmt
	switch p := stack[len(stack)-2].(type) {
	case *ast.FuncDecl:
		decl, body = p, p.Body
	case *ast.FuncLit:
		body = p.Body
	}
	if body != nil && decl == nil && litDictated(stack) {
		return
	}
	if decl != nil {
		if fn, ok := c.pass.TypesInfo.Defs[decl.Name].(*types.Func); ok && c.asValue[fn] {
			return
		}
	}
	site := label(stack)
	suffix, checked := "", false
	pos, ifacePos, ifaces := 0, -1, 0
	var iface ast.Expr
	for _, field := range ft.Results.List {
		k, container := classify(c.pass.Pkg, c.pass.TypesInfo.TypeOf(field.Type))
		if k != concrete && !checked {
			checked = true
			if decl != nil && decl.Recv != nil {
				by, local := c.dictatedBy(decl)
				if by != nil && !local {
					return
				}
				if by != nil {
					suffix = fmt.Sprintf(msgDictated, by.Name())
				}
			}
		}
		for range max(1, len(field.Names)) {
			typ := types.ExprString(field.Type)
			switch k {
			case concrete:
			case anyKind:
				c.report(field.Type, ReturnsAny, fmt.Sprintf(msgAny, site, typ))
			case ifaceKind:
				holds := "an interface type"
				if container {
					holds = "which holds an interface"
				}
				c.report(field.Type, ReturnsInterface, fmt.Sprintf(msgInterface, site, typ, holds, suffix))
				if !container {
					ifaces, ifacePos, iface = ifaces+1, pos, field.Type
				}
			case openKind:
				if !container {
					ifaces, ifacePos, iface = ifaces+1, pos, field.Type
				}
			}
			pos++
		}
	}
	if ifaces != 1 {
		return
	}
	if decl != nil {
		c.soleImplementation(decl, iface)
	}
	if t, ok := c.concreteReturns(body, ifacePos, pos); ok {
		conc := types.TypeString(t, c.qualifier)
		c.report(iface, OneConcrete, fmt.Sprintf(msgOneConcrete, site, types.ExprString(iface), conc, conc, suffix))
	}
}

// classify reports what t is as a result in package self, and whether a
// container holds that kind: a slice, array, map, channel, pointer, iter.Seq or iter.Seq2.
func classify(self *types.Package, t types.Type) (kind, bool) {
	switch u := types.Unalias(t).(type) {
	case *types.Pointer:
		return element(self, u.Elem())
	case *types.Slice:
		return element(self, u.Elem())
	case *types.Array:
		return element(self, u.Elem())
	case *types.Map:
		return element(self, u.Elem())
	case *types.Chan:
		return element(self, u.Elem())
	case *types.Named:
		if obj := u.Obj(); obj.Pkg() != nil && obj.Pkg().Path() == "iter" && (obj.Name() == "Seq" || obj.Name() == "Seq2") {
			out := concrete
			for a := range u.TypeArgs().Types() {
				k, _ := element(self, a)
				out = max(out, k)
			}
			return out, out != concrete
		}
		if it, ok := u.Underlying().(*types.Interface); ok {
			k := interfaceKind(it)
			if k == ifaceKind && (sealed(it) || standard(u.Obj().Pkg()) && u.Obj().Pkg() != self) {
				k = openKind
			}
			return k, false
		}
	case *types.Interface:
		return interfaceKind(u), false
	}
	return concrete, false
}

// sealed reports whether it has an unexported method, so only its own
// package can implement it: a sum type, not an abstraction.
func sealed(it *types.Interface) bool {
	for m := range it.Methods() {
		if !m.Exported() {
			return true
		}
	}
	return false
}

// standard reports whether p is in the standard library: its path's first
// element has no dot.
func standard(p *types.Package) bool {
	if p == nil {
		return false
	}
	first, _, _ := strings.Cut(p.Path(), "/")
	return !strings.Contains(first, ".")
}

// litDictated reports whether the func literal on top of stack (its type
// last) takes its type from where it is used: an argument, a field or an
// existing variable assigned, an element, a returned value. A literal
// defining a new variable, or called where it stands, has its own type.
func litDictated(stack []ast.Node) bool {
	if len(stack) < 3 {
		return false
	}
	lit := stack[len(stack)-2]
	switch p := stack[len(stack)-3].(type) {
	case *ast.AssignStmt:
		return p.Tok != token.DEFINE && !allBlank(p.Lhs)
	case *ast.ValueSpec:
		return p.Type != nil
	case *ast.CallExpr:
		return p.Fun != lit
	case *ast.ParenExpr:
		return false
	}
	return true
}

func allBlank(es []ast.Expr) bool {
	for _, e := range es {
		if id, ok := e.(*ast.Ident); !ok || id.Name != "_" {
			return false
		}
	}
	return true
}

// usedAsValue is every function and method of the package that some
// expression uses other than as the callee of a call.
func usedAsValue(pass *analysis.Pass) map[*types.Func]bool {
	called := map[*ast.Ident]bool{}
	out := map[*types.Func]bool{}
	for _, f := range pass.Files {
		ast.Inspect(f, func(n ast.Node) bool {
			if call, ok := n.(*ast.CallExpr); ok {
				switch fun := ast.Unparen(call.Fun).(type) {
				case *ast.Ident:
					called[fun] = true
				case *ast.SelectorExpr:
					called[fun.Sel] = true
				case *ast.IndexExpr:
					called[identOf(fun.X)] = true
				case *ast.IndexListExpr:
					called[identOf(fun.X)] = true
				}
			}
			return true
		})
	}
	for id, obj := range pass.TypesInfo.Uses {
		if fn, ok := obj.(*types.Func); ok && !called[id] && fn.Pkg() == pass.Pkg {
			out[fn.Origin()] = true
		}
	}
	return out
}

// identOf is the identifier naming e, x or pkg.x.
func identOf(e ast.Expr) *ast.Ident {
	switch t := ast.Unparen(e).(type) {
	case *ast.Ident:
		return t
	case *ast.SelectorExpr:
		return t.Sel
	}
	return nil
}

func element(self *types.Package, t types.Type) (kind, bool) {
	k, _ := classify(self, t)
	return k, k != concrete
}

var errorType = types.Universe.Lookup("error").Type().Underlying()

// interfaceKind is any for an empty method set, concrete for error's and
// for a constraint, else an interface.
func interfaceKind(it *types.Interface) kind {
	switch {
	case !it.IsMethodSet(), types.Identical(it, errorType):
		return concrete
	case it.NumMethods() == 0:
		return anyKind
	}
	return ifaceKind
}

// dictatedBy is an interface the method d's receiver implements that has a
// method of d's name, so fixes its signature; local reports that the
// interface is this package's. One from another package is preferred.
func (c *checker) dictatedBy(d *ast.FuncDecl) (*types.TypeName, bool) {
	fn, ok := c.pass.TypesInfo.Defs[d.Name].(*types.Func)
	if !ok {
		return nil, false
	}
	recv := fn.Signature().Recv().Type()
	var local *types.TypeName
	for _, named := range c.interfaces() {
		it := named.Underlying().(*types.Interface)
		if !hasMethod(it, fn.Name()) {
			continue
		}
		if !types.Implements(recv, it) && !types.Implements(types.NewPointer(recv), it) {
			continue
		}
		if named.Obj().Pkg() != c.pass.Pkg {
			return named.Obj(), false
		}
		local = named.Obj()
	}
	return local, local != nil
}

func hasMethod(it *types.Interface, name string) bool {
	for m := range it.Methods() {
		if m.Name() == name {
			return true
		}
	}
	return false
}

// interfaces lists the named interfaces the package can see: its own and
// those of every package it imports, transitively.
func (c *checker) interfaces() []*types.Named {
	if c.ifaces != nil {
		return c.ifaces
	}
	c.ifaces = []*types.Named{}
	seen := map[*types.Package]bool{}
	todo := []*types.Package{c.pass.Pkg}
	for len(todo) > 0 {
		p := todo[len(todo)-1]
		todo = todo[:len(todo)-1]
		if seen[p] {
			continue
		}
		seen[p] = true
		todo = append(todo, p.Imports()...)
		scope := p.Scope()
		for _, name := range scope.Names() {
			tn, ok := scope.Lookup(name).(*types.TypeName)
			if !ok || tn.IsAlias() {
				continue
			}
			named, ok := tn.Type().(*types.Named)
			if !ok || named.TypeParams().Len() > 0 {
				continue
			}
			if it, ok := named.Underlying().(*types.Interface); ok && it.NumMethods() > 0 {
				c.ifaces = append(c.ifaces, named)
			}
		}
	}
	return c.ifaces
}

// concreteReturns is the one concrete type every non-nil result at pos of
// body's return statements has. A result of interface type, or a return
// statement of another arity (a bare return, a call returning a tuple),
// makes the answer false.
func (c *checker) concreteReturns(body *ast.BlockStmt, pos, n int) (types.Type, bool) {
	if body == nil {
		return nil, false
	}
	var seen types.Type
	known := true
	ast.Inspect(body, func(node ast.Node) bool {
		switch r := node.(type) {
		case *ast.FuncLit:
			return false
		case *ast.ReturnStmt:
			if len(r.Results) != n {
				known = false
				return false
			}
			tv := c.pass.TypesInfo.Types[r.Results[pos]]
			switch {
			case tv.IsNil():
			case tv.Type == nil || types.IsInterface(tv.Type):
				known = false
			case seen == nil:
				seen = tv.Type
			case !types.Identical(seen, tv.Type):
				known = false
			}
		}
		return known
	})
	return seen, known && seen != nil
}

// soleImplementation reports a New* function whose interface result,
// declared in its own package, has exactly one implementing package-level
// type there, counting the package's non-test files only.
func (c *checker) soleImplementation(d *ast.FuncDecl, iface ast.Expr) {
	if d.Recv != nil || !strings.HasPrefix(d.Name.Name, "New") {
		return
	}
	named, ok := types.Unalias(c.pass.TypesInfo.TypeOf(iface)).(*types.Named)
	if !ok || named.Obj().Pkg() != c.pass.Pkg {
		return
	}
	it, ok := named.Underlying().(*types.Interface)
	if !ok || it.NumMethods() == 0 {
		return
	}
	var impls []types.Type
	scope := c.pass.Pkg.Scope()
	for _, name := range scope.Names() {
		tn, ok := scope.Lookup(name).(*types.TypeName)
		if !ok || tn.IsAlias() || c.inTestFile(tn) {
			continue
		}
		t, ok := tn.Type().(*types.Named)
		if !ok || t.TypeParams().Len() > 0 || types.IsInterface(t) {
			continue
		}
		switch {
		case types.Implements(t, it):
			impls = append(impls, t)
		case types.Implements(types.NewPointer(t), it):
			impls = append(impls, types.NewPointer(t))
		}
	}
	if len(impls) == 1 {
		impl := types.TypeString(impls[0], c.qualifier)
		c.report(iface, SoleImplementation, fmt.Sprintf(msgSoleImpl, d.Name.Name, named.Obj().Name(), impl, impl))
	}
}

// qualifier names another package's type by its package name, as source
// does.
func (c *checker) qualifier(p *types.Package) string {
	if p == c.pass.Pkg {
		return ""
	}
	return p.Name()
}

func (c *checker) inTestFile(obj types.Object) bool {
	return strings.HasSuffix(c.pass.Fset.Position(obj.Pos()).Filename, "_test.go")
}

// label names the innermost declaration on stack: F, T.M, T.field, I.M,
// F.param, F.func for a literal; an unnamed parameter or result by position,
// F.param0 or F.result0.
func label(stack []ast.Node) string {
	var parts []string
	for i, n := range stack {
		switch n := n.(type) {
		case *ast.FuncDecl:
			parts = append(parts, funcName(n))
		case *ast.TypeSpec:
			parts = append(parts, n.Name.Name)
		case *ast.ValueSpec:
			parts = append(parts, specName(n, stack[i+1]))
		case *ast.Field:
			if name := fieldName(n, stack[:i]); name != "" {
				parts = append(parts, name)
			}
		case *ast.FuncLit:
			parts = append(parts, "func")
		}
	}
	return strings.Join(parts, ".")
}

// funcName is F, or T.M for a method.
func funcName(d *ast.FuncDecl) string {
	if d.Recv == nil || len(d.Recv.List) == 0 {
		return d.Name.Name
	}
	e := d.Recv.List[0].Type
	for {
		switch t := e.(type) {
		case *ast.StarExpr:
			e = t.X
			continue
		case *ast.ParenExpr:
			e = t.X
			continue
		case *ast.IndexExpr:
			e = t.X
			continue
		case *ast.IndexListExpr:
			e = t.X
			continue
		case *ast.Ident:
			return t.Name + "." + d.Name.Name
		}
		return d.Name.Name
	}
}

// specName is the name of vs that value initialises, or all its names when
// value is the shared type.
func specName(vs *ast.ValueSpec, value ast.Node) string {
	for j, v := range vs.Values {
		if v == value && j < len(vs.Names) {
			return vs.Names[j].Name
		}
	}
	names := make([]string, len(vs.Names))
	for j, n := range vs.Names {
		names[j] = n.Name
	}
	return strings.Join(names, ",")
}

// fieldName is f's names, or, unnamed in a func's parameters or results,
// its role and position there; "" for an embedded field.
func fieldName(f *ast.Field, outer []ast.Node) string {
	if len(f.Names) > 0 {
		names := make([]string, len(f.Names))
		for j, n := range f.Names {
			names[j] = n.Name
		}
		return strings.Join(names, ",")
	}
	if len(outer) < 2 {
		return ""
	}
	fl, ok1 := outer[len(outer)-1].(*ast.FieldList)
	ft, ok2 := outer[len(outer)-2].(*ast.FuncType)
	if !ok1 || !ok2 {
		return ""
	}
	role := "param"
	if fl == ft.Results {
		role = "result"
	}
	pos := 0
	for _, g := range fl.List {
		if g == f {
			break
		}
		pos += max(1, len(g.Names))
	}
	return role + strconv.Itoa(pos)
}
