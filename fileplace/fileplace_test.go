package fileplace_test

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/chrisrob11/gosloplint/fileplace"
)

func TestDefaultRules(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), fileplace.Analyzer, "receiver", "testname", "cmdmain", "grabbag", "tiny")
}

func TestOffByDefaultRules(t *testing.T) {
	for _, pkg := range []string{"typefile", "filename"} {
		for _, d := range diagnostics(t, fileplace.Analyzer, pkg) {
			if strings.HasPrefix(d, "typefile:") || strings.HasPrefix(d, "filename:") {
				t.Errorf("%s, default settings: %q", pkg, d)
			}
		}
	}
	a := fileplace.New(fileplace.Settings{Enable: fileplace.OffByDefault, Disable: []string{fileplace.Tiny}})
	analysistest.Run(t, analysistest.TestData(), a, "typefile", "filename")
}

func TestCleanPackages(t *testing.T) {
	a := fileplace.New(fileplace.Settings{Enable: fileplace.OffByDefault})
	for _, pkg := range []string{"tinypass", "single", "helpers"} {
		if got := diagnostics(t, a, pkg); len(got) != 0 {
			t.Errorf("%s: %q, want none", pkg, got)
		}
	}
}

func TestABigTypeSplitsByConcern(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), fileplace.Analyzer, "split")
	a := fileplace.New(fileplace.Settings{LongLines: 50, Disable: []string{fileplace.Long}})
	if got := diagnostics(t, a, "split"); len(got) != 0 {
		t.Errorf("past long-lines, a concern file: %q, want none", got)
	}
}

func TestLong(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), fileplace.New(fileplace.Settings{LongLines: 50}), "long")
	if got := diagnostics(t, fileplace.Analyzer, "long"); len(got) != 0 {
		t.Errorf("default long-lines: %q, want none", got)
	}
}

func TestTinyLines(t *testing.T) {
	got := diagnostics(t, fileplace.New(fileplace.Settings{TinyLines: 41}), "tinypass")
	if len(got) != 1 || !strings.HasPrefix(got[0], "tiny: long.go is 40 lines") {
		t.Errorf("tiny-lines 41: %q, want long.go", got)
	}
}

func TestGrabBagReplacesTheDefault(t *testing.T) {
	got := diagnostics(t, fileplace.New(fileplace.Settings{GrabBag: []string{"string"}}), "grabbag")
	want := []string{
		"grabbag: string_utils.go has a grab-bag name: move each declaration into the file of the type it serves, then delete the file",
		"testname: helpers_test.go names no source file in its directory: name it for the file whose code it tests (foo_test.go, foo_internal_test.go), or fold its tests into that file's test",
	}
	if slices.Sort(got); !slices.Equal(got, want) {
		t.Errorf("grab-bag [string]: %q, want %q, and helpers_test.go now read as a test name", got, want)
	}
}

func TestDisable(t *testing.T) {
	all := fileplace.New(fileplace.Settings{Enable: fileplace.OffByDefault})
	pkgs := []string{"receiver", "testname", "grabbag", "tiny", "typefile", "filename"}
	before := map[string][]string{}
	for _, pkg := range pkgs {
		before[pkg] = diagnostics(t, all, pkg)
	}
	for _, rule := range []string{"receiver", "testname", "grabbag", "tiny", "typefile", "filename"} {
		var enable []string
		for _, r := range fileplace.OffByDefault {
			if r != rule {
				enable = append(enable, r)
			}
		}
		s := fileplace.Settings{Enable: enable}
		if !slices.Contains(fileplace.OffByDefault, rule) {
			s.Disable = []string{rule}
		}
		a := fileplace.New(s)
		for _, pkg := range pkgs {
			want := 0
			for _, d := range before[pkg] {
				if !strings.HasPrefix(d, rule+":") {
					want++
				}
			}
			got := diagnostics(t, a, pkg)
			if len(got) != want {
				t.Errorf("without %s, %s: %d findings %q, want %d", rule, pkg, len(got), got, want)
			}
		}
	}
	if got := diagnostics(t, fileplace.New(fileplace.Settings{Disable: []string{"long"}, LongLines: 50}), "long"); len(got) != 0 {
		t.Errorf("long disabled: %q", got)
	}
}

func TestExempt(t *testing.T) {
	a := fileplace.New(fileplace.Settings{Exempt: map[string][]string{
		"grabbag":  {"*util*.go", "testhelpers.go"},
		"testname": {"testname"},
	}})
	got := diagnostics(t, a, "grabbag")
	if want := []string{"grabbag: helpers_test.go has a grab-bag name: move each declaration into the file of the type it serves, then delete the file"}; !slices.Equal(got, want) {
		t.Errorf("grabbag exempt by file glob: %q, want %q", got, want)
	}
	if got := diagnostics(t, a, "testname"); len(got) != 1 || !strings.HasPrefix(got[0], "grabbag:") {
		t.Errorf("testname exempt by package: %q, want only the grab-bag finding", got)
	}
	b := fileplace.New(fileplace.Settings{Exempt: map[string][]string{"tiny": {"..."}}})
	if got := diagnostics(t, b, "tiny"); len(got) != 0 {
		t.Errorf("tiny exempt everywhere: %q", got)
	}
}

func TestFlags(t *testing.T) {
	setFlag(t, "long-lines", "50")
	setFlag(t, "disable", "receiver")
	analysistest.Run(t, analysistest.TestData(), fileplace.Analyzer, "long")
	setFlag(t, "exempt", "long=long.go")
	if got := diagnostics(t, fileplace.Analyzer, "long"); len(got) != 0 {
		t.Errorf("-exempt=long=long.go: %q", got)
	}
	setFlag(t, "grab-bag", "string")
	if got := diagnostics(t, fileplace.Analyzer, "grabbag"); len(got) != 2 {
		t.Errorf("-grab-bag=string: %q, want string_utils.go and, now a test name, helpers_test.go", got)
	}
	if err := fileplace.Analyzer.Flags.Set("exempt", "long"); err == nil {
		t.Error("-exempt=long: no error, want rule=pattern")
	}
}

func TestBadSettings(t *testing.T) {
	for _, tc := range []struct {
		s    fileplace.Settings
		want string
	}{
		{fileplace.Settings{Disable: []string{"recevier"}}, `disable[0]: unknown rule "recevier" (known: receiver, testname,`},
		{fileplace.Settings{Enable: []string{"tiny"}}, `enable[0]: "tiny" is not a rule that is off by default (those are: typefile, filename)`},
		{fileplace.Settings{Enable: []string{"typefile"}, Disable: []string{"typefile"}}, `enable[0]: "typefile" is also disabled`},
		{fileplace.Settings{TinyLines: -1}, "tiny-lines: -1 is negative"},
		{fileplace.Settings{LongLines: -1}, "long-lines: -1 is negative"},
		{fileplace.Settings{TinyLines: 50, LongLines: 50}, "tiny-lines 50 is not below long-lines 50"},
		{fileplace.Settings{GrabBag: []string{"util", "test_util"}}, `grab-bag[1]: "test_util" is not one file name part`},
		{fileplace.Settings{Exempt: map[string][]string{"tinny": {"x"}}}, `exempt: unknown rule "tinny"`},
		{fileplace.Settings{Exempt: map[string][]string{"tiny": {""}}}, "exempt.tiny[0]: empty pattern"},
		{fileplace.Settings{Exempt: map[string][]string{"tiny": {"[x"}}}, `exempt.tiny[0]: "[x": syntax error in pattern`},
	} {
		err := tc.s.Validate()
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("Validate(%+v) = %v, want %q", tc.s, err, tc.want)
		}
		func() {
			defer func() {
				if r := recover(); r == nil || !strings.Contains(fmt.Sprint(r), tc.want) {
					t.Errorf("New(%+v) panic = %v, want %q", tc.s, r, tc.want)
				}
			}()
			fileplace.New(tc.s)
		}()
	}
}

func setFlag(t *testing.T, name, v string) {
	t.Helper()
	f := fileplace.Analyzer.Flags.Lookup(name)
	if f == nil {
		t.Fatalf("no flag %s", name)
	}
	old := f.Value.String()
	if err := f.Value.Set(v); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := f.Value.Set(old); err != nil {
			t.Error(err)
		}
	})
}

// diagnostics runs a over the fixture pkg and returns its messages, sorted
// and without duplicates (a non-test file is read by the package's pass and
// its test variant's), without matching want comments.
func diagnostics(t *testing.T, a *analysis.Analyzer, pkg string) []string {
	t.Helper()
	var rec recorder
	var out []string
	for _, r := range analysistest.Run(&rec, analysistest.TestData(), a, pkg) {
		if r.Err != nil {
			t.Fatalf("%s: %v", pkg, r.Err)
		}
		for _, d := range r.Diagnostics {
			out = append(out, r.Pass.Fset.Position(d.Pos).String()+"\x00"+d.Message)
		}
	}
	slices.Sort(out)
	out = slices.Compact(out)
	for i, d := range out {
		_, out[i], _ = strings.Cut(d, "\x00")
	}
	return out
}

type recorder []string

func (r *recorder) Errorf(format string, args ...any) {
	*r = append(*r, fmt.Sprintf(format, args...))
}
