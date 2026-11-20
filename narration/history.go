package narration

import (
	"go/ast"
	"regexp"
	"strings"

	"golang.org/x/tools/go/analysis"
)

// history reports each comment block of f whose prose matches a phrase, at
// the line the earliest match starts on.
func history(pass *analysis.Pass, f *ast.File, phrases []*regexp.Regexp) {
	for _, cg := range f.Comments {
		lines := prose(f, cg)
		var b strings.Builder
		starts := make([]int, len(lines))
		for i, l := range lines {
			starts[i] = b.Len()
			b.WriteString(strings.Join(strings.Fields(l.text), " "))
			b.WriteByte(' ')
		}
		text := b.String()
		first := []int{len(text), 0}
		for _, re := range phrases {
			if m := re.FindStringSubmatchIndex(text); m != nil && m[0] < first[0] {
				first = m
			}
		}
		if first[0] == len(text) {
			continue
		}
		quote := first[:2]
		if len(first) > 2 {
			quote = first[2:4]
		}
		at := 0
		for i, s := range starts {
			if s <= first[0] {
				at = i
			}
		}
		pass.Report(analysis.Diagnostic{Pos: lines[at].pos, Category: History,
			Message: History + `: "` + text[quote[0]:quote[1]] + `" narrates how the code changed: say what it does and why; its history belongs in the commit message`})
	}
}
