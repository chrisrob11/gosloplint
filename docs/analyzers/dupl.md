# dupl

Reports copied code within a package: two runs of statements with the same
syntax tree shape, names and literal values aside, long enough that one
function both call was the better choice. Generated code copies whole
blocks where a person would have extracted a helper, and each copy then
drifts on its own.

```go
// Bad: Monthly is Weekly with the names changed.
func (t Totals) Weekly(days []int) int {
	sum, peak := 0, 0
	for _, d := range days {
		// ...
	}
	return sum / peak
}

func (t Totals) Monthly(weeks []int) int {
	total, top := 0, 0
	for _, w := range weeks {
		// ...
	}
	return total / top
}

// Good: one function, called twice.
func (t Totals) Weekly(days []int) int   { return t.clampedMean(days) }
func (t Totals) Monthly(weeks []int) int { return t.clampedMean(weeks) }
```

A clone is reported once per pair of functions, at the later occurrence (by
file name, then line), naming the lines of the earlier one: `lines 31-48
duplicate lines 8-25 of totals.go (Totals.Weekly)`. Each side's lines cover
every run the pair shares, except that the two sides never overlap: within
one function, a run that would widen one side into the other is left out.

How code is compared: each statement is reduced to the sequence of its
syntax node kinds, keeping every operator, assignment and branch token, and
the kind of every literal, but no identifier name or literal value. So
`a := x + 1` and `total := y + 2` match, while `a := x - 1`, `a = x + 1` and
`a := x + "1"` do not. A run is consecutive statements of one block (a
function body, a nested block, a closure, a `case` body) whose nodes reach
the minimum; two runs match when their statement shapes are equal in order.

Not reported: a composite literal, compared as one node whatever it holds,
since two tables of one shape are data, not a decision made twice; a
switch's or select's list of clauses, a dispatch table (each clause body is
still compared on its own); a run of flag registrations, routes or setters
that only shares its shape: a run whose every statement is a bare call, its
arguments each a name, a literal, a field or its address, or a composite or
function literal, or such a call guarded by `if err := f(x); err != nil {
return err }`, is a copy only when it is also spelled alike, the same
literal values and called names (and, with no literal in a statement, the
same names), since any two such runs of one length match on node kinds; a
statement repeated in place, two overlapping runs of one block; generated files; `_test.go` files unless `tests` is set,
since test tables and setup repeat by design. A finding is suppressed with
[`//gosloplint:ignore dupl <reason>`](../ignore.md)
at the later copy. The alias `dupl-ok` leaves a function out of the comparison
entirely, from a line in its doc comment, and the reason is required:

```go
//gosloplint:dupl-ok the two drivers must stay independent for the benchmark
func (d *Driver) replay() { /* ... */ }
```

Scope: one package pass, so a copy between two packages is not seen; run
[`gosloplint dupl`](../dupl-command.md) for those,
whose check and baseline key is `dupl:cross`.

Flags:

- `-dupl.min-nodes` (default 0, meaning 60): the size a run must reach, in
  syntax nodes, about fifteen lines of ordinary Go. Below 60, route tables
  and repeated setup calls start to match.
- `-dupl.min-statements` (default 0, meaning 1): the length a run must reach,
  in statements.
- `-dupl.tests` (default false): read `_test.go` files too.
- `-dupl.exempt` (default empty): comma-separated patterns of what is not
  read: a package's import path, `...` matching any string, which covers its
  external test package too; or a file's base name, as a `path.Match` glob.

In golangci-lint they are `dupl.min-nodes`, `dupl.min-statements`,
`dupl.tests` and `dupl.exempt`, a list.
