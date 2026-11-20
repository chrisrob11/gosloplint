package clone_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"slices"
	"strings"
	"testing"

	"github.com/chrisrob11/gosloplint/internal/clone"
)

const body = `
	a := in.x + in.y*2
	if a > 10 {
		return a
	}
	b := in.z + a
	for i := 0; i < b; i++ {
		a += i
	}
	b += a * 3
	return a + b
}
`

func parse(t *testing.T, fset *token.FileSet, name, src string) *ast.File {
	t.Helper()
	f, err := parser.ParseFile(fset, name, src, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	return f
}

func TestClonesAcrossFilesNameGenericReceivers(t *testing.T) {
	fset := token.NewFileSet()
	d := clone.NewDetector(fset, clone.Config{MinNodes: 40})
	d.Add(parse(t, fset, "z.go", "package p\n\nfunc (s *stack[T]) push(in input) int {"+body))
	d.Add(parse(t, fset, "a.go", "package p\n\nfunc (m pair[K, V]) get(in input) int {"+body))
	got := d.Clones()
	if len(got) != 1 {
		t.Fatalf("clones = %+v, want one", got)
	}
	c := got[0]
	if c.A.File != "a.go" || c.A.Decl != "pair.get" || c.B.File != "z.go" || c.B.Decl != "stack.push" {
		t.Errorf("clone = %+v, want a.go pair.get before z.go stack.push", c)
	}
	want := "lines 4-12 duplicate lines 4-12 of a.go (pair.get): move the shared statements into one function both call"
	if msg := c.Message("a.go"); msg != want {
		t.Errorf("Message = %q, want %q", msg, want)
	}
}

func TestIncludes(t *testing.T) {
	c := clone.Config{Exempt: []string{"*_gen.go", "example.com/x/..."}}
	for _, tc := range []struct {
		pkg, file string
		want      bool
	}{
		{"example.com/y", "/src/y/y.go", true},
		{"example.com/y", "/src/y/y_test.go", false},
		{"example.com/y", "/src/y/y_gen.go", false},
		{"example.com/x/sub", "/src/x/sub/s.go", false},
		{"example.com/x_test", "/src/x/x.go", false},
	} {
		if got := c.Includes(tc.pkg, tc.file); got != tc.want {
			t.Errorf("Includes(%s, %s) = %v, want %v", tc.pkg, tc.file, got, tc.want)
		}
	}
	c.Tests = true
	if !c.Includes("example.com/y", "/src/y/y_test.go") {
		t.Error("Tests set: a _test.go file is not read")
	}
}

// TestRunsOfPlainCallsAreData repeats one statement twelve times in two
// functions, the second calling other names: a run of bare or guarded calls
// is data then, any other run is a copy.
func TestRunsOfPlainCallsAreData(t *testing.T) {
	for _, tc := range []struct {
		stmt string
		want bool
	}{
		{`fs.Var(&o.name, "name", nil, []string{"x"}, func() {})`, false},
		{`fs.Sub().Var(o.a.b, 1)`, false},
		{`if err := s.step(o.a); err != nil { return err }`, false},
		{`if err := s.step(); err != nil { return fmt.Errorf("step: %w", err) }`, false},
		{`fs.Var(o.a + 1)`, true},
		{`fs.Var(-o.a)`, true},
		{`fs[0].Var(o.a)`, true},
		{`if err := s.step(o.a + 1); err != nil { return err }`, true},
		{`if err := s.step(); err != nil { return err } else { o.a++ }`, true},
		{`if err := s.step(); err != nil { log(err); return err }`, true},
		{`if a, err := s.step(); err != nil { return err }`, true},
		{`if err := o.a; err != nil { return err }`, true},
		{`if err := s.step(); err == nil { return err }`, true},
		{`if err := s.step(); o.err != nil { return err }`, true},
		{`if err := s.step(); err != o.nil { return err }`, true},
		{`if err := s.step(); err != nil { o.a++ }`, true},
		{`if err := s.step(); err != nil { return err + 1 }`, true},
		{`if err := s.step(); err != nil { return s.wrap(err + 1) }`, true},
		{`if s.ok() { return nil }`, true},
		{`o.a++`, true},
	} {
		fset := token.NewFileSet()
		d := clone.NewDetector(fset, clone.Config{MinNodes: 40})
		run := strings.Repeat(tc.stmt+"\n", 12)
		other := strings.NewReplacer("Var", "Set", "step", "other").Replace(run)
		d.Add(parse(t, fset, "a.go", "package p\n\nfunc a() error {\n"+run+"return nil\n}\n\nfunc b() error {\n"+other+"return nil\n}\n"))
		if got := acrossDecls(d.Clones()); got != tc.want {
			t.Errorf("%s: reported %v, want %v", tc.stmt, got, tc.want)
		}
	}
}

func TestRunsOfPlainCallsMatchWhenSpelledAlike(t *testing.T) {
	run := strings.Repeat("fs.Var(&o.host, \"host\", \"localhost\")\n", 5)
	for _, tc := range []struct {
		b    string
		want bool
	}{
		{run, true},
		{strings.ReplaceAll(run, "o.host", "p.addr"), true},
		{strings.ReplaceAll(run, `"host"`, `"addr"`), false},
		{strings.ReplaceAll(run, "fs.Var", "fs.Set"), false},
		{strings.ReplaceAll(run, "fs.Var", "set"), false},
	} {
		fset := token.NewFileSet()
		d := clone.NewDetector(fset, clone.Config{MinNodes: 40})
		d.Add(parse(t, fset, "a.go", "package p\n\nfunc a() {\n"+run+"}\n\nfunc b() {\n"+tc.b+"}\n"))
		if got := acrossDecls(d.Clones()); got != tc.want {
			t.Errorf("b = %q: reported %v, want %v", tc.b[:40], got, tc.want)
		}
	}
}

// acrossDecls reports whether a clone joins two declarations; a run
// repeated inside one is not what these tests are about.
func acrossDecls(cs []clone.Clone) bool {
	return slices.ContainsFunc(cs, func(c clone.Clone) bool { return c.A.Decl != c.B.Decl })
}

func TestRunsOfCallsWithoutLiteralsMatchByName(t *testing.T) {
	run := strings.Repeat("r.Unregister(m.hits)\n", 6)
	for _, tc := range []struct {
		b    string
		want bool
	}{
		{run, true},
		{strings.ReplaceAll(run, "m.hits", "m.misses"), false},
	} {
		fset := token.NewFileSet()
		d := clone.NewDetector(fset, clone.Config{MinNodes: 40})
		d.Add(parse(t, fset, "a.go", "package p\n\nfunc a() {\n"+run+"}\n\nfunc b() {\n"+tc.b+"}\n"))
		if got := acrossDecls(d.Clones()); got != tc.want {
			t.Errorf("b = %q: reported %v, want %v", tc.b[:20], got, tc.want)
		}
	}
}
