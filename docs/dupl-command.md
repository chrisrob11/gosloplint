# Copies across packages: `gosloplint dupl`

Every analyzer reads one package at a time, so it fits the tools you
already run. One check cannot be: a copy that spans two packages. An
analysis pass sees one package, and facts flow only along import edges, so
two packages that do not import each other never meet. `gosloplint dupl` is
the one check that is not an analyzer: it loads the packages with
`go/packages` and runs the [dupl](analyzers/dupl.md) analyzer's clone detector over all
of them at once.

```sh
gosloplint dupl ./...
gosloplint dupl -tests -min-nodes 80 ./internal/...
```

It prints each finding as `file:line:col: message`, with paths relative to
the working directory, and exits 1 on findings, 0 when clean and 2 on an
error. It takes the dupl flags, unprefixed: `-min-nodes`, `-min-statements`,
`-tests` and `-exempt`. A copy within one package is reported both here and
by the dupl analyzer. The package patterns default to `./...`.

Each copy belongs to the package of its later copy, where it is reported. A
copy whose earlier copy is in another package is of the check `dupl:cross`;
one whose two copies share a package is the dupl analyzer's own, `dupl`.
`-baseline file` holds each package to the counts of both in a
[baseline](baseline.md) file, as the
analyzers do: nothing while a package has no more copies of a check than
its count, every one of them, noted, once it has more:

```sh
gosloplint baseline ./...                      # records dupl and dupl:cross
gosloplint dupl -baseline gosloplint-baseline.json ./...
```

`//gosloplint:ignore dupl:cross <reason>` at the later copy suppresses a
copy across packages. Plain `dupl` there suppresses it too, but the dupl
analyzer, which never sees that copy, then reports the directive as stale;
the analyzer accepts `dupl:cross` and never judges it, since no analysis
pass reports that check. A `dupl:cross` directive does not suppress a copy
within the package.

This command is not part of the golangci-lint plugin. golangci-lint runs a
plugin's analyzers one package at a time, the same limit an analysis pass
has, so the plugin's dupl sees only copies within a package; run
`gosloplint dupl` as its own step to gate the rest.
