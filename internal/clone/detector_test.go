package clone_test

import (
	"go/ast"
	"go/parser"
	"go/token"
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
