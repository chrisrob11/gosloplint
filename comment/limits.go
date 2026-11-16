package comment

import "fmt"

// Kind is what a comment block is attached to; each kind has its own limit.
type Kind string

// The kinds, in the order Limits lists them.
const (
	// KindPackage is the package doc, on the package clause.
	KindPackage Kind = "package"
	// KindFile is a top-level comment that documents nothing: a header
	// above the package clause, or a section comment between declarations.
	KindFile Kind = "file"
	// KindType documents a type declaration.
	KindType Kind = "type"
	// KindFunc documents a func or method declaration.
	KindFunc Kind = "func"
	// KindField documents a struct field or interface method.
	KindField Kind = "field"
	// KindValue documents a const or var declaration.
	KindValue Kind = "value"
	// KindInline is a comment inside a declaration, such as a function body,
	// that documents nothing: one a reader meets mid-code.
	KindInline Kind = "inline"
)

// Kinds is every kind.
var Kinds = []Kind{KindPackage, KindFile, KindType, KindFunc, KindField, KindValue, KindInline}

// Limits is the longest a comment block of each kind may be, in lines. A
// zero field means its default.
type Limits struct {
	Package, File, Type, Func, Field, Value, Inline int
}

// DefaultLimits are the limits a zero field takes. A package doc is read once,
// by someone arriving, so it may be long; an inline comment competes with the
// code beside it, so it is the strictest.
var DefaultLimits = Limits{Package: 60, File: 20, Type: 10, Func: 8, Field: 4, Value: 8, Inline: 3}

// For is the limit for kind k, its default when unset.
func (l Limits) For(k Kind) int {
	if n := l.field(k); n > 0 {
		return n
	}
	return DefaultLimits.field(k)
}

func (l Limits) field(k Kind) int {
	switch k {
	case KindPackage:
		return l.Package
	case KindFile:
		return l.File
	case KindType:
		return l.Type
	case KindFunc:
		return l.Func
	case KindField:
		return l.Field
	case KindValue:
		return l.Value
	}
	return l.Inline
}

func (l Limits) validate() error {
	for _, k := range Kinds {
		if n := l.field(k); n < 0 {
			return fmt.Errorf("limits.%s: %d is negative", k, n)
		}
	}
	return nil
}
