// Package plugin registers gosloplint as a golangci-lint module plugin named
// "gosloplint". Import it from a .custom-gcl.yml and enable it in
// .golangci.yml as a linter of type "module"; Settings is its schema.
package plugin

import (
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strings"

	"github.com/golangci/plugin-module-register/register"
	"golang.org/x/tools/go/analysis"

	"github.com/chrisrob11/gosloplint"
	"github.com/chrisrob11/gosloplint/cannotfail"
	"github.com/chrisrob11/gosloplint/comment"
	"github.com/chrisrob11/gosloplint/dupl"
	"github.com/chrisrob11/gosloplint/emptyid"
	"github.com/chrisrob11/gosloplint/fileplace"
	"github.com/chrisrob11/gosloplint/internal/baseline"
	"github.com/chrisrob11/gosloplint/internal/camel"
	"github.com/chrisrob11/gosloplint/internal/ignore"
	"github.com/chrisrob11/gosloplint/narration"
	"github.com/chrisrob11/gosloplint/steer"
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
	// Baseline is a file written by `gosloplint baseline`: each analyzer
	// reports a package's findings of a check only once they outnumber its
	// count there.
	Baseline   string             `json:"baseline"`
	Disable    []string           `json:"disable"`
	Cannotfail CannotfailSettings `json:"cannotfail"`
	Comment    CommentSettings    `json:"comment"`
	Dupl       DuplSettings       `json:"dupl"`
	Emptyid    EmptyidSettings    `json:"emptyid"`
	Fileplace  FileplaceSettings  `json:"fileplace"`
	Narration  NarrationSettings  `json:"narration"`
	Steer      SteerSettings      `json:"steer"`
	Vocab      VocabSettings      `json:"vocab"`
}

// CannotfailSettings mirrors cannotfail.Settings.
type CannotfailSettings struct {
	Disable     []string `json:"disable"`
	Enable      []string `json:"enable"`
	AssertFuncs []string `json:"assert-funcs"`
}

// CommentSettings mirrors comment.Settings. A zero number is its default.
type CommentSettings struct {
	Disable         []string      `json:"disable"`
	Limits          CommentLimits `json:"limits"`
	Density         int           `json:"density"`
	DensityMinLines int           `json:"density-min-lines"`
}

// CommentLimits mirrors comment.Limits, in lines. A zero limit is its
// default.
type CommentLimits struct {
	Package int `json:"package"`
	File    int `json:"file"`
	Type    int `json:"type"`
	Func    int `json:"func"`
	Field   int `json:"field"`
	Value   int `json:"value"`
	Inline  int `json:"inline"`
}

// DuplSettings mirrors dupl.Settings. A zero number is its default.
type DuplSettings struct {
	MinNodes      int      `json:"min-nodes"`
	MinStatements int      `json:"min-statements"`
	Tests         bool     `json:"tests"`
	Exempt        []string `json:"exempt"`
}

// FileplaceSettings mirrors fileplace.Settings. A zero number is its
// default.
type FileplaceSettings struct {
	Disable   []string            `json:"disable"`
	Enable    []string            `json:"enable"`
	TinyLines int                 `json:"tiny-lines"`
	LongLines int                 `json:"long-lines"`
	GrabBag   []string            `json:"grab-bag"`
	Exempt    map[string][]string `json:"exempt"`
}

// NarrationSettings mirrors narration.Settings.
type NarrationSettings struct {
	Disable []string `json:"disable"`
}

// SteerSettings mirrors steer.Settings.
type SteerSettings struct {
	Disable           []string `json:"disable"`
	RecursionPackages []string `json:"recursion-packages"`
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
	cannotfailS, narrationS := cannotfail.Settings(s.Cannotfail), narration.Settings(s.Narration)
	commentS, duplS, fileplaceS, steerS := commentSettings(s.Comment), dupl.Settings(s.Dupl), fileplace.Settings(s.Fileplace), steer.Settings(s.Steer)
	for _, c := range []struct {
		name     string
		zero     bool
		validate func() error
	}{
		{"cannotfail", reflect.ValueOf(s.Cannotfail).IsZero(), cannotfailS.Validate},
		{"comment", reflect.ValueOf(s.Comment).IsZero(), commentS.Validate},
		{"dupl", reflect.ValueOf(s.Dupl).IsZero(), duplS.Validate},
		{"fileplace", reflect.ValueOf(s.Fileplace).IsZero(), fileplaceS.Validate},
		{"narration", reflect.ValueOf(s.Narration).IsZero(), narrationS.Validate},
		{"steer", reflect.ValueOf(s.Steer).IsZero(), steerS.Validate},
	} {
		if disabled[c.name] && !c.zero {
			return nil, fmt.Errorf("%s settings are set but %s is disabled", c.name, c.name)
		}
		if err := c.validate(); err != nil {
			return nil, fmt.Errorf("%s.%w", c.name, err)
		}
	}

	var out []*analysis.Analyzer
	for _, a := range all {
		switch {
		case disabled[a.Name]:
			continue
		case a.Name == "cannotfail":
			a = cannotfail.New(cannotfailS)
		case a.Name == "comment":
			a = comment.New(commentS)
		case a.Name == "dupl":
			a = dupl.New(duplS)
		case a.Name == "emptyid":
			a = emptyid.New(emptyid.Settings{Suffixes: s.Emptyid.Suffixes})
		case a.Name == "fileplace":
			a = fileplace.New(fileplaceS)
		case a.Name == "narration":
			a = narration.New(narrationS)
		case a.Name == "steer":
			a = steer.New(steerS)
		case a.Name == "vocab":
			a = vocab.New(vocab.Settings{Words: s.Vocab.Words})
		}
		out = append(out, a)
	}
	out = ignore.Wrap(out, gosloplint.Checks(), gosloplint.Elsewhere())
	if s.Baseline == "" {
		return out, nil
	}
	counts, err := baseline.Read(s.Baseline)
	if err != nil {
		return nil, fmt.Errorf("baseline: %w", err)
	}
	return baseline.Wrap(out, func() (baseline.Counts, error) { return counts, nil }), nil
}

func commentSettings(c CommentSettings) comment.Settings {
	return comment.Settings{
		Disable:         c.Disable,
		Limits:          comment.Limits(c.Limits),
		Density:         c.Density,
		DensityMinLines: c.DensityMinLines,
	}
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
