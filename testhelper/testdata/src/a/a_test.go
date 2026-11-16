package a

import (
	"testing"

	"other"
)

// Reported: each builds a value and fails the test on an error the caller
// never sees.

func mustX(t *testing.T) int { // want `mustX fails the test on an error it hides`
	t.Helper()
	t.Fatal("x")
	return 1
}

func mustKey(b *testing.B, s string) []byte { // want `mustKey fails the test`
	b.Fatalf("x %s", s)
	return nil
}

func MustVal(tb testing.TB) string { // want `MustVal fails the test`
	tb.FailNow()
	return ""
}

func mustMk(t *testing.T) int { // want `mustMk fails the test`
	func() { t.Fatal("x") }()
	return 1
}

type h struct{}

func (h) mustY(t *testing.T) int { // want `mustY fails the test`
	t.Fatal("x")
	return 1
}

func mustRng(n int, t *testing.T) (int, int) { // want `mustRng fails the test`
	t.Fatal("x")
	return n, n
}

func mustTwo(t *testing.T) (int, bool) { // want `mustTwo fails the test`
	t.Fatal("x")
	return 0, false
}

// Reported: an alias of testing.T is still testing.T.
type tAlias = testing.T

func mustAlias(t *tAlias) int { // want `mustAlias fails the test`
	t.Fatal("x")
	return 1
}

// Not reported: an assertion helper with no result.
func mustAssert(t *testing.T) {
	t.Helper()
	t.Fatal("x")
}

// Not reported: the error is returned to the caller.
func mustBuild(t *testing.T) (int, error) {
	t.Fatal("x")
	return 0, nil
}

func mustRun(t *testing.T) error {
	t.Fatal("x")
	return nil
}

// Not reported: a setup fixture without the must prefix.
func openDB(t *testing.T) int {
	t.Fatal("x")
	return 1
}

// Not reported: "must" inside the name is not the prefix.
func commust(t *testing.T) int {
	t.Fatal("x")
	return 1
}

// Not reported: no fatal call.
func mustGet(t *testing.T) int {
	t.Helper()
	t.Log("x")
	return 1
}

// Not reported: no testing parameter.
type fataler interface{ Fatal(...any) }

func mustGetFrom(s fataler) int {
	s.Fatal("x")
	return 1
}

// Not reported: the testing value is not a parameter, so the helper is not
// handed a test to fail.
var sharedT *testing.T

func mustShared() int {
	sharedT.Fatal("x")
	return 1
}

// Not reported: a T from another package.
func mustOther(t *other.T) int {
	t.Fatal("x")
	return 1
}

// Not reported: Fatal on a value that is not from package testing.
func mustLogged(t *testing.T, s fataler) int {
	t.Helper()
	s.Fatal("x")
	return 1
}

// Not reported: entry points never start with must.
func TestA(t *testing.T) {
	t.Fatal("x")
}

func BenchmarkA(b *testing.B) {
	b.Fatal("x")
}

func FuzzA(f *testing.F) {
	f.Fatal("x")
}

var _ = []any{mustX, mustKey, MustVal, mustMk, h.mustY, mustRng, mustTwo, mustAlias,
	mustAssert, mustBuild, mustRun, openDB, commust, mustGet, mustGetFrom, mustOther, mustLogged, mustShared}
