// Package ignore is the per-site escape hatch every analyzer shares: a
// directive line
//
//	//gosloplint:ignore <analyzer>[:<check>] <reason>
//
// suppresses the named analyzer's findings, of one check or all of them, on
// its own line and the line below; in a declaration's doc comment, the
// whole declaration with its doc. Wrap applies it around each analyzer's
// Run, and reports a directive that cannot be trusted: one with no reason,
// an unknown analyzer or check, or one that suppresses nothing.
package ignore

import (
	"fmt"
	"go/ast"
	"go/token"
	"maps"
	"slices"
	"strings"

	"golang.org/x/tools/go/analysis"
)

// Prefix starts a directive line.
const Prefix = "//gosloplint:ignore"

// Category is the Diagnostic.Category of a finding about a directive.
const Category = "ignore"

// Directive is one directive line and the lines it covers.
type Directive struct {
	Pos      token.Pos
	Analyzer string
	Check    string // "" for every check of Analyzer
	Reason   string
	file     string
	// spans are the inclusive line ranges covered.
	spans [][2]int
}

// Parse returns the directives of f, in source order, nil for a generated
// file, which no analyzer reports on.
func Parse(fset *token.FileSet, f *ast.File) []*Directive {
	if ast.IsGenerated(f) {
		return nil
	}
	decls := docSpans(fset, f)
	var out []*Directive
	for _, cg := range f.Comments {
		for _, c := range cg.List {
			rest, ok := strings.CutPrefix(c.Text, Prefix)
			if !ok || rest != "" && rest[0] != ' ' && rest[0] != '\t' {
				continue
			}
			target, reason := strings.TrimSpace(rest), ""
			if i := strings.IndexAny(target, " \t"); i >= 0 {
				target, reason = target[:i], target[i:]
			}
			name, check, _ := strings.Cut(target, ":")
			pos := fset.Position(c.Pos())
			d := &Directive{Pos: c.Pos(), Analyzer: name, Check: check, Reason: strings.TrimSpace(reason), file: pos.Filename,
				spans: [][2]int{{pos.Line, pos.Line + 1}}}
			if span, ok := decls[cg]; ok {
				d.spans = append(d.spans, span)
			}
			out = append(out, d)
		}
	}
	return out
}

// docSpans maps each doc comment of f's declarations to the lines from the
// doc to the declaration's end.
func docSpans(fset *token.FileSet, f *ast.File) map[*ast.CommentGroup][2]int {
	out := map[*ast.CommentGroup][2]int{}
	add := func(doc *ast.CommentGroup, end token.Pos) {
		if doc != nil {
			out[doc] = [2]int{fset.Position(doc.Pos()).Line, fset.Position(end).Line}
		}
	}
	for _, d := range f.Decls {
		switch d := d.(type) {
		case *ast.FuncDecl:
			add(d.Doc, d.End())
		case *ast.GenDecl:
			add(d.Doc, d.End())
			for _, s := range d.Specs {
				switch s := s.(type) {
				case *ast.TypeSpec:
					add(s.Doc, s.End())
				case *ast.ValueSpec:
					add(s.Doc, s.End())
				}
			}
		}
	}
	return out
}

// Covers reports whether d, well formed, suppresses a finding of check at
// pos.
func (d *Directive) Covers(fset *token.FileSet, check string, pos token.Pos) bool {
	if d.Reason == "" || d.Check != "" && d.Check != check {
		return false
	}
	p := fset.Position(pos)
	return p.Filename == d.file && slices.ContainsFunc(d.spans, func(s [2]int) bool { return s[0] <= p.Line && p.Line <= s[1] })
}

// Wrap returns a copy of each analyzer whose findings the pass's
// directives naming it suppress. known maps every analyzer name, enabled
// or not, to its checks, the categories of its findings. Each copy also
// reports the directives naming its analyzer that have no reason or an
// unknown check, or that suppress nothing when its Run succeeds; the first
// copy reports those naming no known analyzer. elsewhere maps an analyzer
// to those of its checks that no pass reports, so a directive naming one
// is never stale here.
func Wrap(analyzers []*analysis.Analyzer, known, elsewhere map[string][]string) []*analysis.Analyzer {
	out := make([]*analysis.Analyzer, len(analyzers))
	for i, a := range analyzers {
		w := *a
		first := i == 0
		w.Run = func(pass *analysis.Pass) (any, error) {
			var held []analysis.Diagnostic
			inner := *pass
			inner.Report = func(d analysis.Diagnostic) { held = append(held, d) }
			result, err := a.Run(&inner)
			var dirs []*Directive
			for _, f := range pass.Files {
				dirs = append(dirs, Parse(pass.Fset, f)...)
			}
			mine := make([]*Directive, 0, len(dirs))
			for _, d := range dirs {
				if msg := problem(d, a.Name, first, known); msg != "" {
					pass.Report(analysis.Diagnostic{Pos: d.Pos, Category: Category, Message: Category + ": " + msg})
				} else if d.Analyzer == a.Name {
					mine = append(mine, d)
				}
			}
			used := map[*Directive]bool{}
			for _, diag := range held {
				i := slices.IndexFunc(mine, func(d *Directive) bool { return d.Covers(pass.Fset, diag.Category, diag.Pos) })
				if i < 0 {
					pass.Report(diag)
					continue
				}
				used[mine[i]] = true
			}
			for _, d := range mine {
				if !used[d] && err == nil && !slices.Contains(elsewhere[a.Name], d.Check) {
					pass.Report(analysis.Diagnostic{Pos: d.Pos, Category: Category,
						Message: fmt.Sprintf("%s: %s %s suppresses no %s finding: delete it", Category, Prefix, d.target(), a.Name)})
				}
			}
			return result, err
		}
		out[i] = &w
	}
	return out
}

func (d *Directive) target() string {
	if d.Check == "" {
		return d.Analyzer
	}
	return d.Analyzer + ":" + d.Check
}

// problem is what is wrong with d for the copy of analyzer to report, ""
// when nothing is or another copy reports it.
func problem(d *Directive, analyzer string, first bool, known map[string][]string) string {
	checks, ok := known[d.Analyzer]
	switch {
	case !ok && !first, ok && d.Analyzer != analyzer:
		return ""
	case d.Analyzer == "":
		return fmt.Sprintf("%s names no analyzer: write %s <analyzer>[:<check>] <reason>", Prefix, Prefix)
	case !ok:
		return fmt.Sprintf("%s names unknown analyzer %q (known: %s)", Prefix, d.Analyzer, strings.Join(slices.Sorted(maps.Keys(known)), ", "))
	case d.Check != "" && len(checks) == 0:
		return fmt.Sprintf("%s %s: %s has no checks to name; write %s %s <reason>", Prefix, d.target(), analyzer, Prefix, analyzer)
	case d.Check != "" && !slices.Contains(checks, d.Check):
		return fmt.Sprintf("%s %s: %s has no check %q (checks: %s)", Prefix, d.target(), analyzer, d.Check, strings.Join(checks, ", "))
	case d.Reason == "":
		return fmt.Sprintf("%s %s needs a reason after it, saying why the finding is wrong here; without one it suppresses nothing",
			Prefix, d.target())
	}
	return ""
}
