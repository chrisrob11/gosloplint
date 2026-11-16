// Command gosloplint runs every gosloplint analyzer. It is a standard
// multichecker, so it also works as `go vet -vettool=$(which gosloplint)`.
package main

import (
	"golang.org/x/tools/go/analysis/multichecker"

	"github.com/chrisrob11/gosloplint"
)

func main() {
	multichecker.Main(gosloplint.Analyzers()...)
}
