# cannotfail

Reports, in `_test.go` files, tests that cannot fail: they stay green
whatever the code under test does. Each finding is prefixed with its check's
name.

Checks:

- `const-assert`: a testify assertion given only literals
  (`assert.True(t, true)`, `assert.NoError(t, nil)`, `require.Equal(t, 1, 2)`),
  or an `if` whose condition is only literals and whose body fails the test
  (`if false { t.Fatal() }`). Named constants are not literals:
  `assert.Equal(t, wireVersion, protocolV3)` checks that two declarations
  agree. A comparison of a value to itself is selfassert's.
- `no-assert`: a `Test` function that can neither fail nor skip. A test
  fails through a failing `testing` method (`Error`, `Errorf`, `Fatal`,
  `Fatalf`, `Fail`, `FailNow`; `Skip` counts too), any testify call,
  `panic`, `log.Fatal` or `os.Exit`, a function of `assert-funcs`, or a
  function of its package that does one of these, followed through any
  depth of calls. Anything the analyzer cannot follow counts as failing: a
  call handed a testing value (or a struct holding one) outside the
  package, a call through a function value, and a testing value stored,
  returned or sent, so a fake holding `t` is never mistaken for silence.
  A call of a function or method named `Must` or `must` followed by an
  upper-case letter (`MustSave`, `regexp.MustCompile`) counts as failing:
  by Go convention it panics when it fails, and a panic fails the test. A
  `Must` helper declared in a `_test.go` file is judged by its body
  instead.
- `empty-subtest`: a `t.Run` literal that can neither fail nor skip, by the
  same rules. A subtest that assigns a variable of the test around it, or
  sends on a channel, is not reported: the parent checks what it did.
- `dead-err`: in a test file, an error variable assigned and then
  overwritten before anything reads it, or never read at all; and one
  declared by `var err error`, never assigned, then asserted nil. A variable
  a closure names or whose address is taken is not read, nor is any in a
  function with a `goto`.
- `literal-field`: an assertion that a field equals the literal it was set
  to, by `x.f = lit` or a composite literal, with nothing between that could
  change it: no call but a builtin or conversion, no receive, no address
  taken, no assignment but to a plain variable, no compound statement.
- `empty-loop`, off by default: in a `Test` function, a range whose body
  checks its elements (a testify call, a helper, or a `t.Error` inside an
  `if`) when that loop holds every assertion of its test or subtest, and
  nothing shows the collection is non-empty: a non-empty literal, an append
  of elements, any other read of it (`require.Len`, `len(xs)`), or a
  closure's parameter. It is off because it cannot see that a function
  returns a non-empty collection, and ranging over one is how good tests
  check every element; turn it on to audit.

```go
// Bad: nothing here can fail.
func TestNewConfig(t *testing.T) {
	c := Config{Port: 5432}
	assert.Equal(t, 5432, c.Port) // literal-field
	assert.True(t, true)          // const-assert
	rows, err := Load("a")
	_, err = Load("b")            // dead-err: Load("a")'s error is never read
	require.NoError(t, err)
	t.Run("empty", func(t *testing.T) {}) // empty-subtest
	t.Logf("rows: %v", rows)
}

func TestLoad(t *testing.T) { // no-assert: t.Logf never fails
	got, _ := Load("a")
	if len(got) != 1 {
		t.Logf("got %v", got)
	}
}

// Good: assert on what the code under test produced.
func TestLoad(t *testing.T) {
	got, err := Load("a")
	require.NoError(t, err)
	assert.Equal(t, []Row{{ID: "a"}}, got)
}
```

Not read by `no-assert`: benchmarks, examples, fuzz targets, `TestMain`, a
test whose parameter is `_` or unnamed (`func TestNoPanic(_ *testing.T)`
says it only checks that the code survives), and a test or subtest with a
comment, on the line above or inside it, or a name (`TestConcurrentWrites`,
`t.Run("no panic", ...)`), saying it is for a panic, crash, race,
concurrency, hang or deadlock; and an `os/exec` helper process, a test
whose first statement is `if os.Getenv(...) != "1" { return }` (or `==`):
the test that runs it as a subprocess asserts on what it did. Generated
files are skipped.

Flags:

- `-cannotfail.disable` (default empty): comma-separated checks not to run.
- `-cannotfail.enable` (default empty): comma-separated off-by-default checks
  to run: `empty-loop`.
- `-cannotfail.assert-funcs` (default empty): comma-separated functions that
  fail a test without being passed one, such as an assertion library with a
  global fail handler: `importpath.Func`, `importpath.Type.Method`, or an
  import path pattern (`...` matches anything) for every function of the
  packages it matches.

In golangci-lint they are the `cannotfail.disable`, `cannotfail.enable` and
`cannotfail.assert-funcs` settings.
