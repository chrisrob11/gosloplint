package steer

import (
	"fmt"
	"go/ast"
	"go/build/constraint"
	"go/parser"
	"go/token"
	"go/types"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/tools/go/types/typeutil"
)

const (
	msgTwin  = "%s only calls %s with its own parameters: export the one function instead of keeping two names for it"
	msgAlias = "%s only calls %s, of the same signature, with its own parameters: call the original; " +
		"a second name for one function is two things to keep in step"
)

// forwarder finds the package's forwarding functions: each a function or
// method whose whole body is one call, passing its parameters through
// unchanged and in order, to a package-level function from a function, or
// to a method of the same receiver, not promoted, from a method, of an
// identical signature as the call instantiates it.
type forwarder struct {
	c *checker
	// asValue are the functions used as values outside test files.
	asValue map[*types.Func]bool
	// decls are the package's function declarations, each in fileOf.
	decls  map[*types.Func]*ast.FuncDecl
	fileOf map[*ast.FuncDecl]*ast.File
	// linknamed are the names a //go:linkname directive of the package
	// names locally.
	linknamed map[string]bool
	// ignored are the functions, F or T.M, declared in the files the build
	// constraints leave out.
	ignored map[string]bool
}

// forwarding is one forwarder found: fn, declared by decl, calls callee.
type forwarding struct {
	decl       *ast.FuncDecl
	fn, callee *types.Func
}

func newForwarder(c *checker) *forwarder {
	fw := &forwarder{c: c, asValue: usedAsValue(c.pass, false), decls: map[*types.Func]*ast.FuncDecl{}, fileOf: map[*ast.FuncDecl]*ast.File{},
		linknamed: map[string]bool{}, ignored: map[string]bool{}}
	for _, f := range c.pass.Files {
		for _, d := range f.Decls {
			if fd, ok := d.(*ast.FuncDecl); ok {
				if fn, ok := c.pass.TypesInfo.Defs[fd.Name].(*types.Func); ok {
					fw.decls[fn] = fd
					fw.fileOf[fd] = f
				}
			}
		}
		for _, g := range f.Comments {
			for _, cm := range g.List {
				if fields := strings.Fields(cm.Text); len(fields) > 1 && fields[0] == "//go:linkname" {
					fw.linknamed[fields[1]] = true
				}
			}
		}
	}
	fset := token.NewFileSet()
	for _, name := range c.pass.IgnoredFiles {
		src, err := c.pass.ReadFile(name)
		if err != nil {
			continue
		}
		// An ignored file may not parse for this build; what parses counts.
		f, _ := parser.ParseFile(fset, name, src, parser.SkipObjectResolution)
		if f == nil {
			continue
		}
		for _, d := range f.Decls {
			if fd, ok := d.(*ast.FuncDecl); ok {
				fw.ignored[funcName(fd)] = true
			}
		}
	}
	return fw
}

// run reports twin-wrapper and alias over the package's non-test,
// non-generated files.
func (fw *forwarder) run() {
	c := fw.c
	for _, f := range c.pass.Files {
		if ast.IsGenerated(f) || strings.HasSuffix(c.pass.Fset.Position(f.Pos()).Filename, "_test.go") {
			continue
		}
		for _, d := range f.Decls {
			fd, ok := d.(*ast.FuncDecl)
			if !ok {
				continue
			}
			g, ok := fw.forwarding(fd)
			if !ok || fw.kept(g) {
				continue
			}
			wrapper := funcName(fd)
			callee := strings.TrimSuffix(wrapper, fd.Name.Name) + g.callee.Name()
			if g.twin() {
				c.report(fd.Name, TwinWrapper, fmt.Sprintf(msgTwin, wrapper, callee))
			} else {
				c.report(fd.Name, Alias, fmt.Sprintf(msgAlias, wrapper, callee))
			}
		}
	}
}

// twin reports whether g's callee is named for it with the first letter
// lowercased, Fail and fail, so g is exported.
func (g forwarding) twin() bool {
	name := g.fn.Name()
	r, size := utf8.DecodeRuneInString(name)
	return g.callee.Name() == string(unicode.ToLower(r))+name[size:]
}

// kept reports whether the second name has a job its source shows: either
// function is used as a value outside test files or named by //go:linkname; either says
// //go:noinline; the wrapper's doc has a Deprecated: paragraph or it is a
// method an interface names; the callee has no body (assembly), is
// recursive, sits in a file with a //go:build line, or is declared again in
// a file the build leaves out. Each makes the wrapper the one stable name.
func (fw *forwarder) kept(g forwarding) bool {
	callee := g.callee.Origin()
	cd := fw.decls[callee]
	switch {
	case fw.asValue[g.fn], fw.asValue[callee],
		fw.linknamed[g.fn.Name()], fw.linknamed[callee.Name()],
		noinline(g.decl), noinline(cd),
		deprecated(g.decl),
		g.decl.Recv != nil && fw.interfaceMethod(g.fn),
		cd.Body == nil,
		fw.recursive(callee, cd),
		constrained(fw.fileOf[cd]),
		fw.ignored[funcName(cd)]:
		return true
	}
	return false
}

// forwarding is fd as a forwarding function, if it is one.
func (fw *forwarder) forwarding(fd *ast.FuncDecl) (forwarding, bool) {
	info := fw.c.pass.TypesInfo
	if fd.Body == nil || len(fd.Body.List) != 1 {
		return forwarding{}, false
	}
	call := soleCall(fd.Body.List[0])
	fn, ok := info.Defs[fd.Name].(*types.Func)
	if call == nil || !ok {
		return forwarding{}, false
	}
	callee, ok := typeutil.Callee(info, call).(*types.Func)
	if !ok || callee.Pkg() != fw.c.pass.Pkg {
		return forwarding{}, false
	}
	sig := fn.Signature()
	switch fun := ast.Unparen(call.Fun).(type) {
	case *ast.Ident:
		if sig.Recv() != nil || callee.Signature().Recv() != nil {
			return forwarding{}, false
		}
	case *ast.SelectorExpr:
		x, ok := ast.Unparen(fun.X).(*ast.Ident)
		if sig.Recv() == nil || !ok || info.Uses[x] != sig.Recv() || len(info.Selections[fun].Index()) != 1 {
			return forwarding{}, false
		}
	default:
		return forwarding{}, false
	}
	// The callee's type as this call instantiates it, in the wrapper's own
	// type parameters, if it has any.
	target, ok := info.TypeOf(call.Fun).(*types.Signature)
	if !ok || !sameSignature(sig, target) || !fw.passesParams(sig, call) {
		return forwarding{}, false
	}
	return forwarding{decl: fd, fn: fn, callee: callee}, true
}

// sameSignature reports whether a and b take and return identical types,
// receivers and type parameter lists aside; passesParams checks that both
// are variadic or neither.
func sameSignature(a, b *types.Signature) bool {
	return types.Identical(a.Params(), b.Params()) && types.Identical(a.Results(), b.Results())
}

// soleCall is the call s is, as an expression statement or the one result
// of a return; nil for anything else.
func soleCall(s ast.Stmt) *ast.CallExpr {
	var e ast.Expr
	switch s := s.(type) {
	case *ast.ExprStmt:
		e = s.X
	case *ast.ReturnStmt:
		if len(s.Results) != 1 {
			return nil
		}
		e = s.Results[0]
	}
	call, _ := ast.Unparen(e).(*ast.CallExpr)
	return call
}

// passesParams reports whether call's arguments are sig's parameters, each
// named as itself, in order, a variadic one spread. The call type-checked
// against an identical signature, so it has no more arguments than sig has
// parameters.
func (fw *forwarder) passesParams(sig *types.Signature, call *ast.CallExpr) bool {
	params := sig.Params()
	if call.Ellipsis.IsValid() != sig.Variadic() {
		return false
	}
	for i, a := range call.Args {
		id, ok := ast.Unparen(a).(*ast.Ident)
		if !ok || fw.c.pass.TypesInfo.Uses[id] != params.At(i) {
			return false
		}
	}
	return true
}

// interfaceMethod reports whether a named interface the package can see
// has a method of fn's name and signature, so fn may exist to implement
// it. A generic interface's method matches on name and arity alone.
func (fw *forwarder) interfaceMethod(fn *types.Func) bool {
	sig := fn.Signature()
	for _, named := range fw.c.interfaces() {
		for m := range named.Underlying().(*types.Interface).Methods() {
			if m.Name() == fn.Name() && types.Identical(m.Signature(), sig) {
				return true
			}
		}
	}
	for _, named := range fw.c.generic {
		for m := range named.Underlying().(*types.Interface).Methods() {
			ms := m.Signature()
			if m.Name() == fn.Name() && ms.Params().Len() == sig.Params().Len() && ms.Results().Len() == sig.Results().Len() {
				return true
			}
		}
	}
	return false
}

// recursive reports whether fn's body d calls fn.
func (fw *forwarder) recursive(fn *types.Func, d *ast.FuncDecl) bool {
	found := false
	ast.Inspect(d.Body, func(n ast.Node) bool {
		if call, ok := n.(*ast.CallExpr); ok {
			if callee, ok := typeutil.Callee(fw.c.pass.TypesInfo, call).(*types.Func); ok && callee.Origin() == fn {
				found = true
			}
		}
		return !found
	})
	return found
}

// noinline reports whether d's doc holds a //go:noinline directive.
func noinline(d *ast.FuncDecl) bool {
	if d.Doc == nil {
		return false
	}
	for _, cm := range d.Doc.List {
		if strings.TrimSpace(cm.Text) == "//go:noinline" {
			return true
		}
	}
	return false
}

// deprecated reports whether d's doc has a paragraph starting Deprecated:,
// the form go doc and gopls recognise.
func deprecated(d *ast.FuncDecl) bool {
	if d.Doc == nil {
		return false
	}
	for para := range strings.SplitSeq(d.Doc.Text(), "\n\n") {
		if strings.HasPrefix(para, "Deprecated: ") {
			return true
		}
	}
	return false
}

// constrained reports whether f has a //go:build line.
func constrained(f *ast.File) bool {
	for _, g := range f.Comments {
		if g.Pos() > f.Package {
			break
		}
		for _, cm := range g.List {
			if constraint.IsGoBuild(cm.Text) {
				return true
			}
		}
	}
	return false
}
