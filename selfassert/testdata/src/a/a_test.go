package a

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	other "mylib/assert"
)

// Reported: the two compared arguments are one expression written twice.
func TestSelfCompareArgs(t *testing.T) {
	assert.Equal(t, Database("db", "t"), Database("db", "t")) // want `assert.Equal compares Database\("db", "t"\) to itself`
}

// Reported: formatting does not make two expressions different.
func TestSelfCompareFormatting(t *testing.T) {
	require.Equal(t, Database("db", "t"), Database("db", "t")) // want `require.Equal compares Database\("db", "t"\) to itself`
}

// Reported: every equality assertion, not only Equal.
func TestSelfCompareVariants(t *testing.T) {
	x := Database("a", "b")
	assert.Equalf(t, x, x, "msg")   // want `assert.Equalf compares x to itself`
	assert.EqualValues(t, x, x)     // want `assert.EqualValues compares x to itself`
	assert.Same(t, &x, &x)          // want `assert.Same compares &x to itself`
	assert.Exactly(t, x, x)         // want `assert.Exactly compares x to itself`
	require.Samef(t, &x, &x, "msg") // want `require.Samef compares &x to itself`
}

// Reported: the method form on *Assertions, where the compared values are the
// first two arguments.
func TestSelfCompareMethodForm(t *testing.T) {
	is := assert.New(t)
	must := require.New(t)
	is.Equal(Database("db", "t"), Database("db", "t"))   // want `assert.Equal compares Database\("db", "t"\) to itself`
	must.Equal(Database("db", "t"), Database("db", "t")) // want `require.Equal compares Database\("db", "t"\) to itself`
}

// Reported: the same in an if, with == and with !=.
func TestSelfCompareBinary(t *testing.T) {
	if Database("db", "t") == Database("db", "t") { // want `== compares Database\("db", "t"\) to itself`
		t.Error("unreachable")
	}
	n := len("x")
	if n != n { // want `!= compares n to itself`
		t.Error("unreachable")
	}
}

// Reported: two names, one initialiser, compared through an index expression
// rather than directly.
func TestAliasedInitViaIndex(t *testing.T) {
	plain := columnarFamily("root", Database("db", "t"))
	empty := columnarFamily("root", Database("db", "t"))
	for name := range plain {
		assert.Equal(t, plain[name], empty[name]) // want `assert.Equal compares plain and empty, which hold the same initialiser columnarFamily`
	}
}

// Reported: the aliased shape compared with == and declared with var.
func TestAliasedInitBinary(t *testing.T) {
	var a = Database("db", "t")
	var b = Database("db", "t")
	if a == b { // want `== compares a and b, which hold the same initialiser Database`
		return
	}
	t.Error("unreachable")
}

// Not reported: a round trip. The sides differ, and that is the assertion.
func TestRoundTrip(t *testing.T) {
	for _, in := range []string{"a", "bb", "ccc"} {
		assert.Equal(t, in, decode(encode(in)))
	}
}

// Not reported: two different prefixes compared to each other.
func TestRecordPrefix(t *testing.T) {
	all := recordPrefix("")
	one := recordPrefix("rec_1")
	assert.NotEqual(t, all, one)
	assert.True(t, len(all) < len(one))
}

// Not reported: two cases sharing an expected value are not a tautology.
func TestTableSharedWant(t *testing.T) {
	cases := []struct{ in, want string }{
		{"a", "x"},
		{"b", "x"},
	}
	for _, tc := range cases {
		assert.Equal(t, tc.want, normalize(tc.in))
	}
}

// Not reported: two empty containers filled by different routes.
func TestTwoBuilders(t *testing.T) {
	viaA := make(map[string]int, 2)
	viaB := make(map[string]int, 2)
	fillA(viaA)
	fillB(viaB)
	assert.Equal(t, viaA, viaB)
}

// Not reported: two named constants that independently agree.
const wireVersion = 3
const protocolV3 = 3

func TestConstantsAgree(t *testing.T) {
	assert.Equal(t, wireVersion, protocolV3)
}

// Not reported: non-determinism asserted on purpose, with the negated
// assertion, with !=, and with two identical calls.
func TestGeneratesDistinctIDs(t *testing.T) {
	a := newID()
	b := newID()
	assert.NotEqual(t, a, b)
	assert.NotEqual(t, newID(), newID())
	assert.New(t).NotEqual(newID(), newID())
	if a != b {
		return
	}
	t.Error("same id twice")
}

// Not reported: a name reassigned after its initialiser holds something else
// by the comparison.
func TestReassigned(t *testing.T) {
	a := newID()
	b := newID()
	b = a + "x"
	assert.Equal(t, a, b)
}

// Not reported: a local whose address escapes may be written through it.
func TestAddressTaken(t *testing.T) {
	a := newID()
	b := newID()
	fill(&b)
	assert.Equal(t, a, b)
}

// Not reported: x != x on a float is the NaN test.
func TestNaN(t *testing.T) {
	f := math.NaN()
	if f == f {
		t.Error("NaN equals itself")
	}
}

// Not reported: an Equal from a package that is not testify.
func TestOtherAssert(t *testing.T) {
	x := newID()
	_ = other.Equal(t, x, x)
}

// Reported: the inner a is a different variable, so assigning it does not
// make the outer a "assigned twice". Matching by name would miss this.
func TestShadowed(t *testing.T) {
	a := newID()
	b := newID()
	{
		a := recordPrefix("x")
		_ = a
	}
	require.Equal(t, a, b) // want `require.Equal compares a and b, which hold the same initialiser newID\(\)`
}
