package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/packages"

	"github.com/chrisrob11/gosloplint/internal/clone"
)

const duplUsage = `usage: gosloplint dupl [flags] [packages]

Reports copied statement runs across all the packages named (default ./...),
the copies the per-package dupl analyzer cannot see. It exits 1 on findings,
0 when clean and 2 on an error. Flags:
`

// runDupl is `gosloplint dupl`: the dupl analyzer's detector over every
// package at once.
func runDupl(args []string, stdout, stderr io.Writer) int {
	fs, fail := subcommand("dupl", duplUsage, stderr)
	var cfg clone.Config
	clone.BindFlags(fs, &cfg)
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if err := cfg.Validate(); err != nil {
		return fail(err)
	}
	pkgs, err := load(packages.NeedName|packages.NeedFiles|packages.NeedSyntax, cfg.Tests, fs.Args())
	if err != nil {
		return fail(err)
	}
	d := clone.NewDetector(pkgs[0].Fset, cfg)
	seen := map[string]bool{}
	for _, p := range pkgs {
		for _, f := range p.Syntax {
			name := p.Fset.File(f.Pos()).Name()
			if !seen[name] && cfg.Includes(p.PkgPath, name) {
				seen[name] = true
				d.Add(f)
			}
		}
	}
	clones := d.Clones()
	var b strings.Builder
	for _, c := range clones {
		pos := pkgs[0].Fset.Position(c.B.Pos)
		fmt.Fprintf(&b, "%s:%d:%d: %s\n", rel(pos.Filename), pos.Line, pos.Column, c.Message(rel(c.A.File)))
	}
	if _, err := io.WriteString(stdout, b.String()); err != nil {
		return fail(err)
	}
	if len(clones) > 0 {
		return 1
	}
	return 0
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
