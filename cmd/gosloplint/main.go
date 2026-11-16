// Command gosloplint runs every gosloplint analyzer. It is a standard
// multichecker, so it also works as `go vet -vettool=$(which gosloplint)`.
//
// One subcommand is not an analyzer: `gosloplint dupl [packages]` reports
// code copied across packages, which a per-package analysis pass cannot see.
package main

import (
	"os"

	"golang.org/x/tools/go/analysis/multichecker"

	"github.com/chrisrob11/gosloplint"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "dupl" {
		os.Exit(runDupl(os.Args[2:], os.Stdout, os.Stderr))
	}
	multichecker.Main(gosloplint.Analyzers()...)
}
