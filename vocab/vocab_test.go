package vocab_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/chrisrob11/gosloplint/vocab"
)

func TestNoWordsReportsNothing(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), vocab.Analyzer, "unset")
}

func TestWords(t *testing.T) {
	if err := vocab.Analyzer.Flags.Set("words", "age, legacy client"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := vocab.Analyzer.Flags.Set("words", ""); err != nil {
			t.Error(err)
		}
	})
	analysistest.Run(t, analysistest.TestData(), vocab.Analyzer, "a")
}

func TestNewWithoutWordsReportsNothing(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), vocab.New(vocab.Settings{}), "unset")
}

func TestNewWords(t *testing.T) {
	a := vocab.New(vocab.Settings{Words: []string{"age", "legacy client"}})
	analysistest.Run(t, analysistest.TestData(), a, "a")
}
