package tests

import "testing"

// TestA guards the bug where A returned zero; regression for #123.
func TestA(t *testing.T) {
	if A() != 1 {
		t.Fatal("A")
	}
}

// TestB now uses a table. // want `history: "now uses"`
func TestB(t *testing.T) {
	if A() != 1 {
		t.Fatal("A")
	}
}
