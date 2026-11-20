# ctxthread

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
