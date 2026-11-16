package baseline_test

import (
	"errors"
	"fmt"
	"go/ast"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/chrisrob11/gosloplint/internal/baseline"
)

// fake reports each function: x1 and x2 under the check "x", other under
// none.
var fake = &analysis.Analyzer{
	Name: "fake",
	Doc:  "fake: report every function",
	Run: func(pass *analysis.Pass) (any, error) {
		for _, f := range pass.Files {
			for _, d := range f.Decls {
				fd := d.(*ast.FuncDecl)
				cat := ""
				if strings.HasPrefix(fd.Name.Name, "x") {
					cat = "x"
				}
				pass.Report(analysis.Diagnostic{Pos: fd.Pos(), Category: cat, Message: fd.Name.Name})
			}
		}
		return nil, nil
	},
}

func TestUnderOrAtTheBaselineNothingIsReported(t *testing.T) {
	for _, c := range []baseline.Counts{
		{"a": {"fake:x": 2, "fake": 1}},
		{"a": {"fake:x": 5, "fake": 9}},
	} {
		if got := findings(t, c); len(got) != 0 {
			t.Errorf("counts %v: %q, want none", c, got)
		}
	}
}

func TestOverTheBaselineEverythingIsReported(t *testing.T) {
	got := findings(t, baseline.Counts{"a": {"fake:x": 1, "fake": 1}})
	want := []string{
		"x1 (2 fake:x findings in this package, over its baseline of 1)",
		"x2 (2 fake:x findings in this package, over its baseline of 1)",
	}
	if !slices.Equal(got, want) {
		t.Errorf("x over: %q, want %q", got, want)
	}
	// A check the baseline does not name has a baseline of 0, unnoted, and
	// so has a package it does not name.
	if got := findings(t, baseline.Counts{"a": {"fake:x": 2}}); !slices.Equal(got, []string{"other"}) {
		t.Errorf("fake unrecorded: %q, want other", got)
	}
	if got := findings(t, baseline.Counts{"b": {"fake:x": 2, "fake": 1}}); len(got) != 3 {
		t.Errorf("another package's baseline: %q, want all three", got)
	}
}

func TestNoBaselineFiltersNothing(t *testing.T) {
	if got := findings(t, nil); len(got) != 3 {
		t.Errorf("nil counts: %q, want all three", got)
	}
}

func TestALoadErrorFailsThePass(t *testing.T) {
	a := baseline.Wrap([]*analysis.Analyzer{fake}, func() (baseline.Counts, error) { return nil, errors.New("no file") })[0]
	var rec recorder
	for _, r := range analysistest.Run(&rec, analysistest.TestData(), a, "a") {
		if r.Err == nil || r.Err.Error() != "baseline: no file" {
			t.Errorf("Err = %v, want baseline: no file", r.Err)
		}
	}
}

func TestWriteIsSortedAndReadsBack(t *testing.T) {
	path := filepath.Join(t.TempDir(), "b.json")
	c := baseline.Counts{}
	c.Add("z/p", "steer:returns-any", 2)
	c.Add("a/p", "vocab", 1)
	c.Add("a/p", "dupl", 3)
	c.Add("a/p", "dupl", 2)
	c.Add("a/p", "comment", 0)
	if err := c.Write(path); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	want := `{
  "a/p": {
    "dupl": 3,
    "vocab": 1
  },
  "z/p": {
    "steer:returns-any": 2
  }
}
`
	if string(data) != want {
		t.Errorf("file:\n%s\nwant:\n%s", data, want)
	}
	back, err := baseline.Read(path)
	if err != nil || fmt.Sprint(back) != fmt.Sprint(c) {
		t.Errorf("Read = %v, %v, want %v", back, err, c)
	}
	if err := (baseline.Counts(nil)).Write(path); err != nil {
		t.Fatal(err)
	}
	if data, _ := os.ReadFile(path); string(data) != "{}\n" {
		t.Errorf("empty baseline = %q, want {}", data)
	}
}

func TestBadFilesAreErrors(t *testing.T) {
	dir := t.TempDir()
	for name, content := range map[string]string{
		"list.json":     `["a"]`,
		"string.json":   `{"a": {"dupl": "2"}}`,
		"negative.json": `{"a": {"dupl": -1}}`,
	} {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := baseline.Read(path); err == nil || !strings.Contains(err.Error(), name) {
			t.Errorf("Read(%s) = %v, want an error naming the file", name, err)
		}
	}
	if _, err := baseline.Read(filepath.Join(dir, "missing.json")); err == nil {
		t.Error("Read of a missing file: no error")
	}
}

// findings runs fake, wrapped with counts, over testdata/src/a and returns
// its messages, sorted.
func findings(t *testing.T, counts baseline.Counts) []string {
	t.Helper()
	a := baseline.Wrap([]*analysis.Analyzer{fake}, func() (baseline.Counts, error) { return counts, nil })[0]
	var rec recorder
	var out []string
	for _, r := range analysistest.Run(&rec, analysistest.TestData(), a, "a") {
		if r.Err != nil {
			t.Fatal(r.Err)
		}
		for _, d := range r.Diagnostics {
			out = append(out, d.Message)
		}
	}
	slices.Sort(out)
	return out
}

type recorder []string

func (r *recorder) Errorf(format string, args ...any) {
	*r = append(*r, fmt.Sprintf(format, args...))
}
