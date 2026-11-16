package pkgpattern_test

import (
	"testing"

	"github.com/chrisrob11/gosloplint/internal/pkgpattern"
)

func TestMatch(t *testing.T) {
	for _, tc := range []struct {
		pattern, path string
		want          bool
	}{
		{"example.com/x", "example.com/x", true},
		{"example.com/x", "example.com/xy", false},
		{"example.com/x/...", "example.com/x", true},
		{"example.com/x/...", "example.com/x/y/z", true},
		{"example.com/x/...", "example.com/xy", false},
		{".../plan", "example.com/internal/plan", true},
		{".../plan", "example.com/internal/planner", false},
		{"...", "anything/at/all", true},
		{"a.b", "aXb", false},
	} {
		if got := pkgpattern.Match(tc.pattern, tc.path); got != tc.want {
			t.Errorf("Match(%q, %q) = %v, want %v", tc.pattern, tc.path, got, tc.want)
		}
	}
	if !pkgpattern.Any([]string{"x", "y/..."}, "y/z") || pkgpattern.Any(nil, "y") {
		t.Error("Any matches when, and only when, one pattern does")
	}
}
