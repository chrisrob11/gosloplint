package comment

import (
	"cmp"
	"fmt"
	"slices"
	"strings"
)

// The checks, by the name Settings.Disable and each message use.
const (
	Length  = "length"
	Density = "density"
)

// Checks is every check.
var Checks = []string{Length, Density}

// The defaults a zero Settings field takes.
const (
	DefaultDensity         = 20
	DefaultDensityMinLines = 100
)

// Settings configures an Analyzer built by New. The zero value is the
// defaults.
type Settings struct {
	// Disable names checks not to run.
	Disable []string
	// Limits are the per-kind block limits; a zero field is its default.
	Limits Limits
	// Density is the per-package ceiling, in percent of the non-blank lines
	// of its non-test files that are comment. Zero means DefaultDensity.
	Density int
	// DensityMinLines is the size, in non-blank lines, below which a package
	// is not held to the ceiling. Zero means DefaultDensityMinLines.
	DensityMinLines int
}

// Validate reports an unknown check, a negative number, and a ceiling over
// 100%.
func (s Settings) Validate() error {
	for i, d := range s.Disable {
		if !slices.Contains(Checks, d) {
			return fmt.Errorf("disable[%d]: unknown check %q (known: %s)", i, d, strings.Join(Checks, ", "))
		}
	}
	if err := s.Limits.validate(); err != nil {
		return err
	}
	switch {
	case s.Density < 0 || s.Density > 100:
		return fmt.Errorf("density: %d is not a percentage from 1 to 100", s.Density)
	case s.DensityMinLines < 0:
		return fmt.Errorf("density-min-lines: %d is negative", s.DensityMinLines)
	}
	return nil
}

func (s Settings) enabled(check string) bool { return !slices.Contains(s.Disable, check) }

func (s Settings) density() int { return cmp.Or(s.Density, DefaultDensity) }

func (s Settings) densityMinLines() int { return cmp.Or(s.DensityMinLines, DefaultDensityMinLines) }
