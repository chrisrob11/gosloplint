# steer

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
  only its own package implements). Nor is a slice, array or map of an
  interface another module declares when another module's API dictates
  it: the function's body puts values of two or more concrete types into a
  value of the result type (as composite literal elements, `append`
  arguments or index assignments, outside any func literal), or a call of
  the function is passed straight (as a struct literal field, a field
  assigned, or an argument, spread or not) to a field or parameter of
  exactly that type another module declares. `func flags() []cli.Flag`
  holding a `*cli.StringFlag` and a `*cli.BoolFlag` for `cli.Command.Flags`
  has no struct to return instead. One holding a single concrete type and
  never passed straight to such an API is still reported, and so is a lone
  `cli.Flag` result however many types it holds. The module is the one the
  driver reports (`go vet`, the multichecker, golangci-lint); a driver that
  reports none is answered from the import paths, which must share their
  first elements, up to three, to be one module.
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

In golangci-lint they are the `steer.disable` and `steer.recursion-packages`
settings.
