package domainword_test

import (
	"testing"

	"domainword"
)

func TestHelperConfig(t *testing.T) {
	if (domainword.HelperConfig{}).Ours() {
		t.Error("ours")
	}
}
