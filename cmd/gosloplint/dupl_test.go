package main

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
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

func dupl(args ...string) (int, string, string) {
	var out, errOut bytes.Buffer
	code := runDupl(args, &out, &errOut)
	return code, out.String(), errOut.String()
}

func TestDuplReportsCopiesAcrossPackages(t *testing.T) {
	module(t)
	code, out, errOut := dupl("./...")
	want := "b/b.go:8:2: lines 8-20 duplicate lines 8-20 of a/a.go (First): move the shared statements into one function both call\n"
	if code != 1 || out != want {
		t.Fatalf("dupl ./... = %d, %q (stderr %q), want 1, %q", code, out, errOut, want)
	}
	if code, out, _ := dupl(); code != 1 || out != want {
		t.Errorf("dupl with no packages = %d, %q, want ./..., %q", code, out, want)
	}
}

func TestDuplIsCleanWithoutACopy(t *testing.T) {
	module(t)
	if code, out, errOut := dupl("./a", "./c"); code != 0 || out != "" {
		t.Errorf("dupl ./a ./c = %d, %q (stderr %q), want 0 and nothing", code, out, errOut)
	}
}

func TestDuplSettings(t *testing.T) {
	module(t)
	code, out, _ := dupl("-tests", "./a", "./c")
	if code != 1 || !strings.HasPrefix(out, "c/c_test.go:8:2: lines 8-20 duplicate lines 8-20 of a/a.go (First)") {
		t.Errorf("-tests ./a ./c = %d, %q, want c_test.go's copy", code, out)
	}
	for _, args := range [][]string{{"-min-nodes", "1000"}, {"-min-statements", "10"}, {"-exempt", "example.com/m/b"}, {"-exempt", "b.go"}} {
		if code, out, _ := dupl(args...); code != 0 || out != "" {
			t.Errorf("%v = %d, %q, want 0 and nothing", args, code, out)
		}
	}
}

func TestDuplErrors(t *testing.T) {
	module(t)
	for _, args := range [][]string{{"-nope"}, {"-min-nodes", "-1"}, {"-exempt", "[x"}, {"./missing"}} {
		if code, _, errOut := dupl(args...); code != 2 || errOut == "" {
			t.Errorf("%v = %d, stderr %q, want 2 and a message", args, code, errOut)
		}
	}
	write(t, "d/d.go", "package d\n\nfunc broken( {\n")
	if code, _, errOut := dupl("./d"); code != 2 || !strings.Contains(errOut, "d.go") {
		t.Errorf("an unparsable file = %d, stderr %q, want 2 naming it", code, errOut)
	}
}

// TestTheBinaryDispatches builds gosloplint: dupl as the first argument is
// the subcommand, anything else the multichecker, unchanged.
func TestTheBinaryDispatches(t *testing.T) {
	bin := filepath.Join(t.TempDir(), "gosloplint")
	if out, err := exec.Command("go", "build", "-o", bin, ".").CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}
	module(t)
	write(t, "b/b2.go", "package b\n\nfunc Third(in input) (int, error) {\n"+body)
	out, err := exec.Command(bin, "dupl", "./b").Output()
	if code := exitCode(err); code != 1 || !strings.Contains(string(out), "b/b2.go:4:2: lines 4-16 duplicate lines 8-20 of b/b.go (Second)") {
		t.Errorf("gosloplint dupl ./b = %d, %q, want 1 and the copy", code, out)
	}
	out, err = exec.Command(bin, "-dupl", "./b").CombinedOutput()
	if code := exitCode(err); code != 3 || !strings.Contains(string(out), "b2.go:4:2: lines 4-16 duplicate lines 8-20 of b.go (Second)") {
		t.Errorf("gosloplint -dupl ./b = %d, %q, want the multichecker's 3 and the copy", code, out)
	}
}

func exitCode(err error) int {
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		return ee.ExitCode()
	}
	if err != nil {
		return -1
	}
	return 0
}
