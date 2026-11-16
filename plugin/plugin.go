// Package plugin registers gosloplint as a golangci-lint module plugin named
// "gosloplint". Import it from a .custom-gcl.yml and enable it in
// .golangci.yml as a linter of type "module"; Settings is its schema.
package plugin

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/golangci/plugin-module-register/register"
	"golang.org/x/tools/go/analysis"

	"github.com/chrisrob11/gosloplint"
	"github.com/chrisrob11/gosloplint/emptyid"
	"github.com/chrisrob11/gosloplint/internal/camel"
	"github.com/chrisrob11/gosloplint/vocab"
)

// Name is the name the plugin registers under, and the linter name to enable.
const Name = "gosloplint"

func init() {
	register.Plugin(Name, New)
}

// Settings is the plugin's settings block in .golangci.yml. Every analyzer
// runs unless named in Disable.
type Settings struct {
	Disable []string        `json:"disable"`
	Emptyid EmptyidSettings `json:"emptyid"`
	Vocab   VocabSettings   `json:"vocab"`
}

// EmptyidSettings mirrors the emptyid -suffixes flag.
type EmptyidSettings struct {
	// Suffixes replaces emptyid.DefaultSuffixes when set.
	Suffixes []string `json:"suffixes"`
}

// VocabSettings mirrors the vocab -words flag.
type VocabSettings struct {
	Words []string `json:"words"`
}

type gosloplintPlugin struct {
	analyzers []*analysis.Analyzer
}

// New decodes and checks the settings and builds the enabled analyzers, so a
// bad configuration fails when golangci-lint loads it.
func New(raw any) (register.LinterPlugin, error) {
	s, err := register.DecodeSettings[Settings](raw)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", Name, err)
	}
	analyzers, err := build(s)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", Name, err)
	}
	return &gosloplintPlugin{analyzers: analyzers}, nil
}

func (p *gosloplintPlugin) BuildAnalyzers() ([]*analysis.Analyzer, error) {
	return p.analyzers, nil
}

func (p *gosloplintPlugin) GetLoadMode() string {
	return register.LoadModeTypesInfo
}

func build(s Settings) ([]*analysis.Analyzer, error) {
	all := gosloplint.Analyzers()
	var names []string
	for _, a := range all {
		names = append(names, a.Name)
	}
	disabled := map[string]bool{}
	for _, n := range s.Disable {
		if !slices.Contains(names, n) {
			return nil, fmt.Errorf("disable: unknown analyzer %q (known: %s)", n, strings.Join(names, ", "))
		}
		disabled[n] = true
	}
	if err := checkWords("emptyid.suffixes", s.Emptyid.Suffixes, true); err != nil {
		return nil, err
	}
	if err := checkWords("vocab.words", s.Vocab.Words, false); err != nil {
		return nil, err
	}
	if disabled["emptyid"] && len(s.Emptyid.Suffixes) > 0 {
		return nil, errors.New("emptyid.suffixes is set but emptyid is disabled")
	}
	if disabled["vocab"] && len(s.Vocab.Words) > 0 {
		return nil, errors.New("vocab.words is set but vocab is disabled")
	}

	var out []*analysis.Analyzer
	for _, a := range all {
		switch {
		case disabled[a.Name]:
			continue
		case a.Name == "emptyid":
			a = emptyid.New(emptyid.Settings{Suffixes: s.Emptyid.Suffixes})
		case a.Name == "vocab":
			a = vocab.New(vocab.Settings{Words: s.Vocab.Words})
		}
		out = append(out, a)
	}
	return out, nil
}

// checkWords rejects a blank entry, and, when single is set, an entry of more
// than one name word: emptyid compares one word, so "user_id" never matches.
func checkWords(field string, words []string, single bool) error {
	for i, w := range words {
		parts := camel.Words(strings.ReplaceAll(w, " ", "_"))
		switch {
		case len(parts) == 0:
			return fmt.Errorf("%s[%d]: empty word", field, i)
		case single && len(parts) > 1:
			return fmt.Errorf("%s[%d]: %q is %d words; a suffix is one name word", field, i, w, len(parts))
		}
	}
	return nil
}
