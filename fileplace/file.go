package fileplace

import (
	"go/ast"
	"go/token"
	"slices"
	"strings"
)

// file is one file of the pass, with what the rules read of its name.
type file struct {
	ast       *ast.File
	name      string // base name, foo_linux_test.go
	base      string // name without .go
	stem      string // base without _test and any GOOS/GOARCH suffix
	test      bool
	generated bool
	lines     int
}

func newFile(f *ast.File, name string, lines int) *file {
	base := strings.TrimSuffix(name, ".go")
	return &file{
		ast:       f,
		name:      name,
		base:      base,
		stem:      dropPlatform(strings.TrimSuffix(base, "_test")),
		test:      strings.HasSuffix(base, "_test"),
		generated: ast.IsGenerated(f),
		lines:     lines,
	}
}

// platform reports whether the file builds only on some platforms: its name
// ends in a part of platforms, or it has a //go:build line.
func (f *file) platform() bool {
	if dropPlatform(f.base) != f.base {
		return true
	}
	for _, g := range f.ast.Comments {
		if g.Pos() >= f.ast.Package {
			break
		}
		for _, c := range g.List {
			if strings.HasPrefix(c.Text, "//go:build ") {
				return true
			}
		}
	}
	return false
}

// exempt reports whether no rule reads the file: it is generated, or it is
// doc.go.
func (f *file) exempt() bool { return f.generated || f.base == "doc" }

// names are the declared names the file holds: its types, functions,
// methods, variables and constants, and the receiver types of its methods.
func (f *file) names() []string {
	var out []string
	for _, d := range f.ast.Decls {
		switch d := d.(type) {
		case *ast.FuncDecl:
			out = append(out, d.Name.Name)
			if r := recvName(d); r != "" {
				out = append(out, r)
			}
		case *ast.GenDecl:
			for _, s := range d.Specs {
				switch s := s.(type) {
				case *ast.TypeSpec:
					out = append(out, s.Name.Name)
				case *ast.ValueSpec:
					for _, n := range s.Names {
						out = append(out, n.Name)
					}
				}
			}
		}
	}
	return out
}

// declaresType reports whether the file has a top-level type declaration.
func (f *file) declaresType() bool {
	return slices.ContainsFunc(f.ast.Decls, func(d ast.Decl) bool {
		g, ok := d.(*ast.GenDecl)
		return ok && g.Tok == token.TYPE
	})
}

// holdsErrors reports whether the file holds what errors.go holds: a free
// function whose one result is an error, an Err sentinel, or an error type.
func (f *file) holdsErrors() bool {
	for _, d := range f.ast.Decls {
		switch d := d.(type) {
		case *ast.FuncDecl:
			r := d.Type.Results
			if d.Recv != nil || r == nil || len(r.List) != 1 || len(r.List[0].Names) > 1 {
				continue
			}
			if id, ok := r.List[0].Type.(*ast.Ident); ok && id.Name == "error" {
				return true
			}
		case *ast.GenDecl:
			for _, s := range d.Specs {
				switch s := s.(type) {
				case *ast.TypeSpec:
					if strings.HasSuffix(s.Name.Name, "Error") {
						return true
					}
				case *ast.ValueSpec:
					for _, n := range s.Names {
						if strings.HasPrefix(n.Name, "Err") || strings.HasPrefix(n.Name, "err") {
							return true
						}
					}
				}
			}
		}
	}
	return false
}

// namedForType reports whether the file's name is typ's own: it spells all
// of typ's words, a short one skipped (role.go holds RoleID); or the file is
// named for its package and typ is the package's T.
func (f *file) namedForType(typ, pkgName string) bool {
	if typ == "T" && norm(f.stem) == norm(pkgName) {
		return true
	}
	return slices.ContainsFunc(variants(f.stem), func(v string) bool { return covers(v, typ, true) })
}

// concernOf reports whether the file is a <prefix>_<concern>.go file whose
// prefix names typ or a file that declares it.
func (f *file) concernOf(typ string, decls []*file) bool {
	parts := strings.Split(f.stem, "_")
	for k := 1; k < len(parts); k++ {
		prefix := norm(strings.Join(parts[:k], ""))
		if prefix == norm(typ) || slices.ContainsFunc(decls, func(d *file) bool { return prefix == norm(d.stem) }) {
			return true
		}
	}
	return false
}

// recvName is the name of d's receiver type, or "" for a function or a
// receiver no type name reads.
func recvName(d *ast.FuncDecl) string {
	if d.Recv == nil || len(d.Recv.List) != 1 {
		return ""
	}
	e := d.Recv.List[0].Type
	for {
		switch t := e.(type) {
		case *ast.StarExpr:
			e = t.X
		case *ast.ParenExpr:
			e = t.X
		case *ast.IndexExpr:
			e = t.X
		case *ast.IndexListExpr:
			e = t.X
		case *ast.Ident:
			return t.Name
		default:
			return ""
		}
	}
}
