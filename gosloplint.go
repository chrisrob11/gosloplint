// Package gosloplint collects the gosloplint analyzers, each a standard
// golang.org/x/tools/go/analysis Analyzer.
package gosloplint

import (
	"slices"

	"golang.org/x/tools/go/analysis"

	"github.com/chrisrob11/gosloplint/cannotfail"
	"github.com/chrisrob11/gosloplint/comment"
	"github.com/chrisrob11/gosloplint/ctxthread"
	"github.com/chrisrob11/gosloplint/dupl"
	"github.com/chrisrob11/gosloplint/emptyid"
	"github.com/chrisrob11/gosloplint/emptyskip"
	"github.com/chrisrob11/gosloplint/fileplace"
	"github.com/chrisrob11/gosloplint/selfassert"
	"github.com/chrisrob11/gosloplint/steer"
	"github.com/chrisrob11/gosloplint/testhelper"
	"github.com/chrisrob11/gosloplint/vocab"
)

// Analyzers returns every gosloplint analyzer.
func Analyzers() []*analysis.Analyzer {
	return []*analysis.Analyzer{
		cannotfail.Analyzer,
		comment.Analyzer,
		ctxthread.Analyzer,
		dupl.Analyzer,
		emptyid.Analyzer,
		emptyskip.Analyzer,
		fileplace.Analyzer,
		selfassert.Analyzer,
		steer.Analyzer,
		testhelper.Analyzer,
		vocab.Analyzer,
	}
}

// Checks maps each analyzer's name to its checks, the categories of its
// findings that a //gosloplint:ignore directive may name; nil for an
// analyzer of one check.
func Checks() map[string][]string {
	out := map[string][]string{}
	for _, a := range Analyzers() {
		out[a.Name] = nil
	}
	out[comment.Analyzer.Name] = slices.Clone(comment.Checks)
	out[fileplace.Analyzer.Name] = slices.Clone(fileplace.Rules)
	out[steer.Analyzer.Name] = slices.Clone(steer.Checks)
	return out
}
