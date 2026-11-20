package comment_test

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/chrisrob11/gosloplint/comment"
)

// lengthOnly reads block lengths alone: the length fixtures are all comment.
var lengthOnly = comment.New(comment.Settings{Disable: []string{comment.Density}})

func TestLengthByKind(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), lengthOnly, "kinds", "escape", "blockcomment")
}

func TestBlocksWithinTheirLimitsAndGeneratedFilesAreClean(t *testing.T) {
	for _, pkg := range []string{"fits", "gen"} {
		if got := diagnostics(t, lengthOnly, pkg); len(got) != 0 {
			t.Errorf("%s: %q, want none", pkg, got)
		}
	}
	for _, pkg := range []string{"gen", "licensed", "sparse", "tiny", "trailing", "withtests"} {
		if got := diagnostics(t, comment.Analyzer, pkg); len(got) != 0 {
			t.Errorf("%s: %q, want none", pkg, got)
		}
	}
}

func TestAnEscapeWithoutAReasonIsReportedAndExemptsNothing(t *testing.T) {
	got := diagnostics(t, lengthOnly, "noreason")
	want := []string{
		"length: func comment is 9 lines, limit 8: shorten it, or exempt the block with //gosloplint:comment-ok <reason>",
		"length: //gosloplint:comment-ok needs a reason after it, saying why this block needs the room",
	}
	slices.Sort(got)
	slices.Sort(want)
	if !slices.Equal(got, want) {
		t.Errorf("findings = %q, want %q", got, want)
	}
}

func TestDensityIsReportedOnTheFileWithTheMostComments(t *testing.T) {
	a := comment.New(comment.Settings{Disable: []string{comment.Length}})
	analysistest.Run(t, analysistest.TestData(), a, "dense")
}

func TestDensityCeilingAndMinimumAreSettings(t *testing.T) {
	if got := diagnostics(t, comment.New(comment.Settings{Density: 50}), "dense"); len(got) != 0 {
		t.Errorf("ceiling 50%%: %q, want none", got)
	}
	if got := diagnostics(t, comment.New(comment.Settings{Density: 5, DensityMinLines: 1}), "tiny"); len(got) != 1 {
		t.Errorf("ceiling 5%%, no minimum: %q, want one finding", got)
	}
	if got := diagnostics(t, comment.New(comment.Settings{DensityMinLines: 1000}), "dense"); len(got) != 0 {
		t.Errorf("minimum 1000 lines: %q, want none", got)
	}
}

func TestLimitsAreSettings(t *testing.T) {
	a := comment.New(comment.Settings{Limits: comment.Limits{Func: 9, Inline: 10}, Disable: []string{comment.Density}})
	got := diagnostics(t, a, "blockcomment")
	if len(got) != 0 {
		t.Errorf("inline limit 10: %q, want none", got)
	}
	if got := diagnostics(t, a, "noreason"); len(got) != 1 {
		t.Errorf("func limit 9: %q, want only the missing reason", got)
	}
}

func TestFlags(t *testing.T) {
	setFlag(t, "inline", "10")
	setFlag(t, "disable", "density")
	if got := diagnostics(t, comment.Analyzer, "blockcomment"); len(got) != 0 {
		t.Errorf("-inline=10: %q, want none", got)
	}
	setFlag(t, "disable", "length")
	setFlag(t, "density", "50")
	if got := diagnostics(t, comment.Analyzer, "dense"); len(got) != 0 {
		t.Errorf("-density=50: %q, want none", got)
	}
}

func TestDisable(t *testing.T) {
	a := comment.New(comment.Settings{Disable: []string{comment.Length, comment.Density}})
	for _, pkg := range []string{"kinds", "dense", "noreason"} {
		if got := diagnostics(t, a, pkg); len(got) != 0 {
			t.Errorf("both disabled, %s: %q", pkg, got)
		}
	}
	a = comment.New(comment.Settings{Disable: []string{comment.Density}})
	if got := diagnostics(t, a, "dense"); len(got) != 0 {
		t.Errorf("density disabled: %q", got)
	}
}

func TestLimitsForFallsBackToTheDefault(t *testing.T) {
	l := comment.Limits{Func: 2}
	for _, k := range comment.Kinds {
		want := comment.DefaultLimits.For(k)
		if k == comment.KindFunc {
			want = 2
		}
		if got := l.For(k); got != want {
			t.Errorf("For(%s) = %d, want %d", k, got, want)
		}
	}
}

func TestBadSettings(t *testing.T) {
	for _, tc := range []struct {
		s    comment.Settings
		want string
	}{
		{comment.Settings{Disable: []string{"lengths"}}, `disable[0]: unknown check "lengths" (known: length, density)`},
		{comment.Settings{Limits: comment.Limits{Inline: -1}}, "limits.inline: -1 is negative"},
		{comment.Settings{Density: 101}, "density: 101 is not a percentage from 1 to 100"},
		{comment.Settings{Density: -1}, "density: -1 is not a percentage"},
		{comment.Settings{DensityMinLines: -5}, "density-min-lines: -5 is negative"},
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
			comment.New(tc.s)
		}()
	}
}

func setFlag(t *testing.T, name, v string) {
	t.Helper()
	f := comment.Analyzer.Flags.Lookup(name)
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

// diagnostics runs a over the fixture pkg and returns its messages, without
// matching want comments.
func diagnostics(t *testing.T, a *analysis.Analyzer, pkg string) []string {
	t.Helper()
	var rec recorder
	var out []string
	for _, r := range analysistest.Run(&rec, analysistest.TestData(), a, pkg) {
		if r.Err != nil {
			t.Fatalf("%s: %v", pkg, r.Err)
		}
		for _, d := range r.Diagnostics {
			out = append(out, d.Message)
		}
	}
	return out
}

type recorder []string

func (r *recorder) Errorf(format string, args ...any) {
	*r = append(*r, fmt.Sprintf(format, args...))
}
