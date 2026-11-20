package narration

import (
	"fmt"
	"go/ast"
	"go/token"
	"strings"

	"golang.org/x/tools/go/analysis"
)

// unit is the span of source whose warning markers are counted together,
// and its name for a message.
type unit struct {
	start, end token.Pos
	name       string
}

// contains reports whether cg lies inside u.
func (u unit) contains(cg *ast.CommentGroup) bool { return u.start <= cg.Pos() && cg.End() <= u.end }

// warnsign reports every warning marker of f after the first in its unit.
// A unit is the innermost of: a top-level declaration with its doc; one
// spec of a grouped type, const or var declaration with its doc and line
// comment; one field of a struct or one method of an interface declared by
// a type spec, with its doc and line comment. A comment block outside every
// declaration is its own unit.
func warnsign(pass *analysis.Pass, f *ast.File) {
	seen := map[unit]int{}
	for _, cg := range f.Comments {
		u := unitOf(f, cg)
		for _, l := range prose(f, cg) {
			m := marker(l.text)
			if m == "" {
				continue
			}
			seen[u]++
			if n := seen[u]; n > 1 {
				pass.Report(analysis.Diagnostic{Pos: l.pos, Category: Warnsign,
					Message: fmt.Sprintf("%s: %q is warning marker %d in %s: a marker on every comment marks none; keep one per declaration, on the comment a reader must not miss, and say what goes wrong instead",
						Warnsign, m, n, u.name)})
			}
		}
	}
}

// unitOf returns the innermost unit holding cg.
func unitOf(f *ast.File, cg *ast.CommentGroup) unit {
	for _, d := range f.Decls {
		u := unit{declStart(d), d.End(), declName(d)}
		if !u.contains(cg) {
			continue
		}
		g, ok := d.(*ast.GenDecl)
		if !ok {
			return u
		}
		for _, s := range g.Specs {
			su := u
			if g.Lparen.IsValid() {
				su = specUnit(s)
			}
			if !su.contains(cg) {
				continue
			}
			if ts, ok := s.(*ast.TypeSpec); ok {
				return fieldUnit(ts.Name.Name, ts.Type, cg, su)
			}
			return su
		}
		if g.Lparen.IsValid() {
			u.name = "this " + g.Tok.String() + " group"
		}
		return u
	}
	return unit{cg.Pos(), cg.End(), "this comment block"}
}

// specUnit is one spec of a grouped declaration.
func specUnit(s ast.Spec) unit {
	switch s := s.(type) {
	case *ast.TypeSpec:
		return unit{docStart(s.Doc, s.Pos()), lineEnd(s.Comment, s.End()), s.Name.Name}
	case *ast.ValueSpec:
		return unit{docStart(s.Doc, s.Pos()), lineEnd(s.Comment, s.End()), s.Names[0].Name}
	}
	return unit{s.Pos(), s.End(), "this spec"}
}

// fieldUnit returns the unit of the innermost struct field or interface
// method of typ holding cg, named owner.Field, or outer if none holds it.
func fieldUnit(owner string, typ ast.Expr, cg *ast.CommentGroup, outer unit) unit {
	var fields *ast.FieldList
	switch t := typ.(type) {
	case *ast.StructType:
		fields = t.Fields
	case *ast.InterfaceType:
		fields = t.Methods
	default:
		return outer
	}
	for _, fd := range fields.List {
		name := "an embedded field"
		if len(fd.Names) > 0 {
			name = fd.Names[0].Name
		}
		fu := unit{docStart(fd.Doc, fd.Pos()), lineEnd(fd.Comment, fd.End()), owner + "." + name}
		if fu.contains(cg) {
			return fieldUnit(fu.name, fd.Type, cg, fu)
		}
	}
	return outer
}

func docStart(doc *ast.CommentGroup, pos token.Pos) token.Pos {
	if doc != nil {
		return doc.Pos()
	}
	return pos
}

func lineEnd(comment *ast.CommentGroup, end token.Pos) token.Pos {
	if comment != nil {
		return comment.End()
	}
	return end
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
		return docStart(d.Doc, d.Pos())
	case *ast.GenDecl:
		return docStart(d.Doc, d.Pos())
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
