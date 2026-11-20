package narration

import (
	"fmt"
	"go/ast"
	"go/token"
	"strings"

	"golang.org/x/tools/go/analysis"
)

// warnsign reports every warning marker of f after the first in its unit:
// a top-level declaration with its doc, or a comment block outside every
// declaration.
func warnsign(pass *analysis.Pass, f *ast.File) {
	seen := map[token.Pos]int{}
	for _, cg := range f.Comments {
		unit, name := cg.Pos(), "this comment block"
		for _, d := range f.Decls {
			if start := declStart(d); start <= cg.Pos() && cg.End() <= d.End() {
				unit, name = start, declName(d)
				break
			}
		}
		for _, l := range prose(f, cg) {
			m := marker(l.text)
			if m == "" {
				continue
			}
			seen[unit]++
			if n := seen[unit]; n > 1 {
				pass.Report(analysis.Diagnostic{Pos: l.pos, Category: Warnsign,
					Message: fmt.Sprintf("%s: %q is warning marker %d in %s: a marker on every comment marks none; keep one per declaration, on the comment a reader must not miss, and say what goes wrong instead",
						Warnsign, m, n, name)})
			}
		}
	}
}

// marker returns the warning marker text holds, "" if none: ⚠ anywhere, or
// a marker word and colon opening the line, bold or not.
func marker(text string) string {
	if strings.Contains(text, "⚠") {
		return "⚠"
	}
	t := strings.TrimLeft(text, " \t*_")
	for _, w := range []string{"WARNING", "IMPORTANT", "NOTE", "CAUTION", "NB", "Warning", "Important", "Note", "Caution"} {
		if rest, ok := strings.CutPrefix(t, w); ok && strings.HasPrefix(strings.TrimLeft(rest, "*_"), ":") {
			return w + ":"
		}
	}
	return ""
}

func declStart(d ast.Decl) token.Pos {
	switch d := d.(type) {
	case *ast.FuncDecl:
		if d.Doc != nil {
			return d.Doc.Pos()
		}
	case *ast.GenDecl:
		if d.Doc != nil {
			return d.Doc.Pos()
		}
	}
	return d.Pos()
}

// declName names d for a message: its func, Type.Method, or first spec.
func declName(d ast.Decl) string {
	switch d := d.(type) {
	case *ast.FuncDecl:
		if d.Recv != nil && len(d.Recv.List) == 1 {
			t := d.Recv.List[0].Type
			if s, ok := t.(*ast.StarExpr); ok {
				t = s.X
			}
			if id, ok := t.(*ast.Ident); ok {
				return id.Name + "." + d.Name.Name
			}
		}
		return d.Name.Name
	case *ast.GenDecl:
		if len(d.Specs) > 0 {
			switch s := d.Specs[0].(type) {
			case *ast.TypeSpec:
				return s.Name.Name
			case *ast.ValueSpec:
				return s.Names[0].Name
			}
		}
		return "this " + d.Tok.String() + " declaration"
	}
	return "this declaration"
}
