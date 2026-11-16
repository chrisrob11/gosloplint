// Package escape exempts blocks with a reasoned directive.
package escape

// A is exempt: the directive gives its reason.
// line 1
// line 2
// line 3
// line 4
// line 5
// line 6
// line 7
// line 8
// line 9
// line 10
// line 11
// line 12
// line 13
// line 14
// line 15
// line 16
// line 17
// line 18
// line 19
// line 20
//
//gosloplint:comment-ok the table below needs the room
func A() {}

// line 1 // want `length: func comment is 10 lines`
// line 2
// line 3
// line 4
// line 5
// line 6
// line 7
// line 8
// line 9
// a mention of //gosloplint:comment-ok mid-line is prose
func B() {}

// line 1 // want `length: func comment is 11 lines`
// line 2
// line 3
// line 4
// line 5
// line 6
// line 7
// line 8
// line 9
//
//	gosloplint:comment-ok a tab makes it prose
func C() {}

// D's directive lines, and gofmt's blank line before them, are not counted.
// line 1
// line 2
// line 3
// line 4
// line 5
// line 6
// line 7
//
//go:noinline
//lint:ignore X because
func D() {}

// E's blank lines between prose do count. // want `length: func comment is 9 lines`
//
// line 1
// line 2
// line 3
//
// line 1
// line 2
// line 3
//
//go:noinline
func E() {}
