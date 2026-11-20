// Package narration defines an Analyzer that reports comments narrating
// how the code came to be instead of saying what it is.
//
// Checks:
//
//   - history: a comment that tells the story of a change: "this now
//     uses", "was changed to", "we replaced", "previously, it", "after the
//     refactor", "per review", "fixes the bug where", "renamed from",
//     "added in #123". The phrases are anchored as narration, a subject
//     the code is or a change verb, so "the previously stored value" or
//     "is no longer held" is not one. One finding per comment block.
//   - warnsign: a second warning marker (⚠, WARNING:, IMPORTANT:, NOTE:,
//     CAUTION:, NB:) in one declaration, its doc and body together. One
//     marker can say "a reader must not miss this"; a marker on every
//     comment says nothing, so every marker after a declaration's first
//     is reported.
//
// Only comments are read, never string literals. Go directive lines
// (//go:build, //nolint:x, //gosloplint:ignore) are not prose, and a
// comment above the package clause that is not its doc (a license
// header, a build constraint), or any block naming a copyright or an
// SPDX license, is skipped. Generated files are skipped. Test files are
// read like any other: the phrases need a change verb, so a test's
// "regression for #123" or "the bug where" is not narration.
package narration

import (
	"go/ast"
	"go/token"
	"regexp"
	"strings"

	"golang.org/x/tools/go/analysis"

	"github.com/chrisrob11/gosloplint/internal/listflag"
)

// Analyzer reports comments that narrate history. Its settings come from
// the -disable flag; New builds one with explicit settings.
var Analyzer = newAnalyzer(&Settings{}, true)

// New returns a narration Analyzer configured by s rather than by flags. It
// panics if s does not pass Validate.
func New(s Settings) *analysis.Analyzer {
	if err := s.Validate(); err != nil {
		panic("narration.New: " + err.Error())
	}
	return newAnalyzer(&s, false)
}

func newAnalyzer(s *Settings, withFlags bool) *analysis.Analyzer {
	phrases := historyPhrases()
	a := &analysis.Analyzer{
		Name: "narration",
		Doc:  "narration: report comments that narrate a change's history instead of describing the code\n\nChecks: history (\"now uses\", \"was changed to\", \"per review\", ...) and warnsign (a second ⚠, WARNING:, NOTE: ... marker in one declaration). -disable turns checks off.",
		URL:  "https://github.com/chrisrob11/gosloplint#narration",
	}
	a.Run = func(pass *analysis.Pass) (any, error) {
		if err := s.Validate(); err != nil {
			return nil, err
		}
		for _, f := range pass.Files {
			if ast.IsGenerated(f) {
				continue
			}
			if s.enabled(History) {
				history(pass, f, phrases)
			}
			if s.enabled(Warnsign) {
				warnsign(pass, f)
			}
		}
		return nil, nil
	}
	if withFlags {
		a.Flags.Var(listflag.List{Items: &s.Disable}, "disable", "comma-separated checks not to run: history, warnsign")
	}
	return a
}

// line is one line of a comment block's prose.
type line struct {
	pos  token.Pos
	text string
}

// prose returns the lines of cg a reader reads as prose, without comment
// markers and Go directive lines; nil for a header block.
func prose(f *ast.File, cg *ast.CommentGroup) []line {
	if cg != f.Doc && cg.End() < f.Package {
		return nil
	}
	var out []line
	for _, c := range cg.List {
		if isDirective(c.Text) {
			continue
		}
		if body, ok := strings.CutPrefix(c.Text, "//"); ok {
			out = append(out, line{c.Pos(), body})
			continue
		}
		body := strings.TrimSuffix(strings.TrimPrefix(c.Text, "/*"), "*/")
		for _, l := range strings.Split(body, "\n") {
			out = append(out, line{c.Pos(), l})
		}
	}
	for _, l := range out {
		low := strings.ToLower(l.text)
		if strings.Contains(low, "copyright") || strings.Contains(low, "spdx-license-identifier") {
			return nil
		}
	}
	return out
}

// isDirective reports whether text is a Go directive comment, //word:rest
// with word lower-case letters and digits and no space after the colon, or
// //nolint and the text after it.
func isDirective(text string) bool {
	body, ok := strings.CutPrefix(text, "//")
	if !ok {
		return false
	}
	if strings.HasPrefix(body, "nolint ") {
		return true
	}
	i := strings.IndexFunc(body, func(r rune) bool { return (r < 'a' || r > 'z') && (r < '0' || r > '9') })
	return i > 0 && body[i] == ':' && i+1 < len(body) && body[i+1] != ' '
}

// historyPhrases returns the patterns of narration, each matched against
// a comment block's prose joined into one line, case aside. A pattern's
// first group, when it has one, is the phrase a finding quotes.
func historyPhrases() []*regexp.Regexp {
	const (
		noun    = `(?:function|func|method|type|struct|package|code|test|helper|check|logic|implementation|analyzer|handler|wrapper)`
		subject = `(?:this|this ` + noun + `|the ` + noun + `)`
		change  = `(?:changed|switched|updated|converted|renamed|moved|replaced|migrated)`
		edit    = `(?:added|changed|removed|fixed|introduced|moved|renamed|replaced|reverted|updated)`
		ref     = `(?:#|\bpr ?#?|\bissue ?#?|\bpull request ?#?)\d+`
		rework  = `(?:refactor|refactoring|rewrite|rework|cleanup|clean-up)`
		after   = `(?:$|[.,;:)!]| (?:this|it|we|the|that|there|nothing|every|all|each|only|no)\b)`
	)
	patterns := []string{
		`\b` + subject + ` (?:was|were|has been|have been|got) ` + change + ` (?:to|from|into|with|by|here)\b`,
		`\b(?:was|were|has been|have been|got) (?:refactored|rewritten|reworked)\b`,
		`\b(?:this|this ` + noun + `) (?:now|no longer|used to)\b`,
		`\b(?:we|it) used to (?:be|have|return|call|take|use|live|do|work|panic|read|write|store|keep|log|run|check|need)\b`,
		`\bnow (?:uses|delegates|lives|relies on|goes through)\b`,
		`\bwe (?:replaced|refactored|rewrote|reworked|renamed|switched (?:from|to|over)|used to|changed (?:this|it|the ` + noun + `) to)\b`,
		`(?:^|[.!?;:] )(previously(?:,| (?:this|it|we|the|these|they|that)\b))`,
		`(\b(?:after|before|since|during|in|with) (?:the|this|that|a|my|our) ` + rework + `)` + after,
		`\b(?:per|addressing|addresses|addressed) (?:the |a )?(?:code |pr )?review\b`,
		`\b(?:requested|suggested|asked for) (?:in|during|by) (?:the |a )?(?:code |pr )?review\b`,
		`\breview feedback\b`,
		`\b(?:fix|fixes|fixed) (?:the|a) (?:bug|issue|regression) (?:where|in which|that|when|with)\b`,
		`\brenamed from\b`,
		`\bmoved here from\b`,
		`\bformerly (?:called|named|known as)\b`,
		`\b` + edit + ` (?:in|by|for|since|after|with) ` + ref + `\b`,
		ref + `\)?:? ` + edit + `\b`,
		`\b(?:old|previous|earlier|original) (?:implementation|approach|code path|version of this)\b`,
		`\b(?:added|introduced) to (?:fix|address|work around|resolve)\b`,
	}
	out := make([]*regexp.Regexp, len(patterns))
	for i, p := range patterns {
		out[i] = regexp.MustCompile(`(?i)` + p)
	}
	return out
}
