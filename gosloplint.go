// Package gosloplint collects the gosloplint analyzers, each a standard
// golang.org/x/tools/go/analysis Analyzer.
package gosloplint

import (
	"golang.org/x/tools/go/analysis"

	"github.com/chrisrob11/gosloplint/ctxthread"
	"github.com/chrisrob11/gosloplint/emptyid"
	"github.com/chrisrob11/gosloplint/emptyskip"
	"github.com/chrisrob11/gosloplint/selfassert"
	"github.com/chrisrob11/gosloplint/testhelper"
	"github.com/chrisrob11/gosloplint/vocab"
)

// Analyzers returns every gosloplint analyzer.
func Analyzers() []*analysis.Analyzer {
	return []*analysis.Analyzer{
		ctxthread.Analyzer,
		emptyid.Analyzer,
		emptyskip.Analyzer,
		selfassert.Analyzer,
		testhelper.Analyzer,
		vocab.Analyzer,
	}
}
