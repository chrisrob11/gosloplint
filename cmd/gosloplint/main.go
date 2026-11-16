// Command gosloplint runs every gosloplint analyzer. It is a standard
// multichecker, so it also works as `go vet -vettool=$(which gosloplint)`.
// With -baseline, each analyzer is quiet for a package and check until its
// findings outnumber the baseline file's count.
//
// Two subcommands are not analyzers: `gosloplint dupl [packages]` reports
// code copied across packages, which a per-package analysis pass cannot see,
// and `gosloplint baseline [packages]` writes the baseline file.
package main

import (
	"flag"
	"os"

	"golang.org/x/tools/go/analysis/multichecker"

	"github.com/chrisrob11/gosloplint"
	"github.com/chrisrob11/gosloplint/internal/baseline"
)

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "dupl":
			os.Exit(runDupl(os.Args[2:], os.Stdout, os.Stderr))
		case "baseline":
			os.Exit(runBaseline(gosloplint.Analyzers(), os.Args[2:], os.Stdout, os.Stderr))
		}
	}
	path := flag.String("baseline", "", "a file written by gosloplint baseline: report a package's findings of a check only once they outnumber its count")
	multichecker.Main(baseline.Wrap(gosloplint.Analyzers(), func() (baseline.Counts, error) {
		if *path == "" {
			return nil, nil
		}
		return baseline.Read(*path)
	})...)
}
