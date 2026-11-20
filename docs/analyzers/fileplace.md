# fileplace

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
underscores and any GOOS/GOARCH suffix (or `_unix`, `_other`) dropped, spells the declaration's
name or a run of its camelCase words, abbreviated or plural: `volatility.go`
holds `VolatilityOf`, `refs.go` `ReferencedColumns`, `txn.go`
`Transaction`. A file named for its package or directory is named for what
it holds, and so is `errors.go` holding errors.

Not reported: generated files and `doc.go`, under every rule. `receiver`:
a method in a file named for a concern rather than a type, a method of a
type whose files pass `long-lines` in `<type>_<concern>.go`, and methods
`export_test.go` adds, which is what it is for. `testname`: `foo_test.go`
and `foo_<suffix>_test.go` for a source `foo.go`, with the suffixes
`internal`, `integration`, `external`, `bench`, `fuzz` and `example`;
`foo_<aspect>_test.go` for a source `foo.go` other than the one named for
the package (`root_special_unix_test.go` beside `root.go`); a
stem naming a declaration of the package (`newer_test.go` testing `Newer`);
the package's general test file, `<pkg>_test.go`, or one named for the
directory, alone or with one of those suffixes; `<pkg>_<x>_test.go` only
when `<x>` names a source file of the directory or a declaration of the
package, and in package `main` also a subcommand: a string `case` of a
switch on an argument (`os.Args[1]`, `args[0]`, `fs.Arg(0)`), or the first
word of the `Name` or `Use` of a composite literal of a type named
`Command` (urfave/cli, cobra); `example_test.go`, `example_*_test.go`,
`export_test.go` and `main_test.go`; and every test file of a directory
with no non-test `.go` file, an integration suite named for what it
exercises. So `main_ask_test.go` passes in a
command with an `ask` subcommand or an `askCmd` function, and
`gadget_extra_test.go` names nothing. `grabbag`: a
part that names a type of the package (`types.go` holding `Types`), the
package's own name (`helpers.go` in package `helpers`), and a stem of two or
more parts that spells a declaration of the package under test, the word
then being a domain term (`helper_config.go` and `helper_config_test.go`
for `HelperConfig`, `base_repo.go` for `BaseRepoFunc`). `tiny`: a package's
only file, and a file built for some platforms only: one with a
`//go:build` line or a GOOS/GOARCH, `_unix` or `_other` name suffix.

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

In golangci-lint they are `fileplace.disable`, `fileplace.enable`,
`fileplace.tiny-lines`, `fileplace.long-lines`, `fileplace.grab-bag` and
`fileplace.exempt`, a map from rule to a list of patterns.
