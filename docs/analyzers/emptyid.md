# emptyid

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
