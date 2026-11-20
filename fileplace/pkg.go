package fileplace

import (
	"go/ast"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"golang.org/x/tools/go/analysis"
)

// testSuffixes are the suffixes a test file's stem may add to the name of
// the source file it tests.
var testSuffixes = []string{"_internal", "_integration", "_external", "_bench", "_fuzz", "_example"}

// testStems are the test file stems Go names by convention rather than for
// a source file: examples (also example_*_test.go), exports for an external
// test, and TestMain.
var testStems = []string{"example", "export", "main"}

// method is one method declaration and the file it is in.
type method struct {
	f    *file
	decl *ast.FuncDecl
}

// pkg is one pass's files, indexed by where each type and method sits.
type pkg struct {
	pass    *analysis.Pass
	s       Settings
	name    string // the package clause, without _test
	dirName string // the directory's base name
	files   []*file
	decl    map[string][]*file
	spec    map[string]*ast.TypeSpec
	methods map[string][]method
	types   []string // the type names, in declaration order
	// testedNames memoises testedNames.
	testedMemo []string
	// sources are the norm'd stems of the directory's non-test .go files,
	// read from the directory, so an external test package sees them.
	sources map[string]bool
}

// newPkg reads the pass's files that sit in its directory. A file
// elsewhere, as cgo's output is, is left out.
func newPkg(pass *analysis.Pass, s Settings) *pkg {
	p := &pkg{
		pass: pass, s: s, name: strings.TrimSuffix(pass.Pkg.Name(), "_test"),
		decl: map[string][]*file{}, spec: map[string]*ast.TypeSpec{}, methods: map[string][]method{},
		sources: map[string]bool{},
	}
	dirs := map[string]int{}
	for _, f := range pass.Files {
		dirs[filepath.Dir(pass.Fset.Position(f.Package).Filename)]++
	}
	dir := ""
	for d, n := range dirs {
		if n > dirs[dir] || n == dirs[dir] && d < dir {
			dir = d
		}
	}
	p.dirName = filepath.Base(dir)
	for _, f := range pass.Files {
		name := pass.Fset.Position(f.Package).Filename
		if filepath.Dir(name) != dir {
			continue
		}
		p.files = append(p.files, newFile(f, filepath.Base(name), pass.Fset.File(f.Package).LineCount()))
	}
	if entries, err := os.ReadDir(dir); err == nil {
		for _, e := range entries {
			n := e.Name()
			if !e.IsDir() && strings.HasSuffix(n, ".go") && !strings.HasSuffix(n, "_test.go") {
				p.sources[norm(dropPlatform(strings.TrimSuffix(n, ".go")))] = true
			}
		}
	}
	for _, f := range p.files {
		for _, d := range f.ast.Decls {
			switch d := d.(type) {
			case *ast.GenDecl:
				for _, sp := range d.Specs {
					if ts, ok := sp.(*ast.TypeSpec); ok {
						if p.decl[ts.Name.Name] == nil {
							p.types = append(p.types, ts.Name.Name)
							p.spec[ts.Name.Name] = ts
						}
						p.decl[ts.Name.Name] = append(p.decl[ts.Name.Name], f)
					}
				}
			case *ast.FuncDecl:
				if r := recvName(d); r != "" {
					p.methods[r] = append(p.methods[r], method{f, d})
				}
			}
		}
	}
	return p
}

func (p *pkg) run() {
	for _, rule := range []func(){p.receiver, p.testName, p.typeFile, p.fileName, p.grabBag, p.tiny, p.long} {
		rule()
	}
}

// report reports rule's finding at n in f, unless the rule is off or
// exempted there.
func (p *pkg) report(rule string, f *file, n ast.Node, msg string) {
	if !p.s.enabled(rule) || p.s.exempts(rule, p.pass.Pkg.Path(), f.name) {
		return
	}
	pos := f.ast.Package
	if n != nil {
		pos = n.Pos()
	}
	p.pass.Report(analysis.Diagnostic{Pos: pos, Category: rule, Message: rule + ": " + msg})
}

// typeLines is the lines of typ's non-test, non-generated files: those
// declaring it, and those holding its methods; only its concern files among
// those unless every is set.
func (p *pkg) typeLines(typ string, every bool) int {
	seen := map[*file]bool{}
	n := 0
	add := func(f *file) {
		if !seen[f] && !f.test && !f.generated {
			seen[f] = true
			n += f.lines
		}
	}
	decls := p.decl[typ]
	for _, f := range decls {
		add(f)
	}
	for _, m := range p.methods[typ] {
		if every || m.f.concernOf(typ, decls) {
			add(m.f)
		}
	}
	return n
}

// split reports whether typ's methods may live in <type>_<concern>.go
// files: it is a non-test type whose files pass LongLines.
func (p *pkg) split(typ string) bool {
	decls := p.decl[typ]
	return len(decls) > 0 && !decls[0].test && p.typeLines(typ, false) > p.s.longLines()
}

// concern reports whether f is a concern file the split allows: it holds a
// method of a split type, and is named for that type.
func (p *pkg) concern(f *file) bool {
	for _, typ := range p.types {
		if p.split(typ) && f.concernOf(typ, p.decl[typ]) &&
			slices.ContainsFunc(p.methods[typ], func(m method) bool { return m.f == f }) {
			return true
		}
	}
	return false
}

// receiver reports each method declared in the file of another type: a
// file named for a type it declares, not the method's receiver. A method in
// a file named for a concern is not reported, nor, once its type's files
// pass LongLines, one in <type>_<concern>.go. A test file adding a method
// to a non-test type is reported, but for export_test.go, whose purpose
// that is. A method in a generated file, or on a
// type the pass does not declare, is not read.
func (p *pkg) receiver() {
	for _, typ := range p.types {
		decls := p.decl[typ]
		split, every := p.split(typ), p.typeLines(typ, true)
		home := decls[0].name
		concern := norm(typ) + "_<concern>.go"
		for _, m := range p.methods[typ] {
			if m.f.generated || slices.Contains(decls, m.f) || split && !m.f.test && m.f.concernOf(typ, decls) {
				continue
			}
			testAdds := m.f.test && !decls[0].test && m.f.stem != "export"
			if !testAdds && !p.anotherTypesFile(m.f, typ) {
				continue
			}
			name := typ + "." + m.decl.Name.Name
			msg := "method " + name + " is outside " + home + ", the file that declares " + typ
			fix := "move it into " + home
			switch {
			case testAdds:
				fix = "a test adds a method to " + home + "'s type: make " + m.decl.Name.Name + " a function in the test"
			case m.f.test:
			case split:
				fix += " or a " + concern + " file"
			case every > p.s.longLines():
				fix += ", or rename " + m.f.name + " to " + concern + ": the files of " + typ + " hold " +
					strconv.Itoa(every) + " lines, past " + strconv.Itoa(p.s.longLines())
			default:
				msg += " (its files hold " + strconv.Itoa(every) + " lines, at most " + strconv.Itoa(p.s.longLines()) + ", so no concern files)"
			}
			p.report(Receiver, m.f, m.decl, msg+": "+fix)
		}
	}
}

// anotherTypesFile reports whether f is named for a type it declares other
// than typ, and not for typ: a method of typ there sits in another type's
// file. A file of methods named for a concern is typ's own.
func (p *pkg) anotherTypesFile(f *file, typ string) bool {
	if f.namedForType(typ, p.name) {
		return false
	}
	return slices.ContainsFunc(p.types, func(u string) bool {
		return u != typ && slices.Contains(p.decl[u], f) && f.namedForType(u, p.name)
	})
}

// namesASource reports whether a test stem names a source file of the
// directory, alone, with one of testSuffixes, or followed by an aspect:
// root_special_test.go tests one aspect of root.go. An aspect of the file
// named for the package is ownTest's to judge.
func (p *pkg) namesASource(stem string) bool {
	cands := []string{stem}
	for _, suf := range testSuffixes {
		if s, ok := strings.CutSuffix(stem, suf); ok {
			cands = append(cands, s)
		}
	}
	for i := range len(stem) {
		if pre := stem[:i]; stem[i] == '_' && norm(pre) != norm(p.name) && norm(pre) != norm(p.dirName) {
			cands = append(cands, pre)
		}
	}
	return slices.ContainsFunc(cands, func(c string) bool { return p.sources[norm(c)] })
}

// testName reports each _test.go file that names no source file of its
// directory and no declaration of the package it tests. The package's own
// test files and testStems, Go's conventions, are not reported, nor is a
// directory with no source file: an integration suite names its tests for
// what they exercise.
func (p *pkg) testName() {
	if len(p.sources) == 0 {
		return
	}
	var decls []string
	for _, f := range p.files {
		if !f.test || f.exempt() || p.isGrabBag(f.stem) || p.namesASource(f.stem) || slices.Contains(testStems, f.stem) ||
			strings.HasPrefix(f.stem, "example_") {
			continue
		}
		if decls == nil {
			decls = p.tested()
		}
		if slices.ContainsFunc(variants(f.stem), func(v string) bool {
			return slices.ContainsFunc(decls, func(n string) bool { return covers(v, n, false) })
		}) {
			continue
		}
		if p.ownTest(f.stem) {
			continue
		}
		p.report(TestName, f, nil, f.name+" names no source file in its directory: name it for the file whose code it tests "+
			"(foo_test.go, foo_internal_test.go), or fold its tests into that file's test")
	}
}

// ownTest reports whether a test stem is the package's own: the package's
// or directory's name, alone or with one of testSuffixes; or the package's
// name, "_", and a source file's stem or, in package main, a subcommand's
// name. A stem naming a declaration is read before this.
func (p *pkg) ownTest(stem string) bool {
	base := stem
	for _, suf := range testSuffixes {
		base = strings.TrimSuffix(base, suf)
	}
	if norm(base) == norm(p.name) || norm(base) == norm(p.dirName) {
		return true
	}
	rest, ok := strings.CutPrefix(base, p.name+"_")
	if !ok {
		return false
	}
	return p.namesASource(rest) || p.name == "main" && slices.Contains(p.subcommands(), norm(rest))
}

// subcommands are the norm'd names of the subcommands a command's non-test
// files declare: a string case of a switch on an argument (os.Args[1],
// args[0], fs.Arg(0)), and the first word of the Name or Use of a composite
// literal of a type named Command (urfave/cli, cobra, ffcli).
func (p *pkg) subcommands() []string {
	var out []string
	add := func(e ast.Expr) {
		if lit, ok := e.(*ast.BasicLit); ok {
			if s, err := strconv.Unquote(lit.Value); err == nil && lit.Kind == token.STRING {
				if word, _, _ := strings.Cut(strings.TrimSpace(s), " "); word != "" {
					out = append(out, norm(word))
				}
			}
		}
	}
	for _, f := range p.files {
		if f.test {
			continue
		}
		ast.Inspect(f.ast, func(n ast.Node) bool {
			switch n := n.(type) {
			case *ast.SwitchStmt:
				if !isArg(n.Tag) {
					break
				}
				for _, s := range n.Body.List {
					for _, e := range s.(*ast.CaseClause).List {
						add(e)
					}
				}
			case *ast.CompositeLit:
				if !p.isCommand(n) {
					break
				}
				for _, e := range n.Elts {
					if kv, ok := e.(*ast.KeyValueExpr); ok {
						if key, ok := kv.Key.(*ast.Ident); ok && (key.Name == "Name" || key.Name == "Use") {
							add(kv.Value)
						}
					}
				}
			}
			return true
		})
	}
	return out
}

// isArg reports whether e reads a command-line argument: an index of a
// value named Args or args, or a call of a function or method named Arg.
func isArg(e ast.Expr) bool {
	switch e := ast.Unparen(e).(type) {
	case *ast.IndexExpr:
		switch x := e.X.(type) {
		case *ast.Ident:
			return x.Name == "args" || x.Name == "Args"
		case *ast.SelectorExpr:
			return x.Sel.Name == "Args"
		}
	case *ast.CallExpr:
		switch fun := e.Fun.(type) {
		case *ast.Ident:
			return fun.Name == "Arg"
		case *ast.SelectorExpr:
			return fun.Sel.Name == "Arg"
		}
	}
	return false
}

// isCommand reports whether lit is of a struct type named Command, or a
// pointer to one.
func (p *pkg) isCommand(lit *ast.CompositeLit) bool {
	t := p.pass.TypesInfo.TypeOf(lit)
	if ptr, ok := t.(*types.Pointer); ok {
		t = ptr.Elem()
	}
	named, ok := types.Unalias(t).(*types.Named)
	return ok && named.Obj().Name() == "Command"
}

// tested is testedNames, computed once.
func (p *pkg) tested() []string {
	if p.testedMemo == nil {
		p.testedMemo = p.testedNames()
	}
	return p.testedMemo
}

// testedNames are the package-level names, and the method names, that the
// non-test files of the package under test declare: the pass's package, or
// for an external test package, the package it imports under the path
// without _test.
func (p *pkg) testedNames() []string {
	tested := p.pass.Pkg
	if path, ok := strings.CutSuffix(tested.Path(), "_test"); ok {
		for _, imp := range tested.Imports() {
			if imp.Path() == path {
				tested = imp
			}
		}
	}
	out := []string{}
	add := func(obj types.Object) {
		if !strings.HasSuffix(p.pass.Fset.Position(obj.Pos()).Filename, "_test.go") {
			out = append(out, obj.Name())
		}
	}
	scope := tested.Scope()
	for _, name := range scope.Names() {
		obj := scope.Lookup(name)
		add(obj)
		if named, ok := obj.Type().(*types.Named); ok && obj.Pkg() == tested {
			for m := range named.Methods() {
				add(m)
			}
		}
	}
	return out
}

// typeFile reports each non-test type with methods whose declaring file is
// not named for it.
func (p *pkg) typeFile() {
	for _, typ := range p.types {
		decls := p.decl[typ]
		if len(decls) != 1 || len(p.methods[typ]) == 0 {
			continue
		}
		f := decls[0]
		if f.test || f.exempt() || f.namedForType(typ, p.name) || p.namedForPackage(f) {
			continue
		}
		p.report(TypeFile, f, p.spec[typ], "type "+typ+" has "+strconv.Itoa(len(p.methods[typ]))+
			" method(s) but is declared in "+f.name+": move it and its methods into "+norm(typ)+".go")
	}
}

// namedForPackage reports whether f is named for its package or
// directory, or is main.go of a command: the file a package's own types
// live in.
func (p *pkg) namedForPackage(f *file) bool {
	stem := norm(f.stem)
	return stem == norm(p.name) || stem == norm(p.dirName) || p.name == "main" && stem == "main"
}

// namedFor reports whether f's name, or a trailing run of its parts, spells
// a run of words of something it declares; or is its package's or
// directory's name; or names a type whose concern file it is.
func (p *pkg) namedFor(f *file) bool {
	if p.namedForPackage(f) || norm(f.stem) == "errors" && f.holdsErrors() {
		return true
	}
	for _, n := range f.names() {
		if slices.ContainsFunc(variants(f.stem), func(v string) bool { return covers(v, n, false) }) {
			return true
		}
		if decls := p.decl[n]; len(decls) > 0 && f.concernOf(n, decls) {
			return true
		}
	}
	return false
}

// fileName reports each non-test file named for nothing it declares.
func (p *pkg) fileName() {
	for _, f := range p.files {
		if f.test || f.exempt() || p.isGrabBag(f.stem) || p.namedFor(f) {
			continue
		}
		p.report(FileName, f, nil, f.name+" is named for nothing it declares: rename it for the type it declares, "+
			"or move each declaration to the file of the type it serves")
	}
}

// isGrabBag reports whether stem has a grab-bag part that names no type of
// the package's non-test files (types.go holding a type Types is named),
// and is not the package's own name. A stem of two or more parts that spells
// a declaration of the package under test uses the word as a domain term:
// helper_config.go holds HelperConfig.
func (p *pkg) isGrabBag(stem string) bool {
	if norm(stem) == norm(p.name) || norm(stem) == norm(p.dirName) {
		return false
	}
	if strings.Contains(stem, "_") && slices.ContainsFunc(p.tested(), func(n string) bool { return covers(norm(stem), n, false) }) {
		return false
	}
	bag := p.s.grabBag()
	return slices.ContainsFunc(strings.Split(stem, "_"), func(part string) bool {
		if !slices.Contains(bag, part) {
			return false
		}
		return !slices.ContainsFunc(p.types, func(typ string) bool {
			return !p.decl[typ][0].test && covers(part, typ, true)
		})
	})
}

// grabBag reports each file, test or not, with a grab-bag name.
func (p *pkg) grabBag() {
	for _, f := range p.files {
		if !f.exempt() && p.isGrabBag(f.stem) {
			p.report(GrabBag, f, nil, f.name+" has a grab-bag name: move each declaration into the file of the type it serves, then delete the file")
		}
	}
}

// tiny reports each non-test file under TinyLines that declares no type. A
// package's only file has nowhere to fold into, and a file built for some
// platforms only cannot fold into a file the others build.
func (p *pkg) tiny() {
	code := 0
	for _, f := range p.files {
		if !f.test && !f.exempt() {
			code++
		}
	}
	for _, f := range p.files {
		if code <= 1 || f.test || f.exempt() || f.lines >= p.s.tinyLines() || f.declaresType() || p.concern(f) || f.platform() {
			continue
		}
		fix := "fold it into the file of the type its code serves, or its only caller's file"
		if len(f.ast.Decls) == 0 {
			fix = "it holds only the package doc: rename it doc.go"
		}
		p.report(Tiny, f, nil, f.name+" is "+strconv.Itoa(f.lines)+" lines and declares no type: "+fix)
	}
}

// long reports each non-test file over LongLines.
func (p *pkg) long() {
	for _, f := range p.files {
		if !f.test && !f.exempt() && f.lines > p.s.longLines() {
			p.report(Long, f, nil, f.name+" is "+strconv.Itoa(f.lines)+" lines, over "+strconv.Itoa(p.s.longLines())+
				": split it, a file per type, or a <type>_<concern>.go file per concern of a large type")
		}
	}
}
