package a

import (
	"fmt"
	"log"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"mylib/check"
)

func TestEmpty(t *testing.T) {} // want `no-assert: TestEmpty calls no t.Error, t.Fatal or assertion`

func TestOnlyLogs(t *testing.T) { // want `no-assert: TestOnlyLogs calls no`
	got := query()
	if len(got) != 1 {
		t.Logf("got %v", got)
	}
	fmt.Println(got)
}

func TestCallsAQuietHelper(t *testing.T) { // want `no-assert: TestCallsAQuietHelper calls no`
	show(t, query())
}

// show takes t but only logs.
func show(t *testing.T, v any) {
	t.Helper()
	t.Log(v)
}

func TestSubtests(t *testing.T) {
	t.Run("empty", func(t *testing.T) {}) // want `empty-subtest: subtest "empty" calls no t.Error, t.Fatal or assertion`
	t.Run("logs", func(t *testing.T) {    // want `empty-subtest: subtest "logs" calls no`
		t.Log(query())
	})
	name := "computed"
	t.Run(name, func(t *testing.T) { // want `empty-subtest: subtest calls no`
		_ = query()
	})
	t.Run("asserts", func(t *testing.T) {
		assert.NotEmpty(t, query())
	})
	t.Run("skips", func(t *testing.T) {
		t.Skip("later")
	})
	t.Run("smoke", func(_ *testing.T) {
		_ = query()
	})
	t.Run("groups", func(t *testing.T) {
		t.Run("inner", func(t *testing.T) {
			t.Error("x")
		})
	})
	var got []string
	t.Run("feeds the parent", func(t *testing.T) {
		got = query()
	})
	if len(got) != 0 {
		t.Fatal(got)
	}
}

// Each test below can fail, and none is reported.

func TestErrors(t *testing.T) { t.Errorf("x") }

func TestFailNow(t *testing.T) { t.FailNow() }

func TestTestify(t *testing.T) { assert.Equal(t, 1, len(query())) }

func TestTestifyMethods(t *testing.T) {
	is := assert.New(t)
	is.NoError(loadErr())
}

func TestHelperInTheNonTestFile(t *testing.T) { checkOK(t, len(query()) == 0) }

func TestHelperChain(t *testing.T) { outer(t) }

func outer(t *testing.T) { inner(t) }

func inner(tb testing.TB) { tb.Fatal("x") }

func TestHelperFromAnotherPackage(t *testing.T) {
	// The analyzer cannot read it, so it may fail the test.
	fmt.Fprint(writer{t}, "x")
}

type writer struct{ t *testing.T }

func (w writer) Write(p []byte) (int, error) { w.t.Error(string(p)); return len(p), nil }

func TestStoredInAFake(t *testing.T) {
	f := &fake{t: t}
	use(f)
}

type fake struct{ t *testing.T }

type store interface{ get() }

func (f *fake) get() { f.t.Error("unexpected call") }

func use(s store) { s.get() }

func TestMethodOnAHarness(t *testing.T) {
	h := harness{t}
	h.check(query())
}

type harness struct{ t *testing.T }

func (h harness) check(v []string) {
	if v == nil {
		h.t.Error("nil")
	}
}

func TestMethodValue(t *testing.T) {
	report := t.Errorf
	report("x")
}

func TestPanics(t *testing.T) {
	if query() != nil {
		panic("x")
	}
}

func TestExits(t *testing.T) {
	if query() != nil {
		log.Fatal("x")
	}
	os.Exit(0)
}

func TestFuncValue(t *testing.T) {
	cases := []struct{ check func([]string) }{{check: func([]string) {}}}
	for _, c := range cases {
		c.check(query())
	}
}

func TestSubtestByName(t *testing.T) { t.Run("x", subtestBody) }

func subtestBody(t *testing.T) {}

func TestSkipsWhenShort(t *testing.T) {
	if testing.Short() {
		t.Skip("short")
	}
	_ = query()
}

func TestAssertFuncs(t *testing.T) { check.That(len(query()) == 0) } // want `no-assert: TestAssertFuncs calls no`

func TestAssertMethods(t *testing.T) { check.Suite{}.Holds(true) } // want `no-assert: TestAssertMethods calls no`

func TestBlank(_ *testing.T) { _ = query() }

func TestUnnamed(*testing.T) { _ = query() }

// Regression for a crash: calling it must not panic.
func TestDoesNotPanic(t *testing.T) { _ = query() }

func TestRaces(t *testing.T) {
	go query() // run under the race detector
	_ = query()
}

func Testlowercase(t *testing.T) { _ = query() }

func BenchmarkQuery(b *testing.B) {
	for range b.N {
		_ = query()
	}
}

func FuzzQuery(f *testing.F) {
	f.Fuzz(func(t *testing.T, s string) { _ = s })
}

func ExampleConfig() {
	fmt.Println("x")
	// Output: x
}

func TestMain(m *testing.M) { os.Exit(m.Run()) }

func TestRecursiveHelper(t *testing.T) { countdown(t, 3) }

func countdown(t *testing.T, n int) {
	if n == 0 {
		t.Fatal("done")
	}
	countdown(t, n-1)
}

func TestGenericHelper(t *testing.T) { same(t, 1, 1) }

func same[T comparable](t *testing.T, a, b T) {
	if a != b {
		t.Error(a, b)
	}
}

func TestAppendedToASlice(t *testing.T) {
	var all []*testing.T
	all = append(all, t)
	_ = all
}
