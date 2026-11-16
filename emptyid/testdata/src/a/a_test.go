package a

// Not reported: test files are skipped.
func testGuard(userID string) error {
	if userID == "" {
		return nil
	}
	return nil
}

var _ = testGuard
