package emptyskip_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/chrisrob11/gosloplint/emptyskip"
)

func TestAnalyzer(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), emptyskip.Analyzer, "a")
}
