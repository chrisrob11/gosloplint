package clone

import (
	"cmp"
	"go/token"
)

// Span is one side of a clone: lines Start to End of declaration Decl, the
// first statement at Pos.
type Span struct {
	File, Decl string
	Pos        token.Pos
	Start, End int
}

// before reports whether s is the earlier occurrence: by file, then line.
func (s Span) before(t Span) bool { return s.compare(t) < 0 }

func (s Span) compare(t Span) int {
	return cmp.Or(cmp.Compare(s.File, t.File), cmp.Compare(s.Start, t.Start), cmp.Compare(s.End, t.End))
}

// widen is the smallest span covering s and t, both in one declaration.
func (s Span) widen(t Span) Span {
	if t.Start < s.Start {
		s.Start, s.Pos = t.Start, t.Pos
	}
	s.End = max(s.End, t.End)
	return s
}
