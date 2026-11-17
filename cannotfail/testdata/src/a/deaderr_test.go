package a

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeadErrs(t *testing.T) {
	n, err := load("a") // want `dead-err: the error assigned to err here is overwritten on line 12 before anything checks it`
	m, err := load("b")
	require.NoError(t, err)
	_, err = load("c") // want `dead-err: the error assigned to err here is never checked: nothing reads err after it`
	assert.Equal(t, n, m)

	var zero error
	require.NoError(t, zero) // want `dead-err: zero is declared and never assigned, so it is always nil: require.NoError\(t, zero\) cannot fail`
	if zero != nil {         // want `dead-err: zero is declared and never assigned, so it is always nil: if zero != nil cannot fail`
		t.Fatal(zero)
	}
	var shadowed error
	if len(n2()) > 0 {
		_, shadowed := load("d")
		_ = shadowed == nil
		_, shadowed = load("e") // want `dead-err: the error assigned to shadowed here is never checked`
	}
	assert.NoError(t, shadowed) // want `dead-err: shadowed is declared and never assigned`

	var e2 = loadErr() // want `dead-err: the error assigned to e2 here is overwritten on line 31`
	e2 = loadErr()
	assert.NoError(t, e2)
}

func n2() []string { return query() }

func loadErr() error { _, err := load("x"); return err }

// Each error below is read, or may be, before it is overwritten.
func TestErrsChecked(t *testing.T) {
	_, err := load("a")
	if err != nil {
		t.Fatal(err)
	}
	_, err = load("b")
	err = wrap(err) // reads the previous value
	require.NoError(t, err)

	for _, s := range []string{"x", "y"} {
		if err != nil {
			t.Error(err)
		}
		_, err = load(s) // read by the next iteration
	}

	_, err = load("c")
	if len(query()) == 0 {
		return
	}
	require.NoError(t, err)

	_, err = load("g")
	switch {
	case len(query()) > 0:
		_, err = load("h")
	}
	require.NoError(t, err)

	var later error
	if len(query()) > 0 {
		later = loadErr()
	}
	require.NoError(t, later)
}

func TestErrInADeferredClosure(t *testing.T) {
	_, err := load("d")
	defer func() { require.NoError(t, err) }()
	_, err = load("e")
}

func TestErrByAddress(t *testing.T) {
	var filled error
	fill(&filled)
	require.NoError(t, filled)
}

func TestContinueBeforeTheNextWrite(t *testing.T) {
	var err error
	for i := 0; i < 2; i++ {
		_, err = load("e")
		if i == 0 {
			continue
		}
		_, err = load("f")
	}
	require.NoError(t, err)
}

func wrap(err error) error { return err }

func fill(p *error) {}

// A named result is read by the return statement.
func TestNamedResults(t *testing.T) {
	f := func() (err error) {
		_, err = load("a")
		return
	}
	require.NoError(t, f())
}

func TestGoto(t *testing.T) {
	var err error
	n := 0
retry:
	if err != nil && n > 2 {
		t.Fatal(err)
	}
	_, err = load("a")
	n++
	if n < 3 {
		goto retry
	}
}

func TestRange(t *testing.T) {
	for _, err := range []error{loadErr()} {
		require.NoError(t, err)
	}
}

func TestLoopReadsTheLastTurn(t *testing.T) {
	var err error
	for _, s := range []string{"x", "y"} {
		if err != nil {
			t.Fatal(err)
		}
		_, err = load(s)
	}
}
