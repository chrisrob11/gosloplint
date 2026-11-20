// Package warnsign holds warning markers, one per declaration and more.
package warnsign

// One has a single marker, which is what one is for.
//
// WARNING: the caller must hold mu.
func One() {}

// Two marks its doc and its body.
//
// NOTE: n is never negative.
func Two(n int) int {
	// ⚠ the slice aliases the input. // want `warnsign: "⚠" is warning marker 2 in Two: a marker on every comment marks none`
	// **IMPORTANT:** keep this sorted. // want `"IMPORTANT:" is warning marker 3 in Two`
	return n
}

// T has a marker on a field and on a method doc of its own declaration.
type T struct {
	// Note: zero means unlimited.
	Limit int
	// CAUTION: not safe for concurrent use. // want `"CAUTION:" is warning marker 2 in T`
	mu int
}

// Reset is its own declaration, so its one marker is its first.
//
// NB: Reset keeps the limit.
func (t *T) Reset() { t.mu = 0 }

// Grow has a marker on its doc, a second marker in its body.
//
// Important: the receiver grows.
func (t T) Grow() {
	// Warning: t is a copy. // want `"Warning:" is warning marker 2 in T.Grow`
	_ = t
}

const (
	// IMPORTANT: a is first.
	a = 1
	// NOTE: b follows a. // want `"NOTE:" is warning marker 2 in a`
	b = 2
)

// ⚠ a section comment between declarations.
// ⚠ and its second line. // want `"⚠" is warning marker 2 in this comment block`

// None mentions a note, notes: and a warning without a marker, and the
// word NOTE in the middle of a line: NOTE: is not at its start.
func None() {}

var _, _ = a, b
