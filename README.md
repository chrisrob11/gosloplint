# gosloplint

Go analyzers that catch AI slop: code that compiles, passes its tests and
reads as finished, while being wrong or doing nothing. A test whose whole
body is a skip, an assertion that compares a value to itself, a guard that
quietly returns when an ID it needs is missing. Each looks right in a diff,
so review lets it through, and there is more of it than a reviewer can read
closely. An analyzer reads every line on every run, and says what to write
instead.

```go
func (s *Store) DeleteSessions(ctx context.Context, userID string) error {
	if userID == "" {
		return nil
	}
	// ...
}
```

```text
$ gosloplint ./...
store.go:12:2: guard on empty userID exits silently (return nil): return an error or log, so a dropped ID is not reported as success
```

Every check is a standard
[`go/analysis`](https://pkg.go.dev/golang.org/x/tools/go/analysis) Analyzer,
so it runs in the tools you already use: `go vet`, gopls, and golangci-lint.

## Install

```sh
go install github.com/chrisrob11/gosloplint/cmd/gosloplint@latest
```

## Quick start

Through `go vet`:

```sh
go vet -vettool=$(which gosloplint) ./...
```

The binary takes the standard analysis driver flags (`-help` lists them);
each analyzer's own flags are prefixed with its name:

```sh
gosloplint ./...
gosloplint -emptyskip ./...    # only the named analyzers
```

In golangci-lint, as a [module plugin](docs/golangci-lint.md): add a
`.custom-gcl.yml`, build `./custom-gcl` with `golangci-lint custom`, enable
`gosloplint` in `.golangci.yml`, and run `./custom-gcl run ./...`.

```yaml
# .custom-gcl.yml
version: v2.12.2
name: custom-gcl
plugins:
  - module: github.com/chrisrob11/gosloplint
    import: github.com/chrisrob11/gosloplint/plugin
    version: v0.1.0
```

```yaml
# .golangci.yml
version: "2"
linters:
  enable:
    - gosloplint
  settings:
    custom:
      gosloplint:
        type: module
        description: Reports AI slop that compiles and passes its tests.
```

[docs/golangci-lint.md](docs/golangci-lint.md) lists every setting.

## Analyzers

| analyzer | reports |
|---|---|
| [cannotfail](docs/analyzers/cannotfail.md) | tests that cannot fail: literal-only assertions, tests and subtests with no way to fail, error values never read |
| [comment](docs/analyzers/comment.md) | comment blocks over a per-kind line limit, and packages past a comment-density ceiling |
| [ctxthread](docs/analyzers/ctxthread.md) | a `context.Context` made from nothing, stored in a struct or global, or used to pass values |
| [dupl](docs/analyzers/dupl.md) | copied code within a package: statement runs of one shape, names and literal values aside |
| [emptyid](docs/analyzers/emptyid.md) | a guard on an empty ID whose body exits silently |
| [emptyskip](docs/analyzers/emptyskip.md) | a test or subtest whose whole body is a skip |
| [fileplace](docs/analyzers/fileplace.md) | code in the wrong file, and file names and sizes that say nothing about what a file holds |
| [narration](docs/analyzers/narration.md) | comments that narrate how the code came to be, and a second warning marker in one declaration |
| [selfassert](docs/analyzers/selfassert.md) | a test assertion that compares a value to itself |
| [steer](docs/analyzers/steer.md) | results typed `any` or an interface, an interface with one implementation, recursion with no depth bound, a second name for one function |
| [testhelper](docs/analyzers/testhelper.md) | a `must*` test helper that hides an error from the call site |
| [vocab](docs/analyzers/vocab.md) | identifiers that contain a banned word |

## Suppressing one finding

A finding that is wrong at one site is suppressed there, with a required
reason, on the finding's line or the line above it:

```go
//gosloplint:ignore steer:returns-interface the plugin API takes a Handler
func handler() Handler { /* ... */ }
```

The syntax is `//gosloplint:ignore <analyzer>[:<check>] <reason>`. It works
in the binary, `go vet -vettool` and the plugin. A directive with no reason,
an unknown name, or no finding to suppress is itself reported.
[docs/ignore.md](docs/ignore.md) has the full rules.

## Baselines

A baseline holds each package at the findings it has today, so a lint gates
only new ones:

```sh
gosloplint baseline ./...                      # writes gosloplint-baseline.json
gosloplint -baseline gosloplint-baseline.json ./...
```

It counts findings per package and check rather than naming them;
regenerate it after each fix to ratchet it down.
[docs/baseline.md](docs/baseline.md) covers the file and how it counts.

## Copies across packages

An analysis pass sees one package, so the dupl analyzer cannot see a copy
between two packages. `gosloplint dupl` loads them all at once and runs the
same clone detector:

```sh
gosloplint dupl ./...
gosloplint dupl -tests -min-nodes 80 ./internal/...
```

It exits 1 on findings, 0 when clean and 2 on an error. `-baseline file`
holds each package to its recorded `dupl:cross` count, which `gosloplint
baseline` writes. [docs/dupl-command.md](docs/dupl-command.md) has its flags,
output and the `dupl:cross` ignore directive.

## Releases

Each `vX.Y.Z` tag publishes a
[GitHub release](https://github.com/chrisrob11/gosloplint/releases) with
`gosloplint` binaries for darwin and linux (amd64 and arm64), as tar.gz
archives with a `checksums.txt` and the docs, and a changelog.
`go install ...@latest` and the golangci-lint plugin's `version` take the
same tags.

## Development

```sh
make ci          # fmt-check, lint, tests with a coverage floor, build: what the ci job runs
make plugin-e2e  # golangci-lint with the plugin built in, checked against example/want.txt
```

[CONTRIBUTING.md](CONTRIBUTING.md) covers adding an analyzer, commits, pull
requests and releasing. Report a vulnerability privately as
[SECURITY.md](SECURITY.md) describes, not in an issue.

## License

MIT. See [LICENSE](LICENSE).
