// Package vocab defines an Analyzer that reports identifiers using a banned
// word.
//
// A rename only finishes if something counts what is left. Configure the
// words a codebase has retired, or never wants, with -words, and vocab
// reports every identifier that still uses one, declarations and uses alike.
// With no words configured it reports nothing.
//
// Matching is by whole camelCase or snake_case word, case-insensitively, so
// the banned word "age" matches userAge and AGE_LIMIT but not image, manager
// or pageSize. A word with several parts, written "legacy client" or
// "LegacyClient", matches those words in sequence. Generated files are
// skipped; test files are not, since a rename is not done while its tests
// still use the old name.
package vocab

import (
	"go/ast"
	"strings"

	"golang.org/x/tools/go/analysis"

	"github.com/chrisrob11/gosloplint/internal/camel"
)

// Analyzer reports identifiers that use a word set by -words.
var Analyzer = &analysis.Analyzer{
	Name: "vocab",
	Doc:  "vocab: report identifiers that use a banned word\n\nReports every identifier containing one of the words set by -words, matched by whole camelCase or snake_case word. With no words configured it reports nothing.",
	URL:  "https://github.com/chrisrob11/gosloplint#vocab",
	Run:  run,
}

var words wordList

func init() {
	Analyzer.Flags.Var(&words, "words",
		`comma-separated banned words; a word may have parts ("legacy client" or LegacyClient)`)
}

// wordList is a comma-separated flag value.
type wordList []string

func (w *wordList) String() string { return strings.Join(*w, ",") }

func (w *wordList) Set(v string) error {
	*w = nil
	for _, word := range strings.Split(v, ",") {
		if word = strings.TrimSpace(word); word != "" {
			*w = append(*w, word)
		}
	}
	return nil
}

func run(pass *analysis.Pass) (any, error) {
	if len(words) == 0 {
		return nil, nil
	}
	for _, file := range pass.Files {
		if ast.IsGenerated(file) {
			continue
		}
		ast.Inspect(file, func(n ast.Node) bool {
			id, ok := n.(*ast.Ident)
			if !ok || id.Name == "_" {
				return true
			}
			for _, w := range words {
				if !camel.HasWord(id.Name, w) {
					continue
				}
				verb := "uses"
				if pass.TypesInfo.Defs[id] != nil {
					verb = "declares"
				}
				pass.Reportf(id.Pos(), "%s %s, which contains the banned word %q", verb, id.Name, w)
				break
			}
			return true
		})
	}
	return nil, nil
}
