package narration

import (
	"go/ast"
	"regexp"
	"strings"

	"golang.org/x/tools/go/analysis"
)

// history reports each comment block of f whose prose matches a phrase, at
// the line the earliest match starts on. A paragraph from a line opening
// with Deprecated: to the next blank line is godoc's deprecation notice,
// which says what changed by design, so it is not read.
func history(pass *analysis.Pass, f *ast.File, phrases []*regexp.Regexp) {
	for _, cg := range f.Comments {
		lines := prose(f, cg)
		var b strings.Builder
		starts := make([]int, len(lines))
		deprecated := false
		for i, l := range lines {
			starts[i] = b.Len()
			words := strings.Fields(l.text)
			deprecated = len(words) > 0 && (deprecated || words[0] == "Deprecated:")
			if len(words) > 0 && !deprecated {
				b.WriteString(strings.Join(words, " "))
				b.WriteByte(' ')
			}
		}
		text := b.String()
		match := unquoted(text)
		first := []int{len(text), 0}
		for _, re := range phrases {
			if m := re.FindStringSubmatchIndex(match); m != nil && m[0] < first[0] {
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
			if s <= quote[0] {
				at = i
			}
		}
		pass.Report(analysis.Diagnostic{Pos: lines[at].pos, Category: History,
			Message: History + `: "` + text[quote[0]:quote[1]] + `" narrates how the code changed: say what it does and why; its history belongs in the commit message`})
	}
}

// unquoted returns text with the inside of each closed "…", `…` and “…”
// span blanked to spaces of the same byte length, so a quoted example such as
// a reply format is not read as narration and match offsets still index text.
// An opening quote with no close is left as it is.
func unquoted(text string) string {
	b := []byte(text)
	for _, q := range [][2]string{{`"`, `"`}, {"`", "`"}, {"“", "”"}} {
		for from := 0; ; {
			open := strings.Index(text[from:], q[0])
			if open < 0 {
				break
			}
			inner := from + open + len(q[0])
			end := strings.Index(text[inner:], q[1])
			if end < 0 {
				break
			}
			for i := inner; i < inner+end; i++ {
				b[i] = ' '
			}
			from = inner + end + len(q[1])
		}
	}
	return string(b)
}
