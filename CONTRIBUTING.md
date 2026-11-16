# Contributing

Thanks for helping with gosloplint.

## Build and test

```sh
mise install     # the Go version this repo pins
make ci          # fmt-check, lint, tests with a coverage floor, build: exactly what the ci job runs
make plugin-e2e  # builds golangci-lint with the plugin and checks its findings on example/
```

`make ci` must pass before you push. `make plugin-e2e` needs the network (it clones
golangci-lint), so it is a separate CI job; run it when you change an analyzer's findings,
`plugin/` or `example/`.

## Adding an analyzer

Each analyzer is a standard [`go/analysis`](https://pkg.go.dev/golang.org/x/tools/go/analysis)
Analyzer in its own top-level package named for it (`emptyskip/`, `vocab/`, ...):

1. **The package.** Export `Analyzer` with a `Name` equal to the package name, a `Doc` that starts
   with `<name>:`, and a `URL` of `https://github.com/chrisrob11/gosloplint#<name>`
   (`gosloplint_test.go` checks all three). The package comment says what is reported, what is
   deliberately not reported, and how a finding is suppressed, if it can be.
2. **Tests.** Use `analysistest` against `testdata/src/<pkg>`, with a `// want` comment on every
   line that should be reported, and cases for what must not be.
3. **Registration.** Add it to `Analyzers()` in `gosloplint.go`, and to the plugin in `plugin/`
   (its settings, if it has flags, and the analyzer lists in `plugin/plugin_test.go`).
4. **Docs and the example.** Add a section under "Analyzers" in the README, with its flags and
   their golangci-lint settings, and a case in `example/` with its finding in `example/want.txt`.

A false positive is worse than a missed finding: a linter people learn to ignore catches nothing.
Prefer a narrower check that is always right.

## Commits

Use [conventional commits](https://www.conventionalcommits.org/) with a lowercase subject:
`feat: add the dupl analyzer`, `fix: skip generated files in emptyid`, `docs: ...`,
`refactor: ...`, `test: ...`, `ci: ...`, `chore: ...`. Release notes are built from the `feat`
and `fix` commits.

## Pull requests

- Keep each PR small and focused on one change; split unrelated fixes into their own PRs.
- Say what changed and how you tested it (the PR template has a checklist).
- A PR needs a green `ci` check and the owner's review before it merges.

For a security problem, don't open an issue or PR: see [SECURITY.md](SECURITY.md).

## Releasing

Releases are built by GoReleaser (`.goreleaser.yaml`) from a version tag. To cut one, tag a
commit on `main` and push the tag:

```sh
git switch main && git pull
git tag v0.1.0
git push origin v0.1.0
```

The `release` workflow then builds `gosloplint` for darwin and linux (amd64 and arm64) and
publishes tar.gz archives, `checksums.txt` and a changelog of the `feat` and `fix` commits as a
GitHub release. `go install ...@latest` and the golangci-lint plugin resolve the tag from the
module proxy, so they need nothing else.

Before tagging:

- The README's golangci-lint `.custom-gcl.yml` snippet pins the plugin to a gosloplint tag
  (`version: v0.1.0`). Bump it to the tag you are about to push, in the same PR as anything
  else the release needs.
- `make release-check` validates `.goreleaser.yaml`, and
  `go run github.com/goreleaser/goreleaser/v2@latest release --snapshot --clean` builds the
  archives into `dist/` without publishing (delete `dist/` afterwards).
