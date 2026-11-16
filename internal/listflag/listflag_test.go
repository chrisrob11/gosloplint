package listflag_test

import (
	"slices"
	"testing"

	"github.com/chrisrob11/gosloplint/internal/listflag"
)

func TestSetReplacesAndTrims(t *testing.T) {
	items := []string{"old"}
	l := listflag.List{Items: &items}
	if err := l.Set(" a, ,b ,"); err != nil {
		t.Fatal(err)
	}
	if want := []string{"a", "b"}; !slices.Equal(items, want) {
		t.Errorf("items = %q, want %q", items, want)
	}
	if got := l.String(); got != "a,b" {
		t.Errorf("String = %q", got)
	}
	if got := (listflag.List{}).String(); got != "" {
		t.Errorf("zero String = %q", got)
	}
}
