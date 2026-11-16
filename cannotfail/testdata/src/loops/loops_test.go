package loops

import (
	"os"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func rows() []string { return nil }

func TestOnlyInTheLoop(t *testing.T) {
	got := rows()
	for _, r := range got { // want `empty-loop: every assertion here is inside this range over got, and nothing shows it is non-empty`
		assert.NotEqual(t, "", r)
	}
}

func TestGuardedError(t *testing.T) {
	m := map[string]int{}
	for k, v := range m { // want `empty-loop: every assertion here is inside this range over m`
		if v < 0 {
			t.Errorf("%s: %d", k, v)
		}
	}
}

func TestInASubtest(t *testing.T) {
	assert.NotNil(t, rows())
	t.Run("sub", func(t *testing.T) {
		var got []string
		got = rows()
		for _, r := range got { // want `empty-loop: every assertion here is inside this range over got`
			assert.NotEqual(t, "", r)
		}
	})
}

var files, _ = os.ReadDir("testdata")

func TestPackageVar(t *testing.T) {
	for _, f := range files { // want `empty-loop: every assertion here is inside this range over files`
		assert.NotEqual(t, "", f.Name())
	}
}

// Each loop below runs over something shown non-empty, or is not the only
// place the test asserts.

var corpus = []string{"a", "b"}

var more = append(corpus, "c")

func TestShownNonEmpty(t *testing.T) {
	for _, s := range corpus {
		assert.NotEqual(t, "", s)
	}
}

func TestMore(t *testing.T) {
	for _, s := range more {
		assert.NotEqual(t, "", s)
	}
}

func TestLength(t *testing.T) {
	got := rows()
	for _, r := range got {
		assert.NotEqual(t, "", r)
	}
	require.Len(t, got, 2)
}

func TestOtherAssertion(t *testing.T) {
	got := rows()
	require.NotEmpty(t, rows())
	for _, r := range got {
		assert.NotEqual(t, "", r)
	}
}

func TestTable(t *testing.T) {
	cases := []struct{ want []string }{{want: []string{"a"}}}
	for _, tc := range cases {
		for _, w := range tc.want {
			assert.NotEqual(t, "", w)
		}
	}
}

func TestLiterals(t *testing.T) {
	names := []rune("abc")
	for _, r := range names {
		assert.NotEqual(t, 0, r)
	}
	both := append(rows(), "x")
	for _, r := range both {
		assert.NotEqual(t, "", r)
	}
	alias := corpus
	for _, r := range alias {
		assert.NotEqual(t, "", r)
	}
	for _, r := range []string{"a"} {
		assert.NotEqual(t, "", r)
	}
}

func TestReportingLoop(t *testing.T) {
	var offenders []string
	for _, o := range offenders {
		t.Errorf("offender %s", o)
	}
}

func TestClosureParameter(t *testing.T) {
	slices.ContainsFunc([]string{"ab"}, func(s string) bool {
		for _, r := range s {
			assert.NotEqual(t, 0, r)
		}
		return false
	})
}

func TestArrayAndInt(t *testing.T) {
	var a [3]int
	for _, v := range a {
		assert.Equal(t, 0, v)
	}
	n := len(rows())
	for i := range n {
		assert.Equal(t, 0, i)
	}
}

func TestOtherPackage(t *testing.T) {
	for _, a := range os.Args {
		assert.NotEqual(t, "", a)
	}
}

func BenchmarkLoop(b *testing.B) {
	for _, r := range rows() {
		if r == "" {
			b.Fatal("empty")
		}
	}
}
