package steer

import (
	"fmt"
	"slices"
	"strings"
)

// The checks, by the name Settings.Disable and each message use.
const (
	ReturnsAny         = "returns-any"
	ReturnsInterface   = "returns-interface"
	OneConcrete        = "returns-one-concrete"
	SoleImplementation = "sole-implementation"
	UnguardedRecursion = "unguarded-recursion"
	TwinWrapper        = "twin-wrapper"
	Alias              = "alias"
)

// Checks is every check, in report order.
var Checks = []string{ReturnsAny, ReturnsInterface, OneConcrete, SoleImplementation, UnguardedRecursion, TwinWrapper, Alias}

// Settings configures an Analyzer built by New. The zero value runs every
// check but unguarded-recursion, which runs only in RecursionPackages.
type Settings struct {
	// Disable names checks not to run.
	Disable []string
	// RecursionPackages are the import path patterns ("..." matches
	// anything, so "..." is every package) whose recursion
	// unguarded-recursion reads. Empty: the check reads none.
	RecursionPackages []string
}

// Validate reports an unknown check in Disable, and recursion packages set
// while unguarded-recursion is disabled.
func (s Settings) Validate() error {
	for i, d := range s.Disable {
		if !slices.Contains(Checks, d) {
			return fmt.Errorf("disable[%d]: unknown check %q (known: %s)", i, d, strings.Join(Checks, ", "))
		}
	}
	for i, p := range s.RecursionPackages {
		if strings.TrimSpace(p) == "" {
			return fmt.Errorf("recursion-packages[%d]: empty pattern", i)
		}
	}
	if len(s.RecursionPackages) > 0 && slices.Contains(s.Disable, UnguardedRecursion) {
		return fmt.Errorf("recursion-packages is set but %s is disabled", UnguardedRecursion)
	}
	return nil
}

func (s Settings) enabled(check string) bool { return !slices.Contains(s.Disable, check) }
