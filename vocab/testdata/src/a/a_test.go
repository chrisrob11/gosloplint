package a

import "testing"

// Test files are checked: a rename is not done while its tests use the old name.
func TestUserAge(t *testing.T) { // want `declares TestUserAge`
	_ = f("", "", 0, false)
}
