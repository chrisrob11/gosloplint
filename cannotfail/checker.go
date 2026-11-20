package cannotfail

import (
	"go/ast"
	"go/types"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/types/typeutil"

	"github.com/chrisrob11/gosloplint/internal/pkgpattern"
	"github.com/chrisrob11/gosloplint/internal/testapi"
)

// failMethods are the testing methods that fail or end a test. Skip ends it
// too, and a test that may skip is not one that cannot fail.
var failMethods = map[string]bool{
	"Error": true, "Errorf": true, "Fatal": true, "Fatalf": true, "Fail": true, "FailNow": true,
	"Skip": true, "Skipf": true, "SkipNow": true,
}

// exits are the standard library functions that end the process or the
// goroutine, and so fail a test.
var exits = map[string]bool{
	"log.Fatal": true, "log.Fatalf": true, "log.Fatalln": true,
	"log.Panic": true, "log.Panicf": true, "log.Panicln": true,
	"os.Exit": true, "runtime.Goexit": true,
}

// checker runs the checks over one pass.
type checker struct {
	pass *analysis.Pass
	s    Settings
	// decls are the package's function declarations with bodies.
	decls map[*types.Func]*ast.FuncDecl
	// asserts are the package's functions that can fail a test.
	asserts map[*types.Func]bool
	// holds memoises holdsTesting.
	holds map[types.Type]bool
	// parents maps each node of the file being walked to its parent.
	parents map[ast.Node]ast.Node
}

func newChecker(pass *analysis.Pass, s Settings) *checker {
	return &checker{
		pass: pass, s: s,
		decls:   map[*types.Func]*ast.FuncDecl{},
		asserts: map[*types.Func]bool{},
		holds:   map[types.Type]bool{},
		parents: map[ast.Node]ast.Node{},
	}
}

func (c *checker) run() {
	for _, f := range c.pass.Files {
		c.indexParents(f)
		for _, d := range f.Decls {
			if fd, ok := d.(*ast.FuncDecl); ok && fd.Body != nil {
				if fn, ok := c.pass.TypesInfo.Defs[fd.Name].(*types.Func); ok {
					c.decls[fn] = fd
				}
			}
		}
	}
	// A fixed point: each round marks the functions that call one marked in
	// an earlier round.
	for changed := true; changed; {
		changed = false
		for fn, fd := range c.decls {
			if !c.asserts[fn] && c.fails(fd.Body, true) {
				c.asserts[fn], changed = true, true
			}
		}
	}
	for _, f := range c.pass.Files {
		if ast.IsGenerated(f) || !testapi.IsTestFile(c.pass.Fset, f) {
			continue
		}
		constAsserts(c, f)
		noAsserts(c, f)
		literalFields(c, f)
		for _, d := range f.Decls {
			fd, ok := d.(*ast.FuncDecl)
			if !ok || fd.Body == nil {
				continue
			}
			deadErrs(c, fd)
			if c.s.enabled(EmptyLoop) && c.isTest(fd) {
				emptyLoops(c, fd)
			}
		}
	}
}

func (c *checker) indexParents(f *ast.File) {
	var stack []ast.Node
	ast.Inspect(f, func(n ast.Node) bool {
		if n == nil {
			stack = stack[:len(stack)-1]
			return false
		}
		if len(stack) > 0 {
			c.parents[n] = stack[len(stack)-1]
		}
		stack = append(stack, n)
		return true
	})
}

func (c *checker) report(n ast.Node, check, msg string) {
	if !c.s.enabled(check) {
		return
	}
	c.pass.Report(analysis.Diagnostic{Pos: n.Pos(), Category: check, Message: check + ": " + msg})
}

// isTest reports whether fd is func TestXxx(t *testing.T).
func (c *checker) isTest(fd *ast.FuncDecl) bool {
	name := fd.Name.Name
	if fd.Recv != nil || !strings.HasPrefix(name, "Test") {
		return false
	}
	if r, _ := utf8.DecodeRuneInString(name[len("Test"):]); unicode.IsLower(r) {
		return false
	}
	fn, ok := c.pass.TypesInfo.Defs[fd.Name].(*types.Func)
	if !ok || fn.Signature().Params().Len() != 1 {
		return false
	}
	n, ptr := testapi.TestingName(fn.Signature().Params().At(0).Type())
	return ptr && n == "T"
}

// subtest returns the literal of a t.Run(name, func(t *testing.T) {...}).
func (c *checker) subtest(call *ast.CallExpr) (*ast.FuncLit, bool) {
	if len(call.Args) != 2 || !testapi.IsTestingMethod(c.pass.TypesInfo, call, "Run") {
		return nil, false
	}
	lit, ok := call.Args[1].(*ast.FuncLit)
	if !ok || len(lit.Type.Params.List) != 1 {
		return nil, false
	}
	n, ptr := testapi.TestingName(c.pass.TypesInfo.TypeOf(lit.Type.Params.List[0].Type))
	return lit, ptr && n == "T"
}

// fails reports whether n can fail or skip a test. With subtests unset, the
// bodies of t.Run literals are not read.
func (c *checker) fails(n ast.Node, subtests bool) bool {
	return c.walkFails(n, subtests, nil)
}

// failsBesides reports whether n can fail or skip a test outside cut.
func (c *checker) failsBesides(n, cut ast.Node) bool {
	return c.walkFails(n, true, cut)
}

func (c *checker) walkFails(n ast.Node, subtests bool, cut ast.Node) bool {
	found := false
	ast.Inspect(n, func(m ast.Node) bool {
		if found || m == nil || m == cut {
			return false
		}
		switch m := m.(type) {
		case *ast.CallExpr:
			if _, ok := c.subtest(m); ok && !subtests {
				found = c.fails(m.Fun, false) || c.fails(m.Args[0], false)
				return false
			}
			found = c.callFails(m)
		case *ast.Ident, *ast.SelectorExpr:
			found = c.escapes(m.(ast.Expr))
		}
		return !found
	})
	return found
}

// callFails reports whether call itself can fail a test.
func (c *checker) callFails(call *ast.CallExpr) bool {
	info := c.pass.TypesInfo
	switch callee := typeutil.Callee(info, call).(type) {
	case *types.Builtin:
		if callee.Name() == "panic" {
			return true
		}
	case *types.Func:
		if fn, ok := testapi.TestingMethod(info, call); ok {
			if fn.Name() == "Run" && len(call.Args) == 2 {
				_, lit := call.Args[1].(*ast.FuncLit)
				return !lit
			}
			return failMethods[fn.Name()]
		}
		if c.isAssertFunc(callee) {
			return true
		}
		if pkg := callee.Pkg(); pkg != nil && exits[pkg.Path()+"."+callee.Name()] {
			return true
		}
		if mustName(callee.Name()) && !strings.HasSuffix(c.pass.Fset.Position(callee.Pos()).Filename, "_test.go") {
			return true
		}
		if _, local := c.decls[callee.Origin()]; local {
			return c.asserts[callee.Origin()]
		}
	case *types.Var:
		return true // a function value: what it runs is not known here
	case nil:
		if tv, ok := info.Types[call.Fun]; ok && tv.IsType() {
			return false
		}
		_, lit := ast.Unparen(call.Fun).(*ast.FuncLit)
		return !lit
	}
	return c.handsTesting(call)
}

// mustName reports whether name is Must or must followed by an upper-case
// letter: by Go convention such a function panics when it fails, and a
// panic fails the test. Only a test file's own Must helper is judged by its
// body instead, since a test file is where a helper is misnamed most.
func mustName(name string) bool {
	rest, ok := strings.CutPrefix(name, "Must")
	if !ok {
		rest, ok = strings.CutPrefix(name, "must")
	}
	r, _ := utf8.DecodeRuneInString(rest)
	return ok && unicode.IsUpper(r)
}

// handsTesting reports whether call passes a testing value, as an argument
// or as the receiver of a method.
func (c *checker) handsTesting(call *ast.CallExpr) bool {
	for _, a := range call.Args {
		if c.holdsTesting(c.pass.TypesInfo.TypeOf(a)) {
			return true
		}
	}
	if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
		if s, ok := c.pass.TypesInfo.Selections[sel]; ok && s.Kind() == types.MethodVal {
			return c.holdsTesting(c.pass.TypesInfo.TypeOf(sel.X))
		}
	}
	return false
}

// isAssertFunc reports whether fn is testify's or matches Settings.AssertFuncs.
func (c *checker) isAssertFunc(fn *types.Func) bool {
	if fn.Pkg() == nil {
		return false
	}
	path := fn.Pkg().Path()
	if strings.HasPrefix(path, "github.com/stretchr/testify/") {
		return true
	}
	name := fn.Name()
	if recv := fn.Signature().Recv(); recv != nil {
		t := types.Unalias(recv.Type())
		if p, ok := t.(*types.Pointer); ok {
			t = types.Unalias(p.Elem())
		}
		if named, ok := t.(*types.Named); ok {
			name = named.Obj().Name() + "." + name
		}
	}
	for _, e := range c.s.AssertFuncs {
		if pkgpattern.Match(e, path) {
			return true
		}
		slash := strings.LastIndex(e, "/") + 1
		if dot := strings.Index(e[slash:], "."); dot >= 0 && pkgpattern.Match(e[:slash+dot], path) && e[slash+dot+1:] == name {
			return true
		}
	}
	return false
}

// escapes reports whether e is a use that lets a test fail out of sight: a
// testing value stored, returned or sent, a method value of one, or a
// function of the package that can fail used as a value.
func (c *checker) escapes(e ast.Expr) bool {
	info := c.pass.TypesInfo
	parent := c.parents[e]
	call, isCall := parent.(*ast.CallExpr)
	if isCall && call.Fun == e {
		return false
	}
	var obj types.Object
	switch e := e.(type) {
	case *ast.Ident:
		obj = info.Uses[e]
	case *ast.SelectorExpr:
		obj = info.Uses[e.Sel]
		if s, ok := info.Selections[e]; ok && s.Kind() == types.MethodVal {
			return c.holdsTesting(info.TypeOf(e.X))
		}
	}
	switch obj := obj.(type) {
	case *types.Func:
		return c.asserts[obj.Origin()]
	case *types.Var:
		if obj.IsField() && parent != nil {
			if kv, ok := parent.(*ast.KeyValueExpr); ok && kv.Key == e {
				return false
			}
		}
	default:
		return false
	}
	if !c.holdsTesting(info.TypeOf(e)) {
		return false
	}
	switch p := parent.(type) {
	case *ast.CallExpr, *ast.BinaryExpr, *ast.SelectorExpr:
		// An argument is judged with its call, and a selection on its own.
		return false
	case *ast.AssignStmt:
		for i, r := range p.Rhs {
			if r == e && len(p.Lhs) == len(p.Rhs) {
				id, ok := p.Lhs[i].(*ast.Ident)
				return !ok || id.Name != "_"
			}
		}
		return !containsNode(p.Lhs, e)
	}
	return true
}

func containsNode(list []ast.Expr, n ast.Expr) bool {
	for _, x := range list {
		if x == n {
			return true
		}
	}
	return false
}

// holdsTesting reports whether a value of type t is, points to, or has a
// field holding a testing T, B, F or TB.
func (c *checker) holdsTesting(t types.Type) bool {
	if t == nil {
		return false
	}
	if v, ok := c.holds[t]; ok {
		return v
	}
	c.holds[t] = false // a recursive type is decided by its other fields
	var v bool
	if name, _ := testapi.TestingName(t); name == "T" || name == "B" || name == "F" || name == "TB" {
		v = true
	} else {
		u := types.Unalias(t)
		if p, ok := u.(*types.Pointer); ok {
			u = p.Elem()
		}
		if s, ok := u.Underlying().(*types.Struct); ok {
			for f := range s.Fields() {
				if c.holdsTesting(f.Type()) {
					v = true
					break
				}
			}
		}
	}
	c.holds[t] = v
	return v
}
