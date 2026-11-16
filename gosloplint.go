// Package gosloplint collects the gosloplint analyzers, each a standard
// golang.org/x/tools/go/analysis Analyzer.
package gosloplint

import "golang.org/x/tools/go/analysis"

// Analyzers returns every gosloplint analyzer.
func Analyzers() []*analysis.Analyzer {
	return []*analysis.Analyzer{}
}
