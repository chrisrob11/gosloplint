# vocab

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
