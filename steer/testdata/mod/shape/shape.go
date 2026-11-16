// Package shape is in the same module as flags, though its path's third
// element differs.
package shape

// Shape is an interface of this module.
type Shape interface{ Area() int }

// Square is one Shape.
type Square struct{}

// Area implements Shape.
func (Square) Area() int { return 1 }

// Circle is another.
type Circle struct{}

// Area implements Shape.
func (Circle) Area() int { return 2 }
