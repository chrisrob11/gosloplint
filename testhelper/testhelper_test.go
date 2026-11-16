package testhelper_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/chrisrob11/gosloplint/testhelper"
)

func TestAnalyzer(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), testhelper.Analyzer, "a")
}
