package narration

import (
	"fmt"
	"slices"
	"strings"
)

// The checks, by the name Settings.Disable and each message use.
const (
	History  = "history"
	Warnsign = "warnsign"
)

// Checks is every check.
var Checks = []string{History, Warnsign}

// Settings configures an Analyzer built by New. The zero value runs every
// check.
type Settings struct {
	// Disable names checks not to run.
	Disable []string
}

// Validate reports an unknown check in Disable.
func (s Settings) Validate() error {
	for i, d := range s.Disable {
		if !slices.Contains(Checks, d) {
			return fmt.Errorf("disable[%d]: unknown check %q (known: %s)", i, d, strings.Join(Checks, ", "))
		}
	}
	return nil
}

func (s Settings) enabled(check string) bool { return !slices.Contains(s.Disable, check) }
