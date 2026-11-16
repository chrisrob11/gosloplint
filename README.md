# gosloplint

Go analyzers that catch AI slop: code that compiles, passes its tests and
reads as finished, while doing nothing. A test whose whole body is a skip, an
assertion that compares a value to itself, a guard that quietly returns when
an ID it needs is missing.

Every check is a standard
[`go/analysis`](https://pkg.go.dev/golang.org/x/tools/go/analysis) Analyzer,
so it runs in the tools you already use: `go vet`, gopls, and (next)
golangci-lint.

## Install

```sh
go install github.com/chrisrob11/gosloplint/cmd/gosloplint@latest
```

## Usage

Run every analyzer through `go vet`:

```sh
go vet -vettool=$(which gosloplint) ./...
```

Or run the binary directly. It takes the standard analysis driver flags
(`-help` lists them); each analyzer's own flags are prefixed with its name:

```sh
gosloplint ./...
gosloplint -emptyskip ./...    # only the named analyzers
```

golangci-lint plugin support is coming next. Until then, use `go vet`.

## Analyzers

### emptyskip

Reports a test whose whole body is `t.Skip`, `t.Skipf` or `t.SkipNow`, and an
unconditional subtest whose body is only a skip. Such a test occupies the
name, appears in the count and reads as coverage, while running nothing.

```go
// Bad: green, and tests nothing.
func TestSnapshotRead(t *testing.T) {
	t.Skip("needs a live database")
}

// Good: a gated test with a body, or pending work behind the skip.
func TestSnapshotRead(t *testing.T) {
	if testing.Short() {
		t.Skip("needs a live database")
	}
	db := openDB(t)
	// ...
}
```

Not reported: a skip under a condition, a skip with statements after it
(pending work that runs the day the skip is removed), and a subtest created
inside an `if` or `switch` (a documented per-case N/A). A behaviour that
genuinely cannot be tested is exempted with a comment on the line above the
skip, and the reason is required:

```go
//emptyskip:ok this grammar shape cannot be built from source text
t.Skip("unreachable")
```

Flags: none.

## License

MIT. See [LICENSE](LICENSE).
