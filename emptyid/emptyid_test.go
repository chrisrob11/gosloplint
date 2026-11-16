package emptyid_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/chrisrob11/gosloplint/emptyid"
)

func TestDefaultSuffixes(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), emptyid.Analyzer, "a")
}

func TestSuffixesFlag(t *testing.T) {
	setFlag(t, "ref,Prefix")
	analysistest.Run(t, analysistest.TestData(), emptyid.Analyzer, "custom")
}

func setFlag(t *testing.T, v string) {
	t.Helper()
	if err := emptyid.Analyzer.Flags.Set("suffixes", v); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := emptyid.Analyzer.Flags.Set("suffixes", emptyid.DefaultSuffixes); err != nil {
			t.Error(err)
		}
	})
}
