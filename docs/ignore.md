# Suppressing one finding: `//gosloplint:ignore`

A finding that is wrong at one site is suppressed there with a directive,
and the reason is required:

```go
//gosloplint:ignore steer:returns-interface the plugin API takes a Handler
func handler() Handler { /* ... */ }

var userID string //gosloplint:ignore emptyid an empty ID is every user
```

The syntax is `//gosloplint:ignore <analyzer>[:<check>] <reason>`, with no
space after `//`. `<analyzer>` is an analyzer's name; `<check>`, for
cannotfail, comment, dupl, fileplace, narration and steer, narrows it to one of their checks (the
`analyzer:check` names of the [baseline](baseline.md);
dupl's one, `cross`, is the copies across packages that only `gosloplint dupl` reports),
and without it every check of the analyzer is suppressed. A directive
covers:

- its own line, so it can trail the code it is about;
- the line below it, so it can sit above a statement or a declaration;
- in a declaration's doc comment (a func, a `type`, `var` or `const`
  declaration, or one spec of a group), the whole declaration with its
  doc. gofmt moves a directive line to the end of a doc comment, so it
  always sits right above the declaration.

A finding on a file's package clause (fileplace, comment's density) takes
the directive on the line above `package`. A dupl finding is reported at
the later copy, so the directive goes there.

A directive that cannot be trusted is itself reported, with the prefix
`ignore:`, so suppressions do not rot silently:

- one with no reason, which then suppresses nothing;
- one naming no analyzer, an unknown analyzer, or a check its analyzer
  does not have;
- one that suppresses no finding (stale): delete it. Stale is judged
  against the configuration the run has: a directive for a check that is
  disabled, or for a finding that only another setting produces, is stale
  there. A directive for an analyzer that is not running is not judged.

Each analyzer reports the directives naming it; one naming no known
analyzer is reported by the first analyzer of the set, comment unless the
plugin disables it, so a run selecting analyzers by flag (`gosloplint
-steer`) without comment leaves those unreported. Directives in generated files are not read. `gosloplint
dupl` honours `dupl` and `dupl:cross` directives too, and the baseline
counts only the findings no directive suppresses.

It is implemented once, as a wrapper around every analyzer's run, so it
works wherever gosloplint runs: the binary, `go vet -vettool`, and the
golangci-lint plugin. golangci-lint's own `//nolint:gosloplint` also works
in the plugin, but only there: `go vet` and the binary do not read it, it
needs no reason, and it suppresses every gosloplint analyzer at once.

Two other spellings are aliases: `//gosloplint:comment-ok
<reason>` inside a comment block exempts that block from comment's
`length` check, and `//gosloplint:dupl-ok <reason>` in a function's doc
comment leaves the function out of dupl's comparison entirely, on both
sides of a copy. They are reported when they have no reason, but not when
stale.
