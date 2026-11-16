// Package pkgpattern matches import paths against go-command style patterns,
// in which "..." matches any string, slashes included: "example.com/x/..."
// is x and everything below it, ".../internal/plan" any package of that name.
package pkgpattern

import (
	"regexp"
	"strings"
)

// Match reports whether path matches pattern.
func Match(pattern, path string) bool {
	re := "^" + strings.ReplaceAll(regexp.QuoteMeta(pattern), `\.\.\.`, `.*`) + "$"
	// "x/..." also matches x itself, as it does for the go command.
	re = strings.Replace(re, `/.*$`, `(/.*)?$`, 1)
	ok, err := regexp.MatchString(re, path)
	return err == nil && ok
}

// Any reports whether path matches one of patterns.
func Any(patterns []string, path string) bool {
	for _, p := range patterns {
		if Match(p, path) {
			return true
		}
	}
	return false
}
