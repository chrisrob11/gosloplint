// Package listflag is a flag.Value holding a comma-separated list, for the
// analyzers whose settings are lists.
package listflag

import "strings"

// List is a comma-separated flag value. Set replaces the list; blank entries
// are dropped and the rest trimmed.
type List struct {
	Items *[]string
}

func (l List) String() string {
	if l.Items == nil {
		return ""
	}
	return strings.Join(*l.Items, ",")
}

// Set replaces the list with the comma-separated entries of v.
func (l List) Set(v string) error {
	*l.Items = nil
	for _, w := range strings.Split(v, ",") {
		if w = strings.TrimSpace(w); w != "" {
			*l.Items = append(*l.Items, w)
		}
	}
	return nil
}
