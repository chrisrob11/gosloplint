// Package baseline is the per-package finding ratchet: a file recording, for
// each package and analyzer check, how many findings it had, and a wrapper
// that keeps each analyzer quiet for a package and check until its count
// passes the recorded one.
package baseline

import (
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"slices"
	"sync"

	"golang.org/x/tools/go/analysis"
)

// DefaultFile is the file `gosloplint baseline` writes by default.
const DefaultFile = "gosloplint-baseline.json"

// Counts maps a package import path to its finding count per Key.
type Counts map[string]map[string]int

// Key names the check of a finding: the analyzer, and for an analyzer of
// several checks, ":" and the check, its Diagnostic.Category.
func Key(analyzer string, d analysis.Diagnostic) string {
	if d.Category == "" {
		return analyzer
	}
	return analyzer + ":" + d.Category
}

// Add records n findings of key in pkg, keeping the larger of n and what is
// recorded: a package loads once per build variant (with and without its
// test files), and the variant with the most findings is the one to hold.
func (c Counts) Add(pkg, key string, n int) {
	if n <= 0 || n <= c[pkg][key] {
		return
	}
	if c[pkg] == nil {
		c[pkg] = map[string]int{}
	}
	c[pkg][key] = n
}

// Read reads the baseline file at path.
func Read(path string) (Counts, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var c Counts
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	for _, pkg := range slices.Sorted(maps.Keys(c)) {
		for _, key := range slices.Sorted(maps.Keys(c[pkg])) {
			if c[pkg][key] < 0 {
				return nil, fmt.Errorf("%s: %s %s: count %d is negative", path, pkg, key, c[pkg][key])
			}
		}
	}
	return c, nil
}

// Write writes c to path as indented JSON, its keys sorted, so a baseline
// regenerated from the same findings is the same file.
func (c Counts) Write(path string) error {
	if c == nil {
		c = Counts{}
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o600)
}

// Wrap returns a copy of each analyzer whose findings are filtered by the
// counts load returns, read once, on the first pass. For each package and
// Key, a pass reports nothing while its findings number no more than the
// baseline's, and every finding, noting the baseline, once they number more.
// With nil counts nothing is filtered.
func Wrap(analyzers []*analysis.Analyzer, load func() (Counts, error)) []*analysis.Analyzer {
	f := &filter{load: load}
	out := make([]*analysis.Analyzer, len(analyzers))
	for i, a := range analyzers {
		w := *a
		w.Run = func(pass *analysis.Pass) (any, error) {
			counts, err := f.get()
			if err != nil {
				return nil, fmt.Errorf("baseline: %w", err)
			}
			var held []analysis.Diagnostic
			inner := *pass
			inner.Report = func(d analysis.Diagnostic) { held = append(held, d) }
			result, err := a.Run(&inner)
			report(pass, a.Name, counts[pass.Pkg.Path()], held)
			return result, err
		}
		out[i] = &w
	}
	return out
}

// filter reads the counts once for every analyzer it wraps.
type filter struct {
	load   func() (Counts, error)
	once   sync.Once
	counts Counts
	err    error
}

func (f *filter) get() (Counts, error) {
	f.once.Do(func() { f.counts, f.err = f.load() })
	return f.counts, f.err
}

// report reports the findings held from one pass of analyzer whose Key
// outnumbers its count in recorded.
func report(pass *analysis.Pass, analyzer string, recorded map[string]int, held []analysis.Diagnostic) {
	byKey := map[string]int{}
	for _, d := range held {
		byKey[Key(analyzer, d)]++
	}
	for _, d := range held {
		k := Key(analyzer, d)
		n, base := byKey[k], recorded[k]
		if n <= base {
			continue
		}
		if base > 0 {
			d.Message += fmt.Sprintf(" (%d %s findings in this package, over its baseline of %d)", n, k, base)
		}
		pass.Report(d)
	}
}
