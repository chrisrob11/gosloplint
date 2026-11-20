# comment

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
exempts nothing. It is an alias of the general
[`//gosloplint:ignore comment:length <reason>`](../ignore.md),
which works on a doc comment too, and is reported when stale.

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

In golangci-lint they are `comment.disable`, `comment.limits.<kind>`,
`comment.density` and `comment.density-min-lines`.
