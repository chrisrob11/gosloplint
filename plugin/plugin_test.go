package plugin_test

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/golangci/plugin-module-register/register"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/chrisrob11/gosloplint/plugin"
)

func TestRegisteredUnderItsName(t *testing.T) {
	newPlugin, err := register.GetPlugin("gosloplint")
	if err != nil {
		t.Fatalf("GetPlugin: %v", err)
	}
	p, err := newPlugin(nil)
	if err != nil {
		t.Fatalf("new with no settings: %v", err)
	}
	if got := p.GetLoadMode(); got != register.LoadModeTypesInfo {
		t.Errorf("GetLoadMode = %q, want %q", got, register.LoadModeTypesInfo)
	}
}

func TestNoSettingsEnablesEveryAnalyzer(t *testing.T) {
	want := []string{"comment", "ctxthread", "emptyid", "emptyskip", "selfassert", "steer", "testhelper", "vocab"}
	if got := names(t, build(t, nil)); !slices.Equal(got, want) {
		t.Errorf("analyzers = %v, want %v", got, want)
	}
}

func TestDisable(t *testing.T) {
	as := build(t, map[string]any{"disable": []any{"ctxthread", "vocab"}})
	want := []string{"comment", "emptyid", "emptyskip", "selfassert", "steer", "testhelper"}
	if got := names(t, as); !slices.Equal(got, want) {
		t.Errorf("analyzers = %v, want %v", got, want)
	}
}

func TestBadSettingsAreErrors(t *testing.T) {
	for _, tc := range []struct {
		name     string
		settings map[string]any
		want     string
	}{
		{"unknown field", map[string]any{"disabled": []any{"vocab"}}, `unknown field "disabled"`},
		{"unknown nested field", map[string]any{"vocab": map[string]any{"word": []any{"x"}}}, `unknown field "word"`},
		{"wrong type", map[string]any{"disable": "vocab"}, "cannot unmarshal string"},
		{"unknown analyzer", map[string]any{"disable": []any{"emptyids"}}, `disable: unknown analyzer "emptyids" (known: comment, ctxthread, emptyid,`},
		{"blank suffix", map[string]any{"emptyid": map[string]any{"suffixes": []any{"id", " "}}}, "emptyid.suffixes[1]: empty word"},
		{"multi-word suffix", map[string]any{"emptyid": map[string]any{"suffixes": []any{"user_id"}}}, `emptyid.suffixes[0]: "user_id" is 2 words`},
		{"blank word", map[string]any{"vocab": map[string]any{"words": []any{""}}}, "vocab.words[0]: empty word"},
		{"words for disabled vocab", map[string]any{"disable": []any{"vocab"}, "vocab": map[string]any{"words": []any{"age"}}}, "vocab.words is set but vocab is disabled"},
		{"suffixes for disabled emptyid", map[string]any{"disable": []any{"emptyid"}, "emptyid": map[string]any{"suffixes": []any{"ref"}}}, "emptyid.suffixes is set but emptyid is disabled"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := plugin.New(tc.settings)
			if err == nil {
				t.Fatalf("New(%v) = nil error, want one containing %q", tc.settings, tc.want)
			}
			if !strings.HasPrefix(err.Error(), "gosloplint: ") || !strings.Contains(err.Error(), tc.want) {
				t.Errorf("New(%v) error = %q, want gosloplint: ...%s...", tc.settings, err, tc.want)
			}
		})
	}
}

func TestVocabWordsReachTheAnalyzer(t *testing.T) {
	if got := findings(t, nil, "vocab"); len(got) != 0 {
		t.Errorf("no words: findings = %q, want none", got)
	}
	got := findings(t, map[string]any{"vocab": map[string]any{"words": []any{"legacy client"}}}, "vocab")
	want := []string{`declares LegacyClient, which contains the banned word "legacy client"`}
	if !slices.Equal(got, want) {
		t.Errorf("words set: findings = %q, want %q", got, want)
	}
}

func TestEmptyidSuffixesReachTheAnalyzer(t *testing.T) {
	const user = "guard on empty userID exits silently"
	const table = "guard on empty tableRef exits silently"
	if got := findings(t, nil, "emptyid"); len(got) != 1 || !strings.HasPrefix(got[0], user) {
		t.Errorf("default suffixes: findings = %q, want one starting %q", got, user)
	}
	got := findings(t, map[string]any{"emptyid": map[string]any{"suffixes": []any{"Ref"}}}, "emptyid")
	if len(got) != 1 || !strings.HasPrefix(got[0], table) {
		t.Errorf("suffixes [Ref]: findings = %q, want one starting %q", got, table)
	}
}

func build(t *testing.T, settings any) []*analysis.Analyzer {
	t.Helper()
	p, err := plugin.New(settings)
	if err != nil {
		t.Fatalf("New(%v): %v", settings, err)
	}
	as, err := p.BuildAnalyzers()
	if err != nil {
		t.Fatalf("BuildAnalyzers: %v", err)
	}
	return as
}

func names(t *testing.T, as []*analysis.Analyzer) []string {
	t.Helper()
	if err := analysis.Validate(as); err != nil {
		t.Fatalf("Validate: %v", err)
	}
	var out []string
	for _, a := range as {
		out = append(out, a.Name)
	}
	return out
}

// findings runs the analyzer named by name, built from settings, over
// testdata/src/slop and returns its diagnostic messages.
func findings(t *testing.T, settings any, name string) []string {
	t.Helper()
	as := build(t, settings)
	i := slices.IndexFunc(as, func(a *analysis.Analyzer) bool { return a.Name == name })
	if i < 0 {
		t.Fatalf("%s not built", name)
	}
	// slop has no want comments, so analysistest calls every finding
	// unexpected; rec keeps those out of t and the findings are read from
	// the results instead.
	var rec recorder
	results := analysistest.Run(&rec, analysistest.TestData(), as[i], "slop")
	if len(results) == 0 {
		t.Fatalf("no results: %v", rec)
	}
	var out []string
	for _, r := range results {
		if r.Err != nil {
			t.Fatalf("%s: %v", name, r.Err)
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
