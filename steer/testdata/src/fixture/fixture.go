// Package fixture plants every form the return checks report, and the forms
// they must leave alone.
package fixture

import (
	"io"
	"iter"
	"sort"

	"example.com/other"
)

// Shape is a package interface with two implementations.
type Shape interface{ Area() int }

// Square is one Shape.
type Square struct{}

// Area implements Shape.
func (Square) Area() int { return 1 }

// Circle is the other.
type Circle struct{}

// Area implements Shape.
func (*Circle) Area() int { return 2 }

// Any says any and Shape in its doc; only its result counts.
func Any() any { return 1 } // want `returns-any: Any returns any: every caller must type-assert`

func Empty() interface{} { return nil } // want `returns-any: Empty returns interface\{\}`

func AnySlice() []any { return nil } // want `returns-any: AnySlice returns \[\]any`

func AnyMap() map[string]any { return nil } // want `returns-any: AnyMap returns map\[string\]any`

func Pick(b bool) Shape { // want `returns-interface: Pick returns Shape, an interface type: accept interfaces, return structs`
	if b {
		return Square{}
	}
	return &Circle{}
}

func Always() Shape { // want `returns-interface: Always` `returns-one-concrete: Always returns the interface Shape, but every return statement returns \*Circle: return \*Circle\.`
	return &Circle{}
}

func Shapes() []Shape { return nil } // want `returns-interface: Shapes returns \[\]Shape, which holds an interface`

func Grid() [][2]Shape { return nil } // want `returns-interface: Grid`

// The standard library's interfaces are its currency: io.Reader is not
// reported, unless one concrete type is all it hides.
func IOReader() io.Reader { return nil }

type rd struct{}

func (rd) Read([]byte) (int, error) { return 0, nil }

func StdOne() io.Reader { return rd{} } // want `returns-one-concrete: StdOne returns the interface io.Reader, but every return statement returns rd: return rd\.`

// A sealed interface is a sum type: only its package implements it.
func OtherSealed() other.Sealed { return nil }

func SI() sort.Interface { return nil }

func Literal() interface{ Area() int } { return nil } // want `returns-interface: Literal`

func Err() error { return nil }

func Concrete() *Circle { return &Circle{} }

func Other() other.Node { return other.NewLeaf() } // want `returns-interface: Other` `returns-one-concrete: Other .* return \*other.Leaf\.`

func OtherAlias() other.Alias { return nil } // want `returns-interface: OtherAlias`

func OtherDefined() other.Defined { return nil } // want `returns-interface: OtherDefined`

func OtherErr() other.Err { return nil }

func OtherList() other.Shapes { return nil }

// Holder holds a func-typed field.
type Holder struct {
	Make func() Shape // want `returns-interface: Holder.Make returns Shape`
}

// Maker is a named func type.
type Maker func() Shape // want `returns-interface: Maker returns Shape`

// Factory forces its implementations to return an interface.
type Factory interface {
	Build() Shape // want `returns-interface: Factory.Build returns Shape`
}

func Callback(fn func() Shape) { _ = fn } // want `returns-interface: Callback.fn returns Shape`

func Closure() {
	_ = func() Shape { return Square{} } // want `returns-interface: Closure.func` `returns-one-concrete: Closure.func .* return Square\.`
}

func Curried() func() Shape { return nil } // want `returns-interface: Curried.result0`

func Col(func() Shape, int) func() Shape { return nil } // want `returns-interface: Col.param0` `returns-interface: Col.result0`

var vA, vB = func() Shape { return Square{} }, func() Shape { return &Circle{} } // want `returns-interface: vA.func` `returns-one-concrete: vA.func .* return Square\.` `returns-interface: vB.func` `returns-one-concrete: vB.func .* return \*Circle\.`

func Ch() chan Shape { return nil } // want `returns-interface: Ch`

func ChAny() <-chan any { return nil } // want `returns-any: ChAny`

func PtrAny() *any { return nil } // want `returns-any: PtrAny`

func PtrI() *Shape { return nil } // want `returns-interface: PtrI`

func Seq() iter.Seq[Shape] { return nil } // want `returns-interface: Seq`

func Seq2() iter.Seq2[int, any] { return nil } // want `returns-any: Seq2`

func SeqInt() iter.Seq[int] { return nil }

func Generic[T Shape]() T {
	var t T
	return t
}

// Box is generic over its element.
type Box[T any] struct{ v T }

// Value returns the element.
func (b Box[T]) Value() T { return b.v }

func NilOnly() Shape { return nil } // want `returns-interface: NilOnly`

func Mixed() (Shape, error) { return Square{}, nil } // want `returns-interface: Mixed` `returns-one-concrete: Mixed .* return Square\.`

func Conv() Shape { return Square(Square{}) } // want `returns-interface: Conv` `returns-one-concrete: Conv .* return Square\.`

func FromNew() Shape { return new(Circle) } // want `returns-interface: FromNew` `returns-one-concrete: FromNew .* return \*Circle\.`

func ViaCall() Shape { return makeSquare() } // want `returns-interface: ViaCall` `returns-one-concrete: ViaCall .* return Square\.`

func makeSquare() Square { return Square{} }

func ViaVar() Shape { // want `returns-interface: ViaVar` `returns-one-concrete: ViaVar .* return \*Circle\.`
	c := &Circle{}
	return c
}

func Passed(s Shape) Shape { return s } // want `returns-interface: Passed`

func Bare() (s Shape) { // want `returns-interface: Bare`
	s = Square{}
	return
}

func Both() (Shape, Shape) { return Square{}, Square{} } // want `returns-interface: Both` `returns-interface: Both`

func Tuple() (Shape, error) { return pair() } // want `returns-interface: Tuple`

func pair() (*Circle, error) { return nil, nil }

// Store has one implementation, so its constructor should return it.
type Store interface{ Get() int }

type memStore struct{}

func (*memStore) Get() int { return 0 }

func NewStore() Store { return &memStore{} } // want `returns-interface: NewStore` `returns-one-concrete: NewStore` `sole-implementation: NewStore returns the interface Store, which only \*memStore implements in this package: return \*memStore`

func NewShape() Shape { return Square{} } // want `returns-interface: NewShape` `returns-one-concrete: NewShape .* return Square\.`

// Walk recurses, but no recursion package names fixture.
func Walk(n int) int {
	if n == 0 {
		return 0
	}
	return Walk(n - 1)
}
