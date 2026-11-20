package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chrisrob11/gosloplint/internal/baseline"
)

// body is a run of statements long enough to be a clone at the default
// minimum.
const body = `	a := in.x + in.y*2
	if a > 10 {
		return a, nil
	}
	b := in.z + a
	for i := 0; i < b; i++ {
		a += i
	}
	if err := check(a); err != nil {
		return 0, err
	}
	b += a * 3
	return a + b, nil
}
`

const header = "package %s\n\ntype input struct{ x, y, z int }\n\nfunc check(int) error { return nil }\n"

// module writes a module whose packages a and b share a copy, c holds code
// of another shape, and c's test file a copy of a; and moves into it. a has a
// test file too, so it loads twice with -tests.
func module(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	t.Chdir(dir)
	write(t, "go.mod", "module example.com/m\n\ngo 1.22\n")
	write(t, "a/a.go", strings.ReplaceAll(header, "%s", "a")+"\nfunc First(in input) (int, error) {\n"+body)
	write(t, "b/b.go", strings.ReplaceAll(header, "%s", "b")+"\nfunc Second(in input) (int, error) {\n"+body)
	write(t, "a/a_test.go", "package a\n")
	write(t, "c/c.go", "package c\n\nfunc Sum(xs []int) int {\n\tn := 0\n\tfor _, x := range xs {\n\t\tn += x\n\t}\n\treturn n\n}\n")
	write(t, "c/c_test.go", strings.ReplaceAll(header, "%s", "c")+"\nfunc third(in input) (int, error) {\n"+body)
}

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func duplCmd(args ...string) (int, string, string) {
	var out, errOut bytes.Buffer
	code := runDupl(args, &out, &errOut)
	return code, out.String(), errOut.String()
}

func TestDuplReportsCopiesAcrossPackages(t *testing.T) {
	module(t)
	code, out, errOut := duplCmd("./...")
	want := "b/b.go:8:2: lines 8-20 duplicate lines 8-20 of a/a.go (First): move the shared statements into one function both call\n"
	if code != 1 || out != want {
		t.Fatalf("dupl ./... = %d, %q (stderr %q), want 1, %q", code, out, errOut, want)
	}
	if code, out, _ := duplCmd(); code != 1 || out != want {
		t.Errorf("dupl with no packages = %d, %q, want ./..., %q", code, out, want)
	}
}

func TestDuplIsCleanWithoutACopy(t *testing.T) {
	module(t)
	if code, out, errOut := duplCmd("./a", "./c"); code != 0 || out != "" {
		t.Errorf("dupl ./a ./c = %d, %q (stderr %q), want 0 and nothing", code, out, errOut)
	}
}

func TestDuplSettings(t *testing.T) {
	module(t)
	code, out, _ := duplCmd("-tests", "./a", "./c")
	if code != 1 || !strings.HasPrefix(out, "c/c_test.go:8:2: lines 8-20 duplicate lines 8-20 of a/a.go (First)") {
		t.Errorf("-tests ./a ./c = %d, %q, want c_test.go's copy", code, out)
	}
	for _, args := range [][]string{{"-min-nodes", "1000"}, {"-min-statements", "10"}, {"-exempt", "example.com/m/b"}, {"-exempt", "b.go"}} {
		if code, out, _ := duplCmd(args...); code != 0 || out != "" {
			t.Errorf("%v = %d, %q, want 0 and nothing", args, code, out)
		}
	}
}

func TestDuplErrors(t *testing.T) {
	module(t)
	for _, args := range [][]string{{"-nope"}, {"-min-nodes", "-1"}, {"-exempt", "[x"}, {"./missing"}} {
		if code, _, errOut := duplCmd(args...); code != 2 || errOut == "" {
			t.Errorf("%v = %d, stderr %q, want 2 and a message", args, code, errOut)
		}
	}
	write(t, "d/d.go", "package d\n\nfunc broken( {\n")
	if code, _, errOut := duplCmd("./d"); code != 2 || !strings.Contains(errOut, "d.go") {
		t.Errorf("an unparsable file = %d, stderr %q, want 2 naming it", code, errOut)
	}
}

// third writes b/b3.go, a copy of a's First and of b's Second: b then has
// two copies across packages, dupl:cross, and one within, dupl.
func third(t *testing.T) {
	t.Helper()
	write(t, "b/b3.go", "package b\n\nfunc Third(in input) (int, error) {\n"+body)
}

func writeBaseline(t *testing.T, c baseline.Counts) {
	t.Helper()
	if err := c.Write("bl.json"); err != nil {
		t.Fatal(err)
	}
}

func TestDuplBaselineHoldsEachKey(t *testing.T) {
	module(t)
	third(t)
	const (
		b      = "example.com/m/b"
		within = "b/b3.go:4:2: lines 4-16 duplicate lines 8-20 of b/b.go (Second): move the shared statements into one function both call\n"
		note   = " (2 dupl:cross findings in this package, over its baseline of 1)"
	)
	if code, out, _ := duplCmd("./..."); code != 1 || strings.Count(out, "\n") != 3 || !strings.Contains(out, within) {
		t.Fatalf("no baseline = %d, %q, want 1 and three copies, one within b", code, out)
	}
	for _, tc := range []struct {
		name   string
		counts baseline.Counts
		code   int
		lines  int
		want   string
	}{
		{"under", baseline.Counts{b: {"dupl:cross": 3, "dupl": 2}}, 0, 0, ""},
		{"at", baseline.Counts{b: {"dupl:cross": 2, "dupl": 1}}, 0, 0, ""},
		{"across over", baseline.Counts{b: {"dupl:cross": 1, "dupl": 1}}, 1, 2, "of a/a.go (First): move the shared statements into one function both call" + note + "\n"},
		{"within over", baseline.Counts{b: {"dupl:cross": 2}}, 1, 1, within},
		{"dupl does not hold copies across", baseline.Counts{b: {"dupl": 5}}, 1, 2, "of a/a.go (First): move the shared statements into one function both call\n"},
		{"another package's count", baseline.Counts{"example.com/m/a": {"dupl:cross": 5, "dupl": 5}}, 1, 3, within},
	} {
		writeBaseline(t, tc.counts)
		code, out, errOut := duplCmd("-baseline", "bl.json", "./...")
		if code != tc.code || strings.Count(out, "\n") != tc.lines || !strings.Contains(out, tc.want) {
			t.Errorf("%s: dupl -baseline = %d, %q (stderr %q), want %d, %d line(s) holding %q", tc.name, code, out, errOut, tc.code, tc.lines, tc.want)
		}
		if tc.name == "across over" && strings.Count(out, note) != 2 {
			t.Errorf("%s: %q, want both copies across noted", tc.name, out)
		}
	}
}

// TestDuplBaselineOutputIsDeterministic: with two packages over their
// counts, the copies print in the detector's order on every run.
func TestDuplBaselineOutputIsDeterministic(t *testing.T) {
	module(t)
	third(t)
	writeBaseline(t, baseline.Counts{"example.com/m/b": {"dupl:cross": 1}, "example.com/m/c": {"dupl:cross": 2}})
	_, first, _ := duplCmd("-tests", "-baseline", "bl.json", "./...")
	lines := strings.Split(strings.TrimSuffix(first, "\n"), "\n")
	if len(lines) != 6 || !strings.HasPrefix(lines[0], "b/b.go:") || !strings.HasPrefix(lines[5], "c/c_test.go:") ||
		strings.Count(first, "(3 dupl:cross findings in this package, over its baseline of 2)") != 3 {
		t.Fatalf("dupl -tests -baseline = %q, want b's three copies, then c's three, noted", first)
	}
	for range 20 {
		if _, out, _ := duplCmd("-tests", "-baseline", "bl.json", "./..."); out != first {
			t.Fatalf("a second run printed %q, the first %q", out, first)
		}
	}
}

func TestDuplBaselineErrors(t *testing.T) {
	module(t)
	write(t, "bad.json", "{")
	for _, file := range []string{"missing.json", "bad.json"} {
		if code, _, errOut := duplCmd("-baseline", file, "./..."); code != 2 || !strings.Contains(errOut, file) {
			t.Errorf("-baseline %s = %d, stderr %q, want 2 naming it", file, code, errOut)
		}
	}
}
