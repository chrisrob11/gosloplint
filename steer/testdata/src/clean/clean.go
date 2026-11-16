// Package clean returns only concrete types. Its comments say any,
// interface{}, io.Reader and Shape, and none of them is a result.
package clean

// Point is concrete. It could have returned any or an interface; it does not.
type Point struct{ X int }

// New returns a *Point, not an interface{} or any.
func New() *Point { return &Point{} }

// Fail returns error, the one interface allowed.
func Fail() error { return nil }

// Use accepts an interface: allowed.
func Use(v any, f func(any)) { f(v) }
