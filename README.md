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

### comment

Reports comment blocks longer than a limit for their kind, and packages
whose comment density runs past a ceiling. Generated prose that restates the
code costs every later reader, and an agent reading the package pays for it
on every request.

Checks:

- `length`: a comment block over its kind's limit, in lines. The kind is the
  node the block documents, read from the AST:

  | kind | documents | default limit |
  |---|---|---|
  | `package` | the package clause | 60 |
  | `file` | nothing, at the top level: a header above the package clause, a section comment between declarations | 20 |
  | `type` | a type declaration | 10 |
  | `func` | a func or method declaration | 8 |
  | `field` | a struct field or interface method | 4 |
  | `value` | a const or var declaration | 8 |
  | `inline` | nothing, inside a declaration: a comment a reader meets mid-code | 3 |

- `density`: a package whose non-test, non-generated files are more than 20%
  comment lines, over their non-blank lines. A line with code and a trailing
  comment is code. A package under 100 non-blank lines is not held to it.
  It is reported once, on the package clause of the file with the most
  comment lines, preferring a file that declares something over a `doc.go`.

```go
// Bad: eleven lines narrating what the next three say.
func (s *Store) Load(id string) (*Row, error) {
	// First we check the cache for the row. If the row is in the cache we
	// return it straight away, because ...
	// ...
	if r, ok := s.cache[id]; ok {
		return r, nil
	}
	// ...
}

// Good: the comment says what the code cannot.
func (s *Store) Load(id string) (*Row, error) {
	// The cache may hold a row deleted since; callers re-check under the lock.
	if r, ok := s.cache[id]; ok {
		return r, nil
	}
	// ...
}
```

One block that needs the room is exempted with a directive line inside it,
and the reason is required: an escape with no reason is reported, and
exempts nothing.

```go
// Grammar, one production per line:
// ...
//gosloplint:comment-ok the grammar is the spec this parser follows
```

Go directive lines (`//go:generate`, `//nolint:x`, the escape itself) are
not prose and are not counted, nor is the blank `//` gofmt puts before a
directive it moves to the end of a doc comment. Nothing exempts a package
from the density ceiling but disabling the check or raising it. Generated
files are skipped. Density is per package pass, over its non-test files, so a
package and its test variant report the same finding.

Flags:

- `-comment.disable` (default empty): comma-separated checks not to run,
  `length` and `density`.
- `-comment.package`, `-comment.file`, `-comment.type`, `-comment.func`,
  `-comment.field`, `-comment.value`, `-comment.inline` (default 0, meaning
  the limits above): the limit for each kind, in lines.
- `-comment.density` (default 0, meaning 20): the ceiling, in percent.
- `-comment.density-min-lines` (default 0, meaning 100): the package size,
  in non-blank lines, below which density is not checked.

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

### fileplace

Reports code in the wrong file, and file names and sizes that say nothing
about what the file holds: the layout generated code drifts into when each
change lands in whichever file was open.

Rules:

| rule | reports | default |
|---|---|---|
| `receiver` | a method in another type's file: a file named for a type it declares, not the method's receiver; and a test file adding a method to a non-test type | on |
| `testname` | a `_test.go` file named for no source file of its directory and no declaration of the package it tests | on |
| `grabbag` | a file named `util`, `utils`, `helper(s)`, `common`, `misc`, `types`, `base`, `impl`, `manager`, `testutil(s)` or `testhelper(s)`, alone or as an underscore-separated part | on |
| `tiny` | a non-test file under 40 lines that declares no type | on |
| `long` | a non-test file over 1500 lines | on |
| `typefile` | a type with methods declared in a file not named for it | off |
| `filename` | a non-test file named for nothing it declares | off |

```go
// Bad: shape.edge sits in perimeter.go, the file of type perimeter.
// perimeter.go
type perimeter int

func (s *shape) edge() perimeter { /* ... */ }

// Good: the method lives with its type, in shape.go, or in a file named
// for its concern (shape_edges.go, or edges.go holding no type of its own).
```

```text
Bad:  util.go, string_helpers.go, coverage_extra_test.go, twice.go (12 lines, one func)
Good: parse.go holding Parse, parse_test.go, parse_internal_test.go, doc.go
```

A file is named for a declaration when its name, lower-cased with its
underscores and any GOOS/GOARCH suffix dropped, spells the declaration's
name or a run of its camelCase words, abbreviated or plural: `volatility.go`
holds `VolatilityOf`, `refs.go` `ReferencedColumns`, `txn.go`
`Transaction`. A file named for its package or directory is named for what
it holds, and so is `errors.go` holding errors.

Not reported: generated files and `doc.go`, under every rule. `receiver`:
a method in a file named for a concern rather than a type, a method of a
type whose files pass `long-lines` in `<type>_<concern>.go`, and methods
`export_test.go` adds, which is what it is for. `testname`: `foo_test.go`
and `foo_<suffix>_test.go` for a source `foo.go`, with the suffixes
`internal`, `integration`, `external`, `bench`, `fuzz` and `example`; a
stem naming a declaration of the package (`newer_test.go` testing `Newer`);
a stem starting with the package's name; `example_test.go`,
`example_*_test.go`, `export_test.go` and `main_test.go`. `grabbag`: a part
that names a type of the package (`types.go` holding `Types`), and the
package's own name (`helpers.go` in package `helpers`). `tiny`: a package's
only file.

Scope: each rule reads one package pass. `testname` reads the names of the
non-test `.go` files in the package's directory from the directory itself,
so an external `foo_test` package, whose pass holds only its test files, is
checked against the source files beside it, and a file another platform's
build excludes still counts. The declarations it reads are those of the
package under test, which an external test package imports. A whole-file
finding is on the file's package clause.

Flags:

- `-fileplace.disable` (default empty): comma-separated rules not to run.
- `-fileplace.enable` (default empty): comma-separated off-by-default rules
  to run, `typefile` and `filename`.
- `-fileplace.tiny-lines` (default 0, meaning 40) and
  `-fileplace.long-lines` (default 0, meaning 1500): the sizes, in lines.
  `long-lines` is also the size of a type's files past which its methods may
  split into `<type>_<concern>.go`.
- `-fileplace.grab-bag` (default empty, meaning the list above):
  comma-separated file name parts, replacing the default list.
- `-fileplace.exempt` (default empty): comma-separated `rule=pattern`
  exemptions. A pattern matches a package's import path, `...` matching any
  string (`example.com/app/internal/scalar/...`), which covers its external
  test package too; or a file's base name, as a `path.Match` glob
  (`*_cases_test.go`).

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

### steer

Reports the design shapes generated code keeps reaching for: results typed
`any` or an interface, an interface with one implementation, and recursion
with no depth bound. Each finding is prefixed with its check's name, and
each message says what to write instead.

Checks:

- `returns-any`: a result of type `any` or `interface{}`, or a slice, array,
  map, channel, pointer, `iter.Seq` or `iter.Seq2` of it. Every caller has
  to type-assert it.
- `returns-interface`: a result of any other interface type, or a container
  of one. `error` is not reported, nor is an interface from the standard
  library (`io.Reader`, `context.Context`: the abstractions Go code trades
  in) or a sealed interface (one with an unexported method, so a sum type
  only its own package implements).
- `returns-one-concrete`: an interface result, the standard library's
  included, where every `return` returns the same concrete type, so the
  interface hides nothing.
- `sole-implementation`: a `New*` function returning an interface its own
  package declares that exactly one package-level type of that package
  implements.
- `unguarded-recursion`: a function, or a group of mutually recursive
  functions, in which no name says `depth`. Off by default: it runs only in
  the packages `recursion-packages` names, the ones that walk input a user
  controls.

```go
// Bad: the interface hides one type, and callers must assert to reach it.
type Store interface{ Get(id string) (Row, error) }

func NewStore(db *sql.DB) Store { return &sqlStore{db: db} }

// Good: return the struct; declare the interface where a caller accepts one.
func NewStore(db *sql.DB) *SQLStore { return &SQLStore{db: db} }
```

```go
// Bad (with recursion-packages set): deep enough input overflows the stack,
// which is fatal and cannot be recovered.
func eval(e Expr) Value { /* ... */ return eval(e.Left) }

// Good: bound the depth and refuse past it.
func eval(e Expr, depth int) (Value, error) {
	if depth > maxDepth {
		return nil, errTooDeep
	}
	// ...
}
```

A result is that of any func type: functions, methods, closures, func-typed
fields, parameters and named func types, interface methods. A signature the
package does not choose is not reported: a method whose receiver implements
an interface from another package with a method of that name
(`heap.Interface.Pop() any`), a function used as a value rather than called
(passed as a callback, stored in a field), and a func literal that is
passed, assigned to a field or returned. A method implementing an interface
of its own package is reported, and the message names the interface to
change first.

Scope: each check reads one package through its type information. The
interfaces a method may implement are those of the package and everything it
imports; an interface declared in a package that imports this one is not
seen. `sole-implementation` counts the package's own non-test types, so a
test fake is not a second implementation, and neither is a type in another
package. The recursion call graph is the package's static calls, plus, for a
call through an interface method, every method of that name whose receiver
implements the interface. Generated files are skipped.

Flags:

- `-steer.disable` (default empty): comma-separated checks not to run.
- `-steer.recursion-packages` (default empty): comma-separated import path
  patterns in which `unguarded-recursion` runs; `...` matches any string, so
  `...` is every package and `example.com/app/parse/...` a subtree.

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
