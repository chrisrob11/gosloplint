package fileplace

import (
	"cmp"
	"fmt"
	"maps"
	"path"
	"slices"
	"strings"

	"github.com/chrisrob11/gosloplint/internal/pkgpattern"
)

// The rules, by the name Settings.Disable, Settings.Exempt and each message
// use.
const (
	Receiver = "receiver"
	TestName = "testname"
	TypeFile = "typefile"
	FileName = "filename"
	GrabBag  = "grabbag"
	Tiny     = "tiny"
	Long     = "long"
)

// Rules is every rule.
var Rules = []string{Receiver, TestName, TypeFile, FileName, GrabBag, Tiny, Long}

// OffByDefault are the rules that run only when Settings.Enable names them.
var OffByDefault = []string{TypeFile, FileName}

// The defaults a zero Settings field takes.
const (
	DefaultTinyLines = 40
	DefaultLongLines = 1500
)

// DefaultGrabBag are the file name parts grabbag reports by default.
var DefaultGrabBag = []string{
	"util", "utils", "helper", "helpers", "common", "misc", "types", "base", "impl", "manager",
	"testutil", "testutils", "testhelper", "testhelpers",
}

// Settings configures an Analyzer built by New. The zero value is the
// defaults.
type Settings struct {
	// Disable names rules not to run.
	Disable []string
	// Enable names rules of OffByDefault to run.
	Enable []string
	// TinyLines is the size below which a file declaring no type is tiny.
	// Zero means DefaultTinyLines.
	TinyLines int
	// LongLines is the size above which a file is long, and above which a
	// type's files may hold its methods in <type>_<concern>.go files. Zero
	// means DefaultLongLines.
	LongLines int
	// GrabBag replaces DefaultGrabBag when set: the lower-case file name
	// parts, split at underscores, that name no concern.
	GrabBag []string
	// Exempt maps a rule to patterns it does not apply to. A pattern matches
	// a package's import path ("..." matches any string) or a file's base
	// name (a path.Match glob, such as "*_gen.go").
	Exempt map[string][]string
}

// Validate reports an unknown rule, a rule both enabled and disabled, a
// negative size, a tiny size not below
// the long one, a blank grab-bag word, and a malformed exemption.
func (s Settings) Validate() error {
	for i, d := range s.Disable {
		if !slices.Contains(Rules, d) {
			return fmt.Errorf("disable[%d]: unknown rule %q (known: %s)", i, d, strings.Join(Rules, ", "))
		}
	}
	for i, e := range s.Enable {
		switch {
		case !slices.Contains(OffByDefault, e):
			return fmt.Errorf("enable[%d]: %q is not a rule that is off by default (those are: %s)", i, e, strings.Join(OffByDefault, ", "))
		case slices.Contains(s.Disable, e):
			return fmt.Errorf("enable[%d]: %q is also disabled", i, e)
		}
	}
	switch {
	case s.TinyLines < 0:
		return fmt.Errorf("tiny-lines: %d is negative", s.TinyLines)
	case s.LongLines < 0:
		return fmt.Errorf("long-lines: %d is negative", s.LongLines)
	case s.tinyLines() >= s.longLines():
		return fmt.Errorf("tiny-lines %d is not below long-lines %d", s.tinyLines(), s.longLines())
	}
	for i, w := range s.GrabBag {
		if strings.TrimSpace(w) == "" || strings.ContainsAny(w, "_ ") {
			return fmt.Errorf("grab-bag[%d]: %q is not one file name part", i, w)
		}
	}
	for _, rule := range slices.Sorted(maps.Keys(s.Exempt)) {
		if !slices.Contains(Rules, rule) {
			return fmt.Errorf("exempt: unknown rule %q (known: %s)", rule, strings.Join(Rules, ", "))
		}
		for i, p := range s.Exempt[rule] {
			if strings.TrimSpace(p) == "" {
				return fmt.Errorf("exempt.%s[%d]: empty pattern", rule, i)
			}
			if _, err := path.Match(p, ""); err != nil {
				return fmt.Errorf("exempt.%s[%d]: %q: %w", rule, i, p, err)
			}
		}
	}
	return nil
}

func (s Settings) enabled(rule string) bool {
	if slices.Contains(OffByDefault, rule) {
		return slices.Contains(s.Enable, rule)
	}
	return !slices.Contains(s.Disable, rule)
}

func (s Settings) tinyLines() int { return cmp.Or(s.TinyLines, DefaultTinyLines) }

func (s Settings) longLines() int { return cmp.Or(s.LongLines, DefaultLongLines) }

func (s Settings) grabBag() []string {
	if len(s.GrabBag) > 0 {
		return s.GrabBag
	}
	return DefaultGrabBag
}

// exempts reports whether rule is exempted for the file name in the package
// at pkgPath. A pattern matching a package matches its external test
// package too.
func (s Settings) exempts(rule, pkgPath, name string) bool {
	pkgPath = strings.TrimSuffix(pkgPath, "_test")
	for _, p := range s.Exempt[rule] {
		if ok, _ := path.Match(p, name); ok || pkgpattern.Match(p, pkgPath) {
			return true
		}
	}
	return false
}
