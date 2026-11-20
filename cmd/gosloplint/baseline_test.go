package main

import (
	"bytes"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/chrisrob11/gosloplint"
	"github.com/chrisrob11/gosloplint/internal/baseline"
)

func TestBaselineWritesThenFilters(t *testing.T) {
	bin := build(t)
	module(t)
	write(t, "b/b2.go", "package b\n\nfunc Third(in input) (int, error) {\n"+body)
	// b loads twice, with and without its test file; its count is one.
	write(t, "b/b_test.go", "package b\n")
	var out, errOut bytes.Buffer
	if code := runBaseline(gosloplint.Analyzers(), []string{"-o", "bl.json", "./..."}, &out, &errOut); code != 0 {
		t.Fatalf("baseline = %d: %s%s", code, out.String(), errOut.String())
	}
	if !strings.HasPrefix(out.String(), "gosloplint baseline: bl.json records ") {
		t.Errorf("baseline printed %q", out.String())
	}
	counts, err := baseline.Read("bl.json")
	if err != nil {
		t.Fatal(err)
	}
	if n := counts["example.com/m/b"]["dupl"]; n != 1 {
		t.Errorf("b's dupl count = %d, want 1 (b2.go copies b.go): %v", n, counts)
	}
	if again := run(t, bin, "baseline", "-o", "again.json", "./..."); again.code != 0 || read(t, "again.json") != read(t, "bl.json") {
		t.Errorf("a second baseline differs: %s\n%s", read(t, "again.json"), read(t, "bl.json"))
	}

	if r := run(t, bin, "./..."); r.code != 3 {
		t.Fatalf("no baseline = %d, want 3: %s", r.code, r.out)
	}
	if r := run(t, bin, "-baseline", "bl.json", "./..."); r.code != 0 || r.out != "" {
		t.Errorf("at the baseline = %d, %q, want 0 and nothing", r.code, r.out)
	}

	write(t, "b/b2.go", "package b\n\nfunc Third(in input) (int, error) {\n"+body+"\nfunc Fourth(in input) (int, error) {\n"+body)
	r := run(t, bin, "-baseline", "bl.json", "./...")
	if r.code != 3 || strings.Count(r.out, "(3 dupl findings in this package, over its baseline of 1)") != 3 || strings.Contains(r.out, "fileplace") {
		t.Errorf("over the baseline = %d, %q, want b's three dupl findings, noted, and nothing else: b2.go is still b's one tiny file", r.code, r.out)
	}
	if r := run(t, bin, "-baseline", "missing.json", "./..."); r.code == 0 || !strings.Contains(r.out, "missing.json") {
		t.Errorf("a missing baseline = %d, %q, want an error naming it", r.code, r.out)
	}
}

func TestBaselineTakesTheAnalyzerFlags(t *testing.T) {
	bin := build(t)
	module(t)
	write(t, "b/b2.go", "package b\n\nfunc Third(in input) (int, error) {\n"+body)
	if r := run(t, bin, "baseline", "-dupl.min-nodes", "1000", "./..."); r.code != 0 {
		t.Fatalf("baseline -dupl.min-nodes 1000 = %d: %s", r.code, r.out)
	}
	if strings.Contains(read(t, baseline.DefaultFile), `"dupl`) {
		t.Errorf("-dupl.min-nodes 1000: %s records dupl", read(t, baseline.DefaultFile))
	}
	for _, args := range [][]string{{"baseline", "-nope"}, {"baseline", "./missing"}, {"baseline", "-o", "no/such/dir/b.json", "./..."}} {
		if r := run(t, bin, args...); r.code != 2 {
			t.Errorf("%v = %d, want 2: %s", args, r.code, r.out)
		}
	}
}

type result struct {
	code int
	out  string
}

func run(t *testing.T, bin string, args ...string) result {
	t.Helper()
	out, err := exec.Command(bin, args...).CombinedOutput()
	return result{exitCode(err), string(out)}
}

func read(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// TestBaselineRecordsCopiesAcrossPackages: the copies across packages are
// counted as dupl:cross, a copy within one package only as dupl, so dupl
// -baseline is clean on the file the baseline wrote.
func TestBaselineRecordsCopiesAcrossPackages(t *testing.T) {
	bin := build(t)
	module(t)
	third(t)
	if r := run(t, bin, "baseline", "-o", "bl.json", "./..."); r.code != 0 {
		t.Fatalf("baseline = %d: %s", r.code, r.out)
	}
	counts, err := baseline.Read("bl.json")
	if err != nil {
		t.Fatal(err)
	}
	if b := counts["example.com/m/b"]; b["dupl:cross"] != 2 || b["dupl"] != 1 {
		t.Errorf("b's counts = %v, want dupl:cross 2 and dupl 1", b)
	}
	if c := counts["example.com/m/c"]; c["dupl:cross"] != 0 {
		t.Errorf("c's counts = %v, want no dupl:cross without -dupl.tests", c)
	}
	if r := run(t, bin, "dupl", "-baseline", "bl.json", "./..."); r.code != 0 || r.out != "" {
		t.Errorf("dupl -baseline = %d, %q, want 0 and nothing", r.code, r.out)
	}

	if r := run(t, bin, "baseline", "-o", "tests.json", "-dupl.tests", "./..."); r.code != 0 {
		t.Fatalf("baseline -dupl.tests = %d: %s", r.code, r.out)
	}
	if counts, err = baseline.Read("tests.json"); err != nil {
		t.Fatal(err)
	}
	if c := counts["example.com/m/c"]; c["dupl:cross"] != 3 {
		t.Errorf("-dupl.tests: c's counts = %v, want dupl:cross 3, c_test.go's copies", c)
	}
	if r := run(t, bin, "dupl", "-tests", "-baseline", "tests.json", "./..."); r.code != 0 || r.out != "" {
		t.Errorf("dupl -tests -baseline = %d, %q, want 0 and nothing", r.code, r.out)
	}
}
