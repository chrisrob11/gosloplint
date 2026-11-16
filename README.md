# gosloplint

Go analyzers that catch AI slop: code that compiles, passes its tests and
reads as finished, while doing nothing. A test whose whole body is a skip, an
assertion that compares a value to itself, a guard that quietly returns when
an ID it needs is missing.

Every check is a standard
[`go/analysis`](https://pkg.go.dev/golang.org/x/tools/go/analysis) Analyzer,
so it runs in the tools you already use: `go vet`, gopls, and golangci-lint.

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

### With golangci-lint

gosloplint is a golangci-lint
[module plugin](https://golangci-lint.run/docs/plugins/module-plugins/) named
`gosloplint`: `golangci-lint custom` builds a golangci-lint binary with it
compiled in, and `.golangci.yml` enables it like any other linter.

1. Add a `.custom-gcl.yml` next to your `.golangci.yml`. Its top-level
   `version` is the golangci-lint release to build; the plugin's `version`
   must be a released gosloplint tag.

   ```yaml
   version: v2.12.2
   name: custom-gcl
   plugins:
     - module: github.com/chrisrob11/gosloplint
       import: github.com/chrisrob11/gosloplint/plugin
       version: v0.1.0
   ```

2. Build the binary (it needs Go and git, and writes `./custom-gcl`):

   ```sh
   golangci-lint custom
   ```

3. Enable the plugin in `.golangci.yml`:

   ```yaml
   version: "2"
   linters:
     enable:
       - gosloplint
     settings:
       custom:
         gosloplint:
           type: module
           description: Reports AI slop that compiles and passes its tests.
           settings:
             # Analyzers to turn off, by name. Default: none, so all six run.
             disable: [testhelper]
             emptyid:
               # Final name words that mark an ID; replaces the default
               # [id, ids, key, keys, name]. Each entry is one name word.
               suffixes: [id, ids, key, keys, name, ref]
             vocab:
               # Banned words; a word may have parts ("legacy client").
               # Default: none, so vocab reports nothing.
               words: [legacy client]
   ```

4. Run it in place of `golangci-lint`:

   ```sh
   ./custom-gcl run ./...
   ```

Every finding is reported under the `gosloplint` linter, its message
prefixed with the analyzer's name. Every setting is optional. A misspelt or
unknown field, an unknown analyzer in `disable`, a blank word, a suffix of
more than one word (`user_id`), and settings for a disabled analyzer are
errors when golangci-lint loads the configuration, not silently ignored.

[`example/`](example) is a working setup that builds against this checkout;
`make plugin-e2e` builds it and checks its findings against
`example/want.txt`.

## Analyzers

### ctxthread

Reports where code does not thread its `context.Context` from the caller. A
context carries the caller's deadline, cancellation and trace; code that
makes its own drops all three without a visible sign, and code that hides one
in a struct or a global, or passes values through it, hides the data flow
from the call site.

Reported:

- a call to `context.Background` or `context.TODO`;
- each use of a package-level variable initialised by one of them;
- a struct field of type `context.Context`, named or embedded;
- a call to `context.WithValue`;
- a call to `Value` on a `context.Context`.

```go
// Bad
type Store struct{ ctx context.Context }

func (s *Store) Load(id string) (*Row, error) {
	return s.db.Query(context.Background(), id)
}

// Good
func (s *Store) Load(ctx context.Context, id string) (*Row, error) {
	return s.db.Query(ctx, id)
}
```

`_test.go` files, package `main` and generated files are skipped: those are
where a context legitimately starts. The context package is matched by its
resolved import, so an aliased import is still caught, and `Value` is matched
on the `context.Context` method, so an unrelated `Value` method is not.

Flags: none.

### emptyid

Reports a guard on an empty ID whose body exits silently: `return nil`,
`return nil, nil`, a bare `return`, or `continue`. An empty ID means two
things that look the same at the guard: "absent, nothing to do" and "a caller
dropped it". A silent exit turns the second into the first, so a
mis-addressed operation reports success, and the missing value surfaces far
from where it was lost.

```go
// Bad: a dropped userID looks like "nothing to delete".
func (s *Store) DeleteSessions(ctx context.Context, userID string) error {
	if userID == "" {
		return nil
	}
	// ...
}

// Good: say so where it happened.
func (s *Store) DeleteSessions(ctx context.Context, userID string) error {
	if userID == "" {
		return errors.New("DeleteSessions: empty userID")
	}
	// ...
}
```

Every spelling of the test is read alike: `x == ""` in either order or
against a named empty constant, `len(x) == 0` (the builtin, not a local named
`len`), a `case "":` clause, a test joined to others with `&&` or `||`, and a
map entry `m["userID"]`, which is named by its key. A guard with an `else`, a
body that returns a value or an error, panics, or logs a warning or error
before leaving is not reported. `_test.go` files are skipped.

What counts as an ID is the last camelCase or snake_case word of the
expression's name, matched case-insensitively against a suffix list. Matching
whole words keeps `valid` and `grid` out, while `userID`, `userIDs`,
`order_id` and `sessionKey` match.

Flags:

- `-emptyid.suffixes` (default `id,ids,key,keys,name`): comma-separated final
  name words that mark an identifier. Setting it replaces the default list.
  Add your own domain's addressing words, for example
  `-emptyid.suffixes=id,ids,key,ref,prefix,bucket`.
  In golangci-lint it is the `emptyid.suffixes` setting.

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

### selfassert

Reports, in `_test.go` files, an assertion that compares a value to itself and
so cannot fail. It usually arrives through a rewrite: both sides of a
comparison get changed to the same spelling, and the test stays green while
the property it protected is gone.

Three shapes:

- a testify equality assertion (`Equal`, `EqualValues`, `Same`, `Exactly` and
  their `f` forms, in `assert` and `require`, as package functions or
  `*Assertions` methods) whose two compared arguments are the same expression;
- `x == x` or `x != x` (floating-point operands are skipped: `x != x` is the
  NaN test);
- two locals given the same initialiser, each assigned once, then compared,
  directly or inside larger expressions that differ only in which of the two
  they name.

```go
// Bad: both sides are one expression.
assert.Equal(t, Key("db", "users"), Key("db", "users"))

// Bad: two names for one computation.
plain := family("root", Key("db", "users"))
empty := family("root", Key("db", "users"))
assert.Equal(t, plain["cq"], empty["cq"])

// Good: the expected value is written independently of the code under test.
assert.Equal(t, "db/users", Key("db", "users"))
```

"The same expression" is decided on the printed syntax tree, so formatting
does not matter; the aliased-initialiser check follows resolved variables, so
a shadowed name is a different variable. The not-equal assertions are never reported:
`assert.NotEqual(t, newID(), newID())` is how a test says a function does not
return the same thing twice. Two initialisers that only allocate (`make`,
`new`) are not aliases: those containers are usually filled by different
routes before the comparison.

Flags: none.

### testhelper

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

Flags: none.

### vocab

Reports every identifier, declared or used, that contains a banned word. Use
it to finish a rename (a transitional alias makes stopping halfway free, and
only a count of what is left gets a rename done) or to keep a word you have
retired from coming back.

The words come from the `-vocab.words` flag. With none configured, vocab
reports nothing.

```go
// With -vocab.words=age,legacy client
type UserAge struct{ MaxAge int } // reported: UserAge, MaxAge
type LegacyClientID string        // reported
var image, manager, pageSize string // not reported: "age" is not a word in them
```

Matching is by whole camelCase or snake_case word, case-insensitively, never
by substring, so the word `age` does not match `image`. A word with several
parts, written `legacy client` or `LegacyClient`, matches those words in
sequence. Generated files are skipped; test files are not, since a rename is
not done while its tests still use the old name.

Flags:

- `-vocab.words` (default empty): comma-separated banned words.
  In golangci-lint it is the `vocab.words` setting.

## License

MIT. See [LICENSE](LICENSE).
