package slop

import "testing"

func TestTwice(t *testing.T) {
	if twice(2) != 4 {
		t.Fatal(describe(twice(2)))
	}
}
