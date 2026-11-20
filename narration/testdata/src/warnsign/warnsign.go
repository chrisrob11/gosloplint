// Package warnsign holds warning markers, one per declaration and more.
package warnsign

import "fmt"

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

// T has a marker on its doc, one on each of two fields, and two on a third:
// each field is its own unit, apart from the type's doc.
//
// NOTE: T is not copied.
type T struct {
	// Note: zero means unlimited.
	Limit int
	// CAUTION: not safe for concurrent use.
	mu int
	// NOTE: zero is off.
	//
	// NOTE: one is on. // want `"NOTE:" is warning marker 2 in T.Mode`
	Mode  int // ⚠ never two. // want `"⚠" is warning marker 3 in T.Mode`
	Inner struct {
		// NOTE: the inner field is its own unit.
		Deep int
		// NOTE: and so is its sibling.
		Deeper int // NOTE: but not its line comment. // want `"NOTE:" is warning marker 2 in T.Inner.Deeper`
	}
}

// Store is an interface whose methods are each their own unit.
type Store interface {
	// Get reads a row.
	//
	// NOTE: Get may return a stale row.
	Get() int
	// Put writes a row.
	// Caution: Put does not sync.
	Put(int)
	// Del removes a row.
	// NOTE: Del is idempotent.
	// NOTE: and slow. // want `"NOTE:" is warning marker 2 in Store.Del`
	Del()
	// IMPORTANT: an embedded interface is a unit too.
	fmt.Stringer
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
	// NOTE: b follows a, its own unit.
	b = 2
	// NOTE: c is last.
	c = 3 // NOTE: and marked twice. // want `"NOTE:" is warning marker 2 in c`
)

// The group has a marker on its doc, apart from its specs.
//
// NOTE: these types are kept small.
type (
	// U holds one marker.
	//
	// NOTE: U is a counter.
	U int

	// NOTE: a comment between specs belongs to the group. // want `"NOTE:" is warning marker 2 in this type group`

	// V holds one too.
	//
	// Warning: V wraps.
	V uint8
)

// ⚠ a section comment between declarations.
// ⚠ and its second line. // want `"⚠" is warning marker 2 in this comment block`

// None mentions a note, notes: and a warning without a marker, and the
// word NOTE in the middle of a line: NOTE: is not at its start.
func None() {}

var _, _, _ = a, b, c
