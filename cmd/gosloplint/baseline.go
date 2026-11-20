package main

import (
	"errors"
	"flag"
	"fmt"
	"io"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/checker"
	"golang.org/x/tools/go/packages"

	"github.com/chrisrob11/gosloplint/dupl"
	"github.com/chrisrob11/gosloplint/internal/baseline"
	"github.com/chrisrob11/gosloplint/internal/clone"
)

const baselineUsage = `usage: gosloplint baseline [-o file] [analyzer flags] [packages]

Runs every analyzer over the packages named (default ./...) and their tests,
and writes each package's finding count per analyzer check to the file, for
-baseline to hold, with the copies across packages gosloplint dupl reports,
configured by the -dupl flags, as dupl:cross. Pass the analyzer flags the
lint runs with. Flags:
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
	cfg, err := duplConfig(analyzers)
	if err != nil {
		return fail(err)
	}
	cross := map[string]int{}
	for _, c := range duplicates(pkgs, cfg) {
		if c.key != dupl.Analyzer.Name {
			cross[c.pkg]++
		}
	}
	for pkg, n := range cross {
		counts.Add(pkg, baseline.Key(dupl.Analyzer.Name, analysis.Diagnostic{Category: dupl.Cross}), n)
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

// duplConfig is the settings the dupl analyzer among analyzers takes from
// its flags, the defaults without one, so the copies across packages are
// found as the analyzer finds those within one.
func duplConfig(analyzers []*analysis.Analyzer) (clone.Config, error) {
	var cfg clone.Config
	fs := flag.NewFlagSet(dupl.Analyzer.Name, flag.ContinueOnError)
	clone.BindFlags(fs, &cfg)
	var errs []error
	for _, a := range analyzers {
		if a.Name == dupl.Analyzer.Name {
			a.Flags.VisitAll(func(f *flag.Flag) { errs = append(errs, fs.Set(f.Name, f.Value.String())) })
		}
	}
	return cfg, errors.Join(errs...)
}
