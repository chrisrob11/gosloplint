// Package dupl defines an Analyzer that reports copied code within one
// package: two runs of statements with the same syntax tree shape, names and
// literal values aside, long enough that one function both call was the
// better choice.
//
// A statement's shape keeps its node kinds, operators and literal kinds, so
// a copy with renamed variables and changed constants still matches, while
// one operator changed does not. Composite literals and a switch's clauses
// are data, not runs. Each clone is reported once per pair of declarations,
// at the later occurrence, naming the lines of the earlier one. Generated
// files are skipped, and so are _test.go files unless Tests is set.
//
// Copies across packages are invisible to an analysis pass, which sees one
// package; `gosloplint dupl ./...` reports those with the same detector.
package dupl

import (
	"path/filepath"

	"golang.org/x/tools/go/analysis"

	"github.com/chrisrob11/gosloplint/internal/clone"
)

// Directive is the line that, followed by a reason, in a function's doc
// comment exempts the function. Without a reason it exempts nothing.
const Directive = clone.Directive

// Cross is the check of a copy across packages: `gosloplint dupl` reports
// it, never the Analyzer, so `//gosloplint:ignore dupl:cross` suppresses it
// at the later copy and its baseline key is dupl:cross.
const Cross = "cross"

// The defaults a zero Settings field takes.
const (
	DefaultMinNodes      = clone.DefaultMinNodes
	DefaultMinStatements = clone.DefaultMinStatements
)

// Settings configures an Analyzer built by New. The zero value is the
// defaults.
type Settings struct {
	// MinNodes is the size a copied run must reach, in syntax nodes. Zero
	// means DefaultMinNodes.
	MinNodes int
	// MinStatements is the length a copied run must reach, in statements.
	// Zero means DefaultMinStatements.
	MinStatements int
	// Tests includes _test.go files, off by default: test tables and setup
	// repeat by design.
	Tests bool
	// Exempt are patterns of packages and files not read: an import path
	// pattern ("..." matches any string) or a file base name glob
	// ("*_cases.go").
	Exempt []string
}

// Validate reports a negative minimum and a malformed exemption.
func (s Settings) Validate() error { return clone.Config(s).Validate() }

// Analyzer reports copied statement runs. Its settings come from flags; New
// builds one with explicit settings instead.
var Analyzer = newAnalyzer(&Settings{}, true)

// New returns a dupl Analyzer configured by s rather than by flags. It panics
// if s does not pass Validate.
func New(s Settings) *analysis.Analyzer {
	if err := s.Validate(); err != nil {
		panic("dupl.New: " + err.Error())
	}
	return newAnalyzer(&s, false)
}

func newAnalyzer(s *Settings, withFlags bool) *analysis.Analyzer {
	a := &analysis.Analyzer{
		Name: "dupl",
		Doc:  "dupl: report copied statement runs within a package\n\nReports two runs of statements with the same syntax tree shape, names and literal values aside, of at least 60 syntax nodes by default. `gosloplint dupl` finds copies across packages.",
		URL:  "https://github.com/chrisrob11/gosloplint#dupl",
	}
	a.Run = func(pass *analysis.Pass) (any, error) {
		if err := s.Validate(); err != nil {
			return nil, err
		}
		run(pass, clone.Config(*s))
		return nil, nil
	}
	if withFlags {
		clone.BindFlags(&a.Flags, (*clone.Config)(s))
	}
	return a
}

func run(pass *analysis.Pass, cfg clone.Config) {
	d := clone.NewDetector(pass.Fset, cfg)
	for _, f := range pass.Files {
		if cfg.Includes(pass.Pkg.Path(), pass.Fset.File(f.Pos()).Name()) {
			d.Add(f)
		}
	}
	for _, c := range d.Clones() {
		pass.Report(analysis.Diagnostic{Pos: c.B.Pos, Message: c.Message(filepath.Base(c.A.File))})
	}
}
