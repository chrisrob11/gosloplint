package main

import (
	"flag"
	"fmt"
	"io"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/checker"
	"golang.org/x/tools/go/packages"

	"github.com/chrisrob11/gosloplint/internal/baseline"
)

const baselineUsage = `usage: gosloplint baseline [-o file] [analyzer flags] [packages]

Runs every analyzer over the packages named (default ./...) and their tests,
and writes each package's finding count per analyzer check to the file, for
-baseline to hold. Pass the analyzer flags the lint runs with. Flags:
`

// runBaseline is `gosloplint baseline`: it records the findings of
// analyzers, configured by their prefixed flags, as a baseline file.
func runBaseline(analyzers []*analysis.Analyzer, args []string, stdout, stderr io.Writer) int {
	fs, fail := subcommand("baseline", baselineUsage, stderr)
	out := fs.String("o", baseline.DefaultFile, "the baseline file to write")
	for _, a := range analyzers {
		a.Flags.VisitAll(func(f *flag.Flag) {
			fs.Var(f.Value, a.Name+"."+f.Name, f.Usage)
		})
	}
	if err := fs.Parse(args); err != nil {
		return 2
	}
	pkgs, err := load(packages.LoadAllSyntax, true, fs.Args())
	if err != nil {
		return fail(err)
	}
	graph, err := checker.Analyze(analyzers, pkgs, nil)
	if err != nil {
		return fail(err)
	}
	counts, total := baseline.Counts{}, 0
	for _, act := range graph.Roots {
		if act.Err != nil {
			return fail(fmt.Errorf("%s: %w", act, act.Err))
		}
		byKey := map[string]int{}
		for _, d := range act.Diagnostics {
			byKey[baseline.Key(act.Analyzer.Name, d)]++
		}
		for k, n := range byKey {
			counts.Add(act.Package.PkgPath, k, n)
		}
	}
	for _, keys := range counts {
		for _, n := range keys {
			total += n
		}
	}
	if err := counts.Write(*out); err != nil {
		return fail(err)
	}
	_, _ = fmt.Fprintf(stdout, "gosloplint baseline: %s records %d finding(s) in %d package(s)\n", *out, total, len(counts))
	return 0
}
