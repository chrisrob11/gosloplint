// line 1 // want `length: file comment is 21 lines, limit 20: shorten it, or exempt the block with //gosloplint:comment-ok <reason>`
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
// line 21

// line 1 // want `length: package comment is 61 lines, limit 60`
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
// line 21
// line 22
// line 23
// line 24
// line 25
// line 26
// line 27
// line 28
// line 29
// line 30
// line 31
// line 32
// line 33
// line 34
// line 35
// line 36
// line 37
// line 38
// line 39
// line 40
// line 41
// line 42
// line 43
// line 44
// line 45
// line 46
// line 47
// line 48
// line 49
// line 50
// line 51
// line 52
// line 53
// line 54
// line 55
// line 56
// line 57
// line 58
// line 59
// line 60
// line 61
package kinds

// line 1 // want `length: type comment is 11 lines, limit 10`
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
type T struct {
	// line 1 // want `length: field comment is 5 lines, limit 4`
	// line 2
	// line 3
	// line 4
	// line 5
	F int
	// line 1
	// line 2
	// line 3
	// line 4
	G int
}

// line 1 // want `length: func comment is 9 lines, limit 8`
// line 2
// line 3
// line 4
// line 5
// line 6
// line 7
// line 8
// line 9
func F() {
	// line 1 // want `length: inline comment is 4 lines, limit 3`
	// line 2
	// line 3
	// line 4
	_ = 1
	// line 1
	// line 2
	// line 3
	_ = 2
}

// line 1 // want `length: value comment is 9 lines, limit 8`
// line 2
// line 3
// line 4
// line 5
// line 6
// line 7
// line 8
// line 9
const C = 1

var (
	// line 1 // want `length: value comment is 9 lines, limit 8`
	// line 2
	// line 3
	// line 4
	// line 5
	// line 6
	// line 7
	// line 8
	// line 9
	V = 1
)

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
type U int

// line 1
// line 2
// line 3
// line 4
// line 5
// line 6
// line 7
// line 8
func G() {}

// line 1
// line 2
// line 3
// line 4
// line 5
// line 6
// line 7
// line 8
var W = 1

type I interface {
	// line 1 // want `length: field comment is 5 lines, limit 4`
	// line 2
	// line 3
	// line 4
	// line 5
	M()
}

// A comment separated from a declaration by a blank line documents // want `length: file comment is 22 lines, limit 20`
// nothing, so at the top level it is a file comment, held to 20 lines.
// more 0
// more 1
// more 2
// more 3
// more 4
// more 5
// more 6
// more 7
// more 8
// more 9
// more 10
// more 11
// more 12
// more 13
// more 14
// more 15
// more 16
// more 17
// more 18
// over the limit

func H() {}
