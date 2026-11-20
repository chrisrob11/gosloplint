package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/packages"

	"github.com/chrisrob11/gosloplint/dupl"
	"github.com/chrisrob11/gosloplint/internal/baseline"
	"github.com/chrisrob11/gosloplint/internal/clone"
	"github.com/chrisrob11/gosloplint/internal/ignore"
)

const duplUsage = `usage: gosloplint dupl [flags] [packages]

Reports copied statement runs across all the packages named (default ./...),
the copies the per-package dupl analyzer cannot see. With -baseline, a
package's copies are held to its dupl:cross count, and those within it to
its dupl count. It exits 1 on findings, 0 when clean and 2 on an error.
Flags:
`

// runDupl is `gosloplint dupl`: the dupl analyzer's detector over every
// package at once.
func runDupl(args []string, stdout, stderr io.Writer) int {
	fs, fail := subcommand("dupl", duplUsage, stderr)
	var cfg clone.Config
	clone.BindFlags(fs, &cfg)
	file := fs.String("baseline", "", "a file written by gosloplint baseline: report a package's copies only once they outnumber its count")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if err := cfg.Validate(); err != nil {
		return fail(err)
	}
	var counts baseline.Counts
	if *file != "" {
		var err error
		if counts, err = baseline.Read(*file); err != nil {
			return fail(err)
		}
	}
	pkgs, err := load(packages.NeedName|packages.NeedFiles|packages.NeedSyntax, cfg.Tests, fs.Args())
	if err != nil {
		return fail(err)
	}
	found := duplicates(pkgs, cfg)
	notes, shown := make([]string, len(found)), make([]bool, len(found))
	byPkg := map[string][]int{}
	for i, c := range found {
		byPkg[c.pkg] = append(byPkg[c.pkg], i)
	}
	for pkg, idx := range byPkg {
		keys := make([]string, len(idx))
		for j, i := range idx {
			keys[j] = found[i].key
		}
		baseline.Filter(counts[pkg], keys, func(j int, note string) { shown[idx[j]], notes[idx[j]] = true, note })
	}
	var b strings.Builder
	reported := 0
	for i, c := range found {
		if !shown[i] {
			continue
		}
		reported++
		pos := pkgs[0].Fset.Position(c.B.Pos)
		fmt.Fprintf(&b, "%s:%d:%d: %s%s\n", rel(pos.Filename), pos.Line, pos.Column, c.Message(rel(c.A.File)), notes[i])
	}
	if _, err := io.WriteString(stdout, b.String()); err != nil {
		return fail(err)
	}
	if reported > 0 {
		return 1
	}
	return 0
}

// duplicate is a copy `gosloplint dupl` finds: pkg is the import path of
// the package holding the later copy, where it is reported and counted,
// and key its baseline Key: dupl:cross when the earlier copy is in another
// package, else dupl, the count the dupl analyzer records for it.
type duplicate struct {
	clone.Clone
	pkg, key string
}

// duplicates is every copy in pkgs under cfg, in the detector's order, but
// those a //gosloplint:ignore dupl directive (dupl:cross for a copy across
// packages) suppresses at the later copy.
func duplicates(pkgs []*packages.Package, cfg clone.Config) []duplicate {
	fset := pkgs[0].Fset
	d := clone.NewDetector(fset, cfg)
	pkgOf := map[string]string{}
	var dirs []*ignore.Directive
	for _, p := range pkgs {
		for _, f := range p.Syntax {
			name := fset.File(f.Pos()).Name()
			if _, seen := pkgOf[name]; seen || !cfg.Includes(p.PkgPath, name) {
				continue
			}
			pkgOf[name] = p.PkgPath
			d.Add(f)
			dirs = append(dirs, ignore.Parse(fset, f)...)
		}
	}
	var out []duplicate
	for _, c := range d.Clones() {
		check := ""
		if pkgOf[c.A.File] != pkgOf[c.B.File] {
			check = dupl.Cross
		}
		if !slices.ContainsFunc(dirs, func(dir *ignore.Directive) bool {
			return dir.Analyzer == dupl.Analyzer.Name && dir.Covers(fset, check, c.B.Pos)
		}) {
			out = append(out, duplicate{c, pkgOf[c.B.File], baseline.Key(dupl.Analyzer.Name, analysis.Diagnostic{Category: check})})
		}
	}
	return out
}

// load loads patterns (default ./...) with mode, and their tests if tests is
// set, leaving out each test binary's generated main package. Any package
// that fails to load is an error.
func load(mode packages.LoadMode, tests bool, patterns []string) ([]*packages.Package, error) {
	if len(patterns) == 0 {
		patterns = []string{"./..."}
	}
	all, err := packages.Load(&packages.Config{Mode: mode, Tests: tests}, patterns...)
	if err != nil {
		return nil, err
	}
	var errs []string
	packages.Visit(all, nil, func(p *packages.Package) {
		for _, e := range p.Errors {
			errs = append(errs, e.Error())
		}
	})
	if len(errs) > 0 {
		return nil, fmt.Errorf("loading packages:\n\t%s", strings.Join(errs, "\n\t"))
	}
	var out []*packages.Package
	for _, p := range all {
		if !strings.HasSuffix(p.ID, ".test") {
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no packages match %s", strings.Join(patterns, " "))
	}
	return out, nil
}

// rel is name relative to the working directory, when it is below it.
func rel(name string) string {
	wd, err := os.Getwd()
	if err != nil {
		return name
	}
	if r, err := filepath.Rel(wd, name); err == nil && !strings.HasPrefix(r, "..") {
		return r
	}
	return name
}
