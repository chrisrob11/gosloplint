# selfassert

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
  they name, when nothing between could change either: the two definitions
  share a block, and every statement between them and the comparison (and
  before it in any block nested between) has no call but to a builtin or a
  conversion, no receive, no address taken and no assignment to either, and
  is not a compound statement. A call between makes the pair a before/after
  snapshot (`before := checksum(f); rewrite(f); after := checksum(f)`), which
  is a real assertion; so is a pair defined in different blocks, as one
  taken outside a subtest and one inside it.

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
