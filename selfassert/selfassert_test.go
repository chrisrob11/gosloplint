package selfassert_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/chrisrob11/gosloplint/selfassert"
)

func TestAnalyzer(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), selfassert.Analyzer, "a")
}
