package ignore_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/chrisrob11/gosloplint/internal/ignore"
)

// reporter is an analyzer reporting every declared name, package clauses
// included, starting with one of prefixes, its category the prefix when
// categorize is set.
func reporter(name string, categorize bool, prefixes ...string) *analysis.Analyzer {
	return &analysis.Analyzer{
		Name: name,
		Doc:  name + ": test reporter",
		Run: func(pass *analysis.Pass) (any, error) {
			report := func(id *ast.Ident) {
				for _, p := range prefixes {
					if strings.HasPrefix(id.Name, p) {
						cat := ""
						if categorize {
							cat = p
						}
						pass.Report(analysis.Diagnostic{Pos: id.Pos(), Category: cat, Message: p + ": " + id.Name})
					}
				}
			}
			for id := range pass.TypesInfo.Defs {
				report(id)
			}
			return nil, nil
		},
	}
}

var known = map[string][]string{"fake": {"one", "two"}, "plain": nil, "other": nil}

func TestDirectives(t *testing.T) {
	wrapped := ignore.Wrap([]*analysis.Analyzer{reporter("fake", true, "one", "two"), reporter("plain", false, "plain")}, known)
	analysistest.Run(t, analysistest.TestData(), wrapped[0], "onepkg")
	analysistest.Run(t, analysistest.TestData(), wrapped[1], "second")
}

// TestCovers checks a directive's lines directly, as gosloplint dupl reads
// them: one with no reason or of another check covers nothing.
func TestCovers(t *testing.T) {
	fset := token.NewFileSet()
	src := "package p\n\n//gosloplint:ignore dupl\nvar a int\n\n//gosloplint:ignore dupl:x why\nvar b int\n\n//gosloplint:ignore dupl why\nvar c int\n"
	f, err := parser.ParseFile(fset, "p.go", src, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	dirs := ignore.Parse(fset, f)
	if len(dirs) != 3 {
		t.Fatalf("%d directives, want 3", len(dirs))
	}
	for i, want := range []bool{false, false, true} {
		v := f.Decls[i].(*ast.GenDecl).Specs[0].(*ast.ValueSpec).Names[0]
		if got := dirs[i].Covers(fset, "", v.Pos()); got != want {
			t.Errorf("directive %d covers %s: %v, want %v", i, v.Name, got, want)
		}
		if dirs[i].Covers(fset, "", token.NoPos) {
			t.Errorf("directive %d covers no position", i)
		}
	}
}
