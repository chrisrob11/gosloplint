package a

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const wireVersion, protocolV3 = 3, 3

func TestConstAsserts(t *testing.T) {
	assert.True(t, true)          // want `const-assert: assert.True is given only literals \(true\)`
	assert.False(t, 1 > 2)        // want `const-assert: assert.False is given only literals \(1 > 2\)`
	assert.NoError(t, nil)        // want `const-assert: assert.NoError is given only literals \(nil\)`
	require.Equal(t, 1, 2)        // want `const-assert: require.Equal is given only literals \(1 and 2\)`
	assert.NotEqual(t, "a", "b")  // want `const-assert: assert.NotEqual is given only literals \("a" and "b"\)`
	assert.New(t).True(!false)    // want `const-assert: assert.True is given only literals \(!false\)`
	if false {                    // want `const-assert: the condition false is literals only`
		t.Fatal("never")
	}
	if 1 > 2 || (3 < 1) { // want `const-assert: the condition 1 > 2 \|\| \(3 < 1\) is literals only`
		t.Error("never")
	}
}

func TestConstAssertsNotReported(t *testing.T) {
	got := query()
	// Named constants agreeing is a real check of two declarations.
	assert.Equal(t, wireVersion, protocolV3)
	// Build constants differ by platform.
	if runtime.GOOS == "plan9" {
		t.Fatal("unsupported")
	}
	// x == x is selfassert's.
	assert.True(t, 1 == 1)
	require.Equal(t, 1, 1)
	if 1 != 1 {
		t.Fatal("never")
	}
	// A constant switch off of a test is a skip, not a check.
	if true {
		t.Skip("disabled")
	}
	// A constant condition whose body cannot fail checks nothing anyway.
	if false {
		t.Log("never")
	}
	assert.True(t, len(got) == 0)
	assert.Equal(t, 0, len(got))
	assert.Nil(t, got)
}
