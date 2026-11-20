# Baseline: adopting gosloplint on existing code

A baseline holds each package at the findings it has today, so a lint can
gate new findings without fixing the old ones first. Write it with:

```sh
gosloplint baseline ./...                      # writes gosloplint-baseline.json
gosloplint baseline -o lint/baseline.json -dupl.min-nodes=80 ./...
```

It runs every analyzer over the packages and their tests, configured by the
same prefixed flags as the multichecker (pass the settings your lint runs
with, or the counts will not match), and writes a sorted JSON object keyed by
package import path, then by check: the analyzer's name, and for an analyzer
of several checks, `analyzer:check`. Regenerating it from the same code
writes the same file. It also records the copies across packages that
[`gosloplint dupl`](dupl-command.md) reports, found
with the `-dupl.` flags it is passed, as `dupl:cross` under the package of
the later copy. A copy within one package is counted once, as `dupl`, by the
analyzer: `gosloplint dupl -baseline` holds those against the same `dupl`
count, so the two never count one pair twice.

```json
{
  "example.com/app/internal/store": {
    "dupl": 2,
    "dupl:cross": 1,
    "fileplace:tiny": 1,
    "steer:returns-interface": 3
  }
}
```

Then run with it:

```sh
gosloplint -baseline gosloplint-baseline.json ./...
gosloplint dupl -baseline gosloplint-baseline.json ./...
```

For each package and check, an analyzer reports nothing while the package
has no more findings of that check than the baseline records, and every one
of them once it has more, each noting the baseline: `(3 dupl findings in this
package, over its baseline of 2)`. It counts findings rather than naming
them, so it survives edits that move code, and fixing one finding leaves
room for a new one until the baseline is regenerated: regenerate it after
each fix to ratchet it down. A package or check it does not list has a
baseline of 0. A package and its test variant are held to one count, the
larger. The path is relative to the working directory; under `go vet`, which
runs the tool in each package's directory, give an absolute path.

In golangci-lint it is the top-level `baseline` setting of the
[plugin's settings](golangci-lint.md). golangci-lint
users can instead use its own
[`issues.new-from-merge-base`](https://golangci-lint.run/docs/configuration/file/),
which reports only the findings a branch adds, with no file to keep.
