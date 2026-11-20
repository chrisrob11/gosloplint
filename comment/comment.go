// Package comment defines an Analyzer that reports comment blocks longer
// than a per-kind line limit, and packages whose comment density runs past a
// ceiling.
//
// A doc comment is read once, by someone arriving; a comment inside a
// function body competes with the code around it. One limit for both is the
// wrong instrument, so each kind of block carries its own: package, file
// (a top-level comment documenting nothing, such as a header above the
// package clause), type, func, field, value and inline (one inside a
// declaration documenting nothing). What a block documents comes from the
// AST, so a comment separated from a declaration by a blank line is not its
// doc comment.
//
// One block is exempted with a directive line inside it, and the reason is
// required:
//
//	//gosloplint:comment-ok the grammar table below needs the room
//
// Go directive lines (//go:generate, //nolint:x, the escape itself) are not
// prose and are not counted, nor is the blank // gofmt puts before a
// directive it moves to the end of a doc comment.
//
// The density check is per package: comment lines over the non-blank lines
// of the package's non-test, non-generated files. A line holding code and a
// trailing comment is code; a license or copyright header is in no count. A
// package smaller than DensityMinLines is not
// held to it, and nothing exempts a package but disabling the check. It is
// reported once, on the package clause of the file with the most comment
// lines, preferring a file that declares something over a doc.go.
//
// Generated files are skipped.
package comment

import (
	"fmt"
	"go/ast"
	"go/token"
	"strings"

	"golang.org/x/tools/go/analysis"

	"github.com/chrisrob11/gosloplint/internal/listflag"
)

// Escape is the directive that exempts one comment block. A reason must
// follow it.
const Escape = "//gosloplint:comment-ok"

// Analyzer reports long comment blocks and comment-dense packages. Its
// settings come from flags; New builds one with explicit settings instead.
var Analyzer = newAnalyzer(&Settings{}, true)

// New returns a comment Analyzer configured by s rather than by flags. It
// panics if s does not pass Validate.
func New(s Settings) *analysis.Analyzer {
	if err := s.Validate(); err != nil {
		panic("comment.New: " + err.Error())
	}
	return newAnalyzer(&s, false)
}

func newAnalyzer(s *Settings, withFlags bool) *analysis.Analyzer {
	a := &analysis.Analyzer{
		Name: "comment",
		Doc:  "comment: report comment blocks over a per-kind line limit and packages over a comment-density ceiling\n\nChecks: length (package 60, file 20, type 10, func 8, field 4, value 8, inline 3 lines by default) and density (20% of a package's non-blank lines). " + Escape + " <reason> exempts one block.",
		URL:  "https://github.com/chrisrob11/gosloplint#comment",
	}
	a.Run = func(pass *analysis.Pass) (any, error) {
		if err := s.Validate(); err != nil {
			return nil, err
		}
		run(pass, *s)
		return nil, nil
	}
	if withFlags {
		a.Flags.Var(listflag.List{Items: &s.Disable}, "disable", "comma-separated checks not to run: length, density")
		for _, f := range []struct {
			k Kind
			p *int
		}{
			{KindPackage, &s.Limits.Package}, {KindFile, &s.Limits.File}, {KindType, &s.Limits.Type},
			{KindFunc, &s.Limits.Func}, {KindField, &s.Limits.Field}, {KindValue, &s.Limits.Value},
			{KindInline, &s.Limits.Inline},
		} {
			a.Flags.IntVar(f.p, string(f.k), 0, fmt.Sprintf("longest %s comment block, in lines (0: %d)", f.k, DefaultLimits.For(f.k)))
		}
		a.Flags.IntVar(&s.Density, "density", 0, fmt.Sprintf("per-package comment ceiling, in percent (0: %d)", DefaultDensity))
		a.Flags.IntVar(&s.DensityMinLines, "density-min-lines", 0,
			fmt.Sprintf("non-blank lines below which a package is not held to the ceiling (0: %d)", DefaultDensityMinLines))
	}
	return a
}

func run(pass *analysis.Pass, s Settings) {
	if s.enabled(Length) {
		for _, f := range pass.Files {
			if !ast.IsGenerated(f) {
				length(pass, f, s.Limits)
			}
		}
	}
	if s.enabled(Density) {
		density(pass, s.density(), s.densityMinLines())
	}
}

// length reports each block of f over its kind's limit, and each escape
// with no reason.
func length(pass *analysis.Pass, f *ast.File, lim Limits) {
	kinds := classify(f)
	for _, cg := range f.Comments {
		exempt := false
		for _, c := range cg.List {
			reason, ok := escapeReason(c.Text)
			switch {
			case !ok:
			case reason == "":
				pass.Reportf(c.Pos(), "%s: %s needs a reason after it, saying why this block needs the room", Length, Escape)
			default:
				exempt = true
			}
		}
		if exempt {
			continue
		}
		kind := kinds[cg]
		n, limit := countedLines(pass.Fset, cg), lim.For(kind)
		if n > limit {
			pass.Reportf(cg.Pos(), "%s: %s comment is %d lines, limit %d: shorten it, or exempt the block with %s <reason>",
				Length, kind, n, limit, Escape)
		}
	}
}

// escapeReason reports whether text is the escape directive, and the reason
// after it.
func escapeReason(text string) (string, bool) {
	rest, ok := strings.CutPrefix(text, Escape)
	if !ok || rest != "" && rest[0] != ' ' && rest[0] != '\t' {
		return "", false
	}
	return strings.TrimSpace(rest), true
}

// classify maps every comment group of f to the kind of node it documents.
func classify(f *ast.File) map[*ast.CommentGroup]Kind {
	kinds := map[*ast.CommentGroup]Kind{}
	assign := func(k Kind, groups ...*ast.CommentGroup) {
		for _, cg := range groups {
			if _, seen := kinds[cg]; cg != nil && !seen {
				kinds[cg] = k
			}
		}
	}
	assign(KindPackage, f.Doc)
	ast.Inspect(f, func(n ast.Node) bool {
		switch d := n.(type) {
		case *ast.FuncDecl:
			assign(KindFunc, d.Doc)
		case *ast.GenDecl:
			switch d.Tok {
			case token.TYPE:
				assign(KindType, d.Doc)
			case token.IMPORT:
			default:
				assign(KindValue, d.Doc)
			}
		case *ast.TypeSpec:
			assign(KindType, d.Doc, d.Comment)
		case *ast.ValueSpec:
			assign(KindValue, d.Doc, d.Comment)
		case *ast.Field:
			assign(KindField, d.Doc, d.Comment)
		}
		return true
	})
	for _, cg := range f.Comments {
		if _, seen := kinds[cg]; seen {
			continue
		}
		kinds[cg] = KindFile
		for _, d := range f.Decls {
			if d.Pos() <= cg.Pos() && cg.End() <= d.End() {
				kinds[cg] = KindInline
				break
			}
		}
	}
	return kinds
}

// countedLines is the block's length in lines, without its directive lines
// and the blank // gofmt puts before a directive it moves.
func countedLines(fset *token.FileSet, cg *ast.CommentGroup) int {
	n := fset.Position(cg.End()).Line - fset.Position(cg.Pos()).Line + 1
	for i, c := range cg.List {
		if !directive(c.Text) {
			continue
		}
		n -= fset.Position(c.End()).Line - fset.Position(c.Pos()).Line + 1
		if i > 0 && strings.TrimSpace(strings.TrimPrefix(cg.List[i-1].Text, "//")) == "" && !directive(cg.List[i-1].Text) {
			n--
		}
	}
	return max(n, 0)
}

// directive reports whether text is a Go directive comment: //word:rest
// with no space after the slashes, the word lower-case letters and digits.
func directive(text string) bool {
	body, ok := strings.CutPrefix(text, "//")
	if !ok {
		return false
	}
	word, rest, ok := strings.Cut(body, ":")
	if !ok || word == "" || rest == "" || rest[0] == ' ' {
		return false
	}
	for _, r := range word {
		if (r < 'a' || r > 'z') && (r < '0' || r > '9') {
			return false
		}
	}
	return true
}
