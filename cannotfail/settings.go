package cannotfail

import (
	"fmt"
	"slices"
	"strings"
)

// The checks, by the name Settings.Disable and each message use.
const (
	ConstAssert  = "const-assert"
	EmptyLoop    = "empty-loop"
	NoAssert     = "no-assert"
	EmptySubtest = "empty-subtest"
	DeadErr      = "dead-err"
	LiteralField = "literal-field"
)

// Checks is every check.
var Checks = []string{ConstAssert, EmptyLoop, NoAssert, EmptySubtest, DeadErr, LiteralField}

// OffByDefault are the checks that run only when Settings.Enable names them.
// empty-loop cannot see that a function returns a non-empty collection, and
// a range over one is how good tests check every element.
var OffByDefault = []string{EmptyLoop}

// Settings configures an Analyzer built by New. The zero value runs every
// check but those of OffByDefault.
type Settings struct {
	// Disable names checks not to run.
	Disable []string
	// Enable names checks of OffByDefault to run.
	Enable []string
	// AssertFuncs are functions that can fail a test without being handed
	// one, as importpath.Func, importpath.Type.Method, or an import path
	// pattern ("..." matches anything) for every function of the packages
	// it matches. A call to one is an assertion.
	AssertFuncs []string
}

// Validate reports an unknown check in Disable, an Enable entry that is not
// off by default or is also disabled, and a blank AssertFuncs entry.
func (s Settings) Validate() error {
	for i, d := range s.Disable {
		if !slices.Contains(Checks, d) {
			return fmt.Errorf("disable[%d]: unknown check %q (known: %s)", i, d, strings.Join(Checks, ", "))
		}
	}
	for i, e := range s.Enable {
		switch {
		case !slices.Contains(OffByDefault, e):
			return fmt.Errorf("enable[%d]: %q is not a check that is off by default (those are: %s)", i, e, strings.Join(OffByDefault, ", "))
		case slices.Contains(s.Disable, e):
			return fmt.Errorf("enable[%d]: %q is also disabled", i, e)
		}
	}
	for i, f := range s.AssertFuncs {
		if strings.TrimSpace(f) == "" {
			return fmt.Errorf("assert-funcs[%d]: empty entry", i)
		}
	}
	return nil
}

func (s Settings) enabled(check string) bool {
	if slices.Contains(OffByDefault, check) {
		return slices.Contains(s.Enable, check)
	}
	return !slices.Contains(s.Disable, check)
}
