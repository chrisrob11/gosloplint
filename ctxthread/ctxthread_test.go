package ctxthread_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/chrisrob11/gosloplint/ctxthread"
)

func TestAnalyzer(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), ctxthread.Analyzer, "a", "mainpkg")
}
