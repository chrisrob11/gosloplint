# narration

Reports comments that narrate how the code came to be instead of saying what
it is. "This now uses a map" tells a reader about a version of the code they
cannot see; the history belongs in the commit message, and the comment should
say what the code does and why.

Checks:

- `history`: a comment block whose prose, joined across its lines and read
  case aside, holds a phrase of narration. Each phrase is anchored so that the
  words alone do not report: a subject that is the code, or a change verb.

  | phrase | reported | not reported |
  |---|---|---|
  | `this`/`this <code noun>` + `now`, `no longer`, `used to` | this now returns, this function no longer locks | the cache now holds n entries; the lock is no longer held |
  | `this`/`this <noun>`/`the <noun>` + `was`/`has been`/`got` + changed, switched, updated, converted, renamed, moved, replaced, migrated + to/from/into/with/by/here | this function was changed to take ctx | the value was changed to new |
  | `was`/`has been` + refactored, rewritten, reworked | the code was refactored | |
  | `we`/`it used to` + a verb (be, return, call, take, use, ...) | it used to return a slice | the columns used to compute the hash |
  | `now` + uses, delegates, lives, relies on, goes through | Get now uses the cache | |
  | `we` + replaced, refactored, rewrote, reworked, renamed, switched from/to/over, used to, changed this/it/the noun to | we replaced the mutex | we change the key to lower case |
  | `previously` opening a sentence, then a comma or a subject | Previously, it panicked | the previously stored value |
  | after, before, since, during, in, with + the/this/that/my/our + refactor, rewrite, rework, cleanup, ending the clause | simpler after the refactor | after the rewrite rules run; after the cleanup goroutine exits; if it fails after a refactor (no particular change) |
  | per/addressing review, requested/suggested in review, review feedback | per review | the documents waiting for review |
  | fix(es/ed) the/a bug/issue/regression where/that/when/with | this fixes the bug where | the test for the bug where |
  | renamed from, moved here from, formerly called/named/known as | Join was renamed from Concat | rows moved from the source table |
  | a change verb + in/by/for + `#123`/`PR 123`, or `#123` + a change verb | added in #123, (#456) changed | see #123 for why; added in Go 1.21 |
  | the old/previous/original + implementation, approach, code path | the old implementation | the previous value |
  | added/introduced to fix, address, work around | added to fix a flaky test | |

  The code nouns are function, func, method, type, struct, package, code,
  test, helper, check, logic, implementation, analyzer, handler and wrapper.
  One finding per block, at the line its earliest phrase starts on. A
  paragraph from a line opening with `Deprecated:` to the next blank line is
  godoc's deprecation notice, which says what changed by design ("Deprecated:
  this no longer has any effect"), so it is not read; the rest of the block is.

- `warnsign`: a second warning marker in one declaration. A marker is ⚠
  anywhere on a line, or `WARNING:`, `IMPORTANT:`, `NOTE:`, `CAUTION:`,
  `NB:` (or `Warning:`, `Important:`, `Note:`, `Caution:`, bold or not)
  opening one. One marker can say "a reader must not miss this"; a marker on
  every comment marks none. So each unit may hold one, and every marker after
  the first is reported. A unit is the innermost of: a top-level declaration,
  its doc and body together; one spec of a grouped `type ( ... )`,
  `const ( ... )` or `var ( ... )`, with its doc and line comment (a comment
  between specs belongs to the group); one field of a struct type or one
  method of an interface type, with its doc and line comment, named
  `Type.Field`. Each documents a separate thing a reader looks up on its own,
  so a marker on each is not overuse. A function body is one unit with its
  doc, struct literals and test tables included. A comment block outside
  every declaration is its own unit. A single marker is never reported:
  `Note:` is ordinary Go documentation.

```go
// Bad: the comment is a changelog.
// Get now uses the cache; previously, it went to the store (#123) changed.
func (s *Store) Get(id string) (*Row, error)

// Good: what it does, and why.
// Get reads the cache first; a miss goes to the store and fills the cache.
func (s *Store) Get(id string) (*Row, error)
```

Only comments are read, never string literals. Go directive lines
(`//go:build`, `//nolint:x`, `//gosloplint:ignore`) are not prose, and a
comment above the package clause that is not its doc (a license header, a
build constraint's note), or any block naming a copyright or an SPDX license,
is skipped. Generated files are skipped. `history` skips text inside a closed
`"…"`, `` `…` `` or `“…”` pair, so a quoted example such as a reply format
(`"#3: renamed it"`) is not narration; an unclosed quote hides nothing.
Test files are read like any other:
a test's "the bug where" or "regression for #123" says what it guards, and
needs a change verb before it is narration.

Flags:

- `-narration.disable` (default empty): comma-separated checks not to run,
  `history` and `warnsign`. In golangci-lint it is the `narration.disable`
  setting.
