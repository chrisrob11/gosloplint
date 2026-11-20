package header

import (
	"go/parser"
	"go/token"
	"testing"
)

func TestIs(t *testing.T) {
	const src = `// Licensed under the MIT license.

//go:build !never

// Package p is the doc, above the clause.
package p

// F is ordinary.
func F() {}

/* SPDX-License-Identifier: MIT */

// An embedded notice: Copyright 2026 The Authors.
var V int
`
	f, err := parser.ParseFile(token.NewFileSet(), "p.go", src, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	want := []bool{true, true, false, false, true, true}
	if len(f.Comments) != len(want) {
		t.Fatalf("%d comment groups, want %d", len(f.Comments), len(want))
	}
	for i, cg := range f.Comments {
		if got := Is(f, cg); got != want[i] {
			t.Errorf("group %d %q: Is = %v, want %v", i, cg.Text(), got, want[i])
		}
	}
}
