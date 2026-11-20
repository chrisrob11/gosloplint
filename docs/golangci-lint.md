# The golangci-lint plugin

gosloplint is a golangci-lint
[module plugin](https://golangci-lint.run/docs/plugins/module-plugins/) named
`gosloplint`: `golangci-lint custom` builds a golangci-lint binary with it
compiled in, and `.golangci.yml` enables it like any other linter.

1. Add a `.custom-gcl.yml` next to your `.golangci.yml`. Its top-level
   `version` is the golangci-lint release to build; the plugin's `version`
   must be a released gosloplint tag.

   ```yaml
   version: v2.12.2
   name: custom-gcl
   plugins:
     - module: github.com/chrisrob11/gosloplint
       import: github.com/chrisrob11/gosloplint/plugin
       version: v0.1.0
   ```

2. Build the binary (it needs Go and git, and writes `./custom-gcl`):

   ```sh
   golangci-lint custom
   ```

3. Enable the plugin in `.golangci.yml`:

   ```yaml
   version: "2"
   linters:
     enable:
       - gosloplint
     settings:
       custom:
         gosloplint:
           type: module
           description: Reports AI slop that compiles and passes its tests.
           settings:
             # A file written by gosloplint baseline; see baseline.md.
             # Default: none.
             baseline: gosloplint-baseline.json
             # Analyzers to turn off, by name. Default: none, so all twelve run.
             disable: [testhelper]
             cannotfail:
               # Checks to turn off, and off-by-default checks to turn on.
               disable: [dead-err]
               enable: [empty-loop]
               # Functions that fail a test without being passed one.
               assert-funcs: [example.com/app/testkit.Expect]
             comment:
               # Checks to turn off: length, density.
               disable: [density]
               # Per-kind block limits, in lines; an unset kind keeps its
               # default (package 60, file 20, type 10, func 8, field 4,
               # value 8, inline 3).
               limits: {func: 12, inline: 4}
               density: 25            # percent; default 20
               density-min-lines: 200 # default 100
             dupl:
               min-nodes: 80      # default 60
               min-statements: 3  # default 1
               tests: true        # default false
               # Import path patterns or file name globs not read.
               exempt: [example.com/app/internal/gen/..., "*_cases.go"]
             emptyid:
               # Final name words that mark an ID; replaces the default
               # [id, ids, key, keys, name]. Each entry is one name word.
               suffixes: [id, ids, key, keys, name, ref]
             fileplace:
               # Rules to turn off, and off-by-default rules to turn on.
               disable: [tiny]
               enable: [typefile]
               tiny-lines: 30    # default 40
               long-lines: 2000  # default 1500
               # Replaces the default grab-bag name parts.
               grab-bag: [util, utils, helpers, misc]
               # Per rule, import path patterns or file name globs it skips.
               exempt:
                 testname: [example.com/app/internal/sqltest/...]
                 receiver: ["*_gen.go"]
             narration:
               # Checks to turn off: history, warnsign.
               disable: [warnsign]
             steer:
               # Checks to turn off.
               disable: [returns-interface]
               # Packages unguarded-recursion reads; default none.
               recursion-packages: [example.com/app/parse/...]
             vocab:
               # Banned words; a word may have parts ("legacy client").
               # Default: none, so vocab reports nothing.
               words: [legacy client]
   ```

4. Run it in place of `golangci-lint`:

   ```sh
   ./custom-gcl run ./...
   ```

Every finding is reported under the `gosloplint` linter, its message
prefixed with the analyzer's name, and for cannotfail, steer, comment,
fileplace and narration, with the check's. Every setting is optional, and a zero number means its
default. A missing or malformed baseline file, a misspelt or unknown field,
an unknown analyzer, check or rule, a
blank word, a suffix of more than one word (`user_id`), a negative or
out-of-range number, a malformed exemption pattern, and settings for a
disabled analyzer are errors when golangci-lint loads the configuration, not
silently ignored.

[`example/`](../example) is a working setup that builds against this checkout;
`make plugin-e2e` builds it and checks its findings against
`example/want.txt`.
