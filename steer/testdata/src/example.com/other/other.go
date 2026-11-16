// Package other declares the interfaces fixture imports.
package other

// Node is an interface another package declares.
type Node interface{ Kind() int }

// Alias is an alias of Node.
type Alias = Node

// Defined is a type defined from Node.
type Defined Node

// Err is a type defined from error, so it is error.
type Err error

// Shapes is a named slice of interfaces: a concrete type of its own.
type Shapes []Node

// Leaf implements Node.
type Leaf struct{}

// Kind implements Node.
func (Leaf) Kind() int { return 0 }

// NewLeaf returns a concrete leaf.
func NewLeaf() *Leaf { return &Leaf{} }

// Sealed has an unexported method, so only this package implements it.
type Sealed interface{ sealed() }
