package a

import "testing"

// Not reported: a must helper outside a _test.go file is not a test helper
// this analyzer judges.
func mustProd(t *testing.T) int {
	t.Fatal("x")
	return 1
}

var _ = mustProd
