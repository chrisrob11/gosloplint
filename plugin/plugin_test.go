package plugin_test

import (
	"fmt"
	"os"
	"path/filepath"
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
	want := []string{"cannotfail", "comment", "ctxthread", "dupl", "emptyid", "emptyskip", "fileplace", "narration", "selfassert", "steer", "testhelper", "vocab"}
	if got := names(t, build(t, nil)); !slices.Equal(got, want) {
		t.Errorf("analyzers = %v, want %v", got, want)
	}
}

func TestDisable(t *testing.T) {
	as := build(t, map[string]any{"disable": []any{"ctxthread", "vocab"}})
	want := []string{"cannotfail", "comment", "dupl", "emptyid", "emptyskip", "fileplace", "narration", "selfassert", "steer", "testhelper"}
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
		{"unknown analyzer", map[string]any{"disable": []any{"emptyids"}}, `disable: unknown analyzer "emptyids" (known: cannotfail, comment, ctxthread, dupl, emptyid,`},
		{"blank suffix", map[string]any{"emptyid": map[string]any{"suffixes": []any{"id", " "}}}, "emptyid.suffixes[1]: empty word"},
		{"multi-word suffix", map[string]any{"emptyid": map[string]any{"suffixes": []any{"user_id"}}}, `emptyid.suffixes[0]: "user_id" is 2 words`},
		{"blank word", map[string]any{"vocab": map[string]any{"words": []any{""}}}, "vocab.words[0]: empty word"},
		{"words for disabled vocab", map[string]any{"disable": []any{"vocab"}, "vocab": map[string]any{"words": []any{"age"}}}, "vocab.words is set but vocab is disabled"},
		{"suffixes for disabled emptyid", map[string]any{"disable": []any{"emptyid"}, "emptyid": map[string]any{"suffixes": []any{"ref"}}}, "emptyid.suffixes is set but emptyid is disabled"},
		{"unknown steer field", map[string]any{"steer": map[string]any{"recursion_packages": []any{"x"}}}, `unknown field "recursion_packages"`},
		{"unknown steer check", map[string]any{"steer": map[string]any{"disable": []any{"returns-anything"}}}, `steer.disable[0]: unknown check "returns-anything"`},
		{"steer for disabled steer", map[string]any{"disable": []any{"steer"}, "steer": map[string]any{"recursion-packages": []any{"..."}}}, "steer settings are set but steer is disabled"},
		{"recursion with recursion disabled", map[string]any{"steer": map[string]any{"disable": []any{"unguarded-recursion"}, "recursion-packages": []any{"..."}}}, "steer.recursion-packages is set but unguarded-recursion is disabled"},
		{"unknown comment limit", map[string]any{"comment": map[string]any{"limits": map[string]any{"function": 9}}}, `unknown field "function"`},
		{"negative comment limit", map[string]any{"comment": map[string]any{"limits": map[string]any{"inline": -1}}}, "comment.limits.inline: -1 is negative"},
		{"density over 100", map[string]any{"comment": map[string]any{"density": 120}}, "comment.density: 120 is not a percentage"},
		{"density as a string", map[string]any{"comment": map[string]any{"density": "20%"}}, "cannot unmarshal string"},
		{"comment for disabled comment", map[string]any{"disable": []any{"comment"}, "comment": map[string]any{"density": 30}}, "comment settings are set but comment is disabled"},
		{"unknown fileplace rule", map[string]any{"fileplace": map[string]any{"disable": []any{"FP001"}}}, `fileplace.disable[0]: unknown rule "FP001"`},
		{"enable a rule that is on", map[string]any{"fileplace": map[string]any{"enable": []any{"tiny"}}}, `fileplace.enable[0]: "tiny" is not a rule that is off by default`},
		{"bad exempt glob", map[string]any{"fileplace": map[string]any{"exempt": map[string]any{"tiny": []any{"[x"}}}}, `fileplace.exempt.tiny[0]: "[x": syntax error in pattern`},
		{"tiny not below long", map[string]any{"fileplace": map[string]any{"tiny-lines": 100, "long-lines": 90}}, "fileplace.tiny-lines 100 is not below long-lines 90"},
		{"negative dupl size", map[string]any{"dupl": map[string]any{"min-nodes": -5}}, "dupl.min-nodes: -5 is negative"},
		{"bad dupl exempt", map[string]any{"dupl": map[string]any{"exempt": []any{"[x"}}}, `dupl.exempt[0]: "[x": syntax error in pattern`},
		{"unknown dupl field", map[string]any{"dupl": map[string]any{"min_nodes": 9}}, `unknown field "min_nodes"`},
		{"dupl for disabled dupl", map[string]any{"disable": []any{"dupl"}, "dupl": map[string]any{"tests": true}}, "dupl settings are set but dupl is disabled"},
		{"missing baseline", map[string]any{"baseline": "testdata/missing.json"}, "baseline: open testdata/missing.json"},
		{"unknown cannotfail check", map[string]any{"cannotfail": map[string]any{"disable": []any{"no-asserts"}}}, `cannotfail.disable[0]: unknown check "no-asserts"`},
		{"enable a cannotfail check that is on", map[string]any{"cannotfail": map[string]any{"enable": []any{"no-assert"}}}, `cannotfail.enable[0]: "no-assert" is not a check that is off by default`},
		{"blank assert func", map[string]any{"cannotfail": map[string]any{"assert-funcs": []any{""}}}, "cannotfail.assert-funcs[0]: empty entry"},
		{"unknown cannotfail field", map[string]any{"cannotfail": map[string]any{"assert_funcs": []any{"x"}}}, `unknown field "assert_funcs"`},
		{"cannotfail for disabled cannotfail", map[string]any{"disable": []any{"cannotfail"}, "cannotfail": map[string]any{"enable": []any{"empty-loop"}}}, "cannotfail settings are set but cannotfail is disabled"},
		{"fileplace for disabled fileplace", map[string]any{"disable": []any{"fileplace"}, "fileplace": map[string]any{"long-lines": 900}}, "fileplace settings are set but fileplace is disabled"},
		{"unknown narration check", map[string]any{"narration": map[string]any{"disable": []any{"histories"}}}, `narration.disable[0]: unknown check "histories"`},
		{"narration for disabled narration", map[string]any{"disable": []any{"narration"}, "narration": map[string]any{"disable": []any{"history"}}}, "narration settings are set but narration is disabled"},
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

func TestSteerRecursionPackagesReachTheAnalyzer(t *testing.T) {
	if got := findings(t, nil, "steer"); len(got) != 0 {
		t.Errorf("default: findings = %q, want none", got)
	}
	got := findings(t, map[string]any{"steer": map[string]any{"recursion-packages": []any{"slop"}}}, "steer")
	if len(got) != 1 || !strings.HasPrefix(got[0], "unguarded-recursion: walk recurses") {
		t.Errorf("recursion-packages [slop]: findings = %q, want walk", got)
	}
}

func TestCommentLimitsReachTheAnalyzer(t *testing.T) {
	got := findings(t, nil, "comment")
	if len(got) != 1 || !strings.HasPrefix(got[0], "length: func comment is 9 lines, limit 8") {
		t.Errorf("default: findings = %q, want nine's doc", got)
	}
	if got := findings(t, map[string]any{"comment": map[string]any{"limits": map[string]any{"func": 9}}}, "comment"); len(got) != 0 {
		t.Errorf("limits.func 9: findings = %q, want none", got)
	}
	if got := findings(t, map[string]any{"comment": map[string]any{"disable": []any{"length"}}}, "comment"); len(got) != 0 {
		t.Errorf("length disabled: findings = %q, want none", got)
	}
}

func TestNarrationDisableReachesTheAnalyzer(t *testing.T) {
	got := findings(t, nil, "narration")
	if len(got) != 1 || !strings.HasPrefix(got[0], `history: "now uses" narrates`) {
		t.Errorf("default: findings = %q, want cached's doc", got)
	}
	if got := findings(t, map[string]any{"narration": map[string]any{"disable": []any{"history"}}}, "narration"); len(got) != 0 {
		t.Errorf("history disabled: findings = %q, want none", got)
	}
}

func TestFileplaceEnableReachesTheAnalyzer(t *testing.T) {
	if got := findings(t, nil, "fileplace"); len(got) != 0 {
		t.Errorf("default: findings = %q, want none", got)
	}
	got := findings(t, map[string]any{"fileplace": map[string]any{"enable": []any{"typefile"}}}, "fileplace")
	if len(got) != 1 || !strings.HasPrefix(got[0], "typefile: type cache has 1 method(s) but is declared in store.go") {
		t.Errorf("enable [typefile]: findings = %q, want cache", got)
	}
	got = findings(t, map[string]any{"fileplace": map[string]any{"enable": []any{"typefile"}, "exempt": map[string]any{"typefile": []any{"store.go"}}}}, "fileplace")
	if len(got) != 0 {
		t.Errorf("typefile exempt in store.go: findings = %q, want none", got)
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

func TestDuplSettingsReachTheAnalyzer(t *testing.T) {
	got := findings(t, nil, "dupl")
	if len(got) != 1 || !strings.HasPrefix(got[0], "lines 67-84 duplicate lines 46-63 of slop.go (first)") {
		t.Errorf("default: findings = %q, want second", got)
	}
	for _, s := range []map[string]any{{"min-nodes": 1000}, {"min-statements": 20}, {"exempt": []any{"slop.go"}}} {
		if got := findings(t, map[string]any{"dupl": s}, "dupl"); len(got) != 0 {
			t.Errorf("dupl %v: findings = %q, want none", s, got)
		}
	}
}

func TestCannotfailSettingsReachTheAnalyzer(t *testing.T) {
	const nothing, viaCheck = "no-assert: TestNothing calls no", "no-assert: TestViaCheck calls no"
	const rows = "empty-loop: every assertion here is inside this range over got"
	for _, tc := range []struct {
		settings map[string]any
		want     []string
	}{
		{nil, []string{nothing, viaCheck}},
		{map[string]any{"assert-funcs": []any{"tests.check"}}, []string{nothing}},
		{map[string]any{"enable": []any{"empty-loop"}}, []string{nothing, viaCheck, rows}},
		{map[string]any{"disable": []any{"no-assert"}}, nil},
	} {
		got := findingsIn(t, map[string]any{"cannotfail": tc.settings}, "cannotfail", "tests")
		ok := len(got) == len(tc.want)
		for i := 0; ok && i < len(got); i++ {
			ok = strings.HasPrefix(got[i], tc.want[i])
		}
		if !ok {
			t.Errorf("cannotfail %v: findings = %q, want %q", tc.settings, got, tc.want)
		}
	}
}

func TestBaselineHoldsEveryAnalyzer(t *testing.T) {
	path := filepath.Join(t.TempDir(), "baseline.json")
	write := func(content string) {
		t.Helper()
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write(`{"slop": {"dupl": 1, "emptyid": 1, "fileplace:typefile": 1}}`)
	settings := map[string]any{"baseline": path, "fileplace": map[string]any{"enable": []any{"typefile"}}}
	for _, name := range []string{"dupl", "emptyid", "fileplace"} {
		if got := findings(t, settings, name); len(got) != 0 {
			t.Errorf("%s at its baseline: findings = %q, want none", name, got)
		}
	}
	write(`{"slop": {"dupl": 1, "fileplace:typefile": 1}}`)
	got := findings(t, settings, "emptyid")
	if len(got) != 1 || strings.Contains(got[0], "baseline") {
		t.Errorf("emptyid off the baseline: findings = %q, want byUser's, unnoted", got)
	}
	write(`{"slop": {"dupl": 1, "emptyid": 1, "fileplace:typefile": 1}`)
	if _, err := plugin.New(settings); err == nil || !strings.Contains(err.Error(), "baseline: ") {
		t.Errorf("a malformed baseline: New error = %v, want baseline: ...", err)
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

func TestIgnoreDirectivesReachEveryAnalyzer(t *testing.T) {
	got := findingsIn(t, nil, "emptyid", "ignored")
	want := []string{"ignore: //gosloplint:ignore emptyid suppresses no emptyid finding: delete it"}
	if !slices.Equal(got, want) {
		t.Errorf("emptyid over ignored: findings = %q, want %q", got, want)
	}
}

// findings runs the analyzer named by name, built from settings, over
// testdata/src/slop and returns its diagnostic messages.
func findings(t *testing.T, settings any, name string) []string {
	t.Helper()
	return findingsIn(t, settings, name, "slop")
}

func findingsIn(t *testing.T, settings any, name, pkg string) []string {
	t.Helper()
	as := build(t, settings)
	i := slices.IndexFunc(as, func(a *analysis.Analyzer) bool { return a.Name == name })
	if i < 0 {
		t.Fatalf("%s not built", name)
	}
	// The fixtures have no want comments, so analysistest calls every finding
	// unexpected; rec keeps those out of t and the findings are read from
	// the results instead.
	var rec recorder
	results := analysistest.Run(&rec, analysistest.TestData(), as[i], pkg)
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
