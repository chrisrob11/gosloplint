package comment

import (
	"go/ast"
	"go/token"
	"strconv"
	"strings"

	"golang.org/x/tools/go/analysis"
)

// fileLines is one file's count of comment lines and code lines.
type fileLines struct {
	f             *ast.File
	comment, code int
}

// density reports the package when its non-test files are more than
// ceiling percent comment, unless they hold fewer than minLines non-blank
// lines. A file the pass cannot read is left out.
func density(pass *analysis.Pass, ceiling, minLines int) {
	var files []fileLines
	for _, f := range pass.Files {
		name := pass.Fset.Position(f.Package).Filename
		if ast.IsGenerated(f) || strings.HasSuffix(name, "_test.go") {
			continue
		}
		src, err := pass.ReadFile(name)
		if err != nil {
			continue
		}
		files = append(files, measure(pass.Fset, f, src))
	}
	var comment, code int
	for _, fl := range files {
		comment += fl.comment
		code += fl.code
	}
	total := comment + code
	if total == 0 || total < minLines || comment*100 <= ceiling*total {
		return
	}
	at := densest(files)
	pass.Reportf(at.f.Package, "%s: package %s is %s comment (%d comment lines to %d of code), ceiling %d%%: "+
		"cut the comments that restate the code; this file has the most, %d lines",
		Density, pass.Pkg.Name(), percent(comment, total), comment, code, ceiling, at.comment)
}

// densest is the file with the most comment lines, preferring one that
// declares something over a doc.go, and the first on a tie.
func densest(files []fileLines) fileLines {
	best := files[0]
	for _, fl := range files[1:] {
		hasDecls, bestDecls := len(fl.f.Decls) > 0, len(best.f.Decls) > 0
		if hasDecls && !bestDecls || hasDecls == bestDecls && fl.comment > best.comment {
			best = fl
		}
	}
	return best
}

// percent renders part/whole as a percentage to one decimal place.
func percent(part, whole int) string {
	perMille := part * 1000 / whole
	out := strconv.Itoa(perMille / 10)
	if perMille%10 != 0 {
		out += "." + strconv.Itoa(perMille%10)
	}
	return out + "%"
}

// measure counts f's comment lines and code lines. A comment on its own line
// is a comment line, and so is each further line of a /* */ block; a line
// with code before its comment is code. Directive lines are neither, and
// blank lines are in no count.
func measure(fset *token.FileSet, f *ast.File, src []byte) fileLines {
	lines := strings.Split(string(src), "\n")
	text := func(n int) string {
		if n < 1 || n > len(lines) {
			return ""
		}
		return lines[n-1]
	}
	commentLine, directiveLine := map[int]bool{}, map[int]bool{}
	for _, cg := range f.Comments {
		for _, c := range cg.List {
			start := fset.Position(c.Pos())
			end := fset.Position(c.End()).Line
			own := start.Column >= 1 && start.Column-1 <= len(text(start.Line)) &&
				strings.TrimSpace(text(start.Line)[:start.Column-1]) == ""
			switch {
			case own && directive(c.Text):
				directiveLine[start.Line] = true
			case own:
				commentLine[start.Line] = true
			}
			for n := start.Line + 1; n <= end; n++ {
				commentLine[n] = true
			}
		}
	}
	out := fileLines{f: f}
	for n := 1; n <= len(lines); n++ {
		switch {
		case commentLine[n]:
			out.comment++
		case directiveLine[n], strings.TrimSpace(text(n)) == "":
		default:
			out.code++
		}
	}
	return out
}
