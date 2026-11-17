package tests

import "testing"

// TestNothing is reported by cannotfail's no-assert.
func TestNothing(t *testing.T) { nine() }

// TestViaCheck is reported unless cannotfail.assert-funcs names tests.check.
func TestViaCheck(t *testing.T) { check(len(rows()) == 0) }

// TestRows is reported only when cannotfail.enable names empty-loop.
func TestRows(t *testing.T) {
	got := rows()
	for _, r := range got {
		if r == "" {
			t.Error("empty row")
		}
	}
}

func check(bool) {}

func rows() []string { return nil }
