package main

import (
	"errors"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestTheBinaryDispatches builds gosloplint: dupl as the first argument is
// the subcommand, anything else the multichecker, unchanged.
func TestTheBinaryDispatches(t *testing.T) {
	bin := build(t)
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

// build builds gosloplint into a temporary directory; call it before moving
// out of the package's directory.
func build(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "gosloplint")
	if out, err := exec.Command("go", "build", "-o", bin, ".").CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}
	return bin
}
