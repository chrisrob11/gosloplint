package slop

import "testing"

func TestSnapshotRead(t *testing.T) {
	t.Skip("needs a live database")
}

func TestQuery(t *testing.T) {
	err := query(t.Context(), "a")
	if err != err {
		t.Fatal(err)
	}
}

// mustQuery is not reported: .golangci.yml disables testhelper.
func mustQuery(t *testing.T) int {
	if err := query(t.Context(), "a"); err != nil {
		t.Fatal(err)
	}
	return 1
}

var _ = mustQuery

// TestTwiceLogs only logs a wrong result, so it cannot fail.
func TestTwiceLogs(t *testing.T) {
	if got := twice(3); got != 6 {
		t.Logf("twice(3) = %d, want 6", got)
	}
}
