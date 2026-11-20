# testhelper

Reports a `must*` test helper that hides an error from the call site. A helper
that wraps the code under test and calls `t.Fatal` on its error moves the
check away from the line that can fail, so the failure names the helper
rather than the call, and the test reads as if the call cannot fail.

Reported: a function in a `_test.go` file whose name starts with `must` or
`Must`, that takes a `*testing.T`, `*testing.B` or `testing.TB`, returns at
least one result and no `error`, and calls `Fatal`, `Fatalf` or `FailNow` on
a `testing` value, even inside a closure.

```go
// Bad: the error from Parse is checked inside the helper.
func mustParse(t *testing.T, s string) *Doc {
	t.Helper()
	d, err := Parse(s)
	if err != nil {
		t.Fatal(err)
	}
	return d
}

// Good: the error is checked where the call is made.
d, err := Parse(src)
if err != nil {
	t.Fatalf("Parse(%q): %v", src, err)
}
```

Not reported: an assertion helper with no result, a helper that returns an
`error`, and setup fixtures without the `must` prefix (`openDB(t)`), which
build the world around the code under test rather than wrap it.

The rule keys on the `must` prefix, so renaming a helper to dodge it
(`mustGit` to `git`) silences the finding and keeps the defect. Fix the
helper instead: return the error and check it at the call site, or split
setup from reading output. A setup helper that returns nothing may fail
the test itself, since no caller reads a value from it; a helper whose
result a test reads returns `(T, error)`. prfoundry split `mustGit` that
way into `runGit(t, ...)`, for setup, and `gitOut(...) (string, error)`,
for a test that reads git's output and checks its error where it reads.

Flags: none.
